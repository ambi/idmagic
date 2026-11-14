package usecases

// Group CSV の適用。保存済みのプレビューペイロードを現在状態に対して再計画し、
// 受理した行ごとに 1 つの完全な変更集合を原子的な確定ポートへ渡す。行の失敗は
// その行だけを拒否へ落とし、先行して受理した行を巻き戻さない。

import (
	"context"
	"errors"
	"io"
	"slices"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type GroupImportApplyDeps struct {
	Plan      GroupImportPlanDeps
	Committer groupports.GroupImportRowCommitter
	// Emit は Hard Quota で拒否した行の QuotaExceeded の発行先であり、必須である。
	Emit func(spec.DomainEvent) error
}

// ApplyGroupImport は不変なプレビューペイロードを現在状態に対して再計画し、
// 受理した各行を 1 行 1 トランザクションで確定する。
func ApplyGroupImport(
	ctx context.Context,
	deps GroupImportApplyDeps,
	input io.Reader,
	policy idmdomain.CSVTransferPolicy,
	actorUserID string,
	now time.Time,
	emit func(groupdomain.GroupImportRowPlan) error,
) (GroupImportPlanSummary, error) {
	var applied GroupImportPlanSummary
	if deps.Committer == nil || deps.Emit == nil {
		return applied, errors.New("group import apply dependencies are incomplete")
	}
	now = now.UTC()
	_, err := PlanGroupImport(ctx, deps.Plan, input, policy, func(row groupdomain.GroupImportRowPlan) error {
		final := row
		switch row.Action {
		case groupdomain.GroupImportCreate, groupdomain.GroupImportUpdate, groupdomain.GroupImportDeleted:
			mutation, err := prepareGroupImportMutation(row, actorUserID, now)
			if err == nil {
				err = deps.Committer.CommitGroupImportRow(ctx, mutation)
				// 再評価ジョブの active_jobs の拒否は Jobs の投入器が発行するので、
				// ここでは行が消費する groups の拒否だけを記録する。
				idmusecases.EmitQuotaRefusal(ctx, deps.Emit, err, tenancydomain.ResourceGroups, now)
			}
			if err != nil {
				final = groupdomain.RejectedGroupImportRow(row.Row, "", "apply_failed")
			}
		}
		applied.Observe(final)
		if emit != nil {
			return emit(final)
		}
		return nil
	})
	return applied, err
}

func prepareGroupImportMutation(row groupdomain.GroupImportRowPlan, actorUserID string, now time.Time) (groupports.GroupImportRowMutation, error) {
	mutation := groupports.GroupImportRowMutation{
		Before: row.Before, ActorUserID: actorUserID, Changed: row.Changed, Now: now,
	}
	if row.Action == groupdomain.GroupImportDeleted {
		mutation.Delete = true
		mutation.RemovedMemberships = row.DeletedMemberships
		mutation.AuditEventType = "GroupDeleted"
		mutation.ReleasesGroupQuota = true
		return mutation, nil
	}

	after := *row.Group
	after.Roles = append([]string(nil), row.Group.Roles...)
	mutation.After = &after
	if row.Action == groupdomain.GroupImportCreate {
		id, err := groupdomain.NewGroupID()
		if err != nil {
			return groupports.GroupImportRowMutation{}, err
		}
		after.ID = id
		after.CreatedAt = now
		after.UpdatedAt = now
		mutation.AuditEventType = "GroupCreated"
		mutation.ConsumesGroupQuota = true
		mutation.Changed = []string{"group"}
	} else {
		after.UpdatedAt = now
		mutation.AuditEventType = "GroupUpdated"
	}
	if row.Rule != nil {
		rule := *row.Rule
		rule.GroupID = after.ID
		rule.TenantID = after.TenantID
		if rule.CreatedAt.IsZero() {
			rule.CreatedAt = now
		}
		rule.UpdatedAt = now
		if err := rule.Validate(); err != nil {
			return groupports.GroupImportRowMutation{}, err
		}
		mutation.Rule = &rule
		if row.RuleChanged {
			mutation.Changed = appendUnique(mutation.Changed, "dynamic_rule")
			// 規則の版が上がったので、旧版の membership は直ちに実効ロールから外れる。
			// 新版の評価は同じトランザクションへ投入する再評価ジョブが引き受ける。
			if rule.Enabled {
				mutation.ReconcileGroupID = after.ID
				mutation.ReconcileVersion = rule.Version
			}
		}
	}
	if err := after.Validate(); err != nil {
		return groupports.GroupImportRowMutation{}, err
	}
	return mutation, nil
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}
