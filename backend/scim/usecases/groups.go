package usecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/scim/domain"
	"github.com/ambi/idmagic/backend/scim/ports"
)

// CreateGroup validates the full request (displayName uniqueness, member
// resolvability) before any write. If a persistence step still fails after
// validation (rare operational failure, ADR-122), already-completed steps
// are compensated best-effort so a duplicate-key retry doesn't see a
// half-created group.
func (u *Usecases) CreateGroup(ctx context.Context, tenantID string, body map[string]any) (map[string]any, error) {
	w, err := domain.ParseGroupWrite(body)
	if err != nil {
		return nil, err
	}

	existing, err := u.findGroupByDisplayName(ctx, tenantID, w.DisplayName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: displayName %q already exists", ErrDuplicate, w.DisplayName)
	}

	memberUserIDs, err := u.resolveMemberUserIDs(ctx, tenantID, w.MemberScimIDs)
	if err != nil {
		return nil, err
	}

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	id := fmt.Sprintf("group_%s", hex.EncodeToString(idBytes))

	scimIDBytes := make([]byte, 16)
	if _, err := rand.Read(scimIDBytes); err != nil {
		return nil, err
	}
	scimID := hex.EncodeToString(scimIDBytes)

	now := time.Now()
	group := &groupdomain.Group{
		ID:        id,
		TenantID:  tenantID,
		Name:      w.DisplayName,
		Roles:     []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := u.GroupRepo.Save(ctx, group); err != nil {
		return nil, err
	}

	ref := &ports.ScimGroupRef{TenantID: tenantID, ScimID: scimID, GroupID: id}
	if err := u.ScimRepo.SaveGroupRef(ctx, ref); err != nil {
		_ = u.GroupRepo.Delete(ctx, tenantID, id)
		return nil, err
	}

	if err := u.addMembers(ctx, tenantID, id, memberUserIDs); err != nil {
		_ = u.ScimRepo.DeleteGroupRef(ctx, tenantID, scimID)
		_ = u.GroupRepo.Delete(ctx, tenantID, id)
		return nil, err
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
func groupFilterAttrs(group *groupdomain.Group, scimID string) map[string]any {
	return map[string]any{
		"displayname":       group.Name,
		"id":                scimID,
		"meta.created":      group.CreatedAt.Format(time.RFC3339),
		"meta.lastmodified": group.UpdatedAt.Format(time.RFC3339),
	}
}

// findGroupByDisplayName scans the tenant's groups for a displayName
// collision. GroupRepository has no indexed lookup by name, so this is
// O(n) in tenant group count; acceptable at the usecase layer (ListGroups
// already does a full tenant scan for every request).
func (u *Usecases) findGroupByDisplayName(ctx context.Context, tenantID, displayName string) (*groupdomain.Group, error) {
	groups, err := u.GroupRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.Name == displayName {
			return g, nil
		}
	}
	return nil, nil //nolint:nilnil // 契約: 見つからない場合は (nil, nil) (GroupRepo の Find* と同一)。
}

// resolveMemberUserIDs resolves every SCIM member id to an internal User id
// before any write happens (ADR-122 validate-first). An unresolvable
// member is a *domain.MutationError (invalidValue); the caller has not yet
// touched persistence at that point.
func (u *Usecases) resolveMemberUserIDs(ctx context.Context, tenantID string, scimIDs []string) ([]string, error) {
	userIDs := make([]string, 0, len(scimIDs))
	for _, scimID := range scimIDs {
		userRef, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, scimID)
		if err != nil {
			return nil, err
		}
		if userRef == nil {
			return nil, domain.NewMutationError("invalidValue", "member %q does not resolve to a User in this tenant", scimID)
		}
		userIDs = append(userIDs, userRef.UserID)
	}
	return userIDs, nil
}

// addMembers adds each userID, compensating (best-effort) by removing
// already-added members if a later AddMember call fails (ADR-122).
func (u *Usecases) addMembers(ctx context.Context, tenantID, groupID string, userIDs []string) error {
	added := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		if _, err := u.GroupRepo.AddMember(ctx, &groupdomain.GroupMember{GroupID: groupID, UserID: userID, CreatedAt: time.Now()}); err != nil {
			for _, a := range added {
				_, _ = u.GroupRepo.RemoveMember(ctx, tenantID, groupID, a)
			}
			return err
		}
		added = append(added, userID)
	}
	return nil
}

// replaceMembers removes all existing members and adds newUserIDs,
// attempting best-effort compensation to restore the prior membership if a
// step fails partway (ADR-122: no cross-context DB transaction).
func (u *Usecases) replaceMembers(ctx context.Context, tenantID, groupID string, newUserIDs []string) error {
	existing, err := u.GroupRepo.ListMembersByGroup(ctx, tenantID, groupID)
	if err != nil {
		return err
	}

	removed := make([]string, 0, len(existing))
	for _, m := range existing {
		if _, err := u.GroupRepo.RemoveMember(ctx, tenantID, groupID, m.UserID); err != nil {
			for _, r := range removed {
				_, _ = u.GroupRepo.AddMember(ctx, &groupdomain.GroupMember{GroupID: groupID, UserID: r, CreatedAt: time.Now()})
			}
			return err
		}
		removed = append(removed, m.UserID)
	}

	if err := u.addMembers(ctx, tenantID, groupID, newUserIDs); err != nil {
		for _, r := range removed {
			_, _ = u.GroupRepo.AddMember(ctx, &groupdomain.GroupMember{GroupID: groupID, UserID: r, CreatedAt: time.Now()})
		}
		return err
	}
	return nil
}

// removeMembersLenient removes members that resolve to a User; a member
// value that no longer resolves is a no-op (removing something that isn't
// there is already the desired end state), matching pre-wi-239 behavior.
func (u *Usecases) removeMembersLenient(ctx context.Context, tenantID, groupID string, value any) error {
	scimIDs, err := groupMemberScimIDs(value)
	if err != nil {
		return err
	}
	for _, scimID := range scimIDs {
		userRef, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, scimID)
		if err != nil {
			return err
		}
		if userRef == nil {
			continue
		}
		if _, err := u.GroupRepo.RemoveMember(ctx, tenantID, groupID, userRef.UserID); err != nil {
			return err
		}
	}
	return nil
}

func groupMemberScimIDs(value any) ([]string, error) {
	valList, ok := value.([]any)
	if !ok {
		return nil, domain.NewMutationError("invalidValue", "members value must be an array")
	}
	scimIDs := make([]string, 0, len(valList))
	for _, v := range valList {
		vMap, ok := v.(map[string]any)
		if !ok {
			return nil, domain.NewMutationError("invalidValue", "each member must be an object with a value")
		}
		scimID, _ := vMap["value"].(string)
		if scimID == "" {
			return nil, domain.NewMutationError("invalidValue", "member value must be a non-empty string")
		}
		scimIDs = append(scimIDs, scimID)
	}
	return scimIDs, nil
}

// UpdateGroup implements PUT full-replace semantics (ADR-122): displayName
// is required, and members omitted from the body clears the group's
// membership. Uniqueness and member resolvability are validated before any
// write.
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

	w, err := domain.ParseGroupWrite(body)
	if err != nil {
		return nil, err
	}
	if w.DisplayName != group.Name {
		if existing, err := u.findGroupByDisplayName(ctx, tenantID, w.DisplayName); err != nil {
			return nil, err
		} else if existing != nil && existing.ID != group.ID {
			return nil, fmt.Errorf("%w: displayName %q already exists", ErrDuplicate, w.DisplayName)
		}
	}

	targetUserIDs, err := u.resolveMemberUserIDs(ctx, tenantID, w.MemberScimIDs)
	if err != nil {
		return nil, err
	}

	group.Name = w.DisplayName
	group.UpdatedAt = time.Now()
	if err := u.GroupRepo.Save(ctx, group); err != nil {
		return nil, err
	}

	if err := u.replaceMembers(ctx, tenantID, group.ID, targetUserIDs); err != nil {
		return nil, err
	}

	return u.toScimGroup(ctx, group, scimID)
}

// PatchGroup applies RFC 7644 §3.5.2 operations validated by
// domain.ParseGroupPatchOps. Every add/replace member reference is
// resolved before any operation is applied (ADR-122 validate-first).
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

	ops, err := domain.ParseGroupPatchOps(body)
	if err != nil {
		return nil, err
	}

	resolvedMembers := make([][]string, len(ops))
	for i, op := range ops {
		if op.Attr != domain.GroupAttrMembers || (op.Op != "add" && op.Op != "replace") {
			continue
		}
		scimIDs, err := groupMemberScimIDs(op.Value)
		if err != nil {
			return nil, err
		}
		userIDs, err := u.resolveMemberUserIDs(ctx, tenantID, scimIDs)
		if err != nil {
			return nil, err
		}
		resolvedMembers[i] = userIDs
	}

	for i, op := range ops {
		if err := u.applyGroupPatchOp(ctx, tenantID, group, op, resolvedMembers[i]); err != nil {
			return nil, err
		}
	}

	group.UpdatedAt = time.Now()
	if err := u.GroupRepo.Save(ctx, group); err != nil {
		return nil, err
	}

	return u.toScimGroup(ctx, group, scimID)
}

func (u *Usecases) applyGroupPatchOp(ctx context.Context, tenantID string, group *groupdomain.Group, op domain.GroupPatchOp, resolvedUserIDs []string) error {
	switch op.Attr {
	case domain.GroupAttrDisplayName:
		if op.Op == "remove" {
			return domain.NewMutationError("invalidValue", "displayName cannot be removed")
		}
		displayName, _ := op.Value.(string)
		if displayName == "" {
			return domain.NewMutationError("invalidValue", "displayName value must be a non-empty string")
		}
		if displayName != group.Name {
			existing, err := u.findGroupByDisplayName(ctx, tenantID, displayName)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != group.ID {
				return fmt.Errorf("%w: displayName %q already exists", ErrDuplicate, displayName)
			}
		}
		group.Name = displayName
	case domain.GroupAttrMembers:
		switch op.Op {
		case "add":
			return u.addMembers(ctx, tenantID, group.ID, resolvedUserIDs)
		case "remove":
			return u.removeMembersLenient(ctx, tenantID, group.ID, op.Value)
		case "replace":
			return u.replaceMembers(ctx, tenantID, group.ID, resolvedUserIDs)
		}
	}
	return nil
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

func (u *Usecases) toScimGroup(ctx context.Context, group *groupdomain.Group, scimID string) (map[string]any, error) {
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
			"location":     "/scim/v2/Groups/" + scimID,
		},
	}, nil
}
