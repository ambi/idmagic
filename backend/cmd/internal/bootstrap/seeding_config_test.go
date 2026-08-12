package bootstrap

import (
	"strings"
	"testing"
)

// TestLoadSeedConfigRejectsUnknownProfileAndEnvironment covers
// REQ-SYSTEM-016: seed selectors are startup configuration, so unknown
// values must join the aggregated error before Assemble or seed application.
func TestLoadSeedConfigRejectsUnknownProfileAndEnvironment(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"SEED_PROFILE":     "demo-ish",
		"SEED_ENVIRONMENT": "somewhere",
	}))
	LoadSeedConfig(l)
	err := l.Err()
	if err == nil {
		t.Fatal("expected unknown seed selectors to fail startup validation")
	}
	for _, key := range []string{"SEED_PROFILE", "SEED_ENVIRONMENT"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("aggregated error %q does not mention %s", err.Error(), key)
		}
	}
}

func TestLoadSeedConfigRequiresEnvironmentForConfiguredProfile(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"SEED_PROFILE": "development"}))
	LoadSeedConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "SEED_ENVIRONMENT") {
		t.Fatalf("err = %v, want a missing SEED_ENVIRONMENT error", err)
	}
}
