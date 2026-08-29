package usecases

// Group CSV の export。リポジトリのページをそのまま不変な成果物へ書き出し、
// 全体を文字列や base64 として実体化しない。返る成果物は必ず、PlanGroupImport が
// 受理するのと同じポリシーと語彙を満たす。

import (
	"context"
	"errors"
	"io"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

const groupCSVExportPageSize = 500

type GroupCSVExportDeps struct {
	GroupRepo groupports.GroupRepository
	Artifacts idmports.CSVArtifactStore
}

type GroupCSVExportResult struct {
	Artifact  idmports.CSVArtifact
	Columns   []string
	TotalRows int
}

// GroupCSVExporter は data export のジョブ境界へ渡す実装。
type GroupCSVExporter struct {
	Deps   GroupCSVExportDeps
	Policy idmdomain.CSVTransferPolicy
}

func (e GroupCSVExporter) policy() idmdomain.CSVTransferPolicy {
	if e.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return e.Policy
}

func (e GroupCSVExporter) ValidateGroupCSVColumns(_ context.Context, columns []string) error {
	return validateGroupCSVExportColumns(groupdomain.NewGroupCSVSchema(), columns)
}

func (e GroupCSVExporter) ExportGroupCSV(ctx context.Context, columns []string) (idmports.CSVArtifact, int, error) {
	result, err := ExportGroupCSV(ctx, e.Deps, columns, e.policy())
	return result.Artifact, result.TotalRows, err
}

func ExportGroupCSV(ctx context.Context, deps GroupCSVExportDeps, columns []string, policy idmdomain.CSVTransferPolicy) (GroupCSVExportResult, error) {
	var result GroupCSVExportResult
	if deps.GroupRepo == nil || deps.Artifacts == nil {
		return result, errors.New("group CSV export dependencies are incomplete")
	}
	if err := policy.Validate(); err != nil {
		return result, err
	}
	schema := groupdomain.NewGroupCSVSchema()
	if err := validateGroupCSVExportColumns(schema, columns); err != nil {
		return result, err
	}
	columns = append([]string(nil), columns...)

	tenantID := tenancy.TenantID(ctx)
	rules, err := deps.GroupRepo.ListDynamicRules(ctx, tenantID)
	if err != nil {
		return result, err
	}
	byGroup := make(map[string]*groupdomain.DynamicGroupRule, len(rules))
	for _, rule := range rules {
		if rule != nil {
			byGroup[rule.GroupID] = rule
		}
	}

	rowCount := 0
	artifact, err := deps.Artifacts.PutCSVArtifact(ctx, tenantID, func(output io.Writer) error {
		writer, err := idmdomain.NewCSVWriter(output, columns, policy)
		if err != nil {
			return err
		}
		afterName, afterID := "", ""
		for {
			page, err := deps.GroupRepo.ListPage(ctx, tenantID, afterName, afterID, groupCSVExportPageSize)
			if err != nil {
				return err
			}
			if len(page) == 0 {
				break
			}
			for _, group := range page {
				record := make([]string, len(columns))
				for i, key := range columns {
					record[i] = groupCSVExportValue(group, byGroup[group.ID], key)
				}
				if err := writer.WriteRow(record); err != nil {
					return err
				}
				rowCount++
			}
			last := page[len(page)-1]
			afterName, afterID = last.Name, last.ID
			if len(page) < groupCSVExportPageSize {
				break
			}
		}
		return writer.Close()
	})
	if err != nil {
		return result, err
	}
	return GroupCSVExportResult{Artifact: artifact, Columns: columns, TotalRows: rowCount}, nil
}

func validateGroupCSVExportColumns(schema groupdomain.GroupCSVSchema, columns []string) error {
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

// groupCSVExportValue は 1 セルの値。`lifecycle_action` は常に空を書く。これが、
// 無編集の export をそのまま適用しても全行 unchanged になる往復不変条件を成り立たせる。
func groupCSVExportValue(group *groupdomain.Group, rule *groupdomain.DynamicGroupRule, key string) string {
	switch key {
	case "id":
		return group.ID
	case "name":
		return group.Name
	case "description":
		if group.Description == nil {
			return ""
		}
		return *group.Description
	case "membership_type":
		return string(group.MembershipType.Effective())
	case "roles":
		return groupdomain.FormatGroupCSVRoles(group.Roles)
	case "dynamic_rule_expression":
		if rule == nil {
			return ""
		}
		return rule.Expression
	case "dynamic_rule_enabled":
		if rule == nil {
			return ""
		}
		return formatGroupCSVBool(rule.Enabled)
	case "lifecycle_action":
		return ""
	case "created_at":
		return group.CreatedAt.UTC().Format(time.RFC3339)
	case "updated_at":
		return group.UpdatedAt.UTC().Format(time.RFC3339)
	default:
		return ""
	}
}

func formatGroupCSVBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
