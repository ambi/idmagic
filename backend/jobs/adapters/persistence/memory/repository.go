// Package memory is the in-process JobRepository implementation used by the
// memory persistence runtime (and by usecases-layer tests, per this repo's
// convention of exercising a real adapter rather than hand-rolled mocks).
package memory

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// JobRepository is a mutex-protected, map-backed ports.JobRepository. It
// approximates the PostgreSQL adapter's `FOR UPDATE SKIP LOCKED` claim
// (ADR-098) with a single mutex serializing ClaimBatch instead, since there is
// no concurrent-transaction concern to model in-process.
type JobRepository struct {
	mu   sync.Mutex
	byID map[string]*domain.Job
}

func NewJobRepository() *JobRepository {
	return &JobRepository{byID: map[string]*domain.Job{}}
}

var _ ports.JobRepository = (*JobRepository)(nil)

// copyJob returns a struct copy so callers cannot mutate repository state
// through a returned pointer, and concurrent readers/writers of the same Job
// (worker goroutines vs. the map) don't race on the same struct instance.
func copyJob(j *domain.Job) *domain.Job {
	cp := *j
	return &cp
}

func (r *JobRepository) Enqueue(_ context.Context, input ports.EnqueueInput) (*domain.Job, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if input.DedupKey != nil {
		for _, j := range r.byID {
			if j.TenantID == input.TenantID && !domain.IsJobLifecycleTerminal(j.Status) &&
				j.DedupKey != nil && *j.DedupKey == *input.DedupKey {
				return copyJob(j), false, nil
			}
		}
	}

	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, false, err
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	runAt := input.RunAt
	if runAt.IsZero() {
		runAt = now
	}
	job := &domain.Job{
		ID:          id,
		TenantID:    input.TenantID,
		Kind:        input.Kind,
		Lane:        input.Lane,
		Status:      domain.StatusQueued,
		Params:      input.Params,
		Attempts:    0,
		MaxAttempts: input.MaxAttempts,
		DedupKey:    input.DedupKey,
		RunAt:       runAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.byID[job.ID] = job
	return copyJob(job), true, nil
}

func (r *JobRepository) ClaimBatch(_ context.Context, workerID string, lane domain.ExecutionLane, batchSize int, leaseDuration time.Duration, now time.Time) ([]*domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if batchSize <= 0 {
		return nil, nil
	}

	var candidates []*domain.Job
	for _, j := range r.byID {
		if j.Lane != lane {
			continue
		}
		switch {
		case j.Status == domain.StatusQueued && !j.RunAt.After(now):
			candidates = append(candidates, j)
		case j.Status == domain.StatusRunning && j.LeaseExpiresAt != nil && j.LeaseExpiresAt.Before(now):
			// Previous worker's lease expired before it completed the Job
			// (crash, or a drain timeout that outlived the process); reclaim
			// it for a new attempt. Status stays Running, so no
			// JobLifecycle transition applies here.
			candidates = append(candidates, j)
		}
	}
	sort.Slice(candidates, func(i, k int) bool { return candidates[i].RunAt.Before(candidates[k].RunAt) })
	if len(candidates) > batchSize {
		candidates = candidates[:batchSize]
	}

	claimed := make([]*domain.Job, 0, len(candidates))
	for _, j := range candidates {
		if j.Status == domain.StatusQueued {
			next, err := domain.TransitionJobLifecycle(j.Status, domain.EventClaim)
			if err != nil {
				continue
			}
			j.Status = next
		}
		owner := workerID
		expiry := now.Add(leaseDuration)
		j.Attempts++
		j.LeaseOwner = &owner
		j.LeaseExpiresAt = &expiry
		j.UpdatedAt = now
		claimed = append(claimed, copyJob(j))
	}
	return claimed, nil
}

func (r *JobRepository) Heartbeat(_ context.Context, jobID, workerID string, leaseDuration time.Duration, now time.Time) (time.Time, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return time.Time{}, ports.ErrJobNotFound
	}
	if !leaseHeld(j, workerID, now) {
		return time.Time{}, ports.ErrJobLeaseLost
	}
	expiry := now.Add(leaseDuration)
	j.LeaseExpiresAt = &expiry
	j.UpdatedAt = now
	return expiry, nil
}

func (r *JobRepository) Complete(_ context.Context, jobID, workerID string, result json.RawMessage, now time.Time) (*domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, ports.ErrJobNotFound
	}
	if !leaseHeld(j, workerID, now) {
		return nil, ports.ErrJobLeaseLost
	}
	next, err := domain.TransitionJobLifecycle(j.Status, domain.EventComplete)
	if err != nil {
		return nil, ports.ErrJobLeaseLost
	}
	j.Status = next
	j.Result = result
	j.LeaseOwner = nil
	j.LeaseExpiresAt = nil
	j.UpdatedAt = now
	return copyJob(j), nil
}

func (r *JobRepository) Fail(_ context.Context, jobID, workerID string, outcome ports.FailOutcome, now time.Time) (*domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, ports.ErrJobNotFound
	}
	if !leaseHeld(j, workerID, now) {
		return nil, ports.ErrJobLeaseLost
	}
	event := domain.EventFail
	if outcome.NextStatus == domain.StatusQueued {
		event = domain.EventRetry
	}
	next, err := domain.TransitionJobLifecycle(j.Status, event)
	if err != nil {
		return nil, err
	}
	errMsg := outcome.Error
	j.Status = next
	j.Error = &errMsg
	j.LeaseOwner = nil
	j.LeaseExpiresAt = nil
	j.UpdatedAt = now
	if next == domain.StatusQueued {
		j.RunAt = outcome.RunAt
	}
	return copyJob(j), nil
}

func (r *JobRepository) Cancel(_ context.Context, jobID string, now time.Time) (*domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, ports.ErrJobNotFound
	}
	if domain.IsJobLifecycleTerminal(j.Status) {
		return nil, ports.ErrJobAlreadyTerminal
	}
	next, err := domain.TransitionJobLifecycle(j.Status, domain.EventCancel)
	if err != nil {
		return nil, err
	}
	j.Status = next
	j.LeaseOwner = nil
	j.LeaseExpiresAt = nil
	j.UpdatedAt = now
	return copyJob(j), nil
}

func (r *JobRepository) LaneDepths(_ context.Context) ([]ports.LaneDepth, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	byLane := map[domain.ExecutionLane]*ports.LaneDepth{}
	for _, j := range r.byID {
		switch j.Status {
		case domain.StatusQueued, domain.StatusRunning:
		default:
			continue
		}
		d, ok := byLane[j.Lane]
		if !ok {
			d = &ports.LaneDepth{Lane: j.Lane}
			byLane[j.Lane] = d
		}
		if j.Status == domain.StatusQueued {
			d.Queued++
		} else {
			d.Running++
		}
	}
	depths := make([]ports.LaneDepth, 0, len(byLane))
	for _, d := range byLane {
		depths = append(depths, *d)
	}
	return depths, nil
}

func (r *JobRepository) Get(_ context.Context, jobID string) (*domain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, ports.ErrJobNotFound
	}
	return copyJob(j), nil
}

func leaseHeld(j *domain.Job, workerID string, now time.Time) bool {
	return j.Status == domain.StatusRunning &&
		j.LeaseOwner != nil && *j.LeaseOwner == workerID &&
		j.LeaseExpiresAt != nil && !j.LeaseExpiresAt.Before(now)
}
