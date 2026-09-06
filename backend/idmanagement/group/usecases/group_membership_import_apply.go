package usecases

// メンバーシップ CSV の適用。保存済みのプレビューペイロードを現在の所属に対して
// 再計画し、受理した行ごとに 1 つの完全な変更集合を原子的な確定ポートへ渡す。
// 行の失敗はその行だけを拒否へ落とし、先行して受理した行を巻き戻さない。

import (
	"context"
	"errors"
	"io"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

type GroupMembershipImportApplyDeps struct {
	Plan      GroupMembershipImportPlanDeps
	Committer groupports.GroupMembershipImportRowCommitter
}

// ApplyGroupMembershipImport は不変なプレビューペイロードを現在の所属に対して
// 再計画し、受理した各行を 1 行 1 トランザクションで確定する。
func ApplyGroupMembershipImport(
	ctx context.Context,
	deps GroupMembershipImportApplyDeps,
	groupID string,
	input io.Reader,
	policy idmdomain.CSVTransferPolicy,
	actorUserID string,
	now time.Time,
	emit func(groupdomain.GroupMembershipImportRowPlan) error,
) (GroupMembershipImportPlanSummary, error) {
	var applied GroupMembershipImportPlanSummary
	if deps.Committer == nil {
		return applied, errors.New("group membership import apply dependencies are incomplete")
	}
	now = now.UTC()
	tenantID := tenancy.TenantID(ctx)
	_, err := PlanGroupMembershipImport(ctx, deps.Plan, groupID, input, policy,
		func(row groupdomain.GroupMembershipImportRowPlan) error {
			final := row
			switch row.Action {
			case groupdomain.GroupMembershipImportAdded, groupdomain.GroupMembershipImportRemoved:
				mutation := prepareGroupMembershipImportMutation(row, tenantID, groupID, actorUserID, now)
				if err := deps.Committer.CommitGroupMembershipImportRow(ctx, mutation); err != nil {
					final = groupdomain.RejectedGroupMembershipImportRow(row.Row, "", "apply_failed")
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

func prepareGroupMembershipImportMutation(
	row groupdomain.GroupMembershipImportRowPlan,
	tenantID, groupID, actorUserID string,
	now time.Time,
) groupports.GroupMembershipImportRowMutation {
	mutation := groupports.GroupMembershipImportRowMutation{
		TenantID: tenantID, GroupID: groupID, UserID: row.UserID,
		ActorUserID: actorUserID, Now: now,
	}
	if row.Action == groupdomain.GroupMembershipImportRemoved {
		mutation.Release = true
		mutation.AuditEventType = "GroupMemberRemoved"
		return mutation
	}
	mutation.AuditEventType = "GroupMemberAdded"
	// CSV から作る所属は必ず手動である。動的規則が作る所属は規則の評価だけが持ち、
	// そちらの行はそもそも計画器が拒否している。
	mutation.Member = &groupdomain.GroupMember{
		GroupID: groupID, UserID: row.UserID,
		Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	}
	return mutation
}
