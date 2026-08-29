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
	GroupRepo    groupports.GroupRepository
	SchemaReader groupports.EffectiveGroupAttributeSchemaReader
	Artifacts    idmports.CSVArtifactStore
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

func (e GroupCSVExporter) ValidateGroupCSVColumns(ctx context.Context, columns []string) error {
	schema, err := groupCSVSchemaFor(ctx, e.Deps.SchemaReader, tenancy.TenantID(ctx))
	if err != nil {
		return err
	}
	return validateGroupCSVExportColumns(schema, columns)
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
	tenantID := tenancy.TenantID(ctx)
	schema, err := groupCSVSchemaFor(ctx, deps.SchemaReader, tenantID)
	if err != nil {
		return result, err
	}
	if err := validateGroupCSVExportColumns(schema, columns); err != nil {
		return result, err
	}
	columns = append([]string(nil), columns...)
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
					cell, err := groupCSVExportValue(group, byGroup[group.ID], schema.Column(key))
					if err != nil {
						return err
					}
					record[i] = cell
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

// groupCSVSchemaFor はテナントの実効属性定義から列の語彙を組み立てる。リーダーが
// 無いテナントは custom 列を持たない。
func groupCSVSchemaFor(ctx context.Context, reader groupports.EffectiveGroupAttributeSchemaReader, tenantID string) (groupdomain.GroupCSVSchema, error) {
	if reader == nil {
		return groupdomain.NewGroupCSVSchema(nil)
	}
	defs, err := reader.EffectiveGroupAttributeDefs(ctx, tenantID)
	if err != nil {
		return groupdomain.GroupCSVSchema{}, err
	}
	return groupdomain.NewGroupCSVSchema(defs)
}

// groupCSVExportValue は 1 セルの値。`lifecycle_action` は常に空を書く。これが、
// 無編集の export をそのまま適用しても全行 unchanged になる往復不変条件を成り立たせる。
func groupCSVExportValue(group *groupdomain.Group, rule *groupdomain.DynamicGroupRule, column groupdomain.GroupCSVColumn) (string, error) {
	if column.Attribute != nil {
		value, ok := group.Attributes[column.Attribute.Key]
		if !ok {
			return "", nil
		}
		return groupdomain.FormatGroupCSVAttributeCell(value, *column.Attribute)
	}
	switch column.Key {
	case "id":
		return group.ID, nil
	case "name":
		return group.Name, nil
	case "description":
		return optionalGroupCSVString(group.Description), nil
	case "email":
		return optionalGroupCSVString(group.Email), nil
	case "membership_type":
		return string(group.MembershipType.Effective()), nil
	case "roles":
		return groupdomain.FormatGroupCSVRoles(group.Roles), nil
	case "dynamic_rule_expression":
		if rule == nil {
			return "", nil
		}
		return rule.Expression, nil
	case "dynamic_rule_enabled":
		if rule == nil {
			return "", nil
		}
		return formatGroupCSVBool(rule.Enabled), nil
	case "lifecycle_action":
		return "", nil
	case "created_at":
		return group.CreatedAt.UTC().Format(time.RFC3339), nil
	case "updated_at":
		return group.UpdatedAt.UTC().Format(time.RFC3339), nil
	default:
		return "", nil
	}
}

func optionalGroupCSVString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func formatGroupCSVBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
