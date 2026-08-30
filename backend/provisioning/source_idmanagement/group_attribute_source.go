package source_idmanagement

import (
	"context"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

// GroupAttributeSource resolves a Group's attributes for
// spec/contexts/provisioning.yaml models.AttributeMappingRule (source_kind=attribute).
// The resolved keys mirror the User source's shape: `id` and `display_name` are
// always present, with `description` and `email` present only when set.
//
// `display_name` is what a downstream SCIM Group's `displayName` maps from. Which
// IdMagic field it is taken from is the connection's GroupPushConfig.DisplayNameSource;
// `name` is the default and the only value with a meaning today, so an unset or
// unknown source resolves to Group.Name rather than failing the delivery — the
// display name is not a fail-closed decision.
type GroupAttributeSource struct {
	GroupRepo groupports.GroupRepository
	// UserRepo is unused for attribute resolution; membership is pushed by the
	// delivery engine through its own PATCH, not as a mapped attribute.
	UserRepo any
}

var _ ports.AttributeSource = (*GroupAttributeSource)(nil)

func (s *GroupAttributeSource) ResolveAttributes(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, sourceID string) (map[string]any, bool, error) {
	if sourceType != domain.SourceTypeGroup {
		return nil, false, nil
	}
	group, err := s.GroupRepo.FindByID(ctx, tenantID, sourceID)
	if err != nil {
		return nil, false, err
	}
	if group == nil {
		return nil, false, nil
	}
	attrs := map[string]any{
		"id":           group.ID,
		"display_name": group.Name,
		"name":         group.Name,
	}
	if group.Description != nil {
		attrs["description"] = *group.Description
	}
	if group.Email != nil {
		attrs["email"] = *group.Email
	}
	return attrs, true, nil
}

// CombinedAttributeSource dispatches to the source that owns each source type.
// The delivery engine holds one ports.AttributeSource, so a connection that
// pushes both Users and Groups needs the two adapters behind a single value.
type CombinedAttributeSource struct {
	User  ports.AttributeSource
	Group ports.AttributeSource
}

var _ ports.AttributeSource = CombinedAttributeSource{}

func (s CombinedAttributeSource) ResolveAttributes(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, sourceID string) (map[string]any, bool, error) {
	switch sourceType {
	case domain.SourceTypeUser:
		if s.User == nil {
			return nil, false, nil
		}
		return s.User.ResolveAttributes(ctx, tenantID, sourceType, sourceID)
	case domain.SourceTypeGroup:
		if s.Group == nil {
			return nil, false, nil
		}
		return s.Group.ResolveAttributes(ctx, tenantID, sourceType, sourceID)
	default:
		return nil, false, nil
	}
}

// GroupMemberSource reads a Group's direct members. Nested groups are not
// expanded: the inbound policy treats direct membership as the membership, and
// the outbound side follows it (see the work item's Out of Scope).
type GroupMemberSource struct{ GroupRepo groupports.GroupRepository }

var _ ports.GroupMemberSource = (*GroupMemberSource)(nil)

func (s *GroupMemberSource) ListMemberUserIDs(ctx context.Context, tenantID, groupID string) ([]string, error) {
	members, err := s.GroupRepo.ListMembersByGroup(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	return userIDs, nil
}
