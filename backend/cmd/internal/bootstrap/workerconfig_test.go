package bootstrap

import (
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
)

func TestLoadWorkerConfigDefaults(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(nil))
	cfg := LoadWorkerConfig(l)
	if err := l.Err(); err != nil {
		t.Fatalf("LoadWorkerConfig: %v", err)
	}
	if cfg.OTelServiceName != "idmagic-worker" {
		t.Errorf("OTelServiceName = %q", cfg.OTelServiceName)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.PollInterval != 2*time.Second {
		t.Errorf("PollInterval = %s, want 2s", cfg.PollInterval)
	}
	if cfg.LeaseDuration != 5*time.Minute {
		t.Errorf("LeaseDuration = %s, want 5m", cfg.LeaseDuration)
	}
	if cfg.EphemeralSweepInterval != 60*time.Second {
		t.Errorf("EphemeralSweepInterval = %s, want 60s", cfg.EphemeralSweepInterval)
	}
	if cfg.SharedSignalsDeliveryInterval != 5*time.Second {
		t.Errorf("SharedSignalsDeliveryInterval = %s, want 5s", cfg.SharedSignalsDeliveryInterval)
	}
	if cfg.DrainGracePeriod != 5*time.Second {
		t.Errorf("DrainGracePeriod = %s, want 5s", cfg.DrainGracePeriod)
	}
	// Compat mode: an unset JOB_WORKER_LANES keeps one process on every lane.
	if len(cfg.Lanes) != 3 {
		t.Fatalf("Lanes = %v, want every lane", cfg.Lanes)
	}
	for _, lane := range cfg.Lanes {
		if cfg.Concurrency[lane] != 4 {
			t.Errorf("Concurrency[%s] = %d, want 4", lane, cfg.Concurrency[lane])
		}
	}
}

// TestLoadWorkerConfigRejectsMalformedIntervals covers REQ-SYSTEM-016 for the
// worker process: JOB_* durations used to fall back silently on a typo, so a
// deployment could run with the default poll interval while its manifest said
// otherwise.
func TestLoadWorkerConfigRejectsMalformedIntervals(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"JOB_POLL_INTERVAL":                "2",
		"JOB_LEASE_DURATION":               "-5m",
		"EPHEMERAL_SWEEP_INTERVAL":         "soon",
		"SHARED_SIGNALS_DELIVERY_INTERVAL": "0s",
	}))
	LoadWorkerConfig(l)
	err := l.Err()
	if err == nil {
		t.Fatal("expected aggregated worker interval errors")
	}
	for _, key := range []string{
		"JOB_POLL_INTERVAL", "JOB_LEASE_DURATION",
		"EPHEMERAL_SWEEP_INTERVAL", "SHARED_SIGNALS_DELIVERY_INTERVAL",
	} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("aggregated error %q does not mention %s", err.Error(), key)
		}
	}
}

// TestLoadWorkerConfigRejectsInvalidLane covers REQ-SYSTEM-016: an unknown
// lane name must be reported alongside every other problem instead of
// short-circuiting the startup attempt.
func TestLoadWorkerConfigRejectsInvalidLane(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"JOB_WORKER_LANES":       "bulk,turbo",
		"JOB_WORKER_CONCURRENCY": "0",
	}))
	LoadWorkerConfig(l)
	err := l.Err()
	if err == nil {
		t.Fatal("expected an invalid lane error")
	}
	if !strings.Contains(err.Error(), "JOB_WORKER_LANES") || !strings.Contains(err.Error(), "JOB_WORKER_CONCURRENCY") {
		t.Errorf("aggregated error %q, want both the lane and the concurrency problem", err.Error())
	}
}

func TestLoadWorkerConfigLaneSpecificConcurrency(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"JOB_WORKER_CONCURRENCY":                   "8",
		"JOB_WORKER_CONCURRENCY_LATENCY_SENSITIVE": "16",
	}))
	cfg := LoadWorkerConfig(l)
	if err := l.Err(); err != nil {
		t.Fatalf("LoadWorkerConfig: %v", err)
	}
	if got := cfg.Concurrency[domain.LaneLatencySensitive]; got != 16 {
		t.Errorf("latency_sensitive concurrency = %d, want the lane override 16", got)
	}
	if got := cfg.Concurrency[domain.LaneBulk]; got != 8 {
		t.Errorf("bulk concurrency = %d, want the shared default 8", got)
	}
}
