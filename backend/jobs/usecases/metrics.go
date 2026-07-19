package usecases

import (
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
)

// JobsMetrics is the lane-scoped observability surface a Runner records
// against (wi-261 T006). lane is always a bounded domain.ExecutionLane value
// and outcome is always "succeeded" or "failed" — implementations must never
// add a tenant_id, job_id, or other high-cardinality label
// (spec/contexts/system.yaml MetricsExposition, ADR-100).
type JobsMetrics interface {
	// RecordJobClaimLatency records how long a claimed Job waited past its
	// run_at before this Runner claimed it.
	RecordJobClaimLatency(lane domain.ExecutionLane, latency time.Duration)
	// RecordJobOutcome records one terminal-or-not handler outcome for a
	// claimed Job's attempt: outcome is "succeeded" or "failed".
	RecordJobOutcome(lane domain.ExecutionLane, outcome string)
	// RecordJobRetry records one non-terminal failure returned to Queued for
	// a later retry attempt.
	RecordJobRetry(lane domain.ExecutionLane)
}

// jobsMetrics returns deps.Metrics, or a no-op implementation when unset, so
// Runner call sites never need a nil check.
func (rn *Runner) jobsMetrics() JobsMetrics {
	if rn.deps.Metrics != nil {
		return rn.deps.Metrics
	}
	return noopJobsMetrics{}
}

type noopJobsMetrics struct{}

func (noopJobsMetrics) RecordJobClaimLatency(domain.ExecutionLane, time.Duration) {}
func (noopJobsMetrics) RecordJobOutcome(domain.ExecutionLane, string)             {}
func (noopJobsMetrics) RecordJobRetry(domain.ExecutionLane)                       {}
