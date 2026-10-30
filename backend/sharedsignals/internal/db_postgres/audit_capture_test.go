package db_postgres

import (
	"context"
	"testing"
	"time"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func captureFixture(t *testing.T, auditTypes []string) (*SsfStreamRepository, *SecurityEventDeliveryRepository, *ssdomain.SsfStream) {
	t.Helper()
	db := pgtest.Require(t)
	streams := &SsfStreamRepository{Pool: db}
	stream := newStream(t, seedTenant(t, db).ID, ssdomain.SsfStreamDirectionTransmit)
	stream.AuditEventTypes = auditTypes
	if err := streams.Save(context.Background(), stream); err != nil {
		t.Fatal(err)
	}
	return streams, &SecurityEventDeliveryRepository{Pool: db}, stream
}

//spec:covers REQ-SHAREDSIGNALS-015: 監査と同じ Tx の巻き戻しで配送が残らず、確定すれば残る。
//spec:covers EX-SHAREDSIGNALS-015-01: 追記が巻き戻ると配送も残らない。
func TestAuditCaptureRollsBackWithTransaction(t *testing.T) {
	_, deliveries, stream := captureFixture(t, []string{"*"})
	ctx := context.Background()
	now := pgtest.Now()
	event := ssdomain.NewAuditSetEvent("e1", "UserCreated", stream.TenantID, now, map[string]any{"userId": "u1"})
	for _, commit := range []bool{false, true} {
		tx, err := deliveries.Pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := CaptureAuditEventInTx(ctx, tx, event, now); err != nil {
			t.Fatal(err)
		}
		if commit {
			err = tx.Commit(ctx)
		} else {
			err = tx.Rollback(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		got, err := deliveries.ListByStream(ctx, stream.TenantID, stream.ID)
		if err != nil {
			t.Fatal(err)
		}
		if commit {
			if len(got) != 1 || got[0].AuditEvent == nil || got[0].AuditEvent.EventID != "e1" || got[0].SetJTI == "" {
				t.Fatalf("committed: %+v", got)
			}
		} else if len(got) != 0 {
			t.Fatalf("rolled back: %+v", got)
		}
	}
}

//spec:covers REQ-SHAREDSIGNALS-015: 購読しない種類、配送の結果の種類、テナントのない記録は配送を作らない。
func TestAuditCaptureSkips(t *testing.T) {
	_, deliveries, stream := captureFixture(t, []string{"UserCreated"})
	ctx := context.Background()
	now := pgtest.Now()
	tx, err := deliveries.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, event := range []ssdomain.AuditSetEvent{
		{EventID: "a", EventType: "GroupCreated", TenantID: stream.TenantID, OccurredAt: now},
		{EventID: "b", EventType: "SecurityEventDeliveryFailed", TenantID: stream.TenantID, OccurredAt: now},
		{EventID: "c", EventType: "UserCreated", TenantID: "not-a-uuid", OccurredAt: now},
	} {
		if err := CaptureAuditEventInTx(ctx, tx, event, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if got, _ := deliveries.ListByStream(ctx, stream.TenantID, stream.ID); len(got) != 0 {
		t.Fatalf("deliveries = %+v", got)
	}
}

//spec:covers REQ-SHAREDSIGNALS-018: 並行した取得は同じ行を返さず、無効化したストリームを返さず、ストリームごとの上限を守る。
func TestClaimDueLeasesAndIsolates(t *testing.T) {
	streams, deliveries, stream := captureFixture(t, []string{"*"})
	ctx := context.Background()
	now := pgtest.Now()
	for _, id := range []string{"e1", "e2", "e3"} {
		tx, err := deliveries.Pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := CaptureAuditEventInTx(ctx, tx, ssdomain.AuditSetEvent{EventID: id, EventType: "UserCreated", TenantID: stream.TenantID, OccurredAt: now}, now); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// 取得は全テナントが対象で DB はテストの間で共有するので、このストリームの分だけを見る。
	own := func(claimed []*ssdomain.SecurityEventDelivery) []*ssdomain.SecurityEventDelivery {
		out := []*ssdomain.SecurityEventDelivery{}
		for _, d := range claimed {
			if d.StreamID == stream.ID {
				out = append(out, d)
			}
		}
		return out
	}
	claim := func() []*ssdomain.SecurityEventDelivery {
		t.Helper()
		claimed, err := deliveries.ClaimDue(ctx, now, now.Add(time.Minute), 2, 1000)
		if err != nil {
			t.Fatal(err)
		}
		return own(claimed)
	}
	first := claim()
	if len(first) != 2 {
		t.Fatalf("first=%d", len(first))
	}
	second := claim()
	if len(second) != 1 || second[0].ID == first[0].ID || second[0].ID == first[1].ID {
		t.Fatalf("second=%+v", second)
	}
	disabled := *stream
	disabled.Status = ssdomain.SsfStreamStatusDisabled
	if err := streams.Save(ctx, &disabled); err != nil {
		t.Fatal(err)
	}
	got, err := deliveries.ClaimDue(ctx, now.Add(time.Hour), now.Add(2*time.Hour), 10, 1000)
	if err != nil || len(own(got)) != 0 {
		t.Fatalf("disabled claim=%d err=%v", len(own(got)), err)
	}
}

//spec:covers REQ-SHAREDSIGNALS-018: 閾値の両側と、回復による記録の消去を一文で判定する。
func TestRecordDeliveryFailureThreshold(t *testing.T) {
	streams, _, stream := captureFixture(t, nil)
	ctx := context.Background()
	start := pgtest.Now()
	for i := 1; i < ssdomain.AutoDisableFailures; i++ {
		disabled, err := streams.RecordDeliveryFailure(ctx, stream.TenantID, stream.ID, start.Add(2*time.Hour))
		if err != nil || disabled {
			t.Fatalf("failure %d disabled=%v err=%v", i, disabled, err)
		}
	}
	// 回数は満たすが、最初の失敗の時刻が今回と同じなので経過時間を満たさない別の系列を確かめる。
	if err := streams.RecordDeliverySuccess(ctx, stream.TenantID, stream.ID); err != nil {
		t.Fatal(err)
	}
	for range ssdomain.AutoDisableFailures {
		if disabled, err := streams.RecordDeliveryFailure(ctx, stream.TenantID, stream.ID, start); err != nil || disabled {
			t.Fatalf("elapsed not met: disabled=%v err=%v", disabled, err)
		}
	}
	disabled, err := streams.RecordDeliveryFailure(ctx, stream.TenantID, stream.ID, start.Add(time.Hour))
	if err != nil || !disabled {
		t.Fatalf("threshold: disabled=%v err=%v", disabled, err)
	}
	if disabled, err := streams.RecordDeliveryFailure(ctx, stream.TenantID, stream.ID, start.Add(2*time.Hour)); err != nil || disabled {
		t.Fatalf("already disabled must not report again: %v %v", disabled, err)
	}
	got, err := streams.FindByID(ctx, stream.TenantID, stream.ID)
	if err != nil || got.Status != ssdomain.SsfStreamStatusDisabled {
		t.Fatalf("stream=%+v err=%v", got, err)
	}
}
