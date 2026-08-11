package usecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/sourcing/scim/domain"
	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

func (u *Usecases) CreateUser(ctx context.Context, tenantID string, body map[string]any) (map[string]any, error) {
	w, err := domain.ParseUserWrite(body)
	if err != nil {
		return nil, err
	}

	existing, err := u.UserRepo.FindByUsername(ctx, tenantID, w.UserName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: userName %q already exists", ErrDuplicate, w.UserName)
	}

	// server-assigned identifiers (RFC7643-CORE-RESOURCES: id is readOnly,
	// client-supplied values are ignored)
	subBytes := make([]byte, 16)
	if _, err := rand.Read(subBytes); err != nil {
		return nil, err
	}
	sub := fmt.Sprintf("user_%s", hex.EncodeToString(subBytes))

	scimIDBytes := make([]byte, 16)
	if _, err := rand.Read(scimIDBytes); err != nil {
		return nil, err
	}
	scimID := hex.EncodeToString(scimIDBytes)

	now := time.Now()
	user := &userdomain.User{
		ID:                sub,
		TenantID:          tenantID,
		PreferredUsername: w.UserName,
		PasswordHash:      "", // SCIM users usually don't have local password initially
		Name:              &w.Formatted,
		GivenName:         &w.GivenName,
		FamilyName:        &w.FamilyName,
		Email:             &w.Email,
		EmailVerified:     true,
		Roles:             []string{},
		Lifecycle: userdomain.UserLifecycle{
			Status:          userStatusFromActive(w.Active),
			StatusChangedAt: &now,
		},
		Attributes: make(map[string]userdomain.AttributeValue),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.applyEnterpriseExtension(ctx, tenantID, user, w); err != nil {
		return nil, err
	}

	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	ref := &ports.ScimUserRef{TenantID: tenantID, ScimID: scimID, UserID: sub}
	if err := u.ScimRepo.SaveUserRef(ctx, ref); err != nil {
		return nil, err
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

func userStatusFromActive(active bool) idmdomain.UserStatus {
	if active {
		return idmdomain.UserStatusActive
	}
	return idmdomain.UserStatusDisabled
}

func (u *Usecases) GetUser(ctx context.Context, tenantID, scimID string) (map[string]any, error) {
	ref, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNotFound
	}

	user, err := u.UserRepo.FindBySub(ctx, ref.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

// UpdateUser implements PUT full-replace semantics: every
// RFC7643-CORE-RESOURCES mutable attribute is set from body, with omitted
// attributes reset to their default via domain.ParseUserWrite. The User
// aggregate is validated (userName required) before the single Save call,
// so a validation failure never leaves a partial write.
func (u *Usecases) UpdateUser(ctx context.Context, tenantID, scimID string, body map[string]any) (map[string]any, error) {
	ref, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNotFound
	}

	user, err := u.UserRepo.FindBySub(ctx, ref.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	w, err := domain.ParseUserWrite(body)
	if err != nil {
		return nil, err
	}
	if w.UserName != user.PreferredUsername {
		if existing, err := u.UserRepo.FindByUsername(ctx, tenantID, w.UserName); err != nil {
			return nil, err
		} else if existing != nil && existing.ID != user.ID {
			return nil, fmt.Errorf("%w: userName %q already exists", ErrDuplicate, w.UserName)
		}
	}

	user.PreferredUsername = w.UserName
	user.GivenName = &w.GivenName
	user.FamilyName = &w.FamilyName
	user.Name = &w.Formatted
	user.Email = &w.Email
	u.setUserActive(user, w.Active)
	if err := u.applyEnterpriseExtension(ctx, tenantID, user, w); err != nil {
		return nil, err
	}
	user.UpdatedAt = time.Now()

	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

// PatchUser applies RFC 7644 §3.5.2 operations validated by
// domain.ParseUserPatchOps against the User attribute allowlist. All
// operations are validated up front (validate-first) before any
// field is mutated; the aggregate is persisted with a single Save call.
func (u *Usecases) PatchUser(ctx context.Context, tenantID, scimID string, body map[string]any) (map[string]any, error) {
	ref, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNotFound
	}

	user, err := u.UserRepo.FindBySub(ctx, ref.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	ops, err := domain.ParseUserPatchOps(body)
	if err != nil {
		return nil, err
	}

	for _, op := range ops {
		if err := u.applyUserPatchOp(ctx, tenantID, user, op); err != nil {
			return nil, err
		}
	}

	user.UpdatedAt = time.Now()
	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return u.toScimUser(ctx, tenantID, user, scimID)
}

func (u *Usecases) applyUserPatchOp(ctx context.Context, tenantID string, user *userdomain.User, op domain.UserPatchOp) error {
	isRemoveOp := op.Op == "remove"

	switch op.Attr {
	case domain.UserAttrUserName:
		if isRemoveOp {
			return domain.NewMutationError("invalidValue", "userName cannot be removed")
		}
		userName, _ := op.Value.(string)
		if userName == "" {
			return domain.NewMutationError("invalidValue", "userName value must be a non-empty string")
		}
		if userName != user.PreferredUsername {
			existing, err := u.UserRepo.FindByUsername(ctx, tenantID, userName)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != user.ID {
				return fmt.Errorf("%w: userName %q already exists", ErrDuplicate, userName)
			}
		}
		user.PreferredUsername = userName
	case domain.UserAttrName:
		if isRemoveOp {
			empty := ""
			user.GivenName, user.FamilyName, user.Name = &empty, &empty, &empty
			return nil
		}
		nameMap, ok := op.Value.(map[string]any)
		if !ok {
			return domain.NewMutationError("invalidValue", "name value must be an object")
		}
		givenName, _ := nameMap["givenName"].(string)
		familyName, _ := nameMap["familyName"].(string)
		formatted, _ := nameMap["formatted"].(string)
		user.GivenName, user.FamilyName, user.Name = &givenName, &familyName, &formatted
	case domain.UserAttrGivenName:
		user.GivenName = patchStringField(op)
	case domain.UserAttrFamilyName:
		user.FamilyName = patchStringField(op)
	case domain.UserAttrFormatted:
		user.Name = patchStringField(op)
	case domain.UserAttrEmails:
		if isRemoveOp {
			empty := ""
			user.Email = &empty
			return nil
		}
		email, _ := op.Value.(string)
		user.Email = &email
	case domain.UserAttrActive:
		if isRemoveOp {
			u.setUserActive(user, true)
			return nil
		}
		active, _ := op.Value.(bool)
		u.setUserActive(user, active)
	case domain.UserAttrEmployeeNumber:
		ensureAttributes(user)
		if isRemoveOp {
			delete(user.Attributes, "employee_number")
			return nil
		}
		v, _ := op.Value.(string)
		setOrDeleteStringAttr(user.Attributes, "employee_number", v)
	case domain.UserAttrDepartment:
		ensureAttributes(user)
		if isRemoveOp {
			delete(user.Attributes, "department")
			return nil
		}
		v, _ := op.Value.(string)
		setOrDeleteStringAttr(user.Attributes, "department", v)
	case domain.UserAttrManager:
		ensureAttributes(user)
		if isRemoveOp {
			delete(user.Attributes, "manager_sub")
			return nil
		}
		managerScimID, _ := op.Value.(string)
		managerSub, err := u.resolveManagerSub(ctx, tenantID, managerScimID)
		if err != nil {
			return err
		}
		setOrDeleteStringAttr(user.Attributes, "manager_sub", managerSub)
	}
	return nil
}

// ensureAttributes guarantees user.Attributes is non-nil before a sparse
// key is written; User.Attributes is a `map[string]AttributeValue,omitempty`
// so a User loaded without any set attribute has a nil map.
func ensureAttributes(user *userdomain.User) {
	if user.Attributes == nil {
		user.Attributes = make(map[string]userdomain.AttributeValue)
	}
}

// setOrDeleteStringAttr sets attrs[key] to value, or deletes the key when
// value is blank so an attribute reset by PUT/PATCH doesn't leave a stale
// value behind.
func setOrDeleteStringAttr(attrs map[string]userdomain.AttributeValue, key, value string) {
	if strings.TrimSpace(value) == "" {
		delete(attrs, key)
		return
	}
	v := value
	attrs[key] = userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: &v}
}

// stringAttrValue reads a string-typed User.Attributes entry.
func stringAttrValue(attrs map[string]userdomain.AttributeValue, key string) (string, bool) {
	v, ok := attrs[key]
	if !ok || v.Type != idmdomain.AttributeTypeString || v.String == nil {
		return "", false
	}
	return *v.String, true
}

// resolveManagerSub resolves an enterprise extension "manager" SCIM id to
// the manager's internal User.sub, scoped to tenantID so a reference can
// never cross a tenant boundary (wi-247 Risk Notes). An empty
// managerScimID (manager omitted/cleared) resolves to "". An unresolvable
// id is a *domain.MutationError (invalidValue), returned before any
// persistence write (validate-first).
func (u *Usecases) resolveManagerSub(ctx context.Context, tenantID, managerScimID string) (string, error) {
	if strings.TrimSpace(managerScimID) == "" {
		return "", nil
	}
	ref, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, managerScimID)
	if err != nil {
		return "", err
	}
	if ref == nil {
		return "", domain.NewMutationError("invalidValue", "manager %q does not resolve to a User in this tenant", managerScimID)
	}
	return ref.UserID, nil
}

// applyEnterpriseExtension applies the RFC7643-ENTERPRISE-EXTENSION
// adoption:partial subset (employee_number/department/manager_sub) onto
// User.Attributes. w's fields already reflect PUT full-replace/PATCH
// semantics (empty means "clear"), so this only needs to resolve manager
// and write or delete each sparse key.
func (u *Usecases) applyEnterpriseExtension(ctx context.Context, tenantID string, user *userdomain.User, w domain.UserWrite) error {
	managerSub, err := u.resolveManagerSub(ctx, tenantID, w.ManagerValue)
	if err != nil {
		return err
	}
	ensureAttributes(user)
	setOrDeleteStringAttr(user.Attributes, "employee_number", w.EmployeeNumber)
	setOrDeleteStringAttr(user.Attributes, "department", w.Department)
	setOrDeleteStringAttr(user.Attributes, "manager_sub", managerSub)
	return nil
}

func patchStringField(op domain.UserPatchOp) *string {
	if op.Op == "remove" {
		empty := ""
		return &empty
	}
	s, _ := op.Value.(string)
	return &s
}

func (u *Usecases) setUserActive(user *userdomain.User, active bool) {
	now := time.Now()
	if active && user.Lifecycle.Status != idmdomain.UserStatusActive {
		user.Lifecycle.Status = idmdomain.UserStatusActive
		user.Lifecycle.StatusChangedAt = &now
	} else if !active && user.Lifecycle.Status == idmdomain.UserStatusActive {
		user.Lifecycle.Status = idmdomain.UserStatusDisabled
		user.Lifecycle.StatusChangedAt = &now
	}
}

func (u *Usecases) DeleteUser(ctx context.Context, tenantID, scimID string) error {
	ref, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return err
	}
	if ref == nil {
		return errors.New("user not found")
	}

	user, err := u.UserRepo.FindBySub(ctx, ref.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Soft Delete: status = PendingDeletion
	now := time.Now()
	user.Lifecycle.Status = idmdomain.UserStatusPendingDeletion
	user.Lifecycle.StatusChangedAt = &now
	user.UpdatedAt = now

	return u.UserRepo.Save(ctx, user)
}

func (u *Usecases) ListUsers(ctx context.Context, tenantID string, query ListQuery) (ListResult, error) {
	users, err := u.UserRepo.FindAll(ctx, tenantID)
	if err != nil {
		return ListResult{}, err
	}

	var expr domain.FilterExpr
	if query.Filter != "" {
		expr, err = domain.ParseFilter(query.Filter, domain.UserFilterAttributes)
		if err != nil {
			return ListResult{}, err
		}
	}

	var matched []map[string]any
	for _, user := range users {
		ref, err := u.ScimRepo.FindUserRefByUserID(ctx, tenantID, user.ID)
		if err != nil {
			return ListResult{}, err
		}
		scimID := user.ID
		if ref != nil {
			scimID = ref.ScimID
		}

		if expr != nil && !expr.Matches(userFilterAttrs(user, scimID)) {
			continue
		}

		scimUser, err := u.toScimUser(ctx, tenantID, user, scimID)
		if err != nil {
			return ListResult{}, err
		}
		matched = append(matched, scimUser)
	}

	return paginate(matched, query)
}

// userFilterAttrs flattens a User into the lower-cased attribute map
// domain.UserFilterAttributes expects.
func userFilterAttrs(user *userdomain.User, scimID string) map[string]any {
	attrs := map[string]any{
		"username":          user.PreferredUsername,
		"active":            user.Lifecycle.Status == idmdomain.UserStatusActive,
		"id":                scimID,
		"meta.created":      user.CreatedAt.Format(time.RFC3339),
		"meta.lastmodified": user.UpdatedAt.Format(time.RFC3339),
	}
	if user.Name != nil {
		attrs["name.formatted"] = *user.Name
	}
	if user.GivenName != nil {
		attrs["name.givenname"] = *user.GivenName
	}
	if user.FamilyName != nil {
		attrs["name.familyname"] = *user.FamilyName
	}
	if user.Email != nil {
		attrs["emails.value"] = *user.Email
	}
	return attrs
}

// toScimUser projects a User onto its SCIM representation. The enterprise
// extension object (and its URN in schemas) is included only when the User
// carries at least one employee_number/department/manager_sub Attributes
// entry (RFC7643-ENTERPRISE-EXTENSION adoption:partial), matching how emails
// is already omitted when there is no canonical email.
func (u *Usecases) toScimUser(ctx context.Context, tenantID string, user *userdomain.User, scimID string) (map[string]any, error) {
	var givenName, familyName, formattedName string
	if user.GivenName != nil {
		givenName = *user.GivenName
	}
	if user.FamilyName != nil {
		familyName = *user.FamilyName
	}
	if user.Name != nil {
		formattedName = *user.Name
	}

	active := user.Lifecycle.Status == idmdomain.UserStatusActive

	schemas := []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	ext := map[string]any{}
	if v, ok := stringAttrValue(user.Attributes, "employee_number"); ok {
		ext["employeeNumber"] = v
	}
	if v, ok := stringAttrValue(user.Attributes, "department"); ok {
		ext["department"] = v
	}
	if managerSub, ok := stringAttrValue(user.Attributes, "manager_sub"); ok {
		managerRef, err := u.ScimRepo.FindUserRefByUserID(ctx, tenantID, managerSub)
		if err != nil {
			return nil, err
		}
		if managerRef != nil {
			ext["manager"] = map[string]any{"value": managerRef.ScimID}
		}
	}
	if len(ext) > 0 {
		schemas = append(schemas, domain.EnterpriseUserSchemaURN)
	}

	resource := map[string]any{
		"schemas":  schemas,
		"id":       scimID,
		"userName": user.PreferredUsername,
		"name": map[string]any{
			"familyName": familyName,
			"givenName":  givenName,
			"formatted":  formattedName,
		},
		"active": active,
		"meta": map[string]any{
			"resourceType": "User",
			"created":      user.CreatedAt.Format(time.RFC3339),
			"lastModified": user.UpdatedAt.Format(time.RFC3339),
			"location":     "/scim/v2/Users/" + scimID,
		},
	}
	if user.Email != nil && strings.TrimSpace(*user.Email) != "" {
		resource["emails"] = []map[string]any{{
			"value":   *user.Email,
			"type":    "work",
			"primary": true,
		}}
	}
	if len(ext) > 0 {
		resource[domain.EnterpriseUserSchemaURN] = ext
	}
	return resource, nil
}
