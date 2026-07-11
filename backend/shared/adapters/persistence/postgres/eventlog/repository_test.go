package eventlog

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	sharedeventlog "github.com/ambi/idmagic/backend/shared/eventlog"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRepositoryAppendAndFind(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	ctx := context.Background()
	occurredAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	rec := sharedeventlog.Record{
		EventID:        newUUID(t),
		TenantID:       "tenant-eventlog-test",
		Type:           "AdminUserUpdated",
		Classification: sharedeventlog.ClassificationAuditOnly,
		Actor:          "operator",
		Subject:        "alice",
		CorrelationID:  newUUID(t),
		OccurredAt:     occurredAt,
		Payload:        map[string]any{"field": "email"},
	}
	if err := repo.Append(ctx, rec); err != nil {
		t.Fatalf("append: %v", err)
	}

	found, err := repo.FindByID(ctx, rec.EventID)
	if err != nil || found == nil {
		t.Fatalf("find by id: %v %#v", err, found)
	}
	if found.TenantID != rec.TenantID || found.Type != rec.Type ||
		found.Classification != rec.Classification || found.Actor != rec.Actor ||
		found.Subject != rec.Subject || found.CorrelationID != rec.CorrelationID ||
		!found.OccurredAt.Equal(rec.OccurredAt) || found.Payload["field"] != "email" {
		t.Fatalf("round trip mismatch: %#v", found)
	}
}

func TestRepositoryFindByIDMissingReturnsNil(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	found, err := repo.FindByID(context.Background(), newUUID(t))
	if err != nil || found != nil {
		t.Fatalf("expected nil, nil for missing event_id: %v %#v", err, found)
	}
}

func TestRepositoryAppendWithoutActorOrSubject(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	ctx := context.Background()

	rec := sharedeventlog.Record{
		EventID:        newUUID(t),
		TenantID:       "tenant-eventlog-test",
		Type:           "TenantCreated",
		Classification: sharedeventlog.ClassificationPublicIntegration,
		CorrelationID:  newUUID(t),
		OccurredAt:     time.Now().UTC(),
		Payload:        nil,
	}
	if err := repo.Append(ctx, rec); err != nil {
		t.Fatalf("append: %v", err)
	}
	found, err := repo.FindByID(ctx, rec.EventID)
	if err != nil || found == nil {
		t.Fatalf("find by id: %v %#v", err, found)
	}
	if found.Actor != "" || found.Subject != "" {
		t.Fatalf("expected absent actor/subject, got %#v", found)
	}
}

func TestRepositoryAppendDuplicateEventIDFails(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	ctx := context.Background()
	rec := sharedeventlog.Record{
		EventID:        newUUID(t),
		TenantID:       "tenant-eventlog-test",
		Type:           "AdminUserUpdated",
		Classification: sharedeventlog.ClassificationAuditOnly,
		CorrelationID:  newUUID(t),
		OccurredAt:     time.Now().UTC(),
	}
	if err := repo.Append(ctx, rec); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := repo.Append(ctx, rec); err == nil {
		t.Fatal("expected duplicate event_id to fail, got nil error")
	}
}

func TestRepositoryClassificationCheckConstraintRejectsUnknownValue(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	rec := sharedeventlog.Record{
		EventID:        newUUID(t),
		TenantID:       "tenant-eventlog-test",
		Type:           "AdminUserUpdated",
		Classification: sharedeventlog.Classification("not_a_real_classification"),
		CorrelationID:  newUUID(t),
		OccurredAt:     time.Now().UTC(),
	}
	if err := repo.Append(context.Background(), rec); err == nil {
		t.Fatal("expected classification CHECK constraint to reject unknown value")
	}
}

func TestRepositoryAppendDeliveryAndFind(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	ctx := context.Background()
	rec := sharedeventlog.Record{
		EventID:        newUUID(t),
		TenantID:       "tenant-eventlog-test",
		Type:           "TenantCreated",
		Classification: sharedeventlog.ClassificationPublicIntegration,
		CorrelationID:  newUUID(t),
		OccurredAt:     time.Now().UTC(),
	}
	if err := repo.Append(ctx, rec); err != nil {
		t.Fatalf("append event log: %v", err)
	}
	if err := repo.AppendDelivery(ctx, rec.EventID); err != nil {
		t.Fatalf("append delivery: %v", err)
	}

	delivery, err := repo.FindDeliveryByID(ctx, rec.EventID)
	if err != nil || delivery == nil {
		t.Fatalf("find delivery: %v %#v", err, delivery)
	}
	if delivery.Status != sharedeventlog.DeliveryStatusPending || delivery.Attempts != 0 ||
		delivery.DeliveredAt != nil {
		t.Fatalf("unexpected initial delivery state: %#v", delivery)
	}
}

func TestRepositoryAppendDeliveryWithoutEventLogFailsForeignKey(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	if err := repo.AppendDelivery(context.Background(), newUUID(t)); err == nil {
		t.Fatal("expected event_deliveries FK violation for unknown event_id")
	}
}

func TestEventDeliveriesStatusCheckConstraintRejectsUnknownValue(t *testing.T) {
	db := pgtest.Require(t)
	repo := New(db)
	ctx := context.Background()
	rec := sharedeventlog.Record{
		EventID:        newUUID(t),
		TenantID:       "tenant-eventlog-test",
		Type:           "TenantCreated",
		Classification: sharedeventlog.ClassificationPublicIntegration,
		CorrelationID:  newUUID(t),
		OccurredAt:     time.Now().UTC(),
	}
	if err := repo.Append(ctx, rec); err != nil {
		t.Fatalf("append event log: %v", err)
	}
	if err := repo.AppendDelivery(ctx, rec.EventID); err != nil {
		t.Fatalf("append delivery: %v", err)
	}
	// event_deliveries.status has no dedicated sqlc mutation (relay work is
	// wi-184 T005); exercise the CHECK constraint directly against the pool
	// the same way the relay's future UPDATE would.
	_, err := db.Exec(ctx, `UPDATE event_deliveries SET status = 'not_a_real_status' WHERE event_id = $1`, rec.EventID)
	if err == nil {
		t.Fatal("expected status CHECK constraint to reject unknown value")
	}
}
