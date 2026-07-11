package eventlog

import (
	"context"
	"testing"
	"time"

	sharedeventlog "github.com/ambi/idmagic/backend/shared/eventlog"
)

func TestRepositoryAppendAndFind(t *testing.T) {
	repo := New()
	ctx := context.Background()
	rec := sharedeventlog.Record{
		EventID:        "event-1",
		TenantID:       "tenant-1",
		Type:           "AdminUserUpdated",
		Classification: sharedeventlog.ClassificationAuditOnly,
		CorrelationID:  "corr-1",
		OccurredAt:     time.Now().UTC(),
	}
	if err := repo.Append(ctx, rec); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := repo.Append(ctx, rec); err == nil {
		t.Fatal("expected duplicate event_id to fail")
	}
	found, err := repo.FindByID(ctx, rec.EventID)
	if err != nil || found == nil || found.Type != rec.Type {
		t.Fatalf("find by id: %v %#v", err, found)
	}
}

func TestRepositoryAppendDeliveryRequiresEventLog(t *testing.T) {
	repo := New()
	ctx := context.Background()
	if err := repo.AppendDelivery(ctx, "missing"); err == nil {
		t.Fatal("expected error for unknown event_id")
	}

	rec := sharedeventlog.Record{
		EventID:        "event-2",
		TenantID:       "tenant-1",
		Type:           "TenantCreated",
		Classification: sharedeventlog.ClassificationPublicIntegration,
		CorrelationID:  "corr-2",
		OccurredAt:     time.Now().UTC(),
	}
	if err := repo.Append(ctx, rec); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := repo.AppendDelivery(ctx, rec.EventID); err != nil {
		t.Fatalf("append delivery: %v", err)
	}
	if err := repo.AppendDelivery(ctx, rec.EventID); err == nil {
		t.Fatal("expected duplicate delivery to fail")
	}
	delivery, err := repo.FindDeliveryByID(ctx, rec.EventID)
	if err != nil || delivery == nil || delivery.Status != sharedeventlog.DeliveryStatusPending {
		t.Fatalf("find delivery: %v %#v", err, delivery)
	}
}
