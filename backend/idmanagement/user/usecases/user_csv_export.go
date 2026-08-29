package usecases

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

const userCSVExportPageSize = 500

type UserCSVExportDeps struct {
	UserRepo     userports.UserRepository
	SchemaReader userports.EffectiveUserAttributeSchemaReader
	Artifacts    idmports.CSVArtifactStore
}

type UserCSVExportResult struct {
	Artifact  idmports.CSVArtifact
	Columns   []string
	TotalRows int
}

type UserCSVExporter struct {
	Deps   UserCSVExportDeps
	Policy idmdomain.CSVTransferPolicy
}

func (e UserCSVExporter) ValidateUserCSVColumns(ctx context.Context, columns []string) error {
	if e.Deps.SchemaReader == nil {
		return errors.New("user CSV schema reader is unavailable")
	}
	defs, err := e.Deps.SchemaReader.EffectiveUserAttributeDefs(ctx, tenancy.TenantID(ctx))
	if err != nil {
		return err
	}
	schema, err := userdomain.NewUserCSVSchema(defs)
	if err != nil {
		return err
	}
	return validateUserCSVExportColumns(schema, columns)
}

func (e UserCSVExporter) ExportUserCSV(ctx context.Context, columns []string, status string) (idmports.CSVArtifact, int, error) {
	policy := e.Policy
	if policy == (idmdomain.CSVTransferPolicy{}) {
		policy = idmdomain.DefaultCSVTransferPolicy()
	}
	result, err := ExportUserCSV(ctx, e.Deps, columns, status, policy)
	return result.Artifact, result.TotalRows, err
}

// ExportUserCSV writes repository pages directly into an immutable artifact.
// A returned artifact always satisfies the same policy and schema accepted by
// PlanUserImport.
func ExportUserCSV(ctx context.Context, deps UserCSVExportDeps, columns []string, status string, policy idmdomain.CSVTransferPolicy) (UserCSVExportResult, error) {
	var result UserCSVExportResult
	if deps.UserRepo == nil || deps.SchemaReader == nil || deps.Artifacts == nil {
		return result, errors.New("user CSV export dependencies are incomplete")
	}
	if err := policy.Validate(); err != nil {
		return result, err
	}
	tenantID := tenancy.TenantID(ctx)
	defs, err := deps.SchemaReader.EffectiveUserAttributeDefs(ctx, tenantID)
	if err != nil {
		return result, err
	}
	schema, err := userdomain.NewUserCSVSchema(defs)
	if err != nil {
		return result, err
	}
	if len(columns) == 0 {
		for _, column := range schema.Columns() {
			columns = append(columns, column.Key)
		}
	}
	if err := validateUserCSVExportColumns(schema, columns); err != nil {
		return result, err
	}
	columns = append([]string(nil), columns...)
	status = strings.ToLower(strings.TrimSpace(status))

	rowCount := 0
	artifact, err := deps.Artifacts.PutCSVArtifact(ctx, tenantID, func(output io.Writer) error {
		writer, err := idmdomain.NewCSVWriter(output, columns, policy)
		if err != nil {
			return err
		}
		afterUsername, afterID := "", ""
		for {
			page, err := deps.UserRepo.ListPage(ctx, tenantID, afterUsername, afterID, userCSVExportPageSize)
			if err != nil {
				return err
			}
			for _, user := range page {
				if status != "" && string(user.Lifecycle.EffectiveStatus()) != status {
					continue
				}
				if err := userdomain.ValidateAttributes(user.Attributes, defs); err != nil {
					return err
				}
				row, err := userCSVExportRow(user, columns, schema)
				if err != nil {
					return err
				}
				if err := writer.WriteRow(row); err != nil {
					return err
				}
				rowCount++
			}
			if len(page) < userCSVExportPageSize {
				break
			}
			last := page[len(page)-1]
			afterUsername, afterID = last.PreferredUsername, last.ID
		}
		return writer.Close()
	})
	if err != nil {
		return result, err
	}
	return UserCSVExportResult{Artifact: artifact, Columns: columns, TotalRows: rowCount}, nil
}

func validateUserCSVExportColumns(schema userdomain.UserCSVSchema, columns []string) error {
	seen := make(map[string]struct{}, len(columns))
	for _, key := range columns {
		if _, ok := schema.Column(key); !ok {
			return &idmdomain.CSVError{Row: 1, Column: key, Code: idmdomain.CSVErrorInvalidHeader}
		}
		if _, duplicate := seen[key]; duplicate {
			return &idmdomain.CSVError{Row: 1, Column: key, Code: idmdomain.CSVErrorInvalidHeader}
		}
		seen[key] = struct{}{}
	}
	if len(columns) == 0 {
		return &idmdomain.CSVError{Row: 1, Code: idmdomain.CSVErrorInvalidHeader}
	}
	return nil
}

func userCSVExportRow(user *userdomain.User, columns []string, schema userdomain.UserCSVSchema) ([]string, error) {
	row := make([]string, len(columns))
	for i, key := range columns {
		column, _ := schema.Column(key)
		if column.Attribute != nil {
			value, ok := user.Attributes[column.Attribute.Key]
			if !ok {
				row[i] = ""
				continue
			}
			formatted, err := userdomain.FormatUserCSVAttributeCell(value, *column.Attribute)
			if err != nil {
				return nil, err
			}
			row[i] = formatted
			continue
		}
		row[i] = builtinUserCSVExportValue(user, key)
	}
	return row, nil
}

func builtinUserCSVExportValue(user *userdomain.User, key string) string {
	switch key {
	case "id":
		return user.ID
	case "preferred_username":
		return user.PreferredUsername
	case "name":
		return optionalUserCSVString(user.Name)
	case "given_name":
		return optionalUserCSVString(user.GivenName)
	case "family_name":
		return optionalUserCSVString(user.FamilyName)
	case "email":
		return optionalUserCSVString(user.Email)
	case "email_verified":
		return strconv.FormatBool(user.EmailVerified)
	case "roles":
		return strings.Join(user.Roles, "|")
	case "required_actions":
		actions := make([]string, len(user.Lifecycle.RequiredActions))
		for i, action := range user.Lifecycle.RequiredActions {
			actions[i] = string(action)
		}
		return strings.Join(actions, "|")
	case "mfa_enrolled":
		return strconv.FormatBool(user.MfaEnrolled)
	case "status":
		return string(user.Lifecycle.EffectiveStatus())
	case "created_at":
		return user.CreatedAt.UTC().Format(time.RFC3339)
	case "updated_at":
		return user.UpdatedAt.UTC().Format(time.RFC3339)
	default:
		return ""
	}
}

func optionalUserCSVString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
