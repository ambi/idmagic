package bootstrap

import (
	"strings"
	"testing"

	emailConsole "github.com/ambi/idmagic/backend/shared/notification/email_console"
	emailSMTP "github.com/ambi/idmagic/backend/shared/notification/email_smtp"
)

func loadSharedConfigOrFatal(t *testing.T, env map[string]string) SharedConfig {
	t.Helper()
	l := NewConfigLoader(stubEnv(env))
	cfg := LoadSharedConfig(l)
	if err := l.Err(); err != nil {
		t.Fatalf("LoadSharedConfig: %v", err)
	}
	return cfg
}

func TestResolveEmailSenderDefaultsToConsole(t *testing.T) {
	t.Parallel()
	cfg := loadSharedConfigOrFatal(t, map[string]string{})
	sender := ResolveEmailSender(cfg)
	if _, ok := sender.(emailConsole.ConsoleEmailSender); !ok {
		t.Fatalf("default sender = %T, want ConsoleEmailSender", sender)
	}
}

func TestLoadSharedConfigSMTPRequiresHostAndFrom(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"missing host", map[string]string{"EMAIL_SENDER": "smtp", "SMTP_FROM": "a@b"}, "SMTP_HOST"},
		{"missing from", map[string]string{"EMAIL_SENDER": "smtp", "SMTP_HOST": "h"}, "SMTP_FROM"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := NewConfigLoader(stubEnv(tc.env))
			LoadSharedConfig(l)
			err := l.Err()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestResolveEmailSenderSMTPBuildsAdapter(t *testing.T) {
	t.Parallel()
	cfg := loadSharedConfigOrFatal(t, map[string]string{
		"EMAIL_SENDER":  "smtp",
		"SMTP_HOST":     "smtp.example.com",
		"SMTP_FROM":     "noreply@example.com",
		"SMTP_USERNAME": "apikey",
		"SMTP_PASSWORD": "s3cret",
	})
	sender := ResolveEmailSender(cfg)
	if _, ok := sender.(*emailSMTP.SMTPEmailSender); !ok {
		t.Fatalf("smtp sender = %T, want *SMTPEmailSender", sender)
	}
}

func TestLoadSharedConfigRejectsUnknownEmailSender(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"EMAIL_SENDER": "carrier-pigeon"}))
	LoadSharedConfig(l)
	err := l.Err()
	if err == nil || !strings.Contains(err.Error(), "EMAIL_SENDER") {
		t.Fatalf("err=%v, want unsupported EMAIL_SENDER error", err)
	}
}

func TestLoadSharedConfigSMTPPortDefaultsPerMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		mode emailSMTP.SMTPTLSMode
		want int
	}{
		{emailSMTP.SMTPTLSSTARTTLS, 587},
		{emailSMTP.SMTPTLSImplicit, 465},
		{emailSMTP.SMTPTLSNone, 25},
	}
	for _, tc := range cases {
		cfg := loadSharedConfigOrFatal(t, map[string]string{
			"EMAIL_SENDER": "smtp", "SMTP_HOST": "h", "SMTP_FROM": "a@b", "SMTP_TLS": string(tc.mode),
		})
		if cfg.SMTPPort != tc.want {
			t.Errorf("mode=%s port=%d, want %d", tc.mode, cfg.SMTPPort, tc.want)
		}
	}

	cfg := loadSharedConfigOrFatal(t, map[string]string{
		"EMAIL_SENDER": "smtp", "SMTP_HOST": "h", "SMTP_FROM": "a@b", "SMTP_PORT": "2525",
	})
	if cfg.SMTPPort != 2525 {
		t.Errorf("explicit port override = %d, want 2525", cfg.SMTPPort)
	}
}

func TestLoadSharedConfigRejectsMalformedSMTPPort(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"EMAIL_SENDER": "smtp", "SMTP_HOST": "h", "SMTP_FROM": "a@b", "SMTP_PORT": "not-a-port",
	}))
	LoadSharedConfig(l)
	err := l.Err()
	if err == nil || !strings.Contains(err.Error(), "SMTP_PORT") {
		t.Fatalf("err=%v, want malformed SMTP_PORT error", err)
	}
}

func stubEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
