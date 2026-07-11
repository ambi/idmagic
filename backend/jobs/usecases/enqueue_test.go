package usecases_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	memoryjobs "github.com/ambi/idmagic/backend/jobs/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestEnqueue_AppliesDefaultsAndEmits(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	var emitted []spec.DomainEvent
	deps := usecases.EnqueueDeps{Repo: repo, Emit: func(e spec.DomainEvent) { emitted = append(emitted, e) }}
	now := time.Now().UTC()

	job, err := usecases.Enqueue(context.Background(), deps, ports.EnqueueInput{
		TenantID: "tenant-a",
		Kind:     domain.KindNoopEcho,
		Params:   json.RawMessage(`{}`),
	}, now)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if job.MaxAttempts != domain.DefaultMaxAttempts {
		t.Errorf("MaxAttempts = %d, want default %d", job.MaxAttempts, domain.DefaultMaxAttempts)
	}
	if !job.RunAt.Equal(now) {
		t.Errorf("RunAt = %v, want %v", job.RunAt, now)
	}
	if job.Status != domain.StatusQueued {
		t.Errorf("Status = %q, want %q", job.Status, domain.StatusQueued)
	}

	if len(emitted) != 1 {
		t.Fatalf("emitted %d events, want 1", len(emitted))
	}
	got, ok := emitted[0].(*domain.JobEnqueued)
	if !ok {
		t.Fatalf("emitted event type = %T, want *domain.JobEnqueued", emitted[0])
	}
	if got.JobID != job.ID || got.TenantID != job.TenantID || got.Kind != job.Kind {
		t.Errorf("JobEnqueued = %+v, want to match job %+v", got, job)
	}
}

func TestEnqueue_RejectsUnregisteredKind(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	deps := usecases.EnqueueDeps{Repo: repo}

	_, err := usecases.Enqueue(context.Background(), deps, ports.EnqueueInput{
		TenantID: "tenant-a",
		Kind:     domain.JobKind("not_a_registered_kind"),
		Params:   json.RawMessage(`{}`),
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("Enqueue() error = nil, want error for unregistered JobKind")
	}
}

func TestEnqueue_DedupHitDoesNotEmit(t *testing.T) {
	repo := memoryjobs.NewJobRepository()
	var emitted []spec.DomainEvent
	deps := usecases.EnqueueDeps{Repo: repo, Emit: func(e spec.DomainEvent) { emitted = append(emitted, e) }}
	now := time.Now().UTC()
	dedup := "dedup-1"

	input := ports.EnqueueInput{TenantID: "tenant-a", Kind: domain.KindNoopEcho, Params: json.RawMessage(`{}`), DedupKey: &dedup}
	first, err := usecases.Enqueue(context.Background(), deps, input, now)
	if err != nil {
		t.Fatalf("first Enqueue() error = %v", err)
	}
	second, err := usecases.Enqueue(context.Background(), deps, input, now)
	if err != nil {
		t.Fatalf("second Enqueue() error = %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("second Enqueue() returned a new Job, want the deduped existing one")
	}
	if len(emitted) != 1 {
		t.Errorf("emitted %d events, want 1 (no JobEnqueued on dedup hit)", len(emitted))
	}
}
