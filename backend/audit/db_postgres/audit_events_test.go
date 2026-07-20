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
		Payload: map[string]any{"userId": userID}, SearchAttributes: map[string]string{"outcome": "success"},
	}
	second := &ports.AuditEventRecord{
		ID: newUUID(), TenantID: tenantID, Type: "AuthenticationFailed", OccurredAt: base.Add(time.Minute),
		Payload: map[string]any{"userId": userID}, SearchAttributes: map[string]string{"outcome": "failure"},
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
