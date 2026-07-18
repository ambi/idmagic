package usecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/scim/domain"
	"github.com/ambi/idmagic/backend/scim/ports"
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
	user := &idmdomain.User{
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
		Lifecycle: idmdomain.UserLifecycle{
			Status:          userStatusFromActive(w.Active),
			StatusChangedAt: &now,
		},
		Attributes: make(map[string]idmdomain.AttributeValue),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	ref := &ports.ScimUserRef{TenantID: tenantID, ScimID: scimID, UserID: sub}
	if err := u.ScimRepo.SaveUserRef(ctx, ref); err != nil {
		return nil, err
	}

	return u.toScimUser(user, scimID), nil
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

	return u.toScimUser(user, scimID), nil
}

// UpdateUser implements PUT full-replace semantics (ADR-122): every
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
	user.UpdatedAt = time.Now()

	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return u.toScimUser(user, scimID), nil
}

// PatchUser applies RFC 7644 §3.5.2 operations validated by
// domain.ParseUserPatchOps against the User attribute allowlist. All
// operations are validated up front (ADR-122 validate-first) before any
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

	return u.toScimUser(user, scimID), nil
}

func (u *Usecases) applyUserPatchOp(ctx context.Context, tenantID string, user *idmdomain.User, op domain.UserPatchOp) error {
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
		emails, ok := op.Value.([]any)
		if !ok || len(emails) == 0 {
			return domain.NewMutationError("invalidValue", "emails value must be a non-empty array")
		}
		firstEmail, ok := emails[0].(map[string]any)
		if !ok {
			return domain.NewMutationError("invalidValue", "emails[0] must be an object")
		}
		email, _ := firstEmail["value"].(string)
		user.Email = &email
	case domain.UserAttrActive:
		if isRemoveOp {
			u.setUserActive(user, true)
			return nil
		}
		active, _ := op.Value.(bool)
		u.setUserActive(user, active)
	}
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

func (u *Usecases) setUserActive(user *idmdomain.User, active bool) {
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

	// Soft Delete: status = PendingDeletion (ADR-080)
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

		matched = append(matched, u.toScimUser(user, scimID))
	}

	return paginate(matched, query)
}

// userFilterAttrs flattens a User into the lower-cased attribute map
// domain.UserFilterAttributes expects.
func userFilterAttrs(user *idmdomain.User, scimID string) map[string]any {
	attrs := map[string]any{
		"username": user.PreferredUsername,
		"active":   user.Lifecycle.Status == idmdomain.UserStatusActive,
		"id":       scimID,
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

func (u *Usecases) toScimUser(user *idmdomain.User, scimID string) map[string]any {
	var emailVal string
	if user.Email != nil {
		emailVal = *user.Email
	}

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

	return map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"id":       scimID,
		"userName": user.PreferredUsername,
		"name": map[string]any{
			"familyName": familyName,
			"givenName":  givenName,
			"formatted":  formattedName,
		},
		"emails": []map[string]any{
			{
				"value":   emailVal,
				"primary": true,
			},
		},
		"active": active,
		"meta": map[string]any{
			"resourceType": "User",
			"created":      user.CreatedAt.Format(time.RFC3339),
			"lastModified": user.UpdatedAt.Format(time.RFC3339),
			"location":     "/scim/v2/Users/" + scimID,
		},
	}
}
