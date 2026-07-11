package postgres

import (
	"context"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	"github.com/ambi/idmagic/backend/shared/spec"
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

func TestOutboxEventSinkEmit(t *testing.T) {
	db := pgtest.Require(t)
	sink := &OutboxEventSink{Pool: db}
	clientID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Emit(context.Background(), &oauthdomain.ClientRegistered{
		At: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), TenantID: spec.DefaultTenantID, ClientID: clientID, ClientType: spec.ClientConfidential,
	}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	var topic string
	if err := db.QueryRow(context.Background(), "SELECT topic FROM outbox WHERE payload->>'clientId'=$1", clientID).Scan(&topic); err != nil {
		t.Fatalf("find outbox event: %v", err)
	}
	if topic != "oauth2.client.lifecycle.v1" {
		t.Fatalf("topic=%q", topic)
	}
}
