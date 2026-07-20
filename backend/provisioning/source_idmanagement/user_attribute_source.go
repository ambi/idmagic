// Package identitysource adapts IdManagement's User aggregate to
// ports.AttributeSource for the Provisioning delivery engine (ADR-128
// decision 4: Provisioning legitimately depends on IdManagement per
// context_map, so this adapter may import idmanagement/domain and ports).
package source_idmanagement

import (
	"context"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
)

// UserAttributeSource resolves a User's attributes for
// spec/contexts/provisioning.yaml models.AttributeMappingRule (source_kind=attribute).
// The resolved keys match wi-45's default mapping table source column:
// id, preferred_username, display_name, given_name, family_name, email, active.
type UserAttributeSource struct{ UserRepo userports.UserRepository }

var _ ports.AttributeSource = (*UserAttributeSource)(nil)

func (s *UserAttributeSource) ResolveAttributes(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, sourceID string) (map[string]any, bool, error) {
	if sourceType != domain.SourceTypeUser {
		return nil, false, nil
	}
	// FindBySubIncludingDeleted (not FindBySub): a deactivate/delete delivery
	// created for TriggerUserDeleted must still resolve attributes for the
	// now-tombstoned user (its Lifecycle.Status is no longer Active, so
	// "active" resolves to false) so the downstream deactivate/update call
	// actually goes out. Using FindBySub here would make OnDelete=deactivate
	// (the default DeprovisionPolicy) silently no-op forever, since the User
	// row is already gone-per-FindBySub by the time the delivery executes.
	user, err := s.UserRepo.FindBySubIncludingDeleted(ctx, sourceID)
	if err != nil {
		return nil, false, err
	}
	if user == nil || user.TenantID != tenantID {
		return nil, false, nil
	}
	attrs := map[string]any{
		"id":                 user.ID,
		"preferred_username": user.PreferredUsername,
		"active":             user.Lifecycle.EffectiveStatus() == idmdomain.UserStatusActive,
	}
	if user.Name != nil {
		attrs["display_name"] = *user.Name
	}
	if user.GivenName != nil {
		attrs["given_name"] = *user.GivenName
	}
	if user.FamilyName != nil {
		attrs["family_name"] = *user.FamilyName
	}
	if user.Email != nil {
		attrs["email"] = *user.Email
	}
	return attrs, true, nil
}
