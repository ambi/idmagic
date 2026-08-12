package bootstrap

import (
	"strings"
	"time"

	emailSMTP "github.com/ambi/idmagic/backend/shared/notification/email_smtp"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/resilience"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// SharedConfig is the startup configuration read by both the API and worker
// processes (everything reachable from Assemble/AssembleNotification), so a
// typo in a key they share is caught the same way regardless of which
// process starts first. It is parsed once via LoadSharedConfig and passed
// down instead of each adapter re-reading env (wi-103).
type SharedConfig struct {
	Persistence   string
	Observability string

	AuthZEN    string
	AuthZENURL string

	WebAuthnRPID          string
	WebAuthnRPOrigins     []string
	WebAuthnRPDisplayName string

	DatabaseURL Secret
	DB          postgres.DBConfig
	DBBreaker   resilience.Settings

	KeyProvider       string
	VaultAddr         string
	VaultToken        Secret
	VaultTransitMount string
	VaultKeyPrefix    string

	DataKeyProvider      string
	OpenBaoAddr          string
	OpenBaoToken         Secret
	OpenBaoTransitMount  string
	OpenBaoDataKeyPrefix string

	EmailSender string
	SMTPHost    string
	SMTPFrom    string
	SMTPTLSMode emailSMTP.SMTPTLSMode
	SMTPPort    int
	SMTPUser    string
	SMTPPass    Secret
	SMTPHelo    string
	SMTPTimeout time.Duration

	DefaultLocale string

	BreachedPasswordChecker string
}

// LoadSharedConfig parses every SharedConfig field from l and records every
// missing-required-value, malformed-value, and conflicting-combination
// problem on l instead of stopping at the first one, so a caller that
// checks l.Err() once after every Load*Config call sees the full set
// (REQ-SYSTEM-016). It performs no I/O: adapters are constructed from the
// returned struct only after the caller has confirmed l.Err() == nil.
func LoadSharedConfig(l *ConfigLoader) SharedConfig {
	var cfg SharedConfig

	cfg.Persistence = l.Enum("PERSISTENCE", "memory", "memory", "postgres")
	cfg.Observability = l.EnumFold("OBSERVABILITY", "noop", "noop", "otel")

	cfg.AuthZEN = l.Enum("AUTHZEN", "local", "local", "remote")
	cfg.AuthZENURL = l.URL("AUTHZEN_URL", "")
	l.RequiredWhen("AUTHZEN_URL", "AUTHZEN=remote")
	if cfg.AuthZEN == "remote" {
		cfg.AuthZENURL = l.RequiredURL("AUTHZEN_URL")
	}

	cfg.WebAuthnRPID = l.String("WEBAUTHN_RP_ID", "")
	cfg.WebAuthnRPOrigins = l.StringList("WEBAUTHN_RP_ORIGINS", nil)
	l.RequiredWhen("WEBAUTHN_RP_ORIGINS", "WEBAUTHN_RP_ID is set")
	cfg.WebAuthnRPDisplayName = l.String("WEBAUTHN_RP_DISPLAY_NAME", "idmagic")
	if cfg.WebAuthnRPID != "" {
		l.Require("WEBAUTHN_RP_ORIGINS", len(cfg.WebAuthnRPOrigins) > 0,
			"is required when WEBAUTHN_RP_ID is set")
	}

	if cfg.Persistence == "postgres" {
		cfg.DatabaseURL = l.RequiredSecret("DATABASE_URL")
	} else {
		cfg.DatabaseURL = l.Secret("DATABASE_URL")
	}
	l.RequiredWhen("DATABASE_URL", "PERSISTENCE=postgres")
	cfg.DB = postgres.DBConfig{
		MaxConns:        l.NonNegativeInt32("DB_MAX_CONNS", 20),
		MinConns:        l.NonNegativeInt32("DB_MIN_CONNS", 2),
		MaxConnIdleTime: l.PositiveDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Second),
		MaxConnLifetime: l.PositiveDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour),
		ConnectTimeout:  l.PositiveDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
		QueryTimeout:    l.PositiveDuration("DB_QUERY_TIMEOUT", 5*time.Second),
	}
	cfg.DBBreaker = resilience.Settings{
		Name:             "postgres",
		FailureThreshold: l.FloatRange("DB_BREAKER_FAILURE_THRESHOLD", 0.5, 0, 1),
		Cooldown:         l.PositiveDuration("DB_BREAKER_COOLDOWN", 30*time.Second),
		MinRequests:      l.NonNegativeUint32("DB_BREAKER_MIN_REQUESTS", 10),
	}

	cfg.KeyProvider = l.OptionalEnumFold("KEY_PROVIDER", "local", "vault")
	cfg.VaultAddr = l.String("VAULT_ADDR", "")
	cfg.VaultToken = l.Secret("VAULT_TOKEN")
	l.RequiredWhen("VAULT_ADDR", "KEY_PROVIDER=vault")
	l.RequiredWhen("VAULT_TOKEN", "KEY_PROVIDER=vault")
	cfg.VaultTransitMount = l.String("VAULT_TRANSIT_MOUNT", "")
	cfg.VaultKeyPrefix = l.String("VAULT_KEY_PREFIX", "")
	if cfg.KeyProvider == "vault" {
		l.Require("VAULT_ADDR", cfg.VaultAddr != "", "is required when KEY_PROVIDER=vault")
		l.Require("VAULT_TOKEN", !cfg.VaultToken.Empty(), "is required when KEY_PROVIDER=vault")
	}

	cfg.DataKeyProvider = l.OptionalEnumFold("DATA_KEY_PROVIDER", "openbao")
	cfg.OpenBaoAddr = l.String("OPENBAO_ADDR", "")
	cfg.OpenBaoToken = l.Secret("OPENBAO_TOKEN")
	l.RequiredWhen("OPENBAO_ADDR", "DATA_KEY_PROVIDER=openbao")
	l.RequiredWhen("OPENBAO_TOKEN", "DATA_KEY_PROVIDER=openbao")
	cfg.OpenBaoTransitMount = l.String("OPENBAO_TRANSIT_MOUNT", "")
	cfg.OpenBaoDataKeyPrefix = l.String("OPENBAO_DATA_KEY_PREFIX", "idmagic/datakeys")
	if cfg.DataKeyProvider == "openbao" {
		l.Require("OPENBAO_ADDR", cfg.OpenBaoAddr != "", "is required when DATA_KEY_PROVIDER=openbao")
		l.Require("OPENBAO_TOKEN", !cfg.OpenBaoToken.Empty(), "is required when DATA_KEY_PROVIDER=openbao")
	}

	cfg.EmailSender = strings.ToLower(l.Enum("EMAIL_SENDER", "console", "console", "smtp"))
	loadSMTPConfig(l, &cfg, cfg.EmailSender == "smtp")

	cfg.DefaultLocale = l.String("DEFAULT_LOCALE", "")
	if cfg.DefaultLocale == "" {
		cfg.DefaultLocale = template.FallbackLocale
	} else if !template.LocaleSupported(cfg.DefaultLocale) {
		l.fail("DEFAULT_LOCALE", "must be one of "+strings.Join(template.SupportedLocales(), ", "))
	}

	cfg.BreachedPasswordChecker = strings.ToLower(l.Enum("BREACHED_PASSWORD_CHECKER", "noop", "noop", "hibp"))

	return cfg
}

func loadSMTPConfig(l *ConfigLoader, cfg *SharedConfig, required bool) {
	if required {
		cfg.SMTPHost = l.RequiredString("SMTP_HOST")
		cfg.SMTPFrom = l.RequiredString("SMTP_FROM")
	} else {
		cfg.SMTPHost = l.String("SMTP_HOST", "")
		cfg.SMTPFrom = l.String("SMTP_FROM", "")
	}
	l.RequiredWhen("SMTP_HOST", "EMAIL_SENDER=smtp")
	l.RequiredWhen("SMTP_FROM", "EMAIL_SENDER=smtp")
	mode := strings.ToLower(l.Enum("SMTP_TLS", "starttls", "starttls", "implicit", "none"))
	cfg.SMTPTLSMode = emailSMTP.SMTPTLSMode(mode)

	if port := l.NonNegativeInt("SMTP_PORT", 0); port > 0 {
		cfg.SMTPPort = port
	} else {
		switch cfg.SMTPTLSMode {
		case emailSMTP.SMTPTLSImplicit:
			cfg.SMTPPort = 465
		case emailSMTP.SMTPTLSNone:
			cfg.SMTPPort = 25
		default:
			cfg.SMTPPort = 587
		}
	}
	cfg.SMTPUser = l.String("SMTP_USERNAME", "")
	cfg.SMTPPass = l.Secret("SMTP_PASSWORD")
	cfg.SMTPHelo = l.String("SMTP_HELO", "")
	cfg.SMTPTimeout = time.Duration(l.PositiveInt("SMTP_TIMEOUT_SECONDS", 10)) * time.Second
}
