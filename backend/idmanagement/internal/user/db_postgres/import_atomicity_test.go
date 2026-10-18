package db_postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//spec:covers REQ-IDMANAGEMENT-004
func TestUserImportRollsBackWhenHistoryOrAuditFails(t *testing.T) {
	db := pgtest.Require(t)
	for _, table := range []string{"password_history", "audit_events"} {
		t.Run(table, func(t *testing.T) {
			ctx := t.Context()
			tenant := seedTenant(t, db)
			actor := seedUser(t, db, tenant.ID)
			now := testClock()
			user := &userdomain.User{
				ID: newUUID(t), TenantID: tenant.ID, PreferredUsername: uniqueID("atomic-row"),
				PasswordHash: "encoded", Roles: []string{}, CreatedAt: now, UpdatedAt: now,
			}
			committer := NewUserImportRowCommitter(failImportDB{DB: db, table: table}, tenancy.QuotaRepositoryInTx)
			err := committer.CommitUserImportRow(ctx, userports.UserImportRowMutation{
				After: user, ActorUserID: actor.ID, AuditEventType: "UserCreated", Changed: []string{"password"},
				ConsumesUserQuota: true, PasswordHistoryHash: "encoded", Now: now,
			})
			if !errors.Is(err, errImportWrite) {
				t.Fatalf("error=%v, want injected %s failure", err, table)
			}
			for _, query := range []string{
				`SELECT count(*) FROM users WHERE id = $1`,
				`SELECT count(*) FROM password_history WHERE user_id = $1`,
				`SELECT count(*) FROM audit_events WHERE user_id = $1`,
			} {
				var count int
				if err := db.QueryRow(ctx, query, user.ID).Scan(&count); err != nil || count != 0 {
					t.Fatalf("query=%s count=%d error=%v, want zero", query, count, err)
				}
			}
			usage, err := tenancy.NewQuotaRepository(db).GetUsage(ctx, tenant.ID)
			if err != nil || usage.Users != 0 {
				t.Fatalf("usage=%+v error=%v, want zero users", usage, err)
			}
		})
	}
}

var errImportWrite = errors.New("injected import write failure")

type failImportDB struct {
	sharedpg.DB
	table string
}

func (db failImportDB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := db.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return failImportTx{Tx: tx, table: db.table}, nil
}

type failImportTx struct {
	pgx.Tx
	table string
}

func (tx failImportTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "INSERT INTO "+tx.table) {
		return pgconn.CommandTag{}, errImportWrite
	}
	return tx.Tx.Exec(ctx, sql, args...)
}
