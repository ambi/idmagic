package handlers_http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/internal/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// scriptedPusher は受信側の障害と復旧を切り替え、届いた SET の本体を記録する。
type scriptedPusher struct {
	err      error
	received []string
}

func (p *scriptedPusher) Push(_ context.Context, _, _, compactSET string) error {
	if p.err != nil {
		return p.err
	}
	p.received = append(p.received, compactSET)
	return nil
}

func redeliverPath(streamID string) string {
	return "/api/admin/v1/shared-signals/streams/" + streamID + "/deliveries/redeliver"
}

func (s *exampleStack) deliverDue(t *testing.T, pusher *scriptedPusher, now time.Time) {
	t.Helper()
	if _, err := ssusecases.ProcessDueDeliveries(context.Background(), ssusecases.DeliverDeps{
		DeliveryRepo: s.deliveries, TransmitterConfigRepo: s.transmitterConfigs, Pusher: pusher,
	}, now, 10); err != nil {
		t.Fatalf("ProcessDueDeliveries: %v", err)
	}
}

func (s *exampleStack) onlyDelivery(t *testing.T, streamID string) *ssdomain.SecurityEventDelivery {
	t.Helper()
	deliveries, err := s.deliveries.ListByStream(context.Background(), tenancydomain.DefaultTenantID, streamID)
	if err != nil || len(deliveries) != 1 {
		t.Fatalf("deliveries=%+v err=%v, want one", deliveries, err)
	}
	return deliveries[0]
}

// deadLetterOneDelivery は強制終了で配送を作り、受信側の障害で試行を使い切らせる。
func (s *exampleStack) deadLetterOneDelivery(t *testing.T, streamID string) (*ssdomain.SecurityEventDelivery, time.Time) {
	t.Helper()
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	s.killAgent(t)
	now := time.Now().UTC()
	down := &scriptedPusher{err: errors.New("receiver down")}
	for range ssdomain.DefaultMaxDeliveryAttempts {
		now = now.Add(time.Hour)
		s.deliverDue(t, down, now)
	}
	d := s.onlyDelivery(t, streamID)
	if d.Status != ssdomain.SecurityEventDeliveryStatusDeadLetter {
		t.Fatalf("delivery=%+v, want dead_letter", d)
	}
	return d, now
}

func assertRedeliveredCount(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	var body struct {
		RedeliveredCount *int `json:"redelivered_count"`
	}
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") ||
		json.Unmarshal(rec.Body.Bytes(), &body) != nil || body.RedeliveredCount == nil || *body.RedeliveredCount != want {
		t.Fatalf("status=%d headers=%v body=%s, want 200 JSON with redelivered_count=%d", rec.Code, rec.Header(), rec.Body.String(), want)
	}
}

//spec:covers EX-SHAREDSIGNALS-013-01: 管理 API のやり直しで dead_letter の配送が試行の回数 0 の pending に戻り、復旧した受信側へ同じ SET が届いて delivered になる。重複した要求は何も戻さず、イベントも増やさない。
//verify:operation RedeliverDeadLetteredDeliveries: 200 JSON は戻した件数を返し、対象がなければ 0 を返す。
func TestRedeliverDeadLetteredDeliveriesReachRecoveredReceiver(t *testing.T) {
	s := newExampleStack(t)
	streamID := s.registerTransmitter(t, "https://receiver.example/events")
	dead, now := s.deadLetterOneDelivery(t, streamID)

	assertRedeliveredCount(t, s.admin(t, http.MethodPost, redeliverPath(streamID), nil), 1)
	d := s.onlyDelivery(t, streamID)
	if d.ID != dead.ID || d.Status != ssdomain.SecurityEventDeliveryStatusPending || d.AttemptCount != 0 || d.NextAttemptAt != nil || d.LastError != nil {
		t.Fatalf("delivery=%+v, want the same delivery reset to pending", d)
	}
	if got := s.events.count("SecurityEventDeliveryRedeliveryRequested"); got != 1 {
		t.Fatalf("SecurityEventDeliveryRedeliveryRequested=%d, want 1", got)
	}

	assertRedeliveredCount(t, s.admin(t, http.MethodPost, redeliverPath(streamID), nil), 0)
	if got := s.events.count("SecurityEventDeliveryRedeliveryRequested"); got != 1 {
		t.Fatalf("a repeated request emitted again: %d", got)
	}

	recovered := &scriptedPusher{}
	s.deliverDue(t, recovered, now.Add(time.Second))
	d = s.onlyDelivery(t, streamID)
	if d.Status != ssdomain.SecurityEventDeliveryStatusDelivered || d.SetJTI != dead.SetJTI {
		t.Fatalf("delivery=%+v, want delivered with jti %s", d, dead.SetJTI)
	}
	if len(recovered.received) != 1 || recovered.received[0] != dead.Set.Compact {
		t.Fatalf("receiver got %v, want the original SET once", recovered.received)
	}
}

//spec:covers REQ-SHAREDSIGNALS-013: やり直した配送は上限の分だけ再び試行し、また使い切れば dead_letter に戻る。
func TestRedeliveredDeliveryGetsAFullRetryBudget(t *testing.T) {
	s := newExampleStack(t)
	streamID := s.registerTransmitter(t, "https://receiver.example/events")
	_, now := s.deadLetterOneDelivery(t, streamID)
	assertRedeliveredCount(t, s.admin(t, http.MethodPost, redeliverPath(streamID), nil), 1)

	down := &scriptedPusher{err: errors.New("receiver still down")}
	for range ssdomain.DefaultMaxDeliveryAttempts - 1 {
		now = now.Add(time.Hour)
		s.deliverDue(t, down, now)
		if d := s.onlyDelivery(t, streamID); d.Status != ssdomain.SecurityEventDeliveryStatusFailed {
			t.Fatalf("delivery=%+v, want failed before the budget is used up", d)
		}
	}
	s.deliverDue(t, down, now.Add(time.Hour))
	if d := s.onlyDelivery(t, streamID); d.Status != ssdomain.SecurityEventDeliveryStatusDeadLetter {
		t.Fatalf("delivery=%+v, want dead_letter again", d)
	}
}

//spec:covers EX-SHAREDSIGNALS-013-02: 無効化したストリームのやり直しは 409 で拒否され、dead_letter の配送もイベントも変わらない。
//verify:operation RedeliverDeadLetteredDeliveries: 無効化したストリームには 409 ssf_stream_disabled の Problem Details を返す。
func TestRedeliverRefusesDisabledStream(t *testing.T) {
	s := newExampleStack(t)
	streamID := s.registerTransmitter(t, "https://receiver.example/events")
	s.deadLetterOneDelivery(t, streamID)
	assertStreamNoContent(t, s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/"+streamID+"/disable", nil))

	rec := s.admin(t, http.MethodPost, redeliverPath(streamID), nil)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "ssf_stream_disabled") {
		t.Fatalf("status=%d body=%s, want 409 ssf_stream_disabled", rec.Code, rec.Body.String())
	}
	if d := s.onlyDelivery(t, streamID); d.Status != ssdomain.SecurityEventDeliveryStatusDeadLetter {
		t.Fatalf("delivery=%+v, want it untouched", d)
	}
	if got := s.events.count("SecurityEventDeliveryRedeliveryRequested"); got != 0 {
		t.Fatalf("a refused request emitted %d events", got)
	}
}

//spec:covers REQ-SHAREDSIGNALS-013: 存在しないストリームのやり直しは 404 ssf_stream_not_found で拒否する。
//verify:operation RedeliverDeadLetteredDeliveries: 未知のストリームには 404 ssf_stream_not_found の Problem Details を返す。
func TestRedeliverRejectsUnknownStream(t *testing.T) {
	s := newExampleStack(t)
	rec := s.admin(t, http.MethodPost, redeliverPath("00000000-0000-4000-8000-000000000000"), nil)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "ssf_stream_not_found") {
		t.Fatalf("status=%d body=%s, want 404 ssf_stream_not_found", rec.Code, rec.Body.String())
	}
}

//spec:covers REQ-SHAREDSIGNALS-011: 管理者でない利用者の配送のやり直しを拒否し、配送を変えない。
//verify:operation RedeliverDeadLetteredDeliveries: 非管理者には 403 access_denied の Problem Details を返す。
func TestRedeliverRefusesNonAdmin(t *testing.T) {
	s := newExampleStack(t)
	streamID := s.registerTransmitter(t, "https://receiver.example/events")
	s.deadLetterOneDelivery(t, streamID)
	assertAccessDenied(t, s.request(t, exampleNonAdmin, http.MethodPost, redeliverPath(streamID), nil))
	if d := s.onlyDelivery(t, streamID); d.Status != ssdomain.SecurityEventDeliveryStatusDeadLetter {
		t.Fatalf("delivery=%+v, want it untouched", d)
	}
}
