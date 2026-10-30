package usecases

import (
	"context"
	"errors"
	"slices"
	"sync"
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
	// deliveryLease は取得した配送を別の周期が拾わない時間である。worker が止まると時刻の到来で戻る。
	deliveryLease = 2 * time.Minute
	// deliveriesPerStream と deliveryConcurrency は、遅い宛先がほかのストリームを待たせないための上限である。
	deliveriesPerStream = 10
	deliveryConcurrency = 8
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
	// StreamRepo は無効化されたストリームへ送らないことと、連続の失敗の記録に使う。
	StreamRepo ssports.SsfStreamRepository
	// Signer と Issuer は監査の SET を送信時に署名する。
	Signer ssports.SecurityEventTokenSigner
	Issuer string
	// Emit records SecurityEventTransmitted/SecurityEventDeliveryFailed/
	// SecurityEventDeliveryRetried/SecurityEventDeliveryDeadLettered for
	// audit (best-effort; nil skips).
	Emit func(spec.DomainEvent) error
}

// ProcessDueDeliveries implements the SecurityEventDeliveryLifecycle
// retry/backoff/dead-letter worker: it leases deliveries due for an attempt,
// pushes them stream by stream, and persists the outcome — delivered,
// failed-with-backoff, or dead_letter once max_delivery_attempts is
// exhausted. A push failure is expected, ordinary behavior (recorded in the
// delivery's own state); only a repository failure is returned.
func ProcessDueDeliveries(ctx context.Context, deps DeliverDeps, now time.Time, limit int) (int, error) {
	due, err := deps.DeliveryRepo.ClaimDue(ctx, now, now.Add(deliveryLease), deliveriesPerStream, limit)
	if err != nil {
		return 0, err
	}
	slices.SortStableFunc(due, func(a, b *ssdomain.SecurityEventDelivery) int { return a.CreatedAt.Compare(b.CreatedAt) })
	order := []string{}
	groups := map[string][]*ssdomain.SecurityEventDelivery{}
	for _, d := range due {
		key := d.TenantID + "/" + d.StreamID
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], d)
	}
	var wg sync.WaitGroup
	slots := make(chan struct{}, deliveryConcurrency)
	errs := make([]error, len(order))
	for i, key := range order {
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			errs[i] = deliverStream(ctx, deps, groups[key], now)
		})
	}
	wg.Wait()
	return len(due), errors.Join(errs...)
}

// deliverStream は一つのストリームの配送を順に送る。失敗したら残りを送らない。死んだ宛先へ
// 送信の期限の分だけ待ち続けないためで、残りはリースの失効で試行の回数を使わずに戻る。
func deliverStream(ctx context.Context, deps DeliverDeps, deliveries []*ssdomain.SecurityEventDelivery, now time.Time) error {
	first := deliveries[0]
	if deps.StreamRepo != nil {
		stream, err := deps.StreamRepo.FindByID(ctx, first.TenantID, first.StreamID)
		if err != nil {
			return err
		}
		// 無効化されたストリームへは送らない。行はそのまま残り、再有効化で再開する。
		if stream != nil && !stream.IsEnabled() {
			return nil
		}
	}
	for _, d := range deliveries {
		delivered, err := deliverOne(ctx, deps, d, now)
		if err != nil {
			return err
		}
		if !delivered {
			return recordStreamFailure(ctx, deps, first, now)
		}
	}
	if deps.StreamRepo != nil {
		return deps.StreamRepo.RecordDeliverySuccess(ctx, first.TenantID, first.StreamID)
	}
	return nil
}

func recordStreamFailure(ctx context.Context, deps DeliverDeps, d *ssdomain.SecurityEventDelivery, now time.Time) error {
	if deps.StreamRepo == nil {
		return nil
	}
	disabled, err := deps.StreamRepo.RecordDeliveryFailure(ctx, d.TenantID, d.StreamID, now)
	if err != nil || !disabled {
		return err
	}
	return emit(deps.Emit, &ssdomain.SsfStreamAutoDisabled{At: now, TenantID: d.TenantID, StreamID: d.StreamID})
}

// compactFor は送る SET を返す。監査の配送は送信時に署名する。捕捉が監査の Tx の中にあり、
// 鍵の取得と署名を入れないためである。jti は配送の行に固定するので、再試行でも変わらない。
func compactFor(ctx context.Context, deps DeliverDeps, d *ssdomain.SecurityEventDelivery, config *ssdomain.SsfTransmitterConfig) (string, error) {
	if d.AuditEvent == nil {
		return d.Set.Compact, nil
	}
	if config == nil {
		return "", errMissingTransmitterConfig
	}
	return SignAuditSecurityEventToken(ctx, deps.Signer, deps.Issuer, config.Audience, d.SetJTI, *d.AuditEvent)
}

func deliverOne(ctx context.Context, deps DeliverDeps, d *ssdomain.SecurityEventDelivery, now time.Time) (bool, error) {
	// 監査の配送の成功と再試行は発行しない。監査の量が倍になり、配送の行が履歴を持つためである。
	isAudit := d.AuditEvent != nil
	if d.Status == ssdomain.SecurityEventDeliveryStatusFailed && !isAudit {
		if err := emit(deps.Emit, &ssdomain.SecurityEventDeliveryRetried{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI, AttemptCount: d.AttemptCount}); err != nil {
			return false, err
		}
	}

	config, err := deps.TransmitterConfigRepo.FindByStream(ctx, d.TenantID, d.StreamID)
	if err != nil {
		return false, err
	}

	updated := *d
	updated.AttemptCount++

	compact, pushErr := compactFor(ctx, deps, d, config)
	if pushErr == nil {
		pushErr = pushToConfig(ctx, deps.Pusher, config, compact)
	}
	if pushErr == nil {
		updated.Status = ssdomain.SecurityEventDeliveryStatusDelivered
		updated.DeliveredAt = &now
		updated.NextAttemptAt = nil
		updated.LastError = nil
		if err := deps.DeliveryRepo.Save(ctx, &updated); err != nil {
			return false, err
		}
		if isAudit {
			return true, nil
		}
		return true, emit(deps.Emit, &ssdomain.SecurityEventTransmitted{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI})
	}

	errMsg := pushErr.Error()
	updated.LastError = &errMsg
	maxAttempts := ssdomain.DefaultMaxDeliveryAttempts
	if config != nil {
		maxAttempts = config.MaxDeliveryAttempts
	}
	// 失敗した試行はすべて failed を経る (states.md の pending → failed)。上限に
	// 達した試行はそこから dead_letter へ進むので、Failed の後に DeadLettered が続く。
	exhausted := updated.AttemptCount >= maxAttempts
	if exhausted {
		updated.Status = ssdomain.SecurityEventDeliveryStatusDeadLetter
		updated.NextAttemptAt = nil
	} else {
		updated.Status = ssdomain.SecurityEventDeliveryStatusFailed
		next := now.Add(nextAttemptDelay(updated.AttemptCount))
		updated.NextAttemptAt = &next
	}
	if err := deps.DeliveryRepo.Save(ctx, &updated); err != nil {
		return false, err
	}
	if err := emit(deps.Emit, &ssdomain.SecurityEventDeliveryFailed{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI, AttemptCount: updated.AttemptCount}); err != nil {
		return false, err
	}
	if !exhausted {
		return false, nil
	}
	return false, emit(deps.Emit, &ssdomain.SecurityEventDeliveryDeadLettered{At: now, TenantID: d.TenantID, StreamID: d.StreamID, SetJTI: d.SetJTI})
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

// CaptureAuditEvent は監査の記録の追記と同時に、購読するストリームごとの配送を作る。
// メモリの保存先の経路であり、PostgreSQL は同じ規則を一文で監査の Tx の中で行う。
func CaptureAuditEvent(ctx context.Context, streams ssports.SsfStreamRepository, deliveries ssports.SecurityEventDeliveryRepository, event ssdomain.AuditSetEvent, now time.Time) error {
	all, err := streams.ListAll(ctx, event.TenantID)
	if err != nil {
		return err
	}
	for _, stream := range all {
		if stream.Direction != ssdomain.SsfStreamDirectionTransmit || !stream.IsEnabled() || !stream.SubscribesAudit(event.EventType) {
			continue
		}
		id, err := ssdomain.NewSecurityEventDeliveryID()
		if err != nil {
			return err
		}
		jti, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		captured := event
		if err := deliveries.Save(ctx, &ssdomain.SecurityEventDelivery{
			ID: id, TenantID: stream.TenantID, StreamID: stream.ID, SetJTI: jti, AuditEvent: &captured,
			Status: ssdomain.SecurityEventDeliveryStatusPending, CreatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}
