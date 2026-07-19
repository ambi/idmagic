package usecases

import (
	"context"
	"encoding/json"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

// KindProvisioningDelivery is the Jobs.JobKind for one ProvisioningDelivery
// execution attempt (spec/contexts/provisioning.yaml §配送・信頼性). Registered
// via jobsdomain.RegisterKind (caller-owned kind, ADR-117 §5 direction) rather
// than a hardcoded Jobs constant. Lane is default (ADR-129): SCIM delivery
// does not carry the low-latency requirement backchannel_logout_delivery has.
const KindProvisioningDelivery jobsdomain.JobKind = "provisioning_delivery"

func init() {
	jobsdomain.RegisterKind(KindProvisioningDelivery, jobsdomain.LaneDefault)
}

// JobHandlerDeps are ProvisioningDeliveryHandler's dependencies.
type JobHandlerDeps struct {
	DeliverDeps    DeliverDeps
	ConnectionRepo ports.ProvisioningConnectionRepository
	DeliveryRepo   ports.ProvisioningDeliveryRepository
	// Now returns the current time; defaults to time.Now().UTC() when nil.
	Now func() time.Time
}

type provisioningDeliveryParams struct {
	DeliveryID string `json:"delivery_id"`
}

// ProvisioningDeliveryHandler adapts ExecuteDelivery to the Jobs handler
// signature. On success it resets the connection's consecutive failure streak.
// On failure it inspects job.Attempts vs job.MaxAttempts (mirroring
// backend/jobs/usecases.Runner.fail's own terminal check, since the handler
// itself is not otherwise told terminality): a non-terminal failure leaves
// ProvisioningDelivery.status untouched (in_flight, per
// states.ProvisioningDeliveryLifecycle — Jobs owns the retry loop); a terminal
// failure marks the delivery dead_letter and increments the connection's
// consecutive failure count, quarantining the connection once it reaches
// QuarantineAfterConsecutiveFailure (spec/contexts/provisioning.yaml
// events.ConnectionQuarantined). The handler always returns the original error
// on failure so Jobs' own Runner records JobFailed/JobRetried.
func ProvisioningDeliveryHandler(deps JobHandlerDeps) jobsusecases.Handler {
	now := deps.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		var params provisioningDeliveryParams
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return nil, err
		}
		execErr := ExecuteDelivery(ctx, deps.DeliverDeps, job.TenantID, params.DeliveryID, now())
		if execErr == nil {
			if err := resetConsecutiveFailures(ctx, deps, job.TenantID, params.DeliveryID); err != nil {
				return nil, err
			}
			return nil, nil
		}
		if job.Attempts < job.MaxAttempts {
			return nil, execErr // non-terminal: Jobs will retry, delivery stays in_flight
		}
		errMsg := execErr.Error()
		if err := deps.DeliveryRepo.UpdateStatus(ctx, job.TenantID, params.DeliveryID, domain.DeliveryDeadLetter, &errMsg); err != nil {
			return nil, err
		}
		if err := recordConsecutiveFailure(ctx, deps, job.TenantID, params.DeliveryID, errMsg, now()); err != nil {
			return nil, err
		}
		return nil, execErr
	}
}

func resetConsecutiveFailures(ctx context.Context, deps JobHandlerDeps, tenantID, deliveryID string) error {
	conn, err := connectionForDelivery(ctx, deps, tenantID, deliveryID)
	if err != nil || conn == nil {
		return err
	}
	if conn.ConsecutiveFailureCount == 0 {
		return nil
	}
	conn.ConsecutiveFailureCount = 0
	return deps.ConnectionRepo.Update(ctx, conn, nil)
}

func recordConsecutiveFailure(ctx context.Context, deps JobHandlerDeps, tenantID, deliveryID, reason string, now time.Time) error {
	conn, err := connectionForDelivery(ctx, deps, tenantID, deliveryID)
	if err != nil || conn == nil {
		return err
	}
	conn.ConsecutiveFailureCount++
	if conn.ConsecutiveFailureCount >= conn.QuarantineAfterConsecutiveFailure && conn.Health != domain.HealthQuarantined {
		if err := conn.Quarantine(reason, now); err != nil {
			return err
		}
	}
	return deps.ConnectionRepo.Update(ctx, conn, nil)
}

func connectionForDelivery(ctx context.Context, deps JobHandlerDeps, tenantID, deliveryID string) (*domain.ProvisioningConnection, error) {
	delivery, err := deps.DeliveryRepo.Find(ctx, tenantID, deliveryID)
	if err != nil || delivery == nil {
		return nil, err
	}
	return deps.ConnectionRepo.Find(ctx, tenantID, delivery.ConnectionID)
}
