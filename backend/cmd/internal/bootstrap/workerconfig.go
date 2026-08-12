package bootstrap

import (
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
)

// allLanes is the compat-mode default for JOB_WORKER_LANES: a single
// idmagic-worker process claims every lane. Dedicated per-lane deployments
// (production topology) instead set JOB_WORKER_LANES to a single lane.
var allLanes = []domain.ExecutionLane{domain.LaneLatencySensitive, domain.LaneDefault, domain.LaneBulk}

// WorkerConfig is the startup configuration read only by the idmagic-worker
// process: which lanes it claims, how its Runners poll and lease, and the
// cadence of the resident sweep loops it owns. Parsed once via
// LoadWorkerConfig using the same ConfigLoader as LoadSharedConfig, so a
// worker-only typo and a shared-config typo are reported together in one
// startup attempt (REQ-SYSTEM-016).
type WorkerConfig struct {
	OTelServiceName string
	LogLevel        string
	Addr            string
	WorkerID        string

	Lanes       []domain.ExecutionLane
	Concurrency map[domain.ExecutionLane]int

	PollInterval  time.Duration
	LeaseDuration time.Duration
	BackoffBase   time.Duration
	BackoffCap    time.Duration

	EphemeralSweepInterval        time.Duration
	SharedSignalsDeliveryInterval time.Duration
	DrainGracePeriod              time.Duration
}

// LoadWorkerConfig parses every WorkerConfig field from l, recording every
// missing/malformed value on l (see LoadSharedConfig). It performs no I/O:
// WorkerID's hostname fallback is resolved by the caller, after l.Err() has
// been checked.
func LoadWorkerConfig(l *ConfigLoader) WorkerConfig {
	var cfg WorkerConfig

	cfg.OTelServiceName = l.String("OTEL_SERVICE_NAME", "idmagic-worker")
	cfg.LogLevel = l.Enum("LOG_LEVEL", "info", "debug", "info", "warn", "warning", "error")
	cfg.Addr = l.String("ADDR", ":8080")
	cfg.WorkerID = l.String("WORKER_ID", "")

	cfg.Lanes = loadWorkerLanes(l)
	// Every lane's override is read whether or not this process claims that
	// lane, so a typo in an unclaimed lane's key still fails startup and the
	// generated ConfigurationReference lists all three keys.
	shared := l.PositiveInt("JOB_WORKER_CONCURRENCY", 4)
	cfg.Concurrency = make(map[domain.ExecutionLane]int, len(allLanes))
	for _, lane := range allLanes {
		cfg.Concurrency[lane] = l.PositiveInt(laneConcurrencyKey(lane), shared)
	}

	cfg.PollInterval = l.PositiveDuration("JOB_POLL_INTERVAL", 2*time.Second)
	cfg.LeaseDuration = l.PositiveDuration("JOB_LEASE_DURATION", 5*time.Minute)
	cfg.BackoffBase = l.PositiveDuration("JOB_BACKOFF_BASE", domain.DefaultBackoffBase)
	cfg.BackoffCap = l.PositiveDuration("JOB_BACKOFF_CAP", domain.DefaultBackoffCap)

	cfg.EphemeralSweepInterval = l.PositiveDuration("EPHEMERAL_SWEEP_INTERVAL", 60*time.Second)
	cfg.SharedSignalsDeliveryInterval = l.PositiveDuration("SHARED_SIGNALS_DELIVERY_INTERVAL", 5*time.Second)
	cfg.DrainGracePeriod = time.Duration(l.NonNegativeInt("DRAIN_GRACE_PERIOD_SECONDS", 5)) * time.Second

	return cfg
}

// laneConcurrencyKey is the per-lane override for JOB_WORKER_CONCURRENCY
// (e.g. JOB_WORKER_CONCURRENCY_LATENCY_SENSITIVE), which lets a dedicated
// latency_sensitive deployment reserve capacity independently of bulk's.
func laneConcurrencyKey(lane domain.ExecutionLane) string {
	return "JOB_WORKER_CONCURRENCY_" + strings.ToUpper(string(lane))
}

func loadWorkerLanes(l *ConfigLoader) []domain.ExecutionLane {
	allowed := make([]string, len(allLanes))
	fallback := make([]string, len(allLanes))
	for i, lane := range allLanes {
		allowed[i] = string(lane)
		fallback[i] = string(lane)
	}
	names := l.EnumList("JOB_WORKER_LANES", fallback, allowed...)
	lanes := make([]domain.ExecutionLane, 0, len(names))
	for _, name := range names {
		if lane := domain.ExecutionLane(name); lane.Valid() {
			lanes = append(lanes, lane)
		}
	}
	if len(lanes) == 0 {
		return allLanes
	}
	return lanes
}
