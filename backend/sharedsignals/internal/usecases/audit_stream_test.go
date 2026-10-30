package usecases_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/internal/db_memory"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
)

type auditFixture struct {
	streams    *ssmemory.SsfStreamRepository
	configs    *ssmemory.SsfTransmitterConfigRepository
	deliveries *ssmemory.SecurityEventDeliveryRepository
	signer     *capturingSigner
	pusher     *fakePusher
	emitted    *[]spec.DomainEvent
	deps       ssusecases.DeliverDeps
	stream     *ssdomain.SsfStream
	now        time.Time
}

func newAuditFixture(t *testing.T, auditTypes []string) *auditFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	f := &auditFixture{
		streams: ssmemory.NewSsfStreamRepository(), configs: ssmemory.NewSsfTransmitterConfigRepository(),
		deliveries: ssmemory.NewSecurityEventDeliveryRepository(), signer: &capturingSigner{}, pusher: &fakePusher{},
		emitted: &[]spec.DomainEvent{}, now: now,
	}
	f.stream = &ssdomain.SsfStream{ID: "s1", TenantID: "tenant-a", Direction: ssdomain.SsfStreamDirectionTransmit, AuditEventTypes: auditTypes, Status: ssdomain.SsfStreamStatusEnabled, CreatedAt: now}
	if err := f.streams.Save(ctx, f.stream); err != nil {
		t.Fatal(err)
	}
	if err := f.configs.Save(ctx, "tenant-a", &ssdomain.SsfTransmitterConfig{StreamID: "s1", DeliveryEndpoint: "https://receiver.example", Audience: "aud", MaxDeliveryAttempts: 8}); err != nil {
		t.Fatal(err)
	}
	f.deps = ssusecases.DeliverDeps{
		DeliveryRepo: f.deliveries, TransmitterConfigRepo: f.configs, StreamRepo: f.streams, Pusher: f.pusher,
		Signer: f.signer, Issuer: "https://idp.example",
		Emit: func(e spec.DomainEvent) error { *f.emitted = append(*f.emitted, e); return nil },
	}
	return f
}

func (f *auditFixture) capture(t *testing.T, id, eventType string) {
	t.Helper()
	event := ssdomain.NewAuditSetEvent(id, eventType, "tenant-a", f.now, map[string]any{"userId": "u1", "targetUserId": "u2", "email": "a@example.com"})
	if err := ssusecases.CaptureAuditEvent(context.Background(), f.streams, f.deliveries, event, f.now); err != nil {
		t.Fatal(err)
	}
}

func (f *auditFixture) list(t *testing.T) []*ssdomain.SecurityEventDelivery {
	t.Helper()
	deliveries, err := f.deliveries.ListByStream(context.Background(), "tenant-a", "s1")
	if err != nil {
		t.Fatal(err)
	}
	return deliveries
}

//spec:covers REQ-SHAREDSIGNALS-016: 署名する claims は許可した識別子だけで、再試行で jti が変わらず、成功のイベントを発行しない。
func TestAuditDeliverySignsMinimalSet(t *testing.T) {
	f := newAuditFixture(t, []string{"*"})
	f.capture(t, "e1", "UserUpdated")
	f.pusher.err = errors.New("receiver unreachable")
	if _, err := ssusecases.ProcessDueDeliveries(context.Background(), f.deps, f.now, 10); err != nil {
		t.Fatal(err)
	}
	firstJTI := f.signer.claims["jti"]
	claims := f.signer.claims
	events := claims["events"].(map[string]any)
	body, ok := events["urn:idmagic:secevent:audit:UserUpdated"].(map[string]any)
	if !ok {
		t.Fatalf("events = %#v", events)
	}
	if body["audit_event_id"] != "e1" || body["target_id"] != "u2" || body["tenant_id"] != "tenant-a" {
		t.Fatalf("body = %#v", body)
	}
	if strings.Contains(fmt.Sprint(claims), "a@example.com") {
		t.Fatalf("claims contain personal data: %v", claims)
	}
	if claims["sub_id"].(map[string]any)["id"] != "u2" || claims["aud"] != "aud" || claims["iss"] != "https://idp.example" {
		t.Fatalf("claims = %#v", claims)
	}
	f.pusher.err = nil
	if _, err := ssusecases.ProcessDueDeliveries(context.Background(), f.deps, f.now.Add(time.Minute), 10); err != nil {
		t.Fatal(err)
	}
	if f.signer.claims["jti"] != firstJTI {
		t.Fatal("jti changed between attempts")
	}
	got := f.list(t)
	if got[0].Status != ssdomain.SecurityEventDeliveryStatusDelivered {
		t.Fatalf("delivery = %+v", got[0])
	}
	for _, e := range *f.emitted {
		if e.EventType() == "SecurityEventTransmitted" || e.EventType() == "SecurityEventDeliveryRetried" {
			t.Fatalf("emitted %s for an audit delivery", e.EventType())
		}
	}
}

//spec:covers REQ-SHAREDSIGNALS-015: 購読の種類、状態、向きで配送を作るかが決まる。
//spec:covers EX-SHAREDSIGNALS-015-02: 配送の結果のイベントは配送を生まない。
func TestCaptureSelectsSubscribedStreams(t *testing.T) {
	f := newAuditFixture(t, []string{"UserCreated"})
	f.capture(t, "other", "GroupCreated")
	f.capture(t, "outcome", "SecurityEventDeliveryFailed")
	if got := f.list(t); len(got) != 0 {
		t.Fatalf("deliveries = %d", len(got))
	}
	f.capture(t, "match", "UserCreated")
	if got := f.list(t); len(got) != 1 || got[0].AuditEvent == nil || got[0].AuditEvent.EventID != "match" || got[0].SetJTI == "" {
		t.Fatalf("deliveries = %+v", got)
	}
	disabled := *f.stream
	disabled.Status = ssdomain.SsfStreamStatusDisabled
	if err := f.streams.Save(context.Background(), &disabled); err != nil {
		t.Fatal(err)
	}
	f.capture(t, "while-disabled", "UserCreated")
	if got := f.list(t); len(got) != 1 {
		t.Fatalf("captured while disabled: %d", len(got))
	}
}

//spec:covers REQ-SHAREDSIGNALS-018: 閾値で無効化し、配送が止まり、再有効化で再開する。
//spec:covers EX-SHAREDSIGNALS-018-01: 閾値で無効化する。
//spec:covers SSF10-STATUS-NO-TRANSMIT: disabled のストリームへ送らない。
func TestConsecutiveFailuresDisableStream(t *testing.T) {
	f := newAuditFixture(t, []string{"*"})
	f.pusher.err = errors.New("receiver unreachable")
	now := f.now
	for i := range ssdomain.AutoDisableFailures {
		f.capture(t, fmt.Sprintf("e%d", i), "UserCreated")
		if _, err := ssusecases.ProcessDueDeliveries(context.Background(), f.deps, now, 100); err != nil {
			t.Fatal(err)
		}
		now = now.Add(4 * time.Minute)
	}
	stream, err := f.streams.FindByID(context.Background(), "tenant-a", "s1")
	if err != nil || stream.Status != ssdomain.SsfStreamStatusDisabled {
		t.Fatalf("stream = %+v err=%v", stream, err)
	}
	autoDisabled := 0
	for _, e := range *f.emitted {
		if e.EventType() == "SsfStreamAutoDisabled" {
			autoDisabled++
		}
	}
	if autoDisabled != 1 {
		t.Fatalf("SsfStreamAutoDisabled = %d", autoDisabled)
	}
	calls := f.pusher.calls
	later := now.Add(2 * time.Hour)
	if _, err := ssusecases.ProcessDueDeliveries(context.Background(), f.deps, later, 100); err != nil {
		t.Fatal(err)
	}
	if f.pusher.calls != calls {
		t.Fatal("sent to a disabled stream")
	}
	admin := ssusecases.AdminStreamDeps{StreamRepo: f.streams, Emit: f.deps.Emit}
	ctx := adminStreamTestCtx()
	if _, err := ssusecases.EnableSsfStream(ctx, admin, "s1", later); err != nil {
		t.Fatal(err)
	}
	stream, _ = f.streams.FindByID(context.Background(), "tenant-a", "s1")
	if stream.ConsecutiveFailures != 0 || stream.FirstFailureAt != nil {
		t.Fatalf("failures not cleared: %+v", stream)
	}
	f.pusher.err = nil
	if _, err := ssusecases.ProcessDueDeliveries(context.Background(), f.deps, later.Add(3*time.Minute), 100); err != nil {
		t.Fatal(err)
	}
	if f.pusher.calls == calls {
		t.Fatal("held deliveries did not resume")
	}
}

//spec:covers REQ-SHAREDSIGNALS-018: 失敗したらそのストリームの残りを送らず、残りの試行の回数を使わない。
func TestDeliveryStopsAtFirstFailurePerStream(t *testing.T) {
	f := newAuditFixture(t, []string{"*"})
	for i := range 5 {
		f.capture(t, fmt.Sprintf("e%d", i), "UserCreated")
	}
	f.pusher.err = errors.New("receiver unreachable")
	if _, err := ssusecases.ProcessDueDeliveries(context.Background(), f.deps, f.now, 100); err != nil {
		t.Fatal(err)
	}
	if f.pusher.calls != 1 {
		t.Fatalf("calls = %d", f.pusher.calls)
	}
	attempts := 0
	for _, d := range f.list(t) {
		attempts += d.AttemptCount
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}
