package db_postgres

// 受理した 1 行の完全な書き込み集合を、1 つの PostgreSQL トランザクションで確定する。
// Group 本体、動的規則、membership の cascade、グループクォータの増減、監査記録は
// 同じ境界に入る。行の途中で失敗すれば先行する部分的な変更は残らず、他の受理済み行は
// 巻き戻さない (docs/modules/identity-management/features/group-csv.md)。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/jackc/pgx/v5"
)

type GroupImportRowCommitter struct {
	pool          sharedpg.DB
	quotasInTx    func(pgx.Tx) tenantports.QuotaRepository
	enqueuersInTx func(pgx.Tx, tenantports.QuotaRepository, func(spec.DomainEvent)) jobsports.Enqueuer
	emit          func(spec.DomainEvent)
}

// NewGroupImportRowCommitter は、quotasInTx が返す Tenancy の QuotaRepository で、
// 行と同じトランザクションの中でグループの使用量を加減算する committer を返す。
func NewGroupImportRowCommitter(
	pool sharedpg.DB, quotasInTx func(pgx.Tx) tenantports.QuotaRepository,
	enqueuersInTx func(pgx.Tx, tenantports.QuotaRepository, func(spec.DomainEvent)) jobsports.Enqueuer,
	emit func(spec.DomainEvent),
) GroupImportRowCommitter {
	if pool == nil || quotasInTx == nil || enqueuersInTx == nil || emit == nil {
		panic("group import: requires a database, transactional quota and job factories, and an event sink")
	}
	return GroupImportRowCommitter{pool: pool, quotasInTx: quotasInTx, enqueuersInTx: enqueuersInTx, emit: emit}
}

func (c GroupImportRowCommitter) CommitGroupImportRow(ctx context.Context, mutation groupports.GroupImportRowMutation) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return err
	}
	var events []spec.DomainEvent
	defer func() {
		_ = tx.Rollback(ctx)
		// 拒否の通知も Tx の外で監査へ書けるよう、ロールバック後に発行する。
		for _, event := range events {
			if _, refused := event.(*tenancydomain.QuotaExceeded); refused {
				c.emit(event)
			}
		}
	}()
	if mutation.Delete {
		if err := commitGroupImportDeletion(ctx, tx, c.quotasInTx(tx), mutation); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if mutation.After == nil {
		return errors.New("an upsert mutation must carry the resulting group")
	}
	if mutation.ConsumesGroupQuota {
		if err := c.quotasInTx(tx).CheckAndIncrement(ctx, mutation.After.TenantID, tenancydomain.ResourceGroups, 1); err != nil {
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
		enqueuer := c.enqueuersInTx(tx, c.quotasInTx(tx), func(event spec.DomainEvent) { events = append(events, event) })
		if err := enqueueGroupReconcileInTx(ctx, enqueuer, mutation); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	for _, event := range events {
		c.emit(event)
	}
	return nil
}

func commitGroupImportDeletion(ctx context.Context, tx pgx.Tx, quotas tenantports.QuotaRepository, mutation groupports.GroupImportRowMutation) error {
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
		if err := quotas.Decrement(ctx, tenantID, tenancydomain.ResourceGroups, 1); err != nil {
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
		ID: group.ID, TenantID: group.TenantID, Name: group.Name, NameKey: idmdomain.NameKey(group.Name),
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
	return New(tx).InsertGroupImportAuditEvent(ctx, InsertGroupImportAuditEventParams{
		ID: auditID, TenantID: tenantID, Type: mutation.AuditEventType, UserID: mutation.ActorUserID,
		OccurredAt: mutation.Now, Payload: payload,
	})
}

// enqueueGroupReconcileInTx は再評価ジョブを行と同じトランザクションへ入れる。
// レーン、既定値、重複排除と active_jobs の確認は Jobs の操作が担う。
func enqueueGroupReconcileInTx(ctx context.Context, enqueuer jobsports.Enqueuer, mutation groupports.GroupImportRowMutation) error {
	params, err := json.Marshal(map[string]any{
		"group_id": mutation.ReconcileGroupID, "rule_version": mutation.ReconcileVersion,
	})
	if err != nil {
		return err
	}
	dedupKey := fmt.Sprintf("dynamic-group:%s:v%d", mutation.ReconcileGroupID, mutation.ReconcileVersion)
	_, err = enqueuer.Enqueue(ctx, jobsports.EnqueueInput{
		TenantID: mutation.After.TenantID, Kind: jobsdomain.KindDynamicGroupReconcile,
		Params: params, DedupKey: &dedupKey,
	}, mutation.Now)
	return err
}

var _ groupports.GroupImportRowCommitter = GroupImportRowCommitter{}
