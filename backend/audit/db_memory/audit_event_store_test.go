package db_memory

import (
	"context"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/audit/ports"
)

func newAuditEvent(t *testing.T, tenantID, typ string, occurredAt time.Time, userID string) *ports.AuditEventRecord {
	t.Helper()
	return &ports.AuditEventRecord{
		ID:         tenantID + ":" + typ + ":" + userID + ":" + occurredAt.Format(time.RFC3339Nano),
		TenantID:   tenantID,
		Type:       typ,
		OccurredAt: occurredAt,
		Payload:    map[string]any{"userId": userID, "tenantId": tenantID},
	}
}

func TestAuditEventStoreFiltersAndOrders(t *testing.T) {
	store := NewAuditEventStore(0)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, ev := range []*ports.AuditEventRecord{
		newAuditEvent(t, "acme", "UserAuthenticated", base, "alice"),
		newAuditEvent(t, "acme", "AccessTokenIssued", base.Add(2*time.Second), "alice"),
		newAuditEvent(t, tenancydomain.DefaultTenantID, "UserAuthenticated", base.Add(3*time.Second), "bob"),
		newAuditEvent(t, "acme", "UserAuthenticated", base.Add(4*time.Second), "carol"),
	} {
		if err := store.Append(context.Background(), ev); err != nil {
			t.Fatalf("append #%d: %v", i, err)
		}
	}

	// 全テナントを暗黙に閉じた acme フィルタは acme のみ降順で返す。
	out, err := store.List(context.Background(), ports.AuditEventQuery{TenantID: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("expected 3 acme events, got %d", len(out))
	}
	if out[0].OccurredAt.Before(out[len(out)-1].OccurredAt) {
		t.Fatalf("results must be in descending OccurredAt order: %+v", out)
	}

	// type フィルタ + sub フィルタの結合。
	filtered, err := store.List(context.Background(), ports.AuditEventQuery{
		TenantID: "acme", Type: "UserAuthenticated", UserID: "alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].Payload["userId"] != "alice" {
		t.Fatalf("filter mismatch: %+v", filtered)
	}

	// AllTenants=true は default を含めて全件を返す。
	all, err := store.List(context.Background(), ports.AuditEventQuery{AllTenants: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("AllTenants must include default events, got %d", len(all))
	}

	// After フィルタは境界を含む (BeforeForce: rec.Before(After) を弾く)。
	after, err := store.List(context.Background(), ports.AuditEventQuery{
		TenantID: "acme", After: base.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].Payload["userId"] != "carol" {
		t.Fatalf("After filter: %+v", after)
	}
}

func TestAuditEventStoreEvictsBeyondCapacity(t *testing.T) {
	// maxEvents=3 で 4 件追加すると一番古い 1 件が落ちる。byID も同期する。
	store := NewAuditEventStore(3)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first := newAuditEvent(t, "acme", "X", base, "a")
	for _, ev := range []*ports.AuditEventRecord{
		first,
		newAuditEvent(t, "acme", "X", base.Add(time.Second), "b"),
		newAuditEvent(t, "acme", "X", base.Add(2*time.Second), "c"),
		newAuditEvent(t, "acme", "X", base.Add(3*time.Second), "d"),
	} {
		if err := store.Append(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := store.FindByID(context.Background(), first.ID); got != nil {
		t.Fatal("oldest event must be evicted")
	}
	out, err := store.List(context.Background(), ports.AuditEventQuery{TenantID: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("capacity not enforced: len=%d", len(out))
	}
}

func TestAuditEventStoreLimitCapsAt1000(t *testing.T) {
	store := NewAuditEventStore(0)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 1100 {
		_ = store.Append(context.Background(), &ports.AuditEventRecord{
			ID: "e" + time.Duration(i).String(), TenantID: "acme", Type: "X",
			OccurredAt: base.Add(time.Duration(i) * time.Second),
			Payload:    map[string]any{"userId": "u"},
		})
	}
	out, _ := store.List(context.Background(), ports.AuditEventQuery{TenantID: "acme", Limit: 10000})
	if len(out) != 1000 {
		t.Fatalf("limit must cap at 1000, got %d", len(out))
	}
}

func TestAuditEventStoreKeysetPagination(t *testing.T) {
	store := NewAuditEventStore(0)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 5 {
		if err := store.Append(context.Background(), &ports.AuditEventRecord{
			ID: "e" + string(rune('0'+i)), TenantID: "acme", Type: "X",
			OccurredAt: base.Add(time.Duration(i) * time.Second),
			Payload:    map[string]any{"userId": "u"},
		}); err != nil {
			t.Fatal(err)
		}
	}

	first, err := store.List(context.Background(), ports.AuditEventQuery{TenantID: "acme", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	// OccurredAt DESC: e4, e3, e2, e1, e0
	if len(first) != 2 || first[0].ID != "e4" || first[1].ID != "e3" {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := store.List(context.Background(), ports.AuditEventQuery{
		TenantID: "acme", Limit: 2, AfterOccurredAt: last.OccurredAt, AfterID: last.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 2 || next[0].ID != "e2" || next[1].ID != "e1" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}
	total, err := store.Count(context.Background(), ports.AuditEventQuery{TenantID: "acme", Type: "X"})
	if err != nil || total != 5 {
		t.Fatalf("Count = %d, err=%v", total, err)
	}
	fromEnd, err := store.List(context.Background(), ports.AuditEventQuery{TenantID: "acme", Limit: 2, FromEnd: true})
	if err != nil || len(fromEnd) != 2 || fromEnd[0].ID != "e1" || fromEnd[1].ID != "e0" {
		t.Fatalf("end page = %+v, err=%v", fromEnd, err)
	}
}

// REQ-AUDIT-006: 多値の検索属性は、いずれか 1 つの参加者が一致すればそのイベントを返す。
// PostgreSQL 側の EXISTS 照合と同じ意味論であることを、memory store でも保つ。
func TestAuditEventStoreMatchesAnyValueOfAMultiValuedAttribute(t *testing.T) {
	store := NewAuditEventStore(0)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	chained := newAuditEvent(t, "acme", "TokenExchanged", base, "alice")
	chained.SearchAttributes = map[string][]string{
		"delegation.actor": {"app-b", "app-a", "alice"},
		"delegation.mode":  {"on_behalf_of"},
	}
	direct := newAuditEvent(t, "acme", "TokenExchanged", base.Add(time.Second), "bob")
	direct.SearchAttributes = map[string][]string{
		"delegation.actor": {"bob"},
		"delegation.mode":  {"direct"},
	}
	for i, ev := range []*ports.AuditEventRecord{chained, direct} {
		if err := store.Append(context.Background(), ev); err != nil {
			t.Fatalf("append #%d: %v", i, err)
		}
	}

	// チェーンの中間にいる参加者からでも引ける。
	events, err := store.List(context.Background(), ports.AuditEventQuery{
		TenantID: "acme",
		Filters:  []ports.AuditFilterExpression{{Field: "delegation.actor", Operator: ports.OpEq, Values: []string{"app-a"}}},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 1 || events[0].ID != chained.ID {
		t.Fatalf("delegation.actor=app-a returned %d event(s), want only the chained one", len(events))
	}

	// 参加していない主体では引けない。
	events, err = store.List(context.Background(), ports.AuditEventQuery{
		TenantID: "acme",
		Filters:  []ports.AuditFilterExpression{{Field: "delegation.actor", Operator: ports.OpEq, Values: []string{"app-c"}}},
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("delegation.actor=app-c returned %d event(s), want 0", len(events))
	}
}
