package usecases_test

import (
	"context"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

type fakeEnqueuer struct {
	calls   []struct{ tenantID, dedupKey, deliveryID string }
	nextJob string
	err     error
}

func (f *fakeEnqueuer) EnqueueProvisioningDelivery(_ context.Context, tenantID, dedupKey, deliveryID string) (string, error) {
	f.calls = append(f.calls, struct{ tenantID, dedupKey, deliveryID string }{tenantID, dedupKey, deliveryID})
	if f.err != nil {
		return "", f.err
	}
	return f.nextJob, nil
}

func TestDispatchPendingDeliveries_AttachesJobToEachUnenqueuedDelivery(t *testing.T) {
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	ctx := context.Background()
	d := &domain.ProvisioningDelivery{ID: "delivery-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.DeliveryPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := deliveryRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	enqueuer := &fakeEnqueuer{nextJob: "job-1"}
	dispatched, err := usecases.DispatchPendingDeliveries(ctx, usecases.DispatcherDeps{DeliveryRepo: deliveryRepo, Enqueuer: enqueuer}, 10, time.Now())
	if err != nil {
		t.Fatalf("DispatchPendingDeliveries() error = %v", err)
	}
	if dispatched != 1 {
		t.Errorf("DispatchPendingDeliveries() dispatched = %d, want 1", dispatched)
	}
	if len(enqueuer.calls) != 1 || enqueuer.calls[0].deliveryID != "delivery-1" || enqueuer.calls[0].dedupKey != d.IdempotencyKey() {
		t.Errorf("enqueuer.calls = %+v, want a single call for delivery-1 with dedupKey %q", enqueuer.calls, d.IdempotencyKey())
	}
	found, _ := deliveryRepo.Find(ctx, "tenant-a", "delivery-1")
	if found.JobID == nil || *found.JobID != "job-1" {
		t.Errorf("delivery.JobID = %v, want job-1", found.JobID)
	}
}

func TestDispatchPendingDeliveries_SkipsAlreadyAttached(t *testing.T) {
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	ctx := context.Background()
	d := &domain.ProvisioningDelivery{ID: "delivery-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.DeliveryPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_, _ = deliveryRepo.Save(ctx, d)
	_, _ = deliveryRepo.AttachJob(ctx, "tenant-a", "delivery-1", "job-existing")

	enqueuer := &fakeEnqueuer{nextJob: "job-new"}
	dispatched, err := usecases.DispatchPendingDeliveries(ctx, usecases.DispatcherDeps{DeliveryRepo: deliveryRepo, Enqueuer: enqueuer}, 10, time.Now())
	if err != nil {
		t.Fatalf("DispatchPendingDeliveries() error = %v", err)
	}
	if dispatched != 0 || len(enqueuer.calls) != 0 {
		t.Errorf("DispatchPendingDeliveries() dispatched = %d, calls = %d, want 0 (already attached, not in ListUnenqueued)", dispatched, len(enqueuer.calls))
	}
}

//spec:covers EX-PROVISIONING-017-01: 未関連付けの pending 配信を再走査してジョブを関連付け、配信を in_flight にして ProvisioningDeliveryStarted を発行する。
func TestDispatchPendingDeliveries_StartsTheDeliveryAndEmitsStartedAfterAttaching(t *testing.T) {
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	ctx := context.Background()
	d := &domain.ProvisioningDelivery{ID: "delivery-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.DeliveryPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := deliveryRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	recorder := &eventRecorder{}
	deps := usecases.DispatcherDeps{DeliveryRepo: deliveryRepo, Enqueuer: &fakeEnqueuer{nextJob: "job-1"}, Emit: recorder.emit}

	if _, err := usecases.DispatchPendingDeliveries(ctx, deps, 10, time.Now()); err != nil {
		t.Fatalf("DispatchPendingDeliveries() error = %v", err)
	}
	found, _ := deliveryRepo.Find(ctx, "tenant-a", "delivery-1")
	started, ok := onlyEvent[*domain.ProvisioningDeliveryStarted](t, recorder)
	if !ok || found.Status != domain.DeliveryInFlight ||
		started.DeliveryID != "delivery-1" || started.JobID != "job-1" || started.ConnectionID != "app-1" || started.TenantID != "tenant-a" {
		t.Fatalf("after dispatch: status = %v, event = %+v; want in_flight and one Started for delivery-1/job-1", found.Status, started)
	}
}

// 別の worker が同じ配信へ先にジョブを関連付けた場合、関連付けに負けた側は発行しない。
func TestDispatchPendingDeliveries_EmitsNothingWhenAnotherWorkerAttachedFirst(t *testing.T) {
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	ctx := context.Background()
	d := &domain.ProvisioningDelivery{ID: "delivery-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.DeliveryPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := deliveryRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	recorder := &eventRecorder{}
	racing := racingEnqueuer{attach: func() { _, _ = deliveryRepo.AttachJob(ctx, "tenant-a", "delivery-1", "job-1") }}
	deps := usecases.DispatcherDeps{DeliveryRepo: deliveryRepo, Enqueuer: racing, Emit: recorder.emit}

	if _, err := usecases.DispatchPendingDeliveries(ctx, deps, 10, time.Now()); err != nil {
		t.Fatalf("DispatchPendingDeliveries() error = %v", err)
	}
	if len(recorder.events) != 0 {
		t.Fatalf("events = %v, want none from the worker that lost the attach", recorder.types())
	}
}

// racingEnqueuer は投入の間に別の worker の関連付けを割り込ませる。
type racingEnqueuer struct{ attach func() }

func (r racingEnqueuer) EnqueueProvisioningDelivery(context.Context, string, string, string) (string, error) {
	r.attach()
	return "job-1", nil
}
