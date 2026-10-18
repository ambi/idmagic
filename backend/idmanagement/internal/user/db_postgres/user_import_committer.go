package db_postgres

import (
	"context"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/jackc/pgx/v5"
)

// UserImportRowCommitter persists the complete row write set and its safe
// audit record in one PostgreSQL transaction.
type UserImportRowCommitter struct {
	pool        sharedpg.DB
	quotasInTx  func(pgx.Tx) tenantports.QuotaRepository
	auditInTx   func(pgx.Tx) auditports.AuditEventWriter
	historyInTx func(pgx.Tx) passwordports.PasswordHistoryWriter
}

// NewUserImportCommitter は、quotasInTx が返す Tenancy の QuotaRepository で、
// 行と同じトランザクションの中でユーザーの使用量を加算する committer を返す。
func NewUserImportCommitter(pool sharedpg.DB, quotasInTx func(pgx.Tx) tenantports.QuotaRepository,
	auditInTx func(pgx.Tx) auditports.AuditEventWriter, historyInTx func(pgx.Tx) passwordports.PasswordHistoryWriter,
) UserImportRowCommitter {
	if pool == nil || quotasInTx == nil || auditInTx == nil || historyInTx == nil {
		panic("user import: requires a database and transactional quota, audit, and history writers")
	}
	return UserImportRowCommitter{pool: pool, quotasInTx: quotasInTx, auditInTx: auditInTx, historyInTx: historyInTx}
}

func (c UserImportRowCommitter) CommitUserImportRow(ctx context.Context, mutation userports.UserImportRowMutation) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	if mutation.ConsumesUserQuota {
		if err := c.quotasInTx(tx).CheckAndIncrement(ctx, mutation.After.TenantID, tenancydomain.ResourceUsers, 1); err != nil {
			return err
		}
	}
	if err := internalSaveUser(ctx, tx, mutation.After); err != nil {
		return err
	}
	if mutation.PasswordHistoryHash != "" {
		if err := c.historyInTx(tx).Add(ctx, mutation.After.ID, mutation.PasswordHistoryHash, mutation.Now); err != nil {
			return err
		}
	}
	auditID, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	if err := c.auditInTx(tx).Append(ctx, auditports.AuditEventRecord{
		ID: auditID, TenantID: mutation.After.TenantID, Type: mutation.AuditEventType, OccurredAt: mutation.Now,
		Payload: map[string]any{"actorUserId": mutation.ActorUserID, "targetUserId": mutation.After.ID, "changedFields": mutation.Changed},
	}, mutation.After.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ userports.UserImportRowCommitter = UserImportRowCommitter{}
