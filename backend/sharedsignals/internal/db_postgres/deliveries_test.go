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

//spec:covers REQ-SHAREDSIGNALS-013: 指定したテナントとストリームの dead_letter だけが pending に戻り、二度目の要求は何も戻さない。
func TestSecurityEventDeliveryRepository_RedeliverDeadLettered(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	streamRepo := &SsfStreamRepository{Pool: db}
	repo := &SecurityEventDeliveryRepository{Pool: db}
	tenant := seedTenant(t, db)
	stream := newStream(t, tenant.ID, ssdomain.SsfStreamDirectionTransmit)
	otherStream := newStream(t, tenant.ID, ssdomain.SsfStreamDirectionTransmit)
	otherTenant := seedTenant(t, db)
	foreignStream := newStream(t, otherTenant.ID, ssdomain.SsfStreamDirectionTransmit)
	for _, s := range []*ssdomain.SsfStream{stream, otherStream, foreignStream} {
		if err := streamRepo.Save(ctx, s); err != nil {
			t.Fatalf("Save stream: %v", err)
		}
	}

	lastError := "receiver down"
	dead := newDelivery(t, tenant.ID, stream.ID, "dead_letter", nil)
	dead.AttemptCount, dead.LastError = 8, &lastError
	untouched := []*ssdomain.SecurityEventDelivery{
		newDelivery(t, tenant.ID, stream.ID, "pending", nil),
		newDelivery(t, tenant.ID, stream.ID, "delivered", nil),
		newDelivery(t, tenant.ID, stream.ID, "failed", nil),
		newDelivery(t, tenant.ID, otherStream.ID, "dead_letter", nil),
		newDelivery(t, otherTenant.ID, foreignStream.ID, "dead_letter", nil),
	}
	for _, d := range append([]*ssdomain.SecurityEventDelivery{dead}, untouched...) {
		if err := repo.Save(ctx, d); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	got, err := repo.RedeliverDeadLettered(ctx, tenant.ID, stream.ID)
	if err != nil {
		t.Fatalf("RedeliverDeadLettered: %v", err)
	}
	if len(got) != 1 || got[0].ID != dead.ID || got[0].Status != ssdomain.SecurityEventDeliveryStatusPending ||
		got[0].AttemptCount != 0 || got[0].NextAttemptAt != nil || got[0].LastError != nil || got[0].Set.Compact != dead.Set.Compact {
		t.Fatalf("RedeliverDeadLettered = %+v, want %s reset with its SET kept", got, dead.ID)
	}
	for _, d := range untouched {
		list, err := repo.ListByStream(ctx, d.TenantID, d.StreamID)
		if err != nil {
			t.Fatalf("ListByStream: %v", err)
		}
		for _, row := range list {
			if row.ID == d.ID && row.Status != d.Status {
				t.Fatalf("delivery %s changed to %s, want %s", d.ID, row.Status, d.Status)
			}
		}
	}

	again, err := repo.RedeliverDeadLettered(ctx, tenant.ID, stream.ID)
	if err != nil || len(again) != 0 {
		t.Fatalf("second RedeliverDeadLettered = %+v err=%v, want nothing", again, err)
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

	got, err := repo.ClaimDue(context.Background(), now, now.Add(time.Minute), 100, 100)
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
