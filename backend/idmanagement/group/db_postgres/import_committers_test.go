package db_postgres

import (
	"context"
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

type importAuditRow struct {
	tenantID, eventType, userID string
}

func importAuditEvents(ctx context.Context, t *testing.T, tenantID string) []importAuditRow {
	t.Helper()
	rows, err := pgtest.Require(t).Query(ctx,
		`SELECT tenant_id, type, user_id::text FROM audit_events WHERE tenant_id = $1 ORDER BY occurred_at, id`, tenantID)
	if err != nil {
		t.Fatalf("query audit events: %v", err)
	}
	defer rows.Close()
	var out []importAuditRow
	for rows.Next() {
		var row importAuditRow
		if err := rows.Scan(&row.tenantID, &row.eventType, &row.userID); err != nil {
			t.Fatalf("scan audit event: %v", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGroupImportRowCommitterRecordsAuditAndEnqueuesOneReconcile(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenant := seedTenant(t, db)
	actor := seedUser(t, db, tenant.ID)
	now := testClock()
	group := &groupdomain.Group{
		ID: newUUID(t), TenantID: tenant.ID, Name: uniqueID("imported-group"), Roles: []string{},
		CreatedAt: now, UpdatedAt: now,
	}
	mutation := groupports.GroupImportRowMutation{
		After: group, Changed: []string{"name"}, ActorUserID: actor.ID, AuditEventType: "group.imported",
		ReconcileGroupID: group.ID, ReconcileVersion: 3, Now: now,
	}
	committer := GroupImportRowCommitter{Pool: db}
	for range 2 {
		if err := committer.CommitGroupImportRow(ctx, mutation); err != nil {
			t.Fatalf("CommitGroupImportRow: %v", err)
		}
	}

	events := importAuditEvents(ctx, t, tenant.ID)
	want := importAuditRow{tenantID: tenant.ID, eventType: "group.imported", userID: actor.ID}
	if len(events) != 2 || events[0] != want || events[1] != want {
		t.Fatalf("audit events = %+v, want two of %+v", events, want)
	}

	var count int
	var kind, lane, status, dedupKey string
	var attempts, maxAttempts int
	err := db.QueryRow(ctx, `SELECT count(*) OVER (), kind, lane, status, dedup_key, attempts, max_attempts
		FROM jobs WHERE tenant_id = $1`, tenant.ID).Scan(&count, &kind, &lane, &status, &dedupKey, &attempts, &maxAttempts)
	if err != nil {
		t.Fatalf("query reconcile job: %v", err)
	}
	if count != 1 {
		t.Fatalf("reconcile jobs = %d, want 1 (the second commit must hit the dedup key)", count)
	}
	if kind != string(jobsdomain.KindDynamicGroupReconcile) || lane != string(jobsdomain.LaneBulk) ||
		status != "queued" || dedupKey != "dynamic-group:"+group.ID+":v3" ||
		attempts != 0 || maxAttempts != jobsdomain.DefaultMaxAttempts {
		t.Fatalf("job = kind %s lane %s status %s dedup %s attempts %d/%d", kind, lane, status, dedupKey, attempts, maxAttempts)
	}
}

func TestGroupMembershipImportRowCommitterRecordsAudit(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenant := seedTenant(t, db)
	actor := seedUser(t, db, tenant.ID)
	member := seedUser(t, db, tenant.ID)
	group := seedGroup(t, db, tenant.ID)
	now := testClock()
	err := GroupMembershipImportRowCommitter{Pool: db}.CommitGroupMembershipImportRow(ctx, groupports.GroupMembershipImportRowMutation{
		TenantID: tenant.ID, GroupID: group.ID, UserID: member.ID,
		Member:      &groupdomain.GroupMember{GroupID: group.ID, UserID: member.ID, CreatedAt: now},
		ActorUserID: actor.ID, AuditEventType: "group.membership_imported", Now: now,
	})
	if err != nil {
		t.Fatalf("CommitGroupMembershipImportRow: %v", err)
	}
	events := importAuditEvents(ctx, t, tenant.ID)
	want := importAuditRow{tenantID: tenant.ID, eventType: "group.membership_imported", userID: actor.ID}
	if len(events) != 1 || events[0] != want {
		t.Fatalf("audit events = %+v, want [%+v]", events, want)
	}
}
