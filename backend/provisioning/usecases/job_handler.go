package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// KindProvisioningDelivery is the Jobs.JobKind for one ProvisioningDelivery
// execution attempt (spec/contexts/provisioning.yaml §配送・信頼性). Registered
// via jobsdomain.RegisterKind (caller-owned kind, §5 direction) rather
// than a hardcoded Jobs constant. Lane is default: SCIM delivery
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
	// Emit は配信の終端遷移と接続の隔離を、それぞれの保存の後に発行する。
	Emit func(spec.DomainEvent)
}

type provisioningDeliveryParams struct {
	DeliveryID string `json:"delivery_id"`
}

// ProvisioningDeliveryHandler adapts ExecuteDelivery to the Jobs handler
// signature. On success it resets the connection's consecutive failure streak
// and then emits the delivery's transition event, so a reset that fails and is
// retried by Jobs emits on the retry instead of twice.
// On failure it inspects job.Attempts vs job.MaxAttempts (mirroring
// backend/jobs/usecases.Runner.fail's own terminal check, since the handler
// itself is not otherwise told terminality): a non-terminal failure leaves
// ProvisioningDelivery.status untouched (in_flight, per
// states.ProvisioningDeliveryLifecycle — Jobs owns the retry loop); a terminal
// failure marks the delivery dead_letter, emits UserProvisioningFailed, and
// increments the connection's consecutive failure count, quarantining the
// connection once it reaches QuarantineAfterConsecutiveFailure. The handler
// returns the original error on a downstream failure so Jobs' own Runner records
// JobFailed/JobRetried.
//
// A required attribute mapping that cannot be resolved is terminal on any
// attempt (fail-closed): retrying cannot produce the missing value. The delivery
// is settled as dead_letter and the handler returns nil so Jobs does not retry
// it. It does not count toward quarantine, which measures the downstream's
// health, not one User's missing attribute.
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
		event, execErr := ExecuteDelivery(ctx, deps.DeliverDeps, job.TenantID, params.DeliveryID, now())
		if execErr == nil {
			if err := resetConsecutiveFailures(ctx, deps, job.TenantID, params.DeliveryID); err != nil {
				return nil, err
			}
			if event != nil {
				emit(deps.Emit, event)
			}
			return nil, nil
		}
		failsClosed := errors.Is(execErr, ports.ErrRequiredAttributeUnresolved)
		if !failsClosed && job.Attempts < job.MaxAttempts {
			return nil, execErr // non-terminal: Jobs will retry, delivery stays in_flight
		}
		delivery, err := deadLetter(ctx, deps, job.TenantID, params.DeliveryID, execErr.Error(), now())
		if err != nil {
			return nil, err
		}
		if failsClosed {
			return nil, nil
		}
		if err := recordConsecutiveFailure(ctx, deps, job.TenantID, delivery.ConnectionID, execErr.Error(), now()); err != nil {
			return nil, err
		}
		return nil, execErr
	}
}

// deadLetter settles the delivery as dead_letter and emits UserProvisioningFailed
// once that status is saved.
func deadLetter(ctx context.Context, deps JobHandlerDeps, tenantID, deliveryID, reason string, now time.Time) (*domain.ProvisioningDelivery, error) {
	delivery, err := deps.DeliveryRepo.Find(ctx, tenantID, deliveryID)
	if err != nil {
		return nil, err
	}
	if delivery == nil {
		return nil, ErrDeliveryNotFound
	}
	if err := deps.DeliveryRepo.UpdateStatus(ctx, tenantID, deliveryID, domain.DeliveryDeadLetter, &reason); err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.UserProvisioningFailed{
		At: now, TenantID: tenantID, ConnectionID: delivery.ConnectionID, DeliveryID: deliveryID,
		SourceType: delivery.SourceType, SourceID: delivery.SourceID, Error: reason,
	})
	return delivery, nil
}

func resetConsecutiveFailures(ctx context.Context, deps JobHandlerDeps, tenantID, deliveryID string) error {
	delivery, err := deps.DeliveryRepo.Find(ctx, tenantID, deliveryID)
	if err != nil || delivery == nil {
		return err
	}
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, delivery.ConnectionID)
	if err != nil || conn == nil {
		return err
	}
	if conn.ConsecutiveFailureCount == 0 {
		return nil
	}
	conn.ConsecutiveFailureCount = 0
	return deps.ConnectionRepo.Update(ctx, conn, nil)
}

// recordConsecutiveFailure counts a terminal failure against the connection and
// quarantines it at the threshold. ConnectionQuarantined is emitted only on the
// failure that moves the connection into quarantine, after that is saved.
func recordConsecutiveFailure(ctx context.Context, deps JobHandlerDeps, tenantID, connectionID, reason string, now time.Time) error {
	conn, err := deps.ConnectionRepo.Find(ctx, tenantID, connectionID)
	if err != nil || conn == nil {
		return err
	}
	conn.ConsecutiveFailureCount++
	quarantining := conn.ConsecutiveFailureCount >= conn.QuarantineAfterConsecutiveFailure && conn.Health != domain.HealthQuarantined
	if quarantining {
		if err := conn.Quarantine(reason, now); err != nil {
			return err
		}
	}
	if err := deps.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		return err
	}
	if quarantining {
		emit(deps.Emit, &domain.ConnectionQuarantined{
			At: now, TenantID: tenantID, ApplicationID: conn.ApplicationID, Reason: reason, ConsecutiveFailures: conn.ConsecutiveFailureCount,
		})
	}
	return nil
}
