package bootstrap

import (
	"context"
	"testing"
	"time"

	auditmemory "github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
)

//spec:covers REQ-AUDIT-002: 共通配信点は tenant、時刻、行為者と変更項目を監査へ投影し、内容が同じ別発生を省かない。
func TestCharacterizeAuditEmission(t *testing.T) {
	deps, _ := newSecurityNotificationDeps(t)
	repo := auditmemory.NewAuditEventStore(100)
	deps.Audit.AuditEventRepo = repo
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	event := &idmdomain.UserUpdated{At: now, TenantID: "tenant-a", ActorUserID: "admin", TargetUserID: "alice", ChangedFields: []string{"email"}}
	emit := deps.NewEmitFunc(logging.Default())
	emit(event)
	emit(event)
	rows, err := repo.List(context.Background(), auditports.AuditEventQuery{TenantID: "tenant-a"})
	if err != nil || len(rows) != 2 {
		t.Fatalf("projection = %+v,%v", rows, err)
	}
	for _, row := range rows {
		if row.Type != "UserUpdated" || !row.OccurredAt.Equal(now) || row.Payload["actorUserId"] != "admin" || row.Payload["targetUserId"] != "alice" {
			t.Fatalf("record = %+v", row)
		}
		fields, ok := row.Payload["changedFields"].([]any)
		if !ok || len(fields) != 1 || fields[0] != "email" {
			t.Fatalf("fields = %v", fields)
		}
	}
	if rows[0].ID == rows[1].ID {
		t.Fatal("distinct occurrences share an ID")
	}
	if rows, _ := repo.List(context.Background(), auditports.AuditEventQuery{TenantID: "tenant-b"}); len(rows) != 0 {
		t.Fatal("tenant isolation lost")
	}
}
