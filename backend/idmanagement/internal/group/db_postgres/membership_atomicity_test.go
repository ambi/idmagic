package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/audit"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-IDMANAGEMENT-026
func TestMembershipImportRollsBackWhenAuditFails(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	group := seedGroup(t, db, tenant.ID)
	err := NewGroupMembershipImportRowCommitter(db, audit.RecordWriterInTx).CommitGroupMembershipImportRow(ctx, groupports.GroupMembershipImportRowMutation{
		TenantID: tenant.ID, GroupID: group.ID, UserID: user.ID,
		Member:      &groupdomain.GroupMember{GroupID: group.ID, UserID: user.ID, CreatedAt: testClock()},
		ActorUserID: "invalid-uuid", AuditEventType: "group.membership_imported", Now: testClock(),
	})
	if err == nil {
		t.Fatal("invalid audit actor accepted")
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM group_members WHERE group_id = $1 AND user_id = $2`, group.ID, user.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("members=%d error=%v, want none", count, err)
	}
	if events := importAuditEvents(ctx, t, tenant.ID); len(events) != 0 {
		t.Fatalf("audit=%+v, want none", events)
	}
}

func TestMembershipImportRequiresTransactionalDependencies(t *testing.T) {
	db := pgtest.Require(t)
	for name, construct := range map[string]func(){
		"database": func() { NewGroupMembershipImportRowCommitter(nil, audit.RecordWriterInTx) },
		"audit":    func() { NewGroupMembershipImportRowCommitter(db, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("missing required dependency accepted")
				}
			}()
			construct()
		})
	}
}
