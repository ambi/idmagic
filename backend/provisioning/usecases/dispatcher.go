package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Enqueuer submits a pending ProvisioningDelivery as a durable Jobs.Job
// (spec/contexts/provisioning.yaml §配送・信頼性, kind provisioning_delivery).
// dedupKey is the delivery's idempotency key, so a duplicate dispatch of the
// same delivery is a no-op at the Jobs layer too.
type Enqueuer interface {
	EnqueueProvisioningDelivery(ctx context.Context, tenantID, dedupKey, deliveryID string) (jobID string, err error)
}

// DispatcherDeps are DispatchPendingDeliveries's dependencies.
type DispatcherDeps struct {
	DeliveryRepo ports.ProvisioningDeliveryRepository
	Enqueuer     Enqueuer
	// Emit は関連付けに成功した配信の ProvisioningDeliveryStarted を発行する。
	Emit func(spec.DomainEvent)
}

// DispatchPendingDeliveries associates up to limit pending, unattached
// deliveries with a Jobs.Job (LifecycleWorkflowRunLifecycle's dispatcher
// precedent: recovers from an API-process enqueue failure via periodic
// re-scan). It is safe to call repeatedly and from multiple worker processes:
// AttachJob only succeeds once per delivery (job_id IS NULL guard), so only the
// worker that attached emits ProvisioningDeliveryStarted.
//
// 関連付けの前に、now の時点で期限に達した ScheduledDeprovision を pending の delete
// 配信へ変える。猶予期間つきの削除も、ほかの配信と同じ経路で下流へ届く。
func DispatchPendingDeliveries(ctx context.Context, deps DispatcherDeps, limit int, now time.Time) (dispatched int, err error) {
	if err := materializeDueDeprovisions(ctx, deps.DeliveryRepo, limit, now); err != nil {
		return 0, err
	}
	deliveries, err := deps.DeliveryRepo.ListUnenqueued(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, d := range deliveries {
		jobID, err := deps.Enqueuer.EnqueueProvisioningDelivery(ctx, d.TenantID, d.IdempotencyKey(), d.ID)
		if err != nil {
			return dispatched, err
		}
		attached, err := deps.DeliveryRepo.AttachJob(ctx, d.TenantID, d.ID, jobID)
		if err != nil {
			return dispatched, err
		}
		if attached {
			dispatched++
			emit(deps.Emit, &domain.ProvisioningDeliveryStarted{
				At: now.UTC(), TenantID: d.TenantID, ConnectionID: d.ConnectionID, DeliveryID: d.ID, JobID: jobID,
			})
		}
	}
	return dispatched, nil
}

// materializeDueDeprovisions は期限に達した予約を配信へ変える。取消と競合した予約は
// MaterializeDeprovision が false を返すので、配信を作らずに読み飛ばす。
func materializeDueDeprovisions(ctx context.Context, repo ports.ProvisioningDeliveryRepository, limit int, now time.Time) error {
	due, err := repo.ListDueDeprovisions(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, reservation := range due {
		id, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		if _, err := repo.MaterializeDeprovision(ctx, reservation, reservation.Delivery(id, now)); err != nil {
			return err
		}
	}
	return nil
}
