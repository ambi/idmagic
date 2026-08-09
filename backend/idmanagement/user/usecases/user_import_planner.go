package usecases

import (
	"context"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/tenancy"
)

type UserImportPlanDeps struct {
	UserRepo       userports.UserRepository
	SchemaReader   userports.EffectiveUserAttributeSchemaReader
	OwnershipGuard userports.UserSourceOwnershipGuard
	PageSize       int
}

type userImportIndex struct {
	byID                 map[string]*userdomain.User
	byUsername           map[string]*userdomain.User
	sourceManaged        map[string]bool
	ownershipUnavailable bool
}

func (d UserImportPlanDeps) pageSize() int {
	if d.PageSize > 0 {
		return d.PageSize
	}
	return 1000
}

func loadUserImportIndex(ctx context.Context, deps UserImportPlanDeps, tenantID string) (userImportIndex, error) {
	index := userImportIndex{
		byID:          map[string]*userdomain.User{},
		byUsername:    map[string]*userdomain.User{},
		sourceManaged: map[string]bool{},
	}
	afterUsername, afterID := "", ""
	for {
		page, err := deps.UserRepo.ListPage(ctx, tenantID, afterUsername, afterID, deps.pageSize())
		if err != nil {
			return userImportIndex{}, err
		}
		if len(page) == 0 {
			break
		}
		ids := make([]string, 0, len(page))
		for _, user := range page {
			if user == nil || user.TenantID != tenantID {
				continue
			}
			index.byID[user.ID] = user
			index.byUsername[user.PreferredUsername] = user
			ids = append(ids, user.ID)
		}
		if deps.OwnershipGuard == nil {
			index.ownershipUnavailable = true
		} else if managed, err := deps.OwnershipGuard.SourceManagedUserIDs(ctx, tenantID, ids); err != nil {
			index.ownershipUnavailable = true
		} else {
			for id, value := range managed {
				if value {
					index.sourceManaged[id] = true
				}
			}
		}
		last := page[len(page)-1]
		if last == nil || (last.PreferredUsername == afterUsername && last.ID == afterID) {
			return userImportIndex{}, errors.New("user import pagination did not advance")
		}
		afterUsername, afterID = last.PreferredUsername, last.ID
		if len(page) < deps.pageSize() {
			break
		}
	}
	return index, nil
}

// PlanUserImport streams rows through one deterministic planner. emit can
// persist row plans/errors incrementally; the returned summary is bounded.
func PlanUserImport(
	ctx context.Context,
	deps UserImportPlanDeps,
	input io.Reader,
	policy userdomain.UserCSVTransferPolicy,
	emit func(userdomain.UserImportRowPlan) error,
) (userdomain.UserImportPlanSummary, error) {
	var summary userdomain.UserImportPlanSummary
	if deps.UserRepo == nil || deps.SchemaReader == nil {
		return summary, errors.New("user import planner dependencies are incomplete")
	}
	tenantID := tenancy.TenantID(ctx)
	defs, err := deps.SchemaReader.EffectiveUserAttributeDefs(ctx, tenantID)
	if err != nil {
		return summary, err
	}
	schema, err := userdomain.NewUserCSVSchema(defs)
	if err != nil {
		return summary, err
	}
	reader, err := userdomain.NewUserCSVReader(input, schema, policy)
	if err != nil {
		return summary, err
	}
	index, err := loadUserImportIndex(ctx, deps, tenantID)
	if err != nil {
		return summary, err
	}
	seenIDs := map[string]struct{}{}
	seenUsernames := map[string]struct{}{}
	for {
		record, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return summary, nil
		}
		if err != nil {
			return summary, err
		}
		var planned userdomain.UserImportRowPlan
		if record.Error != nil {
			planned = rejectedUserImportRow(record.Error.Row, record.Error.Column, record.Error.Code)
		} else {
			planned = planUserImportRow(*record.Row, schema, defs, index, tenantID, seenIDs, seenUsernames)
		}
		summary.Observe(planned)
		if emit != nil {
			if err := emit(planned); err != nil {
				return summary, err
			}
		}
	}
}

func rejectedUserImportRow(row int, column string, code userdomain.UserCSVErrorCode) userdomain.UserImportRowPlan {
	return userdomain.UserImportRowPlan{
		Row: row, Action: userdomain.UserImportRejected,
		Error: &userdomain.UserCSVError{Row: row, Column: column, Code: code},
	}
}

func planUserImportRow(
	row userdomain.UserCSVRow,
	schema userdomain.UserCSVSchema,
	defs []userdomain.UserAttributeDef,
	index userImportIndex,
	tenantID string,
	seenIDs, seenUsernames map[string]struct{},
) userdomain.UserImportRowPlan {
	identifier, code := row.Identifier()
	if code != "" {
		return rejectedUserImportRow(row.Number, "", code)
	}
	if identifier.ID != "" {
		if _, duplicate := seenIDs[identifier.ID]; duplicate {
			return rejectedUserImportRow(row.Number, "id", "duplicate_target")
		}
		seenIDs[identifier.ID] = struct{}{}
	}
	if identifier.PreferredUsername != "" {
		if _, duplicate := seenUsernames[identifier.PreferredUsername]; duplicate {
			return rejectedUserImportRow(row.Number, "preferred_username", "duplicate_username")
		}
		seenUsernames[identifier.PreferredUsername] = struct{}{}
	}

	existing, resolveCode := resolveUserImportTarget(identifier, index)
	if resolveCode != "" {
		return rejectedUserImportRow(row.Number, "", resolveCode)
	}
	if existing != nil && (index.ownershipUnavailable || index.sourceManaged[existing.ID]) {
		return rejectedUserImportRow(row.Number, "", "source_managed")
	}
	candidate := newUserImportCandidate(existing, tenantID, identifier.PreferredUsername)
	column, applyCode := applyUserImportCells(&candidate, row, schema, defs)
	if applyCode != "" {
		return rejectedUserImportRow(row.Number, column, applyCode)
	}
	if existing == nil {
		candidate.ID = ""
		candidate.PasswordHash = ""
		candidate.CreatedAt = time.Time{}
		candidate.UpdatedAt = time.Time{}
		return userdomain.UserImportRowPlan{Row: row.Number, Action: userdomain.UserImportCreate, Identifier: identifier, User: &candidate}
	}
	action := userdomain.UserImportUpdate
	if userImportMutationEqual(*existing, candidate) {
		action = userdomain.UserImportUnchanged
	}
	return userdomain.UserImportRowPlan{Row: row.Number, Action: action, Identifier: identifier, Before: existing, User: &candidate}
}

func resolveUserImportTarget(identifier userdomain.UserCSVIdentifier, index userImportIndex) (*userdomain.User, userdomain.UserCSVErrorCode) {
	if identifier.ID != "" {
		user := index.byID[identifier.ID]
		if user == nil {
			return nil, "target_not_found"
		}
		if identifier.PreferredUsername != "" {
			if named := index.byUsername[identifier.PreferredUsername]; named != nil && named.ID != user.ID {
				return nil, "identifier_mismatch"
			}
		}
		return user, ""
	}
	return index.byUsername[identifier.PreferredUsername], ""
}

func newUserImportCandidate(existing *userdomain.User, tenantID, username string) userdomain.User {
	if existing != nil {
		return cloneImportUser(*existing)
	}
	stamp := time.Unix(1, 0).UTC()
	return userdomain.User{
		ID: "preview", TenantID: tenantID, PreferredUsername: username, PasswordHash: "preview",
		Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]userdomain.AttributeValue{}, CreatedAt: stamp, UpdatedAt: stamp,
	}
}

func cloneImportUser(user userdomain.User) userdomain.User {
	user.Roles = append([]string(nil), user.Roles...)
	user.Lifecycle.RequiredActions = append([]idmdomain.RequiredAction(nil), user.Lifecycle.RequiredActions...)
	user.Attributes = cloneImportAttributes(user.Attributes)
	return user
}

func cloneImportAttributes(values map[string]userdomain.AttributeValue) map[string]userdomain.AttributeValue {
	out := make(map[string]userdomain.AttributeValue, len(values))
	for key, value := range values {
		value.StringArray = append([]string(nil), value.StringArray...)
		out[key] = value
	}
	return out
}

func applyUserImportCells(candidate *userdomain.User, row userdomain.UserCSVRow, schema userdomain.UserCSVSchema, defs []userdomain.UserAttributeDef) (string, userdomain.UserCSVErrorCode) {
	if cell, ok := row.Cell("preferred_username"); ok {
		candidate.PreferredUsername = strings.TrimSpace(cell.Raw)
		if candidate.PreferredUsername == "" {
			return "preferred_username", "required"
		}
	}
	setOptionalImportString(row, "name", &candidate.Name)
	setOptionalImportString(row, "given_name", &candidate.GivenName)
	setOptionalImportString(row, "family_name", &candidate.FamilyName)
	setOptionalImportString(row, "email", &candidate.Email)
	if cell, ok := row.Cell("email_verified"); ok {
		switch cell.Raw {
		case "true":
			candidate.EmailVerified = true
		case "false":
			candidate.EmailVerified = false
		default:
			return "email_verified", "invalid_boolean"
		}
	}
	if cell, ok := row.Cell("roles"); ok {
		roles, err := parseImportList(cell.Raw)
		if err != nil {
			return "roles", "invalid_roles"
		}
		roles, err = idmusecases.NormalizeRoles(roles)
		if err != nil {
			return "roles", "invalid_roles"
		}
		candidate.Roles = roles
	}
	if cell, ok := row.Cell("required_actions"); ok {
		actions, err := parseImportRequiredActions(cell.Raw)
		if err != nil {
			return "required_actions", "invalid_required_actions"
		}
		candidate.Lifecycle.RequiredActions = actions
	}
	for _, key := range schema.Columns() {
		if key.Attribute == nil {
			continue
		}
		cell, present := row.Cell(key.Key)
		if !present {
			continue
		}
		value, shouldClear, err := userdomain.ParseUserCSVAttributeCell(cell.Raw, *key.Attribute)
		if err != nil {
			return key.Key, "invalid_custom_attribute"
		}
		if shouldClear {
			delete(candidate.Attributes, key.Attribute.Key)
		} else {
			candidate.Attributes[key.Attribute.Key] = value
		}
	}
	if err := userdomain.ValidateAttributes(candidate.Attributes, defs); err != nil {
		return "", "invalid_custom_attribute"
	}
	if err := candidate.Validate(); err != nil {
		if _, ok := row.Cell("email"); ok {
			return "email", "invalid_email"
		}
		return "", "invalid_user"
	}
	return "", ""
}

func setOptionalImportString(row userdomain.UserCSVRow, key string, target **string) {
	cell, ok := row.Cell(key)
	if !ok {
		return
	}
	if cell.Raw == "" {
		*target = nil
		return
	}
	value := cell.Raw
	*target = &value
}

func parseImportList(raw string) ([]string, error) {
	if raw == "" {
		return []string{}, nil
	}
	parts := strings.Split(raw, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, errors.New("empty list value")
		}
	}
	return parts, nil
}

func parseImportRequiredActions(raw string) ([]idmdomain.RequiredAction, error) {
	values, err := parseImportList(raw)
	if err != nil {
		return nil, err
	}
	actions := make([]idmdomain.RequiredAction, 0, len(values))
	for _, value := range values {
		action := idmdomain.RequiredAction(value)
		if !action.Valid() {
			return nil, errors.New("invalid required action")
		}
		if !slices.Contains(actions, action) {
			actions = append(actions, action)
		}
	}
	slices.Sort(actions)
	return actions, nil
}

func userImportMutationEqual(left, right userdomain.User) bool {
	return left.PreferredUsername == right.PreferredUsername &&
		reflect.DeepEqual(left.Name, right.Name) &&
		reflect.DeepEqual(left.GivenName, right.GivenName) &&
		reflect.DeepEqual(left.FamilyName, right.FamilyName) &&
		reflect.DeepEqual(left.Email, right.Email) &&
		left.EmailVerified == right.EmailVerified &&
		slices.Equal(left.Roles, right.Roles) &&
		slices.Equal(left.Lifecycle.RequiredActions, right.Lifecycle.RequiredActions) &&
		reflect.DeepEqual(left.Attributes, right.Attributes)
}
