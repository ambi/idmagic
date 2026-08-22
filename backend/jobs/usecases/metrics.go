package usecases

import (
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
)

// JobsMetrics is the lane-scoped observability surface a Runner records
// against (wi-261 T006). lane is always a bounded domain.ExecutionLane value
// and outcome is always "succeeded" or "failed" — implementations must never
// add a tenant_id, job_id, or other high-cardinality label
// (spec/contexts/system.yaml MetricsExposition).
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
	// RecordJobDuration records how long one attempt's handler ran, from the
	// moment the Runner started it to the moment it returned (wi-157). Claim
	// latency says how long a Job waited; this says how long it worked, and
	// only the two together explain where a slow queue is spending its time.
	// outcome is "succeeded" or "failed".
	RecordJobDuration(lane domain.ExecutionLane, outcome string, duration time.Duration)
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

func (noopJobsMetrics) RecordJobClaimLatency(domain.ExecutionLane, time.Duration)     {}
func (noopJobsMetrics) RecordJobOutcome(domain.ExecutionLane, string)                 {}
func (noopJobsMetrics) RecordJobRetry(domain.ExecutionLane)                           {}
func (noopJobsMetrics) RecordJobDuration(domain.ExecutionLane, string, time.Duration) {}
