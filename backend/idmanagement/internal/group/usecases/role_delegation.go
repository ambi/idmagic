package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type SetGroupRoleDelegationInput struct {
	ActorUserID string
	GroupID     string
	Roles       []string
	Expression  string
	Now         time.Time
}

func SetGroupRoleDelegation(ctx context.Context, deps AdminGroupDeps, in SetGroupRoleDelegationInput) (*groupdomain.Group, error) {
	tenantID := tenantports.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, in.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	if !group.AutomaticMembership() {
		return nil, idmusecases.ErrInvalidRole
	}
	roles, err := idmusecases.NormalizeRoles(in.Roles)
	if err != nil {
		return nil, err
	}
	currentExpression := ""
	if group.MembershipType.Effective() == groupdomain.GroupMembershipDynamic {
		rule, err := deps.GroupRepo.FindDynamicRule(ctx, tenantID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if rule == nil && len(roles) > 0 {
			return nil, idmusecases.ErrInvalidRole
		}
		if rule != nil {
			currentExpression = rule.Expression
		}
	}
	now := idmusecases.NormalizedNow(in.Now)
	var delegation *groupdomain.GroupRoleDelegation
	if len(roles) > 0 {
		if err := group.ValidateAutomaticRoles(); err != nil {
			return nil, errors.Join(idmusecases.ErrInvalidRole, err)
		}
		if in.Expression != currentExpression || !slices.Equal(roles, group.Roles) {
			return nil, idmusecases.ErrInvalidRole
		}
		delegation = &groupdomain.GroupRoleDelegation{Roles: roles, Expression: in.Expression, ApprovedBy: in.ActorUserID, ApprovedAt: now}
	}
	if err := deps.GroupRepo.SetRoleDelegation(ctx, tenantID, in.GroupID, delegation, now); err != nil {
		if errors.Is(err, groupdomain.ErrDelegationGroupNotFound) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	group.RoleDelegation, group.UpdatedAt = delegation, now
	digest := sha256.Sum256([]byte(currentExpression))
	if roles == nil {
		roles = []string{}
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupUpdated{
		At: now, TenantID: tenantID, ActorUserID: in.ActorUserID, GroupID: in.GroupID,
		ChangedFields: []string{"role_delegation"}, DelegatedRoles: &roles, DelegationRuleSHA256: hex.EncodeToString(digest[:]),
	}); err != nil {
		return nil, err
	}
	if err := notifyProvisioning(ctx, deps, tenantID, in.GroupID, groupports.ProvisioningGroupChanged, now); err != nil {
		return nil, err
	}
	return group, nil
}
