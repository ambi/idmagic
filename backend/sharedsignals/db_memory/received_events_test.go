package db_memory_test

import (
	"context"
	"testing"
	"time"

	dbmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// TestReceivedSecurityEventRepository_ExistsByJTIDetectsReplay — scenario
// `重複jtiのSETは一度だけ反映される` の前提となる replay 検知を検証する。
func TestReceivedSecurityEventRepository_ExistsByJTIDetectsReplay(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewReceivedSecurityEventRepository()

	exists, err := repo.ExistsByJTI(ctx, "tenant-a", "stream_1", "jti_1")
	if err != nil {
		t.Fatalf("ExistsByJTI: %v", err)
	}
	if exists {
		t.Fatal("unseen jti must not exist yet")
	}

	if err := repo.Save(ctx, &ssdomain.ReceivedSecurityEvent{
		ID: "received_1", TenantID: "tenant-a", StreamID: "stream_1", SetJTI: "jti_1",
		EventType: ssdomain.CaepEventTypeSessionRevoked,
		Subject: ssdomain.SsfSubject{
			SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: "tenant-a", PrincipalID: "agent_1",
		},
		VerificationResult: ssdomain.SecurityEventVerificationAccepted,
		ReceivedAt:         time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err = repo.ExistsByJTI(ctx, "tenant-a", "stream_1", "jti_1")
	if err != nil {
		t.Fatalf("ExistsByJTI: %v", err)
	}
	if !exists {
		t.Fatal("saved jti must now be detected as replay")
	}

	// 他テナント・他 stream のスコープには漏れない。
	if exists, err := repo.ExistsByJTI(ctx, "tenant-b", "stream_1", "jti_1"); err != nil || exists {
		t.Fatalf("ExistsByJTI cross-tenant = (%v, %v), want (false, nil)", exists, err)
	}
	if exists, err := repo.ExistsByJTI(ctx, "tenant-a", "stream_2", "jti_1"); err != nil || exists {
		t.Fatalf("ExistsByJTI cross-stream = (%v, %v), want (false, nil)", exists, err)
	}
}
