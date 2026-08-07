package db_memory_test

import (
	"context"
	"testing"
	"time"

	dbmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

const testDeliveryTenantID = "tenant-a"

func testDelivery(id, streamID, status string, nextAttemptAt *time.Time) *ssdomain.SecurityEventDelivery {
	return &ssdomain.SecurityEventDelivery{
		ID: id, TenantID: testDeliveryTenantID, StreamID: streamID, SetJTI: id,
		Set: ssdomain.SecurityEventToken{
			JTI: id, Issuer: "https://idmagic.example", Audience: "https://receiver.example",
			IssuedAt: time.Now().UTC(),
			Event: ssdomain.CaepEvent{
				EventType: ssdomain.CaepEventTypeSessionRevoked,
				Subject: ssdomain.SsfSubject{
					SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: testDeliveryTenantID, PrincipalID: "agent_1",
				},
				EventTimestamp:   time.Now().UTC(),
				InitiatingEntity: ssdomain.InitiatingEntityAdmin,
			},
			Compact: "eyJ...",
		},
		Status:        ssdomain.SecurityEventDeliveryStatus(status),
		NextAttemptAt: nextAttemptAt,
		CreatedAt:     time.Now().UTC(),
	}
}

// TestSecurityEventDeliveryRepository_ListDue — scenario
// `配送失敗は再試行され上限超過でdead_letterへ遷移する` の前提: pending/failed かつ
// next_attempt_at <= now の配送だけが対象になり、terminal (delivered/dead_letter) は除外される。
func TestSecurityEventDeliveryRepository_ListDue(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewSecurityEventDeliveryRepository()
	now := time.Now().UTC()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	due := testDelivery("due_pending", "stream_1", "pending", nil)
	dueFailed := testDelivery("due_failed", "stream_1", "failed", &past)
	notYetDue := testDelivery("not_yet_due", "stream_1", "failed", &future)
	delivered := testDelivery("already_delivered", "stream_1", "delivered", nil)
	deadLettered := testDelivery("already_dead", "stream_1", "dead_letter", nil)

	for _, d := range []*ssdomain.SecurityEventDelivery{due, dueFailed, notYetDue, delivered, deadLettered} {
		if err := repo.Save(ctx, d); err != nil {
			t.Fatalf("Save(%s): %v", d.ID, err)
		}
	}

	got, err := repo.ListDue(ctx, now, 0)
	if err != nil {
		t.Fatalf("ListDue: %v", err)
	}
	gotIDs := map[string]bool{}
	for _, d := range got {
		gotIDs[d.ID] = true
	}
	if !gotIDs["due_pending"] || !gotIDs["due_failed"] {
		t.Fatalf("ListDue missing due deliveries: %v", gotIDs)
	}
	if gotIDs["not_yet_due"] || gotIDs["already_delivered"] || gotIDs["already_dead"] {
		t.Fatalf("ListDue returned ineligible deliveries: %v", gotIDs)
	}
}

func TestSecurityEventDeliveryRepository_ListByStream(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewSecurityEventDeliveryRepository()
	if err := repo.Save(ctx, testDelivery("d1", "stream_1", "pending", nil)); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Save(ctx, testDelivery("d2", "stream_2", "pending", nil)); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.ListByStream(ctx, "tenant-a", "stream_1")
	if err != nil {
		t.Fatalf("ListByStream: %v", err)
	}
	if len(got) != 1 || got[0].ID != "d1" {
		t.Fatalf("ListByStream = %+v, want only d1", got)
	}
}
