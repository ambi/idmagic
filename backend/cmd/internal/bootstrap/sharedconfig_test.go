package bootstrap

import (
	"strings"
	"testing"
)

func TestLoadSharedConfigPostgresRequiresDatabaseURL(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"PERSISTENCE": "postgres"}))
	LoadSharedConfig(l)
	err := l.Err()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("err=%v, want a DATABASE_URL required error", err)
	}
}

func TestLoadSharedConfigMemoryDoesNotRequireDatabaseURL(t *testing.T) {
	t.Parallel()
	cfg := loadSharedConfigOrFatal(t, map[string]string{})
	if cfg.Persistence != "memory" {
		t.Fatalf("Persistence = %q, want default memory", cfg.Persistence)
	}
}

func TestLoadSharedConfigRejectsUnknownPersistence(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"PERSISTENCE": "sqlite"}))
	LoadSharedConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "PERSISTENCE") {
		t.Fatalf("err=%v, want unsupported PERSISTENCE error", err)
	}
}

func TestLoadSharedConfigAuthZENRemoteRequiresURL(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"AUTHZEN": "remote"}))
	LoadSharedConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "AUTHZEN_URL") {
		t.Fatalf("err=%v, want an AUTHZEN_URL required error", err)
	}
}

func TestLoadSharedConfigAuthZENRemoteRejectsNonAbsoluteURL(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"AUTHZEN": "remote", "AUTHZEN_URL": "not-a-url"}))
	LoadSharedConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "AUTHZEN_URL") {
		t.Fatalf("err=%v, want an AUTHZEN_URL absolute-URL error", err)
	}
}

func TestLoadSharedConfigWebAuthnRequiresOriginsWhenRPIDSet(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"WEBAUTHN_RP_ID": "idp.example.com"}))
	LoadSharedConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "WEBAUTHN_RP_ORIGINS") {
		t.Fatalf("err=%v, want a WEBAUTHN_RP_ORIGINS required error", err)
	}
}

func TestLoadSharedConfigKeyProviderVaultRequiresAddrAndToken(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"KEY_PROVIDER": "vault"}))
	LoadSharedConfig(l)
	err := l.Err()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "VAULT_ADDR") || !strings.Contains(err.Error(), "VAULT_TOKEN") {
		t.Fatalf("err=%v, want both VAULT_ADDR and VAULT_TOKEN reported together", err)
	}
}

func TestLoadSharedConfigKeyProviderVaultCaseInsensitive(t *testing.T) {
	t.Parallel()
	cfg := loadSharedConfigOrFatal(t, map[string]string{
		"KEY_PROVIDER": "VAULT", "VAULT_ADDR": "https://vault.example.com", "VAULT_TOKEN": "t",
	})
	if cfg.KeyProvider != "vault" {
		t.Fatalf("KeyProvider = %q, want normalized \"vault\"", cfg.KeyProvider)
	}
}

func TestLoadSharedConfigDataKeyProviderOpenBaoRequiresAddrAndToken(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"DATA_KEY_PROVIDER": "openbao"}))
	LoadSharedConfig(l)
	err := l.Err()
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "OPENBAO_ADDR") || !strings.Contains(err.Error(), "OPENBAO_TOKEN") {
		t.Fatalf("err=%v, want both OPENBAO_ADDR and OPENBAO_TOKEN reported together", err)
	}
}

// TestLoadSharedConfigReportsEveryUnrelatedProblemTogether guards
// REQ-SYSTEM-016: independent problems across unrelated fields must all
// surface from one LoadSharedConfig call instead of stopping at the first.
func TestLoadSharedConfigReportsEveryUnrelatedProblemTogether(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"PERSISTENCE":  "postgres",
		"KEY_PROVIDER": "vault",
		"EMAIL_SENDER": "smtp",
	}))
	LoadSharedConfig(l)
	err := l.Err()
	if err == nil {
		t.Fatal("expected an aggregated error")
	}
	for _, key := range []string{"DATABASE_URL", "VAULT_ADDR", "VAULT_TOKEN", "SMTP_HOST", "SMTP_FROM"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("aggregated error %q does not mention %s", err.Error(), key)
		}
	}
}

func TestLoadSharedConfigSecretsAreRedactedInLoaderErrors(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"PERSISTENCE":  "postgres",
		"DATABASE_URL": "postgres://user:hunter2@db.internal:5432/idmagic",
	}))
	cfg := LoadSharedConfig(l)
	if err := l.Err(); err != nil {
		t.Fatalf("LoadSharedConfig: %v", err)
	}
	if cfg.DatabaseURL.String() == "postgres://user:hunter2@db.internal:5432/idmagic" ||
		strings.Contains(cfg.DatabaseURL.String(), "hunter2") {
		t.Fatalf("DatabaseURL.String() leaked the DSN: %q", cfg.DatabaseURL.String())
	}
	if cfg.DatabaseURL.Value() != "postgres://user:hunter2@db.internal:5432/idmagic" {
		t.Fatal("DatabaseURL.Value() must still return the real DSN for the adapter that opens the pool")
	}
}
