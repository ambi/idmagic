package db_postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestAuditEventRepositoryAppendAndList(t *testing.T) {
	db := pgtest.Require(t)
	newUUID := func() string {
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	tenantID, userID := "tenant-audit-test", newUUID()
	repo := &AuditEventRepository{Pool: db}
	ctx := context.Background()
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	first := &ports.AuditEventRecord{
		ID: newUUID(), TenantID: tenantID, Type: "UserAuthenticated", OccurredAt: base,
		Payload: map[string]any{"userId": userID}, SearchAttributes: map[string][]string{"outcome": {"success"}},
	}
	second := &ports.AuditEventRecord{
		ID: newUUID(), TenantID: tenantID, Type: "AuthenticationFailed", OccurredAt: base.Add(time.Minute),
		Payload: map[string]any{"userId": userID}, SearchAttributes: map[string][]string{"outcome": {"failure"}},
	}
	for _, event := range []*ports.AuditEventRecord{first, second, first} {
		if err := repo.Append(ctx, event); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	events, err := repo.List(ctx, ports.AuditEventQuery{TenantID: tenantID, Filters: []ports.AuditFilterExpression{{
		Field: "outcome", Operator: ports.OpEq, Values: []string{"failure"},
	}}})
	if err != nil || len(events) != 1 || events[0].ID != second.ID {
		t.Fatalf("filtered list: %v %#v", err, events)
	}
	found, err := repo.FindByID(ctx, first.ID)
	if err != nil || found == nil || found.Payload["userId"] != userID {
		t.Fatalf("find by ID: %v %#v", err, found)
	}
}

func TestAuditEventRepositoryListKeysetPagination(t *testing.T) {
	db := pgtest.Require(t)
	newUUID := func() string {
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	tenantID := "tenant-audit-keyset-test"
	repo := &AuditEventRepository{Pool: db}
	ctx := context.Background()
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	events := make([]*ports.AuditEventRecord, 5)
	for i := range events {
		ev := &ports.AuditEventRecord{
			ID: newUUID(), TenantID: tenantID, Type: "UserAuthenticated",
			OccurredAt: base.Add(time.Duration(i) * time.Second),
			Payload:    map[string]any{"userId": newUUID()},
		}
		if err := repo.Append(ctx, ev); err != nil {
			t.Fatalf("append #%d: %v", i, err)
		}
		events[i] = ev
	}

	// OccurredAt DESC: events[4], events[3], events[2], events[1], events[0]
	first, err := repo.List(ctx, ports.AuditEventQuery{TenantID: tenantID, Limit: 2})
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].ID != events[4].ID || first[1].ID != events[3].ID {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.List(ctx, ports.AuditEventQuery{
		TenantID: tenantID, Limit: 2, AfterOccurredAt: last.OccurredAt, AfterID: last.ID,
	})
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].ID != events[2].ID || next[1].ID != events[1].ID {
		t.Fatalf("unexpected continuation page: %+v", next)
	}
}

func TestAuditEventRepositoryListRejectsMalformedUserIDAsNoMatch(t *testing.T) {
	// wi-147: user_id は UUID 列。typo や実在しない ID を入力しても 500 にせず 0 件を返す。
	db := pgtest.Require(t)
	repo := &AuditEventRepository{Pool: db}
	events, err := repo.List(context.Background(), ports.AuditEventQuery{
		TenantID: "tenant-audit-test", UserID: "not-a-uuid",
	})
	if err != nil {
		t.Fatalf("expected no error for malformed user_id, got %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events for malformed user_id, got %d", len(events))
	}
}

// REQ-AUDIT-005 / REQ-AUDIT-006: 委譲の軸で、エージェントが代行した操作を本人の操作と
// 区別して引ける。チェーンの参加者は多値なので、どの段からでも同じイベントに当たる。
func TestAuditEventRepositoryDelegationAxes(t *testing.T) {
	db := pgtest.Require(t)
	newUUID := func() string {
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	tenantID := "tenant-audit-delegation-test"
	repo := &AuditEventRepository{Pool: db}
	ctx := context.Background()
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	// audit_events.user_id は UUID 列なので、利用者の識別子は実在する形の UUID を使う。
	alice := newUUID()

	delegated := &ports.AuditEventRecord{
		ID: newUUID(), TenantID: tenantID, Type: "TokenExchanged", OccurredAt: base,
		Payload: map[string]any{"actorUserId": "app-b", "subjectUserId": alice},
		SearchAttributes: map[string][]string{
			"actor.type":       {ports.ActorTypeAgent},
			"agent.id":         {"agent-a"},
			"actor.id":         {"app-b"},
			"target.id":        {alice},
			"delegation.actor": {"app-b", "app-a", alice},
			"delegation.mode":  {"on_behalf_of"},
			"delegation.depth": {"2"},
		},
	}
	inPerson := &ports.AuditEventRecord{
		ID: newUUID(), TenantID: tenantID, Type: "UserAuthenticated", OccurredAt: base.Add(time.Minute),
		Payload: map[string]any{"userId": alice},
		SearchAttributes: map[string][]string{
			"actor.type": {ports.ActorTypeUser},
			"actor.id":   {alice},
		},
	}
	// 二重の Append が多値の行を重複させないこと (冪等) も併せて確かめる。
	for _, ev := range []*ports.AuditEventRecord{delegated, inPerson, delegated} {
		if err := repo.Append(ctx, ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	cases := []struct {
		name    string
		filters []ports.AuditFilterExpression
		wantID  string
		wantLen int
	}{
		{
			name: "the agent's own action",
			filters: []ports.AuditFilterExpression{
				{Field: "actor.type", Operator: ports.OpEq, Values: []string{ports.ActorTypeAgent}},
				{Field: "agent.id", Operator: ports.OpEq, Values: []string{"agent-a"}},
			},
			wantID: delegated.ID, wantLen: 1,
		},
		{
			name:    "the user's own action does not include what was done for them",
			filters: []ports.AuditFilterExpression{{Field: "actor.id", Operator: ports.OpEq, Values: []string{alice}}},
			wantID:  inPerson.ID, wantLen: 1,
		},
		{
			name:    "an intermediate participant of the chain",
			filters: []ports.AuditFilterExpression{{Field: "delegation.actor", Operator: ports.OpEq, Values: []string{"app-a"}}},
			wantID:  delegated.ID, wantLen: 1,
		},
		{
			name:    "the subject is a participant of the chain too",
			filters: []ports.AuditFilterExpression{{Field: "delegation.actor", Operator: ports.OpIn, Values: []string{alice, "nobody"}}},
			wantID:  delegated.ID, wantLen: 1,
		},
		{
			name:    "a principal outside the chain matches nothing",
			filters: []ports.AuditFilterExpression{{Field: "delegation.actor", Operator: ports.OpEq, Values: []string{"app-c"}}},
			wantLen: 0,
		},
		{
			name:    "the delegation mode comes from the payload of the exchange",
			filters: []ports.AuditFilterExpression{{Field: "delegation.mode", Operator: ports.OpEq, Values: []string{"on_behalf_of"}}},
			wantID:  delegated.ID, wantLen: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events, err := repo.List(ctx, ports.AuditEventQuery{TenantID: tenantID, Filters: tc.filters})
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(events) != tc.wantLen {
				t.Fatalf("got %d event(s), want %d: %#v", len(events), tc.wantLen, events)
			}
			if tc.wantLen == 1 && events[0].ID != tc.wantID {
				t.Fatalf("got event %s, want %s", events[0].ID, tc.wantID)
			}
		})
	}
}
