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
	dispatched, err := usecases.DispatchPendingDeliveries(ctx, usecases.DispatcherDeps{DeliveryRepo: deliveryRepo, Enqueuer: enqueuer}, 10)
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
	dispatched, err := usecases.DispatchPendingDeliveries(ctx, usecases.DispatcherDeps{DeliveryRepo: deliveryRepo, Enqueuer: enqueuer}, 10)
	if err != nil {
		t.Fatalf("DispatchPendingDeliveries() error = %v", err)
	}
	if dispatched != 0 || len(enqueuer.calls) != 0 {
		t.Errorf("DispatchPendingDeliveries() dispatched = %d, calls = %d, want 0 (already attached, not in ListUnenqueued)", dispatched, len(enqueuer.calls))
	}
}
