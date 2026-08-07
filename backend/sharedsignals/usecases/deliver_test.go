package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
)

// fakePusher records Push calls and returns a scripted error (nil = success).
type fakePusher struct {
	err   error
	calls int
}

func (f *fakePusher) Push(_ context.Context, _, _, _ string) error {
	f.calls++
	return f.err
}

const deliveryTestTenantID = "tenant-a"

func seedDelivery(t *testing.T, repo *ssmemory.SecurityEventDeliveryRepository, streamID string, status ssdomain.SecurityEventDeliveryStatus, attemptCount int) {
	t.Helper()
	const id = "d1"
	d := &ssdomain.SecurityEventDelivery{
		ID: id, TenantID: deliveryTestTenantID, StreamID: streamID, SetJTI: "jti_" + id,
		Set: ssdomain.SecurityEventToken{
			JTI: "jti_" + id, Issuer: "https://idp.example", Audience: "https://receiver.example", IssuedAt: time.Now().UTC(),
			Event: ssdomain.CaepEvent{
				EventType:      ssdomain.CaepEventTypeSessionRevoked,
				Subject:        ssdomain.SsfSubject{SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: deliveryTestTenantID, PrincipalID: "agent_1"},
				EventTimestamp: time.Now().UTC(), InitiatingEntity: ssdomain.InitiatingEntityAdmin,
			},
			Compact: "header.payload.sig",
		},
		Status: status, AttemptCount: attemptCount, CreatedAt: time.Now().UTC(),
	}
	if err := repo.Save(context.Background(), d); err != nil {
		t.Fatalf("seed delivery %s: %v", id, err)
	}
}

func seedConfigForStream(t *testing.T, repo *ssmemory.SsfTransmitterConfigRepository, maxAttempts int) {
	t.Helper()
	const streamID = "stream_1"
	if err := repo.Save(context.Background(), deliveryTestTenantID, &ssdomain.SsfTransmitterConfig{
		StreamID: streamID, DeliveryEndpoint: "https://receiver.example/" + streamID, Audience: "https://receiver.example/" + streamID,
		MaxDeliveryAttempts: maxAttempts,
	}); err != nil {
		t.Fatalf("seed config for %s: %v", streamID, err)
	}
}

// TestProcessDueDeliveries_SuccessMarksDelivered — RED: push が成功したら
// delivered へ遷移し SecurityEventTransmitted を emit する。
func TestProcessDueDeliveries_SuccessMarksDelivered(t *testing.T) {
	ctx := context.Background()
	deliveryRepo := ssmemory.NewSecurityEventDeliveryRepository()
	configRepo := ssmemory.NewSsfTransmitterConfigRepository()
	seedConfigForStream(t, configRepo, 3)
	seedDelivery(t, deliveryRepo, "stream_1", ssdomain.SecurityEventDeliveryStatusPending, 0)

	var emitted []spec.DomainEvent
	pusher := &fakePusher{}
	deps := ssusecases.DeliverDeps{
		DeliveryRepo: deliveryRepo, TransmitterConfigRepo: configRepo, Pusher: pusher,
		Emit: func(e spec.DomainEvent) error { emitted = append(emitted, e); return nil },
	}
	now := time.Now().UTC()
	n, err := ssusecases.ProcessDueDeliveries(ctx, deps, now, 10)
	if err != nil {
		t.Fatalf("ProcessDueDeliveries: %v", err)
	}
	if n != 1 || pusher.calls != 1 {
		t.Fatalf("processed=%d pusher.calls=%d, want 1/1", n, pusher.calls)
	}

	deliveries, _ := deliveryRepo.ListByStream(ctx, "tenant-a", "stream_1")
	if len(deliveries) != 1 || deliveries[0].Status != ssdomain.SecurityEventDeliveryStatusDelivered || deliveries[0].DeliveredAt == nil {
		t.Fatalf("unexpected delivery state: %+v", deliveries)
	}
	if len(emitted) != 1 || emitted[0].EventType() != "SecurityEventTransmitted" {
		t.Fatalf("expected SecurityEventTransmitted, got %+v", emitted)
	}
}

// TestProcessDueDeliveries_FailureBelowMaxSchedulesBackoffRetry — RED: 失敗かつ
// max_delivery_attempts 未満なら failed へ遷移し next_attempt_at を exponential
// backoff で設定、SecurityEventDeliveryFailed を emit する。
func TestProcessDueDeliveries_FailureBelowMaxSchedulesBackoffRetry(t *testing.T) {
	ctx := context.Background()
	deliveryRepo := ssmemory.NewSecurityEventDeliveryRepository()
	configRepo := ssmemory.NewSsfTransmitterConfigRepository()
	seedConfigForStream(t, configRepo, 5)
	seedDelivery(t, deliveryRepo, "stream_1", ssdomain.SecurityEventDeliveryStatusPending, 0)

	var emitted []spec.DomainEvent
	pusher := &fakePusher{err: errors.New("receiver unreachable")}
	deps := ssusecases.DeliverDeps{
		DeliveryRepo: deliveryRepo, TransmitterConfigRepo: configRepo, Pusher: pusher,
		Emit: func(e spec.DomainEvent) error { emitted = append(emitted, e); return nil },
	}
	now := time.Now().UTC()
	if _, err := ssusecases.ProcessDueDeliveries(ctx, deps, now, 10); err != nil {
		t.Fatalf("ProcessDueDeliveries: %v", err)
	}

	deliveries, _ := deliveryRepo.ListByStream(ctx, "tenant-a", "stream_1")
	d := deliveries[0]
	if d.Status != ssdomain.SecurityEventDeliveryStatusFailed || d.AttemptCount != 1 {
		t.Fatalf("unexpected delivery state: %+v", d)
	}
	if d.NextAttemptAt == nil || !d.NextAttemptAt.After(now) {
		t.Fatalf("expected next_attempt_at scheduled in the future, got %+v", d.NextAttemptAt)
	}
	if d.LastError == nil || *d.LastError == "" {
		t.Fatalf("expected last_error to be recorded")
	}
	if len(emitted) != 1 || emitted[0].EventType() != "SecurityEventDeliveryFailed" {
		t.Fatalf("expected SecurityEventDeliveryFailed, got %+v", emitted)
	}
}

// TestProcessDueDeliveries_ExhaustingMaxAttemptsDeadLetters — RED: SCL シナリオ
// 「配送失敗は再試行され上限超過でdead_letterへ遷移する」: max_delivery_attempts=3
// で3回連続失敗すると3回目で dead_letter へ遷移する。
func TestProcessDueDeliveries_ExhaustingMaxAttemptsDeadLetters(t *testing.T) {
	ctx := context.Background()
	deliveryRepo := ssmemory.NewSecurityEventDeliveryRepository()
	configRepo := ssmemory.NewSsfTransmitterConfigRepository()
	seedConfigForStream(t, configRepo, 3)
	seedDelivery(t, deliveryRepo, "stream_1", ssdomain.SecurityEventDeliveryStatusPending, 0)

	var emitted []spec.DomainEvent
	pusher := &fakePusher{err: errors.New("receiver unreachable")}
	deps := ssusecases.DeliverDeps{
		DeliveryRepo: deliveryRepo, TransmitterConfigRepo: configRepo, Pusher: pusher,
		Emit: func(e spec.DomainEvent) error { emitted = append(emitted, e); return nil },
	}

	// now を毎回 1 時間ずつ進めて呼ぶ: backoff の上限 (30分) より確実に大きい
	// 間隔にすることで、直前の next_attempt_at が次の呼び出し時点で必ず due に
	// なるようにする。
	now := time.Now().UTC()

	// 1回目・2回目: failed へ。
	for i := range 2 {
		if _, err := ssusecases.ProcessDueDeliveries(ctx, deps, now, 10); err != nil {
			t.Fatalf("attempt %d: ProcessDueDeliveries: %v", i+1, err)
		}
		now = now.Add(time.Hour)
	}
	mid, _ := deliveryRepo.ListByStream(ctx, "tenant-a", "stream_1")
	if mid[0].Status != ssdomain.SecurityEventDeliveryStatusFailed || mid[0].AttemptCount != 2 {
		t.Fatalf("expected failed/attempt=2 after 2 failures, got %+v", mid[0])
	}

	// 3回目: max_delivery_attempts に到達し dead_letter へ。
	if _, err := ssusecases.ProcessDueDeliveries(ctx, deps, now, 10); err != nil {
		t.Fatalf("attempt 3: ProcessDueDeliveries: %v", err)
	}
	final, _ := deliveryRepo.ListByStream(ctx, "tenant-a", "stream_1")
	if final[0].Status != ssdomain.SecurityEventDeliveryStatusDeadLetter || final[0].AttemptCount != 3 {
		t.Fatalf("expected dead_letter/attempt=3, got %+v", final[0])
	}
	if pusher.calls != 3 {
		t.Fatalf("expected 3 push attempts, got %d", pusher.calls)
	}
	lastEvent := emitted[len(emitted)-1]
	if lastEvent.EventType() != "SecurityEventDeliveryDeadLettered" {
		t.Fatalf("expected the last emitted event to be SecurityEventDeliveryDeadLettered, got %s", lastEvent.EventType())
	}
}

// TestProcessDueDeliveries_RetryEmitsRetriedBeforeOutcome — RED: 直前が failed
// だった delivery を再試行するときは、結果イベントの前に
// SecurityEventDeliveryRetried を emit する。
func TestProcessDueDeliveries_RetryEmitsRetriedBeforeOutcome(t *testing.T) {
	ctx := context.Background()
	deliveryRepo := ssmemory.NewSecurityEventDeliveryRepository()
	configRepo := ssmemory.NewSsfTransmitterConfigRepository()
	seedConfigForStream(t, configRepo, 5)
	seedDelivery(t, deliveryRepo, "stream_1", ssdomain.SecurityEventDeliveryStatusFailed, 1)

	var emitted []spec.DomainEvent
	pusher := &fakePusher{}
	deps := ssusecases.DeliverDeps{
		DeliveryRepo: deliveryRepo, TransmitterConfigRepo: configRepo, Pusher: pusher,
		Emit: func(e spec.DomainEvent) error { emitted = append(emitted, e); return nil },
	}
	if _, err := ssusecases.ProcessDueDeliveries(ctx, deps, time.Now().UTC(), 10); err != nil {
		t.Fatalf("ProcessDueDeliveries: %v", err)
	}
	if len(emitted) != 2 || emitted[0].EventType() != "SecurityEventDeliveryRetried" || emitted[1].EventType() != "SecurityEventTransmitted" {
		t.Fatalf("expected [Retried, Transmitted], got %+v", emitted)
	}
}

// TestProcessDueDeliveries_MissingTransmitterConfigFailsClosed — RED: stream の
// SsfTransmitterConfig が消えている (整合性が崩れたデータ) 場合は push を試みず
// 通常の失敗として扱う (エラーで batch 全体を止めない)。
func TestProcessDueDeliveries_MissingTransmitterConfigFailsClosed(t *testing.T) {
	ctx := context.Background()
	deliveryRepo := ssmemory.NewSecurityEventDeliveryRepository()
	configRepo := ssmemory.NewSsfTransmitterConfigRepository() // no config seeded
	seedDelivery(t, deliveryRepo, "orphan_stream", ssdomain.SecurityEventDeliveryStatusPending, 0)

	pusher := &fakePusher{}
	deps := ssusecases.DeliverDeps{DeliveryRepo: deliveryRepo, TransmitterConfigRepo: configRepo, Pusher: pusher}
	if _, err := ssusecases.ProcessDueDeliveries(ctx, deps, time.Now().UTC(), 10); err != nil {
		t.Fatalf("ProcessDueDeliveries: %v", err)
	}
	if pusher.calls != 0 {
		t.Fatalf("expected push not to be attempted without a transmitter config, got %d calls", pusher.calls)
	}
	deliveries, _ := deliveryRepo.ListByStream(ctx, "tenant-a", "orphan_stream")
	if deliveries[0].Status != ssdomain.SecurityEventDeliveryStatusFailed {
		t.Fatalf("expected failed status, got %+v", deliveries[0])
	}
}
