package usecases

import (
	"context"

	"github.com/ambi/idmagic/backend/provisioning/ports"
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
}

// DispatchPendingDeliveries associates up to limit pending, unattached
// deliveries with a Jobs.Job (LifecycleWorkflowRunLifecycle's dispatcher
// precedent: recovers from an API-process enqueue failure via periodic
// re-scan). It is safe to call repeatedly and from multiple worker processes:
// AttachJob only succeeds once per delivery (job_id IS NULL guard).
func DispatchPendingDeliveries(ctx context.Context, deps DispatcherDeps, limit int) (dispatched int, err error) {
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
		}
	}
	return dispatched, nil
}
