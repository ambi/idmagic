package usecases

// メンバーシップ CSV の計画器。プレビューと適用が共有する 1 個の決定的な計算であり、
// リポジトリの読み取り以外に効果を持たない。適用はこの計画器を現在の所属に対して
// 再び走らせるため、プレビューが暗黙の楽観的ロックの迂回路にならない
// (docs/contexts/identity-management/internals.md)。

import (
	"context"
	"errors"
	"io"
	"strings"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

const groupMembershipImportPageSize = 1000

type GroupMembershipImportPlanDeps struct {
	GroupRepo groupports.GroupRepository
	UserRepo  userports.UserRepository
	// GroupOwnershipGuard は対象 Group が外部管理かどうかを返す。nil、または
	// 失敗は「外部管理」として fail-closed に扱う。
	GroupOwnershipGuard groupports.GroupSourceOwnershipGuard
	// UserOwnershipGuard は行の対象 User について同じことを返す。
	UserOwnershipGuard userports.UserSourceOwnershipGuard
	PageSize           int
}

func (d GroupMembershipImportPlanDeps) pageSize() int {
	if d.PageSize > 0 {
		return d.PageSize
	}
	return groupMembershipImportPageSize
}

// GroupMembershipImportPlanSummary は行操作ごとの件数。解除は所属していた User の
// 実効ロールを変えるため、他の操作と分けて持つ。
type GroupMembershipImportPlanSummary struct {
	TotalRows     int `json:"total_rows"`
	AddedRows     int `json:"added_rows"`
	RemovedRows   int `json:"removed_rows"`
	UnchangedRows int `json:"unchanged_rows"`
	RejectedRows  int `json:"rejected_rows"`
}

func (s *GroupMembershipImportPlanSummary) Observe(row groupdomain.GroupMembershipImportRowPlan) {
	s.TotalRows++
	switch row.Action {
	case groupdomain.GroupMembershipImportAdded:
		s.AddedRows++
	case groupdomain.GroupMembershipImportRemoved:
		s.RemovedRows++
	case groupdomain.GroupMembershipImportUnchanged:
		s.UnchangedRows++
	case groupdomain.GroupMembershipImportRejected:
		s.RejectedRows++
	}
}

// groupMembershipImportIndex は 1 回のページ走査で作る上限付きの索引。行ごとに
// リポジトリを引かないためであり、所有権の判定も同じ走査でまとめて解決する。
type groupMembershipImportIndex struct {
	group                *groupdomain.Group
	membersByUserID      map[string]*groupdomain.GroupMember
	usersByID            map[string]*userdomain.User
	usersByUsername      map[string]*userdomain.User
	sourceManagedUsers   map[string]bool
	ownershipUnavailable bool
}

// loadGroupMembershipImportIndex は対象 Group、現在の所属、テナントの User 索引を
// 読む。Group 側の拒否理由はここで CSVError として確定し、行の走査を始める前に
// ファイル全体を止める。
func loadGroupMembershipImportIndex(
	ctx context.Context,
	deps GroupMembershipImportPlanDeps,
	tenantID, groupID string,
) (groupMembershipImportIndex, error) {
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, groupID)
	if err != nil {
		return groupMembershipImportIndex{}, err
	}
	if group == nil {
		return groupMembershipImportIndex{}, &idmdomain.CSVError{Column: "group_id", Code: "target_not_found"}
	}
	// 動的グループは手動の追加と解除を受け付けない。行ごとに拒否しても全行が同じ
	// 理由で落ちるだけなので、ファイルごと断る。
	if group.MembershipType.Effective() == groupdomain.GroupMembershipDynamic {
		return groupMembershipImportIndex{}, &idmdomain.CSVError{Column: "group_id", Code: "dynamic_group"}
	}
	if deps.GroupOwnershipGuard == nil {
		return groupMembershipImportIndex{}, &idmdomain.CSVError{Column: "group_id", Code: "source_managed"}
	}
	managedGroups, err := deps.GroupOwnershipGuard.SourceManagedGroupIDs(ctx, tenantID, []string{group.ID})
	if err != nil || managedGroups[group.ID] {
		return groupMembershipImportIndex{}, &idmdomain.CSVError{Column: "group_id", Code: "source_managed"}
	}

	index := groupMembershipImportIndex{
		group:              group,
		membersByUserID:    map[string]*groupdomain.GroupMember{},
		usersByID:          map[string]*userdomain.User{},
		usersByUsername:    map[string]*userdomain.User{},
		sourceManagedUsers: map[string]bool{},
	}
	members, err := deps.GroupRepo.ListMembersByGroup(ctx, tenantID, group.ID)
	if err != nil {
		return groupMembershipImportIndex{}, err
	}
	for _, member := range members {
		if member != nil {
			index.membersByUserID[member.UserID] = member
		}
	}
	if err := loadGroupMembershipUserIndex(ctx, deps, tenantID, &index); err != nil {
		return groupMembershipImportIndex{}, err
	}
	return index, nil
}

func loadGroupMembershipUserIndex(
	ctx context.Context,
	deps GroupMembershipImportPlanDeps,
	tenantID string,
	index *groupMembershipImportIndex,
) error {
	afterUsername, afterID := "", ""
	for {
		page, err := deps.UserRepo.ListPage(ctx, tenantID, afterUsername, afterID, deps.pageSize())
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		ids := make([]string, 0, len(page))
		for _, user := range page {
			if user == nil || user.TenantID != tenantID {
				continue
			}
			index.usersByID[user.ID] = user
			index.usersByUsername[user.PreferredUsername] = user
			ids = append(ids, user.ID)
		}
		if deps.UserOwnershipGuard == nil {
			index.ownershipUnavailable = true
		} else if managed, err := deps.UserOwnershipGuard.SourceManagedUserIDs(ctx, tenantID, ids); err != nil {
			index.ownershipUnavailable = true
		} else {
			for id, value := range managed {
				if value {
					index.sourceManagedUsers[id] = true
				}
			}
		}
		last := page[len(page)-1]
		if last == nil || (last.PreferredUsername == afterUsername && last.ID == afterID) {
			return errors.New("group membership import pagination did not advance")
		}
		afterUsername, afterID = last.PreferredUsername, last.ID
		if len(page) < deps.pageSize() {
			return nil
		}
	}
}

// PlanGroupMembershipImport は 1 個の決定的な計画器で行を流す。対象 Group は
// groupID だけが決め、CSV の `group_id` / `group_name` は照合にしか使わない。
func PlanGroupMembershipImport(
	ctx context.Context,
	deps GroupMembershipImportPlanDeps,
	groupID string,
	input io.Reader,
	policy idmdomain.CSVTransferPolicy,
	emit func(groupdomain.GroupMembershipImportRowPlan) error,
) (GroupMembershipImportPlanSummary, error) {
	var summary GroupMembershipImportPlanSummary
	if deps.GroupRepo == nil || deps.UserRepo == nil {
		return summary, errors.New("group membership import planner dependencies are incomplete")
	}
	tenantID := tenancy.TenantID(ctx)
	schema := groupdomain.NewGroupMembershipCSVSchema()
	reader, err := idmdomain.NewCSVReader(input, schema.Accepts, policy)
	if err != nil {
		return summary, err
	}
	if missing := schema.MissingRequiredColumn(reader.Header()); missing != "" {
		return summary, &idmdomain.CSVError{Row: 1, Column: missing, Code: idmdomain.CSVErrorInvalidHeader}
	}
	index, err := loadGroupMembershipImportIndex(ctx, deps, tenantID, groupID)
	if err != nil {
		return summary, err
	}
	seenUsers := map[string]struct{}{}
	for {
		record, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return summary, nil
		}
		if err != nil {
			return summary, err
		}
		var planned groupdomain.GroupMembershipImportRowPlan
		if record.Error != nil {
			planned = groupdomain.RejectedGroupMembershipImportRow(record.Error.Row, record.Error.Column, record.Error.Code)
		} else {
			planned = planGroupMembershipImportRow(*record.Row, index, seenUsers)
		}
		summary.Observe(planned)
		if emit != nil {
			if err := emit(planned); err != nil {
				return summary, err
			}
		}
	}
}

func planGroupMembershipImportRow(
	row idmdomain.CSVRow,
	index groupMembershipImportIndex,
	seenUsers map[string]struct{},
) groupdomain.GroupMembershipImportRowPlan {
	if column, code := verifyGroupMembershipRowTarget(row, index.group); code != "" {
		return groupdomain.RejectedGroupMembershipImportRow(row.Number, column, code)
	}
	identifier, code := groupdomain.GroupMembershipCSVIdentifierOf(row)
	if code != "" {
		return groupdomain.RejectedGroupMembershipImportRow(row.Number, "", code)
	}
	state, err := groupdomain.ParseGroupMembershipCSVState(row.TrimmedCell("membership_state"))
	if err != nil {
		return groupdomain.RejectedGroupMembershipImportRow(row.Number, "membership_state", "invalid_membership_state")
	}

	user, column, resolveCode := resolveGroupMembershipTargetUser(identifier, index)
	if resolveCode != "" {
		return groupdomain.RejectedGroupMembershipImportRow(row.Number, column, resolveCode)
	}
	// 同じ User を複数の行が指すファイルは、どの意図が勝つかをファイルの並び順に
	// 委ねないため拒否する。
	if _, duplicate := seenUsers[user.ID]; duplicate {
		return groupdomain.RejectedGroupMembershipImportRow(row.Number, "user_id", "duplicate_target")
	}
	seenUsers[user.ID] = struct{}{}
	// 所有権の判定不能と外部管理は、追加でも解除でも同じく fail-closed に拒否する。
	if index.ownershipUnavailable || index.sourceManagedUsers[user.ID] {
		return groupdomain.RejectedGroupMembershipImportRow(row.Number, "user_id", "source_managed")
	}

	current := index.membersByUserID[user.ID]
	// 動的規則が作った所属は、`present` でも `absent` でも書き換えない。`present`
	// は何も書き込まないが、規則が次に一致しなくなればその所属は消えるため、
	// 変更なしと答えると管理者が受け取る保証と実際の寿命が食い違う。
	if current != nil && current.Source.Effective() == groupdomain.MembershipSourceDynamicRule {
		return groupdomain.RejectedGroupMembershipImportRow(row.Number, "membership_state", "dynamic_membership")
	}

	planned := groupdomain.GroupMembershipImportRowPlan{
		Row: row.Number, Identifier: identifier, UserID: user.ID,
		Action: groupdomain.GroupMembershipImportUnchanged,
	}
	switch {
	case state == groupdomain.GroupMembershipStatePresent && current == nil:
		planned.Action = groupdomain.GroupMembershipImportAdded
	case state == groupdomain.GroupMembershipStateAbsent && current != nil:
		planned.Action = groupdomain.GroupMembershipImportRemoved
	}
	return planned
}

// verifyGroupMembershipRowTarget は読み取り専用の照合列を見る。対象を決めるのは
// URL であり、別の Group を指す行はその行だけを拒否して、指された Group には
// 何も書かない。
func verifyGroupMembershipRowTarget(row idmdomain.CSVRow, group *groupdomain.Group) (string, idmdomain.CSVErrorCode) {
	if declared := row.TrimmedCell("group_id"); declared != "" && declared != group.ID {
		return "group_id", "group_mismatch"
	}
	declaredName := row.TrimmedCell("group_name")
	if declaredName != "" && groupdomain.GroupMembershipNameKey(declaredName) != groupdomain.GroupMembershipNameKey(group.Name) {
		return "group_name", "group_mismatch"
	}
	return "", ""
}

// resolveGroupMembershipTargetUser は `user_id` を優先し、無ければ
// `preferred_username` で解決する。両方が別の User を指す行は、どちらを選ぶかを
// 決めずに拒否する。
func resolveGroupMembershipTargetUser(
	identifier groupdomain.GroupMembershipCSVIdentifier,
	index groupMembershipImportIndex,
) (*userdomain.User, string, idmdomain.CSVErrorCode) {
	if identifier.UserID != "" {
		user := index.usersByID[identifier.UserID]
		if user == nil {
			return nil, "user_id", "target_not_found"
		}
		if identifier.PreferredUsername != "" && !strings.EqualFold(user.PreferredUsername, identifier.PreferredUsername) {
			named := index.usersByUsername[identifier.PreferredUsername]
			if named == nil || named.ID != user.ID {
				return nil, "preferred_username", "identifier_mismatch"
			}
		}
		return user, "", ""
	}
	user := index.usersByUsername[identifier.PreferredUsername]
	if user == nil {
		return nil, "preferred_username", "target_not_found"
	}
	return user, "", ""
}
