package db_postgres

import (
	"context"
	"testing"
	"time"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func newDelivery(t *testing.T, tenantID, streamID, status string, nextAttemptAt *time.Time) *ssdomain.SecurityEventDelivery {
	t.Helper()
	set := newSecurityEventToken("agent_1", tenantID)
	return &ssdomain.SecurityEventDelivery{
		ID: newUUID(t), TenantID: tenantID, StreamID: streamID, SetJTI: set.JTI, Set: set,
		Status: ssdomain.SecurityEventDeliveryStatus(status), NextAttemptAt: nextAttemptAt,
		CreatedAt: testClock(),
	}
}

func TestSecurityEventDeliveryRepository_SaveAndRoundTripSetPayload(t *testing.T) {
	db := pgtest.Require(t)
	streamRepo := &SsfStreamRepository{Pool: db}
	repo := &SecurityEventDeliveryRepository{Pool: db}
	tenant := seedTenant(t, db)
	stream := newStream(t, tenant.ID, ssdomain.SsfStreamDirectionTransmit)
	if err := streamRepo.Save(context.Background(), stream); err != nil {
		t.Fatalf("Save stream: %v", err)
	}

	d := newDelivery(t, tenant.ID, stream.ID, "pending", nil)
	if err := repo.Save(context.Background(), d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	list, err := repo.ListByStream(context.Background(), tenant.ID, stream.ID)
	if err != nil {
		t.Fatalf("ListByStream: %v", err)
	}
	if len(list) != 1 || list[0].Set.JTI != d.Set.JTI || list[0].Set.Event.Subject.PrincipalID != "agent_1" {
		t.Fatalf("ListByStream = %+v, want SecurityEventToken round-tripped from %+v", list, d)
	}
}

func TestSecurityEventDeliveryRepository_ListDue(t *testing.T) {
	db := pgtest.Require(t)
	streamRepo := &SsfStreamRepository{Pool: db}
	repo := &SecurityEventDeliveryRepository{Pool: db}
	tenant := seedTenant(t, db)
	stream := newStream(t, tenant.ID, ssdomain.SsfStreamDirectionTransmit)
	if err := streamRepo.Save(context.Background(), stream); err != nil {
		t.Fatalf("Save stream: %v", err)
	}

	now := testClock()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	due := newDelivery(t, tenant.ID, stream.ID, "pending", nil)
	dueFailed := newDelivery(t, tenant.ID, stream.ID, "failed", &past)
	notYetDue := newDelivery(t, tenant.ID, stream.ID, "failed", &future)
	delivered := newDelivery(t, tenant.ID, stream.ID, "delivered", nil)

	for _, d := range []*ssdomain.SecurityEventDelivery{due, dueFailed, notYetDue, delivered} {
		if err := repo.Save(context.Background(), d); err != nil {
			t.Fatalf("Save(%s): %v", d.ID, err)
		}
	}

	got, err := repo.ListDue(context.Background(), now, 0)
	if err != nil {
		t.Fatalf("ListDue: %v", err)
	}
	gotIDs := map[string]bool{}
	for _, d := range got {
		gotIDs[d.ID] = true
	}
	if !gotIDs[due.ID] || !gotIDs[dueFailed.ID] {
		t.Fatalf("ListDue missing due deliveries: %v", gotIDs)
	}
	if gotIDs[notYetDue.ID] || gotIDs[delivered.ID] {
		t.Fatalf("ListDue returned ineligible deliveries: %v", gotIDs)
	}
}
