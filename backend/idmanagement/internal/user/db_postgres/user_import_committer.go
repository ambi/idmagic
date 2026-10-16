package db_postgres

import (
	"context"
	"encoding/json"

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
	pool       sharedpg.DB
	quotasInTx func(pgx.Tx) tenantports.QuotaRepository
}

// NewUserImportRowCommitter は、quotasInTx が返す Tenancy の QuotaRepository で、
// 行と同じトランザクションの中でユーザーの使用量を加算する committer を返す。
func NewUserImportRowCommitter(pool sharedpg.DB, quotasInTx func(pgx.Tx) tenantports.QuotaRepository) UserImportRowCommitter {
	return UserImportRowCommitter{pool: pool, quotasInTx: quotasInTx}
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
		historyID, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		if err := New(tx).InsertImportedPasswordHistory(ctx, InsertImportedPasswordHistoryParams{
			ID: historyID, UserID: mutation.After.ID, Encoded: mutation.PasswordHistoryHash, CreatedAt: mutation.Now,
		}); err != nil {
			return err
		}
	}
	auditID, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"actorUserId": mutation.ActorUserID, "targetUserId": mutation.After.ID, "changedFields": mutation.Changed,
	})
	if err != nil {
		return err
	}
	if err := New(tx).InsertUserImportAuditEvent(ctx, InsertUserImportAuditEventParams{
		ID: auditID, TenantID: mutation.After.TenantID, Type: mutation.AuditEventType, UserID: mutation.After.ID,
		OccurredAt: mutation.Now, Payload: payload,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ userports.UserImportRowCommitter = UserImportRowCommitter{}
