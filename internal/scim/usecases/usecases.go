package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	idmports "github.com/ambi/idmagic/internal/identitymanagement/ports"
	"github.com/ambi/idmagic/internal/scim/ports"
	"github.com/ambi/idmagic/internal/shared/spec"
)

var ErrNotFound = errors.New("SCIM resource not found")

type Usecases struct {
	ScimRepo  ports.ScimRepository
	UserRepo  idmports.UserRepository
	GroupRepo idmports.GroupRepository
	Emit      func(spec.DomainEvent)
}

func NewUsecases(
	scimRepo ports.ScimRepository,
	userRepo idmports.UserRepository,
	groupRepo idmports.GroupRepository,
	emit func(spec.DomainEvent),
) *Usecases {
	return &Usecases{
		ScimRepo:  scimRepo,
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
		Emit:      emit,
	}
}

// Token Management
func (u *Usecases) GenerateToken(ctx context.Context, tenantID, description string, expiryDays int) (string, *ports.ScimToken, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", nil, err
	}
	tokenStr := hex.EncodeToString(rawBytes)

	hash := sha256.Sum256([]byte(tokenStr))
	hashStr := hex.EncodeToString(hash[:])

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", nil, err
	}
	id := hex.EncodeToString(idBytes)

	var expiresAt *time.Time
	if expiryDays > 0 {
		exp := time.Now().AddDate(0, 0, expiryDays)
		expiresAt = &exp
	}

	tok := &ports.ScimToken{
		ID:          id,
		TenantID:    tenantID,
		TokenHash:   hashStr,
		Description: description,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	if err := u.ScimRepo.SaveToken(ctx, tok); err != nil {
		return "", nil, err
	}

	return tokenStr, tok, nil
}

func (u *Usecases) AuthenticateToken(ctx context.Context, tokenStr string) (string, error) {
	hash := sha256.Sum256([]byte(tokenStr))
	hashStr := hex.EncodeToString(hash[:])

	tok, err := u.ScimRepo.FindToken(ctx, hashStr)
	if err != nil {
		return "", err
	}
	if tok == nil {
		return "", errors.New("invalid token")
	}

	if tok.ExpiresAt != nil && tok.ExpiresAt.Before(time.Now()) {
		return "", errors.New("token expired")
	}

	return tok.TenantID, nil
}

func (u *Usecases) ListTokens(ctx context.Context, tenantID string) ([]*ports.ScimToken, error) {
	return u.ScimRepo.ListTokens(ctx, tenantID)
}

func (u *Usecases) RevokeToken(ctx context.Context, tenantID, id string) error {
	return u.ScimRepo.DeleteToken(ctx, tenantID, id)
}

// Config Management
func (u *Usecases) GetConfig(ctx context.Context, tenantID string) (*ports.ScimConfig, error) {
	cfg, err := u.ScimRepo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		// default disabled
		return &ports.ScimConfig{TenantID: tenantID, Enabled: false}, nil
	}
	return cfg, nil
}

func (u *Usecases) UpdateConfig(ctx context.Context, tenantID string, enabled bool) (*ports.ScimConfig, error) {
	now := time.Now().UTC()
	cfg := &ports.ScimConfig{TenantID: tenantID, Enabled: enabled, CreatedAt: now, UpdatedAt: now}
	existing, err := u.ScimRepo.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if existing != nil && !existing.CreatedAt.IsZero() {
		cfg.CreatedAt = existing.CreatedAt
	}
	if err := u.ScimRepo.SaveConfig(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SCIM API Handlers mapping to IdP Core

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
	status := spec.UserStatusActive
	if !activeVal {
		status = spec.UserStatusDisabled
	}

	user := &spec.User{
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
		Lifecycle: spec.UserLifecycle{
			Status:          status,
			StatusChangedAt: &now,
		},
		Attributes: make(map[string]spec.AttributeValue),
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

func (u *Usecases) setUserActive(user *spec.User, active bool) {
	now := time.Now()
	if active && user.Lifecycle.Status != spec.UserStatusActive {
		user.Lifecycle.Status = spec.UserStatusActive
		user.Lifecycle.StatusChangedAt = &now
	} else if !active && user.Lifecycle.Status == spec.UserStatusActive {
		user.Lifecycle.Status = spec.UserStatusDisabled
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
	user.Lifecycle.Status = spec.UserStatusPendingDeletion
	user.Lifecycle.StatusChangedAt = &now
	user.UpdatedAt = now

	return u.UserRepo.Save(ctx, user)
}

func (u *Usecases) ListUsers(ctx context.Context, tenantID, filter string) ([]map[string]any, error) {
	users, err := u.UserRepo.FindAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var out []map[string]any
	for _, user := range users {
		ref, err := u.ScimRepo.FindUserRefByUserID(ctx, tenantID, user.ID)
		if err != nil {
			return nil, err
		}
		scimID := user.ID
		if ref != nil {
			scimID = ref.ScimID
		}

		// Apply simple filter (userName eq "...")
		if filter != "" {
			// very naive parsing: userName eq "bjensen@example.com"
			if len(filter) > 13 && filter[:11] == "userName eq" {
				expected := filter[12 : len(filter)-1] // strip quotes
				if user.PreferredUsername != expected {
					continue
				}
			}
		}

		out = append(out, u.toScimUser(user, scimID))
	}
	return out, nil
}

func (u *Usecases) toScimUser(user *spec.User, scimID string) map[string]any {
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

	active := user.Lifecycle.Status == spec.UserStatusActive

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

// Groups

func (u *Usecases) CreateGroup(ctx context.Context, tenantID string, body map[string]any) (map[string]any, error) {
	displayName, _ := body["displayName"].(string)
	if displayName == "" {
		return nil, errors.New("displayName is required")
	}

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	id := fmt.Sprintf("group_%s", hex.EncodeToString(idBytes))

	now := time.Now()
	group := &spec.Group{
		ID:          id,
		TenantID:    tenantID,
		Name:        displayName,
		Description: nil,
		Roles:       []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.GroupRepo.Save(ctx, group); err != nil {
		return nil, err
	}

	scimID, _ := body["id"].(string)
	if scimID == "" {
		scimID = id
	}

	ref := &ports.ScimGroupRef{
		TenantID: tenantID,
		ScimID:   scimID,
		GroupID:  id,
	}
	if err := u.ScimRepo.SaveGroupRef(ctx, ref); err != nil {
		return nil, err
	}

	// メンバーがいる場合、追加
	if members, ok := body["members"].([]any); ok {
		for _, mVal := range members {
			if mMap, ok := mVal.(map[string]any); ok {
				userScimID, _ := mMap["value"].(string)
				if userScimID != "" {
					userRef, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, userScimID)
					if err == nil && userRef != nil {
						if _, err := u.GroupRepo.AddMember(ctx, &spec.GroupMember{
							GroupID:   id,
							UserID:    userRef.UserID,
							CreatedAt: time.Now(),
						}); err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	return u.toScimGroup(ctx, group, scimID)
}

func (u *Usecases) GetGroup(ctx context.Context, tenantID, scimID string) (map[string]any, error) {
	ref, err := u.ScimRepo.FindGroupRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNotFound
	}

	group, err := u.GroupRepo.FindByID(ctx, tenantID, ref.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrNotFound
	}

	return u.toScimGroup(ctx, group, scimID)
}

func (u *Usecases) ListGroups(ctx context.Context, tenantID string) ([]map[string]any, error) {
	groups, err := u.GroupRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var out []map[string]any
	for _, g := range groups {
		ref, err := u.ScimRepo.FindGroupRefByGroupID(ctx, tenantID, g.ID)
		if err != nil {
			return nil, err
		}
		scimID := g.ID
		if ref != nil {
			scimID = ref.ScimID
		}

		scimGrp, err := u.toScimGroup(ctx, g, scimID)
		if err != nil {
			return nil, err
		}
		out = append(out, scimGrp)
	}

	return out, nil
}

func (u *Usecases) UpdateGroup(ctx context.Context, tenantID, scimID string, body map[string]any) (map[string]any, error) {
	ref, err := u.ScimRepo.FindGroupRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNotFound
	}

	group, err := u.GroupRepo.FindByID(ctx, tenantID, ref.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrNotFound
	}

	displayName, _ := body["displayName"].(string)
	if displayName != "" {
		group.Name = displayName
	}

	// メンバー置換
	if members, ok := body["members"].([]any); ok {
		existingMembers, err := u.GroupRepo.ListMembersByGroup(ctx, tenantID, group.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range existingMembers {
			if _, err := u.GroupRepo.RemoveMember(ctx, tenantID, group.ID, m.UserID); err != nil {
				return nil, err
			}
		}

		for _, mVal := range members {
			if mMap, ok := mVal.(map[string]any); ok {
				userScimID, _ := mMap["value"].(string)
				if userScimID != "" {
					userRef, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, userScimID)
					if err == nil && userRef != nil {
						if _, err := u.GroupRepo.AddMember(ctx, &spec.GroupMember{
							GroupID:   group.ID,
							UserID:    userRef.UserID,
							CreatedAt: time.Now(),
						}); err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	now := time.Now()
	group.UpdatedAt = now
	if err := u.GroupRepo.Save(ctx, group); err != nil {
		return nil, err
	}

	return u.toScimGroup(ctx, group, scimID)
}

func (u *Usecases) PatchGroup(ctx context.Context, tenantID, scimID string, body map[string]any) (map[string]any, error) {
	ref, err := u.ScimRepo.FindGroupRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNotFound
	}

	group, err := u.GroupRepo.FindByID(ctx, tenantID, ref.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
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

		if path == "members" || path == "" {
			switch op {
			case "add":
				if err := u.patchAddMembers(ctx, tenantID, group.ID, value); err != nil {
					return nil, err
				}
			case "remove":
				if err := u.patchRemoveMembers(ctx, tenantID, group.ID, value); err != nil {
					return nil, err
				}
			case "replace":
				if err := u.patchReplaceMembers(ctx, tenantID, group.ID, value); err != nil {
					return nil, err
				}
			}
		}
	}

	now := time.Now()
	group.UpdatedAt = now
	if err := u.GroupRepo.Save(ctx, group); err != nil {
		return nil, err
	}

	return u.toScimGroup(ctx, group, scimID)
}

func (u *Usecases) patchAddMembers(ctx context.Context, tenantID, groupID string, value any) error {
	if valList, ok := value.([]any); ok {
		for _, v := range valList {
			if vMap, ok := v.(map[string]any); ok {
				userScimID, _ := vMap["value"].(string)
				if userScimID != "" {
					userRef, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, userScimID)
					if err == nil && userRef != nil {
						if _, err := u.GroupRepo.AddMember(ctx, &spec.GroupMember{
							GroupID:   groupID,
							UserID:    userRef.UserID,
							CreatedAt: time.Now(),
						}); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func (u *Usecases) patchRemoveMembers(ctx context.Context, tenantID, groupID string, value any) error {
	if valList, ok := value.([]any); ok {
		for _, v := range valList {
			if vMap, ok := v.(map[string]any); ok {
				userScimID, _ := vMap["value"].(string)
				if userScimID != "" {
					userRef, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, userScimID)
					if err == nil && userRef != nil {
						if _, err := u.GroupRepo.RemoveMember(ctx, tenantID, groupID, userRef.UserID); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func (u *Usecases) patchReplaceMembers(ctx context.Context, tenantID, groupID string, value any) error {
	existingMembers, err := u.GroupRepo.ListMembersByGroup(ctx, tenantID, groupID)
	if err != nil {
		return err
	}
	for _, m := range existingMembers {
		if _, err := u.GroupRepo.RemoveMember(ctx, tenantID, groupID, m.UserID); err != nil {
			return err
		}
	}
	return u.patchAddMembers(ctx, tenantID, groupID, value)
}

func (u *Usecases) DeleteGroup(ctx context.Context, tenantID, scimID string) error {
	ref, err := u.ScimRepo.FindGroupRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return err
	}
	if ref == nil {
		return errors.New("group not found")
	}

	if err := u.GroupRepo.Delete(ctx, tenantID, ref.GroupID); err != nil {
		return err
	}
	return u.ScimRepo.DeleteGroupRef(ctx, tenantID, scimID)
}

func (u *Usecases) toScimGroup(ctx context.Context, group *spec.Group, scimID string) (map[string]any, error) {
	members, err := u.GroupRepo.ListMembersByGroup(ctx, group.TenantID, group.ID)
	if err != nil {
		return nil, err
	}

	scimMembers := []map[string]any{}
	for _, m := range members {
		ref, err := u.ScimRepo.FindUserRefByUserID(ctx, group.TenantID, m.UserID)
		userScimID := m.UserID
		if err == nil && ref != nil {
			userScimID = ref.ScimID
		}
		scimMembers = append(scimMembers, map[string]any{
			"value":   userScimID,
			"display": m.UserID,
		})
	}

	var updatedStr string
	if !group.UpdatedAt.IsZero() {
		updatedStr = group.UpdatedAt.Format(time.RFC3339)
	}

	return map[string]any{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"id":          scimID,
		"displayName": group.Name,
		"members":     scimMembers,
		"meta": map[string]any{
			"resourceType": "Group",
			"created":      group.CreatedAt.Format(time.RFC3339),
			"lastModified": updatedStr,
		},
	}, nil
}
