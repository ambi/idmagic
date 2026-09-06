package db_postgres

// 受理した 1 行の完全な書き込み集合を、1 つの PostgreSQL トランザクションで確定する。
// メンバーシップの追加または解除と監査記録は同じ境界に入る。行の途中で失敗すれば
// 部分的な変更は残らず、他の受理済み行は巻き戻さない
// (docs/contexts/identity-management/internals.md)。

import (
	"context"
	"encoding/json"
	"errors"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/jackc/pgx/v5"
)

var errMembershipMutationWithoutMember = errors.New("an add mutation must carry the membership it creates")

type GroupMembershipImportRowCommitter struct{ Pool sharedpg.DB }

func (c GroupMembershipImportRowCommitter) CommitGroupMembershipImportRow(
	ctx context.Context, mutation groupports.GroupMembershipImportRowMutation,
) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	if mutation.Release {
		if _, err := New(tx).RemoveGroupMember(ctx, RemoveGroupMemberParams{
			TenantID: mutation.TenantID, GroupID: mutation.GroupID, UserID: mutation.UserID,
		}); err != nil {
			return err
		}
	} else {
		if mutation.Member == nil {
			return errMembershipMutationWithoutMember
		}
		if _, err := New(tx).AddGroupMember(ctx, AddGroupMemberParams{
			GroupID: mutation.Member.GroupID, UserID: mutation.Member.UserID,
			Source: string(mutation.Member.Source.Effective()), CreatedAt: mutation.Member.CreatedAt,
		}); err != nil {
			return err
		}
	}
	if err := writeGroupMembershipImportAudit(ctx, tx, mutation); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// writeGroupMembershipImportAudit は行番号もセル値も載せない。監査に残るのは、
// 誰が、どの Group の、どの User の所属を、どちらへ変えたかだけである。
func writeGroupMembershipImportAudit(
	ctx context.Context, tx pgx.Tx, mutation groupports.GroupMembershipImportRowMutation,
) error {
	auditID, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"actorUserId": mutation.ActorUserID, "groupId": mutation.GroupID, "userId": mutation.UserID,
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events (id, tenant_id, type, user_id, occurred_at, payload)
        VALUES ($1, $2, $3, $4, $5, $6)`,
		auditID, mutation.TenantID, mutation.AuditEventType, mutation.ActorUserID, mutation.Now, payload)
	return err
}

var _ groupports.GroupMembershipImportRowCommitter = GroupMembershipImportRowCommitter{}
