// Package testing_contract defines the shared durable job repository contract.
package testing_contract

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
)

type Fixture struct {
	Repository ports.JobRepository
	TenantID   string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	dedup := "contract-dedup"
	job, created, err := f.Repository.Enqueue(ctx, ports.EnqueueInput{TenantID: f.TenantID, Kind: domain.KindNoopEcho, Lane: domain.LaneDefault, Params: json.RawMessage(`{"ok":true}`), DedupKey: &dedup, MaxAttempts: 3, RunAt: f.Now, Now: f.Now})
	if err != nil || !created || job == nil || job.Status != domain.StatusQueued {
		t.Fatalf("Enqueue = (%+v, %v, %v)", job, created, err)
	}
	duplicate, created, err := f.Repository.Enqueue(ctx, ports.EnqueueInput{TenantID: f.TenantID, Kind: domain.KindNoopEcho, Lane: domain.LaneDefault, Params: json.RawMessage(`{"ok":true}`), DedupKey: &dedup, MaxAttempts: 3, RunAt: f.Now, Now: f.Now})
	if err != nil || created || duplicate == nil || duplicate.ID != job.ID {
		t.Fatalf("deduplicated Enqueue = (%+v, %v, %v)", duplicate, created, err)
	}
	claimed, err := f.Repository.ClaimBatch(ctx, "worker-1", domain.LaneDefault, 1, time.Minute, f.Now)
	if err != nil || len(claimed) != 1 || claimed[0].ID != job.ID || claimed[0].Status != domain.StatusRunning {
		t.Fatalf("ClaimBatch = (%+v, %v)", claimed, err)
	}
	completed, err := f.Repository.Complete(ctx, job.ID, "worker-1", json.RawMessage(`{"done":true}`), f.Now.Add(time.Minute))
	if err != nil || completed == nil || completed.Status != domain.StatusSucceeded {
		t.Fatalf("Complete = (%+v, %v)", completed, err)
	}
	if _, err := f.Repository.ClaimBatch(ctx, "worker-2", domain.LaneDefault, 1, time.Minute, f.Now); err != nil {
		t.Fatalf("ClaimBatch after terminal: %v", err)
	}
	if _, err := f.Repository.Get(ctx, "00000000-0000-0000-0000-0000000000ff"); !errors.Is(err, ports.ErrJobNotFound) {
		t.Fatalf("Get missing = %v", err)
	}
}
