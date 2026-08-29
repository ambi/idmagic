package db_postgres

// 受理した 1 行の完全な書き込み集合を、1 つの PostgreSQL トランザクションで確定する。
// Group 本体、動的規則、membership の cascade、グループクォータの増減、監査記録は
// 同じ境界に入る。行の途中で失敗すれば先行する部分的な変更は残らず、他の受理済み行は
// 巻き戻さない (docs/contexts/identity-management/internals.md)。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/jackc/pgx/v5"
)

type GroupImportRowCommitter struct{ Pool sharedpg.DB }

func (c GroupImportRowCommitter) CommitGroupImportRow(ctx context.Context, mutation groupports.GroupImportRowMutation) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	if mutation.Delete {
		if err := commitGroupImportDeletion(ctx, tx, mutation); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if mutation.After == nil {
		return errors.New("an upsert mutation must carry the resulting group")
	}
	if mutation.ConsumesGroupQuota {
		if err := tenancypostgres.NewQuotaRepository(tx).CheckAndIncrement(ctx, mutation.After.TenantID, tenancydomain.ResourceGroups, 1); err != nil {
			return err
		}
	}
	if err := saveGroupInTx(ctx, tx, mutation.After); err != nil {
		return err
	}
	if mutation.Rule != nil {
		refs, err := json.Marshal(mutation.Rule.ReferencedAttributes)
		if err != nil {
			return err
		}
		if err := New(tx).SaveDynamicGroupRule(ctx, SaveDynamicGroupRuleParams{
			GroupID: mutation.Rule.GroupID, TenantID: mutation.Rule.TenantID, Expression: mutation.Rule.Expression,
			Enabled: mutation.Rule.Enabled, Version: mutation.Rule.Version, ReferencedAttributes: refs,
			CreatedAt: mutation.Rule.CreatedAt, UpdatedAt: mutation.Rule.UpdatedAt,
		}); err != nil {
			return err
		}
	}
	if err := writeGroupImportAudit(ctx, tx, mutation, mutation.After.TenantID, mutation.After.ID); err != nil {
		return err
	}
	// 規則の版が上がった行は、新版の評価を同じトランザクションへ投入する。別の
	// 経路にすると、行が確定したのに再評価だけが失われる窓ができる。
	if mutation.ReconcileGroupID != "" {
		if err := enqueueGroupReconcileInTx(ctx, tx, mutation); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func commitGroupImportDeletion(ctx context.Context, tx pgx.Tx, mutation groupports.GroupImportRowMutation) error {
	if mutation.Before == nil {
		return errors.New("a delete mutation must name the group it removes")
	}
	tenantID, groupID := mutation.Before.TenantID, mutation.Before.ID
	for _, userID := range mutation.RemovedMemberships {
		if _, err := New(tx).RemoveGroupMember(ctx, RemoveGroupMemberParams{
			TenantID: tenantID, GroupID: groupID, UserID: userID,
		}); err != nil {
			return err
		}
	}
	if err := New(tx).DeleteGroup(ctx, DeleteGroupParams{TenantID: tenantID, ID: groupID}); err != nil {
		return err
	}
	if mutation.ReleasesGroupQuota {
		if err := tenancypostgres.NewQuotaRepository(tx).Decrement(ctx, tenantID, tenancydomain.ResourceGroups, 1); err != nil {
			return err
		}
	}
	return writeGroupImportAudit(ctx, tx, mutation, tenantID, groupID)
}

func saveGroupInTx(ctx context.Context, tx pgx.Tx, group *groupdomain.Group) error {
	roles := group.Roles
	if roles == nil {
		roles = []string{}
	}
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return err
	}
	attributes := group.Attributes
	if attributes == nil {
		attributes = map[string]userdomain.AttributeValue{}
	}
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return err
	}
	return New(tx).SaveGroup(ctx, SaveGroupParams{
		ID: group.ID, TenantID: group.TenantID, Name: group.Name,
		Description: textOrNil(group.Description), Email: textOrNil(group.Email),
		Attributes: attributesJSON, Roles: rolesJSON,
		MembershipType: string(group.MembershipType.Effective()),
		CreatedAt:      group.CreatedAt, UpdatedAt: group.UpdatedAt,
	})
}

// writeGroupImportAudit は行番号もセル値も載せない。監査に残るのは、誰が、どの
// Group を、どの項目について変えたかだけである。
func writeGroupImportAudit(ctx context.Context, tx pgx.Tx, mutation groupports.GroupImportRowMutation, tenantID, groupID string) error {
	auditID, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"actorUserId": mutation.ActorUserID, "groupId": groupID, "changedFields": mutation.Changed,
		"removedMemberships": len(mutation.RemovedMemberships),
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_events (id, tenant_id, type, user_id, occurred_at, payload)
        VALUES ($1, $2, $3, $4, $5, $6)`, auditID, tenantID, mutation.AuditEventType, mutation.ActorUserID, mutation.Now, payload)
	return err
}

// enqueueGroupReconcileInTx は再評価ジョブを行と同じトランザクションへ入れる。
// Jobs のリポジトリは接続プールを要求するため、ここでは投入だけを同じ SQL 形で
// 直接行う。dedup key は Jobs の部分一意索引がそのまま効く。
func enqueueGroupReconcileInTx(ctx context.Context, tx pgx.Tx, mutation groupports.GroupImportRowMutation) error {
	params, err := json.Marshal(map[string]any{
		"group_id": mutation.ReconcileGroupID, "rule_version": mutation.ReconcileVersion,
	})
	if err != nil {
		return err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	dedupKey := fmt.Sprintf("dynamic-group:%s:v%d", mutation.ReconcileGroupID, mutation.ReconcileVersion)
	_, err = tx.Exec(ctx, `INSERT INTO jobs (id, tenant_id, kind, lane, status, params, attempts, max_attempts, dedup_key, run_at, created_at, updated_at)
        VALUES ($1, $2, $3, $4, 'queued', $5, 0, $6, $7, $8, $8, $8)
        ON CONFLICT (tenant_id, dedup_key) WHERE dedup_key IS NOT NULL AND status IN ('queued', 'running')
        DO NOTHING`,
		id, mutation.After.TenantID, string(jobsdomain.KindDynamicGroupReconcile), string(jobsdomain.LaneBulk),
		params, jobsdomain.DefaultMaxAttempts, dedupKey, mutation.Now)
	return err
}

var _ groupports.GroupImportRowCommitter = GroupImportRowCommitter{}
