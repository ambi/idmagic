package db_postgres

import (
	"context"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// TestReceivedSecurityEventRepository_UniqueJTIPerStream — scenario
// `重複jtiのSETは一度だけ反映される` の前提: DB 制約 (UNIQUE (stream_id, set_jti)) が
// 二重挿入を吸収し、ExistsByJTI が既存を検知する。
func TestReceivedSecurityEventRepository_UniqueJTIPerStream(t *testing.T) {
	db := pgtest.Require(t)
	streamRepo := &SsfStreamRepository{Pool: db}
	repo := &ReceivedSecurityEventRepository{Pool: db}
	tenant := seedTenant(t, db)
	stream := newStream(t, tenant.ID, ssdomain.SsfStreamDirectionReceive)
	if err := streamRepo.Save(context.Background(), stream); err != nil {
		t.Fatalf("Save stream: %v", err)
	}

	exists, err := repo.ExistsByJTI(context.Background(), tenant.ID, stream.ID, "jti_1")
	if err != nil {
		t.Fatalf("ExistsByJTI: %v", err)
	}
	if exists {
		t.Fatal("unseen jti must not exist yet")
	}

	event := &ssdomain.ReceivedSecurityEvent{
		ID: newUUID(t), TenantID: tenant.ID, StreamID: stream.ID, SetJTI: "jti_1",
		EventType: ssdomain.CaepEventTypeSessionRevoked,
		Subject: ssdomain.SsfSubject{
			SubjectType: ssdomain.SsfSubjectTypeAgent, TenantID: tenant.ID, PrincipalID: "agent_1",
		},
		VerificationResult: ssdomain.SecurityEventVerificationAccepted, ReceivedAt: testClock(),
	}
	if err := repo.Save(context.Background(), event); err != nil {
		t.Fatalf("Save: %v", err)
	}

	exists, err = repo.ExistsByJTI(context.Background(), tenant.ID, stream.ID, "jti_1")
	if err != nil {
		t.Fatalf("ExistsByJTI: %v", err)
	}
	if !exists {
		t.Fatal("saved jti must now be detected as replay")
	}

	// 同じ (stream_id, set_jti) を再度 Save してもエラーにはならず (ON CONFLICT DO
	// NOTHING)、レコードは1件のまま (DB 制約が二重挿入そのものを吸収する)。
	duplicate := *event
	duplicate.ID = newUUID(t)
	if err := repo.Save(context.Background(), &duplicate); err != nil {
		t.Fatalf("Save duplicate: %v", err)
	}
	var count int
	if err := db.QueryRow(context.Background(),
		"SELECT count(*) FROM received_security_events WHERE stream_id=$1 AND set_jti=$2",
		stream.ID, "jti_1",
	).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 row after duplicate Save, got %d", count)
	}
}
