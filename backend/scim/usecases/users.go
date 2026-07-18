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
	userName, _ := body["userName"].(string)
	if userName == "" {
		return nil, errors.New("userName is required")
	}

	var emailVal string
	if emails, ok := body["emails"].([]any); ok && len(emails) > 0 {
		if firstEmail, ok := emails[0].(map[string]any); ok {
			emailVal, _ = firstEmail["value"].(string)
		}
	}

	var givenName, familyName, displayName string
	if nameMap, ok := body["name"].(map[string]any); ok {
		givenName, _ = nameMap["givenName"].(string)
		familyName, _ = nameMap["familyName"].(string)
		displayName, _ = nameMap["formatted"].(string)
	}
	if displayName == "" {
		displayName = userName
	}

	activeVal := true
	if act, exists := body["active"].(bool); exists {
		activeVal = act
	}

	// ユーザー作成
	subBytes := make([]byte, 16)
	if _, err := rand.Read(subBytes); err != nil {
		return nil, err
	}
	sub := fmt.Sprintf("user_%s", hex.EncodeToString(subBytes))

	now := time.Now()
	status := idmdomain.UserStatusActive
	if !activeVal {
		status = idmdomain.UserStatusDisabled
	}

	user := &idmdomain.User{
		ID:                sub,
		TenantID:          tenantID,
		PreferredUsername: userName,
		PasswordHash:      "", // SCIM users usually don't have local password initially
		Name:              &displayName,
		GivenName:         &givenName,
		FamilyName:        &familyName,
		Email:             &emailVal,
		EmailVerified:     true,
		Roles:             []string{},
		Lifecycle: idmdomain.UserLifecycle{
			Status:          status,
			StatusChangedAt: &now,
		},
		Attributes: make(map[string]idmdomain.AttributeValue),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	scimID, _ := body["id"].(string)
	if scimID == "" {
		// generate unique ID
		idBytes := make([]byte, 16)
		if _, err := rand.Read(idBytes); err != nil {
			return nil, err
		}
		scimID = hex.EncodeToString(idBytes)
	}

	ref := &ports.ScimUserRef{
		TenantID: tenantID,
		ScimID:   scimID,
		UserID:   sub,
	}
	if err := u.ScimRepo.SaveUserRef(ctx, ref); err != nil {
		return nil, err
	}

	return u.toScimUser(user, scimID), nil
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

	userName, _ := body["userName"].(string)
	if userName != "" {
		user.PreferredUsername = userName
	}

	var emailVal string
	if emails, ok := body["emails"].([]any); ok && len(emails) > 0 {
		if firstEmail, ok := emails[0].(map[string]any); ok {
			emailVal, _ = firstEmail["value"].(string)
		}
	}
	if emailVal != "" {
		user.Email = &emailVal
	}

	if nameMap, ok := body["name"].(map[string]any); ok {
		givenName, _ := nameMap["givenName"].(string)
		familyName, _ := nameMap["familyName"].(string)
		displayName, _ := nameMap["formatted"].(string)
		if givenName != "" {
			user.GivenName = &givenName
		}
		if familyName != "" {
			user.FamilyName = &familyName
		}
		if displayName != "" {
			user.Name = &displayName
		}
	}

	if act, exists := body["active"].(bool); exists {
		u.setUserActive(user, act)
	}

	user.UpdatedAt = time.Now()
	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return u.toScimUser(user, scimID), nil
}

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

	ops, _ := body["Operations"].([]any)
	for _, opVal := range ops {
		opMap, ok := opVal.(map[string]any)
		if !ok {
			continue
		}
		op, _ := opMap["op"].(string)
		path, _ := opMap["path"].(string)
		value := opMap["value"]

		// active field patch
		if path == "active" || op == "replace" && path == "active" || path == "" && op == "replace" {
			switch val := value.(type) {
			case map[string]any:
				if act, exists := val["active"].(bool); exists {
					u.setUserActive(user, act)
				}
			case bool:
				u.setUserActive(user, val)
			}
		}
	}

	user.UpdatedAt = time.Now()
	if err := u.UserRepo.Save(ctx, user); err != nil {
		return nil, err
	}

	return u.toScimUser(user, scimID), nil
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
		},
	}
}
