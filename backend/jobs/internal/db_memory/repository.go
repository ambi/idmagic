// Package memory is the in-process JobRepository implementation used by the
// memory persistence runtime (and by usecases-layer tests, per this repo's
// convention of exercising a real adapter rather than hand-rolled mocks).
package db_memory

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// JobRepository is a mutex-protected, map-backed ports.JobRepository. It
// approximates the PostgreSQL adapter's `FOR UPDATE SKIP LOCKED` claim
// with a single mutex serializing ClaimBatch instead, since there is
// no concurrent-transaction concern to model in-process.
type JobRepository struct {
	mu   sync.Mutex
	byID map[string]*jobsdomain.Job
}

func NewJobRepository() *JobRepository {
	return &JobRepository{byID: map[string]*jobsdomain.Job{}}
}

var _ jobsports.JobRepository = (*JobRepository)(nil)

// copyJob returns a struct copy so callers cannot mutate repository state
// through a returned pointer, and concurrent readers/writers of the same Job
// (worker goroutines vs. the map) don't race on the same struct instance.
func copyJob(j *jobsdomain.Job) *jobsdomain.Job {
	cp := *j
	return &cp
}

func (r *JobRepository) Enqueue(_ context.Context, input jobsports.EnqueueInput) (*jobsdomain.Job, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if input.DedupKey != nil {
		for _, j := range r.byID {
			if j.TenantID == input.TenantID && !jobsdomain.IsJobLifecycleTerminal(j.Status) &&
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
	job := &jobsdomain.Job{
		ID:          id,
		TenantID:    input.TenantID,
		Kind:        input.Kind,
		Lane:        input.Lane,
		Status:      jobsdomain.StatusQueued,
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

func (r *JobRepository) ClaimBatch(_ context.Context, workerID string, lane jobsdomain.ExecutionLane, batchSize int, leaseDuration time.Duration, now time.Time) ([]*jobsdomain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if batchSize <= 0 {
		return nil, nil
	}

	var candidates []*jobsdomain.Job
	for _, j := range r.byID {
		if j.Lane != lane {
			continue
		}
		switch {
		case j.Status == jobsdomain.StatusQueued && !j.RunAt.After(now):
			candidates = append(candidates, j)
		case j.Status == jobsdomain.StatusRunning && j.LeaseExpiresAt != nil && j.LeaseExpiresAt.Before(now):
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

	claimed := make([]*jobsdomain.Job, 0, len(candidates))
	for _, j := range candidates {
		if j.Status == jobsdomain.StatusQueued {
			next, err := jobsdomain.TransitionJobLifecycle(j.Status, jobsdomain.EventClaim)
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
		return time.Time{}, jobsports.ErrJobNotFound
	}
	if !leaseHeld(j, workerID, now) {
		return time.Time{}, jobsports.ErrJobLeaseLost
	}
	expiry := now.Add(leaseDuration)
	j.LeaseExpiresAt = &expiry
	j.UpdatedAt = now
	return expiry, nil
}

func (r *JobRepository) Complete(_ context.Context, jobID, workerID string, result json.RawMessage, now time.Time) (*jobsdomain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, jobsports.ErrJobNotFound
	}
	if !leaseHeld(j, workerID, now) {
		return nil, jobsports.ErrJobLeaseLost
	}
	next, err := jobsdomain.TransitionJobLifecycle(j.Status, jobsdomain.EventComplete)
	if err != nil {
		return nil, jobsports.ErrJobLeaseLost
	}
	j.Status = next
	j.Result = result
	j.LeaseOwner = nil
	j.LeaseExpiresAt = nil
	j.UpdatedAt = now
	return copyJob(j), nil
}

func (r *JobRepository) Fail(_ context.Context, jobID, workerID string, outcome jobsports.FailOutcome, now time.Time) (*jobsdomain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, jobsports.ErrJobNotFound
	}
	if !leaseHeld(j, workerID, now) {
		return nil, jobsports.ErrJobLeaseLost
	}
	event := jobsdomain.EventFail
	if outcome.NextStatus == jobsdomain.StatusQueued {
		event = jobsdomain.EventRetry
	}
	next, err := jobsdomain.TransitionJobLifecycle(j.Status, event)
	if err != nil {
		return nil, err
	}
	errMsg := outcome.Error
	j.Status = next
	j.Error = &errMsg
	j.LeaseOwner = nil
	j.LeaseExpiresAt = nil
	j.UpdatedAt = now
	if next == jobsdomain.StatusQueued {
		j.RunAt = outcome.RunAt
	}
	return copyJob(j), nil
}

func (r *JobRepository) Cancel(_ context.Context, jobID string, now time.Time) (*jobsdomain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, jobsports.ErrJobNotFound
	}
	if jobsdomain.IsJobLifecycleTerminal(j.Status) {
		return nil, jobsports.ErrJobAlreadyTerminal
	}
	next, err := jobsdomain.TransitionJobLifecycle(j.Status, jobsdomain.EventCancel)
	if err != nil {
		return nil, err
	}
	j.Status = next
	j.LeaseOwner = nil
	j.LeaseExpiresAt = nil
	j.UpdatedAt = now
	return copyJob(j), nil
}

func (r *JobRepository) LaneDepths(_ context.Context) ([]jobsports.LaneDepth, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	byLane := map[jobsdomain.ExecutionLane]*jobsports.LaneDepth{}
	for _, j := range r.byID {
		switch j.Status {
		case jobsdomain.StatusQueued, jobsdomain.StatusRunning:
		default:
			continue
		}
		d, ok := byLane[j.Lane]
		if !ok {
			d = &jobsports.LaneDepth{Lane: j.Lane}
			byLane[j.Lane] = d
		}
		if j.Status == jobsdomain.StatusQueued {
			d.Queued++
		} else {
			d.Running++
		}
	}
	depths := make([]jobsports.LaneDepth, 0, len(byLane))
	for _, d := range byLane {
		depths = append(depths, *d)
	}
	return depths, nil
}

func (r *JobRepository) ListByTenantAndKinds(_ context.Context, tenantID string, kinds []jobsdomain.JobKind, limit int) ([]*jobsdomain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	kindSet := make(map[jobsdomain.JobKind]struct{}, len(kinds))
	for _, k := range kinds {
		kindSet[k] = struct{}{}
	}
	var out []*jobsdomain.Job
	for _, j := range r.byID {
		if j.TenantID != tenantID {
			continue
		}
		if _, ok := kindSet[j.Kind]; !ok {
			continue
		}
		out = append(out, copyJob(j))
	}
	sort.Slice(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListForAdmin implements the wi-157 console read. The PostgreSQL adapter does
// the same narrowing in SQL; keeping the two in step matters because the memory
// runtime is what the usecases tests exercise.
func (r *JobRepository) ListForAdmin(_ context.Context, filter jobsports.AdminJobFilter) ([]*jobsdomain.Job, error) {
	if filter.TenantID == "" && !filter.AllTenants {
		return nil, jobsports.ErrAdminJobFilterUnscoped
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	statuses := make(map[jobsdomain.JobStatus]struct{}, len(filter.Statuses))
	for _, s := range filter.Statuses {
		statuses[s] = struct{}{}
	}
	kinds := make(map[jobsdomain.JobKind]struct{}, len(filter.Kinds))
	for _, k := range filter.Kinds {
		kinds[k] = struct{}{}
	}

	var out []*jobsdomain.Job
	for _, j := range r.byID {
		if !filter.AllTenants && j.TenantID != filter.TenantID {
			continue
		}
		if len(statuses) > 0 {
			if _, ok := statuses[j.Status]; !ok {
				continue
			}
		}
		if len(kinds) > 0 {
			if _, ok := kinds[j.Kind]; !ok {
				continue
			}
		}
		if filter.Lane != "" && j.Lane != filter.Lane {
			continue
		}
		if !afterKeyset(j, filter.BeforeCreatedAt, filter.BeforeID) {
			continue
		}
		out = append(out, copyJob(j))
	}
	sort.Slice(out, func(i, k int) bool {
		if !out[i].CreatedAt.Equal(out[k].CreatedAt) {
			return out[i].CreatedAt.After(out[k].CreatedAt)
		}
		return out[i].ID > out[k].ID
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// afterKeyset reports whether j sorts strictly after the (created_at DESC,
// id DESC) anchor. A zero anchor starts from the newest row. The id is part of
// the comparison because two Jobs enqueued in the same instant would otherwise
// let a page boundary drop or repeat one of them.
func afterKeyset(j *jobsdomain.Job, beforeCreatedAt time.Time, beforeID string) bool {
	if beforeCreatedAt.IsZero() && beforeID == "" {
		return true
	}
	if j.CreatedAt.Before(beforeCreatedAt) {
		return true
	}
	return j.CreatedAt.Equal(beforeCreatedAt) && j.ID < beforeID
}

func (r *JobRepository) Get(_ context.Context, jobID string) (*jobsdomain.Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, ok := r.byID[jobID]
	if !ok {
		return nil, jobsports.ErrJobNotFound
	}
	return copyJob(j), nil
}

func leaseHeld(j *jobsdomain.Job, workerID string, now time.Time) bool {
	return j.Status == jobsdomain.StatusRunning &&
		j.LeaseOwner != nil && *j.LeaseOwner == workerID &&
		j.LeaseExpiresAt != nil && !j.LeaseExpiresAt.Before(now)
}
