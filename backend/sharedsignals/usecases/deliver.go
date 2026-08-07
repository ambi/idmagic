package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

var errMissingTransmitterConfig = errors.New("sharedsignals: delivery has no transmitter config for its stream")

// deliveryBackoffBase/Cap mirror jobs.domain's exponential-backoff shape
// (base * 2^(attempts-1), capped) but are scoped to SecurityEventDelivery's
// own retry semantics rather than importing the Jobs context for a five-line
// formula.
const (
	deliveryBackoffBase = 30 * time.Second
	deliveryBackoffCap  = 30 * time.Minute
)

// nextAttemptDelay returns the exponential backoff delay before the next
// delivery attempt, given the attempt count that just failed.
func nextAttemptDelay(attemptCount int) time.Duration {
	if attemptCount < 1 {
		attemptCount = 1
	}
	delay := deliveryBackoffBase
	for i := 1; i < attemptCount; i++ {
		delay *= 2
		if delay >= deliveryBackoffCap {
			return deliveryBackoffCap
		}
	}
	return delay
}

// DeliverDeps holds what ProcessDueDeliveries needs.
type DeliverDeps struct {
	DeliveryRepo          ssports.SecurityEventDeliveryRepository
	TransmitterConfigRepo ssports.SsfTransmitterConfigRepository
	Pusher                ssports.SecurityEventPusher
	// Emit records SecurityEventTransmitted/SecurityEventDeliveryFailed/
	// SecurityEventDeliveryRetried/SecurityEventDeliveryDeadLettered for
	// audit (best-effort; nil skips).
	Emit func(spec.DomainEvent) error
}

// ProcessDueDeliveries implements the SecurityEventDeliveryLifecycle
// retry/backoff/dead-letter worker: it finds deliveries due for an attempt
// (SecurityEventDeliveryRepository.ListDue already applies the due
// condition), pushes each one, and persists the outcome — delivered,
// failed-with-backoff, or dead_letter once max_delivery_attempts is
// exhausted. A single delivery's push failure is expected, ordinary
// behavior (recorded in the delivery's own state) and never aborts the
// batch; only a repository failure (an infrastructure problem) does.
func ProcessDueDeliveries(ctx context.Context, deps DeliverDeps, now time.Time, limit int) (int, error) {
	due, err := deps.DeliveryRepo.ListDue(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	for i, d := range due {
		if err := deliverOne(ctx, deps, d, now); err != nil {
			return i, err
		}
	}
	return len(due), nil
}

func deliverOne(ctx context.Context, deps DeliverDeps, d *ssdomain.SecurityEventDelivery, now time.Time) error {
	wasFailed := d.Status == ssdomain.SecurityEventDeliveryStatusFailed
	if wasFailed {
		if err := emit(deps.Emit, &ssdomain.SecurityEventDeliveryRetried{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI, AttemptCount: d.AttemptCount}); err != nil {
			return err
		}
	}

	config, err := deps.TransmitterConfigRepo.FindByStream(ctx, d.TenantID, d.StreamID)
	if err != nil {
		return err
	}

	updated := *d
	updated.AttemptCount++

	pushErr := pushToConfig(ctx, deps.Pusher, config, d.Set.Compact)
	if pushErr == nil {
		updated.Status = ssdomain.SecurityEventDeliveryStatusDelivered
		updated.DeliveredAt = &now
		updated.NextAttemptAt = nil
		updated.LastError = nil
		if err := deps.DeliveryRepo.Save(ctx, &updated); err != nil {
			return err
		}
		return emit(deps.Emit, &ssdomain.SecurityEventTransmitted{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI})
	}

	errMsg := pushErr.Error()
	updated.LastError = &errMsg
	maxAttempts := ssdomain.DefaultMaxDeliveryAttempts
	if config != nil {
		maxAttempts = config.MaxDeliveryAttempts
	}
	if updated.AttemptCount >= maxAttempts {
		updated.Status = ssdomain.SecurityEventDeliveryStatusDeadLetter
		updated.NextAttemptAt = nil
		if err := deps.DeliveryRepo.Save(ctx, &updated); err != nil {
			return err
		}
		return emit(deps.Emit, &ssdomain.SecurityEventDeliveryDeadLettered{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI})
	}

	updated.Status = ssdomain.SecurityEventDeliveryStatusFailed
	next := now.Add(nextAttemptDelay(updated.AttemptCount))
	updated.NextAttemptAt = &next
	if err := deps.DeliveryRepo.Save(ctx, &updated); err != nil {
		return err
	}
	return emit(deps.Emit, &ssdomain.SecurityEventDeliveryFailed{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI, AttemptCount: updated.AttemptCount})
}

// pushToConfig pushes to config's delivery endpoint, or fails closed if the
// stream's transmitter config was deleted out from under a pending delivery
// (data inconsistency, not a normal transport failure).
func pushToConfig(ctx context.Context, pusher ssports.SecurityEventPusher, config *ssdomain.SsfTransmitterConfig, compactSET string) error {
	if config == nil {
		return errMissingTransmitterConfig
	}
	authorization := ""
	if config.DeliveryAuthorization != nil {
		authorization = *config.DeliveryAuthorization
	}
	return pusher.Push(ctx, config.DeliveryEndpoint, authorization, compactSET)
}
