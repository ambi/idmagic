package bootstrap

import (
	"bytes"
	"testing"
)

func TestLoadPaginationCursorSecretUsesExplicitEnv(t *testing.T) {
	t.Setenv("PAGINATION_CURSOR_SECRET", "explicit-secret")
	got := LoadPaginationCursorSecret()
	if string(got) != "explicit-secret" {
		t.Fatalf("got %q, want %q", got, "explicit-secret")
	}
}

func TestLoadPaginationCursorSecretFallsBackToRandom(t *testing.T) {
	t.Setenv("PAGINATION_CURSOR_SECRET", "")
	a := LoadPaginationCursorSecret()
	b := LoadPaginationCursorSecret()
	if len(a) == 0 {
		t.Fatal("expected a non-empty generated secret")
	}
	if bytes.Equal(a, b) {
		t.Fatal("expected two independently generated fallback secrets to differ")
	}
}
