package usecases_test

// 主要ユースケース追跡: REQ-PROVISIONING-006。

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

//spec:covers EX-PROVISIONING-006-01: 予約は期限の直前には配信へ変わらず、期限に達した周期処理で delete の配信になってジョブへ関連付けられ、以後の周期処理では重複しない。
func TestDispatchPendingDeliveries_MaterializesAScheduledDeprovisionOnlyOnceDue(t *testing.T) {
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	ctx := context.Background()
	deletedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	reservation := domain.NewScheduledDeprovision("reservation-1", "tenant-a", "app-1", "user-1", 42, deletedAt, 7)
	if _, err := deliveryRepo.ScheduleDeprovision(ctx, reservation); err != nil {
		t.Fatalf("ScheduleDeprovision() error = %v", err)
	}
	other := domain.NewScheduledDeprovision("reservation-2", "tenant-a", "app-2", "user-1", 42, deletedAt, 7)
	if _, err := deliveryRepo.ScheduleDeprovision(ctx, other); err != nil {
		t.Fatalf("ScheduleDeprovision(other) error = %v", err)
	}
	enqueuer := &fakeEnqueuer{nextJob: "job-1"}
	deps := usecases.DispatcherDeps{DeliveryRepo: deliveryRepo, Enqueuer: enqueuer}

	if _, err := usecases.DispatchPendingDeliveries(ctx, deps, 10, reservation.DueAt.Add(-time.Second)); err != nil {
		t.Fatalf("DispatchPendingDeliveries(before due) error = %v", err)
	}
	if deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10); len(deliveries) != 0 || len(enqueuer.calls) != 0 {
		t.Fatalf("before due: deliveries = %+v, enqueues = %+v; want neither", deliveries, enqueuer.calls)
	}

	if _, err := usecases.DispatchPendingDeliveries(ctx, deps, 10, reservation.DueAt); err != nil {
		t.Fatalf("DispatchPendingDeliveries(at due) error = %v", err)
	}
	deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(deliveries) != 1 {
		t.Fatalf("at due: deliveries = %+v, want one", deliveries)
	}
	d := deliveries[0]
	if d.Operation != domain.OperationDelete || d.SourceID != "user-1" || d.SourceVersion != 42 || d.Status != domain.DeliveryInFlight {
		t.Errorf("materialized delivery = %+v, want an in-flight delete of user-1 at version 42", d)
	}
	if others, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-2", nil, 10); len(others) != 1 {
		t.Errorf("at due: deliveries on app-2 = %+v, want every due reservation materialized in one pass", others)
	}
	if len(enqueuer.calls) != 2 {
		t.Errorf("enqueuer.calls = %+v, want one call per materialized delivery", enqueuer.calls)
	}

	if _, err := usecases.DispatchPendingDeliveries(ctx, deps, 10, reservation.DueAt.Add(time.Hour)); err != nil {
		t.Fatalf("DispatchPendingDeliveries(after due) error = %v", err)
	}
	if deliveries, _ := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10); len(deliveries) != 1 || len(enqueuer.calls) != 2 {
		t.Errorf("after due: deliveries = %d, enqueues = %d; want the one delivery only", len(deliveries), len(enqueuer.calls))
	}
}
