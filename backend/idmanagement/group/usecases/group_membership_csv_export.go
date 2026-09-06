package usecases

// メンバーシップ CSV の export。1 つの Group の所属をそのまま不変な成果物へ書き出し、
// 全体を文字列や base64 として実体化しない。返る成果物は必ず、
// PlanGroupMembershipImport が受理するのと同じポリシーと語彙を満たす。

import (
	"context"
	"errors"
	"io"
	"slices"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

const groupMembershipCSVExportPageSize = 1000

type GroupMembershipCSVExportDeps struct {
	GroupRepo groupports.GroupRepository
	UserRepo  userports.UserRepository
	Artifacts idmports.CSVArtifactStore
}

type GroupMembershipCSVExportResult struct {
	Artifact  idmports.CSVArtifact
	Columns   []string
	TotalRows int
}

// GroupMembershipCSVExporter は data export のジョブ境界へ渡す実装。対象 Group は
// フィルターではなく引数で渡し、種別ごとの実装がそれを解釈する。
type GroupMembershipCSVExporter struct {
	Deps   GroupMembershipCSVExportDeps
	Policy idmdomain.CSVTransferPolicy
}

func (e GroupMembershipCSVExporter) policy() idmdomain.CSVTransferPolicy {
	if e.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return e.Policy
}

func (e GroupMembershipCSVExporter) ValidateGroupMembershipCSVColumns(_ context.Context, columns []string) error {
	return validateGroupMembershipCSVExportColumns(groupdomain.NewGroupMembershipCSVSchema(), columns)
}

func (e GroupMembershipCSVExporter) ExportGroupMembershipCSV(
	ctx context.Context, groupID string, columns []string,
) (idmports.CSVArtifact, int, error) {
	result, err := ExportGroupMembershipCSV(ctx, e.Deps, groupID, columns, e.policy())
	return result.Artifact, result.TotalRows, err
}

func ExportGroupMembershipCSV(
	ctx context.Context,
	deps GroupMembershipCSVExportDeps,
	groupID string,
	columns []string,
	policy idmdomain.CSVTransferPolicy,
) (GroupMembershipCSVExportResult, error) {
	var result GroupMembershipCSVExportResult
	if deps.GroupRepo == nil || deps.UserRepo == nil || deps.Artifacts == nil {
		return result, errors.New("group membership CSV export dependencies are incomplete")
	}
	if err := policy.Validate(); err != nil {
		return result, err
	}
	schema := groupdomain.NewGroupMembershipCSVSchema()
	if err := validateGroupMembershipCSVExportColumns(schema, columns); err != nil {
		return result, err
	}
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, groupID)
	if err != nil {
		return result, err
	}
	if group == nil {
		return result, &idmdomain.CSVError{Column: "group_id", Code: "target_not_found"}
	}
	members, err := deps.GroupRepo.ListMembersByGroup(ctx, tenantID, group.ID)
	if err != nil {
		return result, err
	}
	usernames, err := loadGroupMembershipUsernames(ctx, deps, tenantID, columns)
	if err != nil {
		return result, err
	}
	columns = append([]string(nil), columns...)

	rowCount := 0
	artifact, err := deps.Artifacts.PutCSVArtifact(ctx, tenantID, func(output io.Writer) error {
		writer, err := idmdomain.NewCSVWriter(output, columns, policy)
		if err != nil {
			return err
		}
		for _, member := range members {
			if member == nil {
				continue
			}
			record := make([]string, len(columns))
			for i, key := range columns {
				record[i] = groupMembershipCSVExportValue(group, member, usernames[member.UserID], key)
			}
			if err := writer.WriteRow(record); err != nil {
				return err
			}
			rowCount++
		}
		return writer.Close()
	})
	if err != nil {
		return result, err
	}
	return GroupMembershipCSVExportResult{Artifact: artifact, Columns: columns, TotalRows: rowCount}, nil
}

// loadGroupMembershipUsernames は `preferred_username` 列を要求されたときだけ
// テナントの User 索引を作る。所属 1 件ごとにリポジトリを引くと、10,000 件の
// エクスポートが 10,000 回の検索になる。
func loadGroupMembershipUsernames(
	ctx context.Context,
	deps GroupMembershipCSVExportDeps,
	tenantID string,
	columns []string,
) (map[string]string, error) {
	usernames := map[string]string{}
	if !slices.Contains(columns, "preferred_username") {
		return usernames, nil
	}
	afterUsername, afterID := "", ""
	for {
		page, err := deps.UserRepo.ListPage(ctx, tenantID, afterUsername, afterID, groupMembershipCSVExportPageSize)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return usernames, nil
		}
		for _, user := range page {
			if user != nil && user.TenantID == tenantID {
				usernames[user.ID] = user.PreferredUsername
			}
		}
		last := page[len(page)-1]
		if last == nil || (last.PreferredUsername == afterUsername && last.ID == afterID) {
			return nil, errors.New("group membership export pagination did not advance")
		}
		afterUsername, afterID = last.PreferredUsername, last.ID
		if len(page) < groupMembershipCSVExportPageSize {
			return usernames, nil
		}
	}
}

func validateGroupMembershipCSVExportColumns(schema groupdomain.GroupMembershipCSVSchema, columns []string) error {
	if len(columns) == 0 {
		return &idmdomain.CSVError{Row: 1, Code: idmdomain.CSVErrorInvalidHeader}
	}
	seen := make(map[string]struct{}, len(columns))
	for _, key := range columns {
		if !schema.Accepts(key) {
			return &idmdomain.CSVError{Row: 1, Column: key, Code: idmdomain.CSVErrorInvalidHeader}
		}
		if _, duplicate := seen[key]; duplicate {
			return &idmdomain.CSVError{Row: 1, Column: key, Code: idmdomain.CSVErrorInvalidHeader}
		}
		seen[key] = struct{}{}
	}
	return nil
}

// groupMembershipCSVExportValue は 1 セルの値。`membership_state` は常に `present`
// を書く。これが、無編集の export をそのまま適用しても全行 unchanged になる往復
// 不変条件を成り立たせる。
func groupMembershipCSVExportValue(
	group *groupdomain.Group,
	member *groupdomain.GroupMember,
	username, key string,
) string {
	switch key {
	case "group_id":
		return group.ID
	case "group_name":
		return group.Name
	case "user_id":
		return member.UserID
	case "preferred_username":
		return username
	case "membership_state":
		return string(groupdomain.GroupMembershipStatePresent)
	case "source":
		return string(member.Source.Effective())
	case "created_at":
		return member.CreatedAt.UTC().Format(time.RFC3339)
	default:
		return ""
	}
}
