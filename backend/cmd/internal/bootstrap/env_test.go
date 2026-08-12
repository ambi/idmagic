package bootstrap

import (
	"testing"
	"time"
)

func TestEnvIntRejectsNegativeValues(t *testing.T) {
	t.Setenv("BOOTSTRAP_INT", "-1")

	if got := EnvInt("BOOTSTRAP_INT", 20); got != 20 {
		t.Fatalf("EnvInt negative value = %d, want fallback 20", got)
	}
}

func TestEnvDurationRejectsNonPositiveValues(t *testing.T) {
	t.Setenv("BOOTSTRAP_DURATION", "0s")

	if got := EnvDuration("BOOTSTRAP_DURATION", 5*time.Second); got != 5*time.Second {
		t.Fatalf("EnvDuration non-positive value = %v, want fallback 5s", got)
	}
}
