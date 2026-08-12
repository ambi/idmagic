package bootstrap

import (
	"bytes"
	"fmt"
	"testing"
)

func TestPaginationCursorSecretUsesConfiguredValue(t *testing.T) {
	t.Parallel()
	got := PaginationCursorSecret(NewSecret("explicit-secret"))
	if string(got) != "explicit-secret" {
		t.Fatalf("got %q, want %q", got, "explicit-secret")
	}
}

func TestPaginationCursorSecretFallsBackToRandom(t *testing.T) {
	t.Parallel()
	a := PaginationCursorSecret(NewSecret(""))
	b := PaginationCursorSecret(NewSecret(""))
	if len(a) == 0 {
		t.Fatal("expected a non-empty generated secret")
	}
	if bytes.Equal(a, b) {
		t.Fatal("expected two independently generated fallback secrets to differ")
	}
}

// TestPaginationCursorSecretIsRedactedWhenLogged guards REQ-SYSTEM-016's
// secret rule for the one API secret that has no adapter of its own: the
// value must never reach a log line through the config struct.
func TestPaginationCursorSecretIsRedactedWhenLogged(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"PAGINATION_CURSOR_SECRET": "shared-hmac-key"}))
	cfg := LoadAPIConfig(l)
	if rendered := fmt.Sprintf("%+v", cfg.PaginationCursorSecret); rendered != "[REDACTED]" {
		t.Fatalf("rendered = %q, want [REDACTED]", rendered)
	}
	if cfg.PaginationCursorSecret.Value() != "shared-hmac-key" {
		t.Fatal("Value() must still expose the configured secret to its consumer")
	}
}
