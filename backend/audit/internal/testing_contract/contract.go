// Package testing_contract defines the shared audit-event persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type Fixture struct {
	Repository ports.AuditEventRepository
	TenantA    string
	TenantB    string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func newID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	first := &ports.AuditEventRecord{ID: newID(t), TenantID: f.TenantA, Type: "user.created", OccurredAt: f.Now, Payload: map[string]any{"user": "alice"}}
	second := &ports.AuditEventRecord{ID: newID(t), TenantID: f.TenantA, Type: "user.disabled", OccurredAt: f.Now.Add(time.Minute), Payload: map[string]any{"user": "alice"}}
	other := &ports.AuditEventRecord{ID: newID(t), TenantID: f.TenantB, Type: "user.created", OccurredAt: f.Now.Add(2 * time.Minute)}
	for _, event := range []*ports.AuditEventRecord{first, second, other} {
		if err := f.Repository.Append(ctx, event); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	got, err := f.Repository.FindByID(ctx, first.ID)
	if err != nil || got == nil || got.Type != first.Type {
		t.Fatalf("FindByID = (%+v, %v)", got, err)
	}
	list, err := f.Repository.List(ctx, ports.AuditEventQuery{TenantID: f.TenantA, Limit: 10})
	if err != nil || len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("tenant list = (%+v, %v), want newest first and isolated", list, err)
	}
	count, err := f.Repository.Count(ctx, ports.AuditEventQuery{TenantID: f.TenantA})
	if err != nil || count != 2 {
		t.Fatalf("tenant count = (%d, %v), want 2", count, err)
	}
	if err := f.Repository.Append(ctx, first); err != nil {
		t.Fatalf("duplicate Append: %v", err)
	}
	count, err = f.Repository.Count(ctx, ports.AuditEventQuery{TenantID: f.TenantA})
	if err != nil || count != 2 {
		t.Fatalf("duplicate append changed count = (%d, %v)", count, err)
	}
	runLatestByAgent(ctx, t, f)
}

// runLatestByAgent は、Agent ごとの最新の発生時刻が、名指した種別だけから、テナントの中で、
// 発生時刻の最大として決まることを確かめる。追記の順序と発生時刻の順序をわざと逆にする。
func runLatestByAgent(ctx context.Context, t *testing.T, f Fixture) {
	t.Helper()
	agentA, agentB, agentC := newID(t), newID(t), newID(t)
	record := func(tenantID, eventType, agentID string, offset time.Duration) {
		t.Helper()
		rec := &ports.AuditEventRecord{
			ID: newID(t), TenantID: tenantID, Type: eventType, OccurredAt: f.Now.Add(offset),
			Payload: map[string]any{"agentId": agentID}, SearchAttributes: map[string][]string{"agent.id": {agentID}},
		}
		if err := f.Repository.Append(ctx, rec); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	record(f.TenantA, "AccessTokenIssued", agentA, 10*time.Minute)
	record(f.TenantA, "AccessTokenIssued", agentA, 5*time.Minute)
	record(f.TenantA, "TokenExchanged", agentA, 20*time.Minute)
	record(f.TenantA, "AgentDisabled", agentA, 30*time.Minute) // 名指していない種別は数えない
	record(f.TenantB, "AccessTokenIssued", agentB, 40*time.Minute)
	record(f.TenantA, "AgentRegistered", agentC, 50*time.Minute)

	latest, err := f.Repository.LatestByAgent(ctx, f.TenantA, []string{agentA, agentB, agentC}, []string{"AccessTokenIssued", "TokenExchanged"})
	if err != nil {
		t.Fatalf("LatestByAgent: %v", err)
	}
	if got := latest[agentA]; !got.Equal(f.Now.Add(20 * time.Minute)) {
		t.Errorf("agentA latest = %s, want %s", got, f.Now.Add(20*time.Minute))
	}
	if _, ok := latest[agentB]; ok {
		t.Error("別のテナントの記録が数えられた")
	}
	if _, ok := latest[agentC]; ok {
		t.Error("名指していない種別だけの Agent が結果に含まれた")
	}
}
