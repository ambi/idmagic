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
	group := &idmdomain.Group{
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
						if _, err := u.GroupRepo.AddMember(ctx, &idmdomain.GroupMember{
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

func (u *Usecases) ListGroups(ctx context.Context, tenantID string, query ListQuery) (ListResult, error) {
	groups, err := u.GroupRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return ListResult{}, err
	}

	var expr domain.FilterExpr
	if query.Filter != "" {
		expr, err = domain.ParseFilter(query.Filter, domain.GroupFilterAttributes)
		if err != nil {
			return ListResult{}, err
		}
	}

	var matched []map[string]any
	for _, g := range groups {
		ref, err := u.ScimRepo.FindGroupRefByGroupID(ctx, tenantID, g.ID)
		if err != nil {
			return ListResult{}, err
		}
		scimID := g.ID
		if ref != nil {
			scimID = ref.ScimID
		}

		if expr != nil && !expr.Matches(groupFilterAttrs(g, scimID)) {
			continue
		}

		scimGrp, err := u.toScimGroup(ctx, g, scimID)
		if err != nil {
			return ListResult{}, err
		}
		matched = append(matched, scimGrp)
	}

	return paginate(matched, query)
}

// groupFilterAttrs flattens a Group into the lower-cased attribute map
// domain.GroupFilterAttributes expects.
func groupFilterAttrs(group *idmdomain.Group, scimID string) map[string]any {
	return map[string]any{
		"displayname": group.Name,
		"id":          scimID,
	}
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
						if _, err := u.GroupRepo.AddMember(ctx, &idmdomain.GroupMember{
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
						if _, err := u.GroupRepo.AddMember(ctx, &idmdomain.GroupMember{
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

func (u *Usecases) toScimGroup(ctx context.Context, group *idmdomain.Group, scimID string) (map[string]any, error) {
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
