package db_postgres_test

import (
	"context"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/db_postgres"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestOutboxEventSinkEmit(t *testing.T) {
	db := pgtest.Require(t)
	sink := &oauth2postgres.OutboxEventSink{Pool: db}
	clientID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.Emit(context.Background(), &oauthdomain.ClientRegistered{
		At: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), TenantID: tenancydomain.DefaultTenantID, ClientID: clientID, ClientType: spec.ClientConfidential,
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
