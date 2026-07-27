package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var ErrEmailConflict = errors.New("email already exists")

type ProvisionFederatedUserInput struct {
	PreferredUsername string
	Name              *string
	Email             *string
	EmailVerified     bool
	Attributes        map[string]userdomain.AttributeValue
	Now               time.Time
}

// ProvisionFederatedUser is IdManagement's published creation surface for an
// Authentication-verified JIT identity. It deliberately creates no password
// credential; password authentication therefore remains fail-closed.
func ProvisionFederatedUser(
	ctx context.Context,
	deps AdminUserDeps,
	in ProvisionFederatedUserInput,
) (*userdomain.User, error) {
	username := strings.TrimSpace(in.PreferredUsername)
	if username == "" {
		return nil, errors.New("preferred username is required")
	}
	tenantID := tenancy.TenantID(ctx)
	existing, err := deps.UserRepo.FindByUsername(ctx, tenantID, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameConflict
	}
	if in.Email != nil {
		email := strings.TrimSpace(*in.Email)
		if email == "" {
			in.Email = nil
		} else {
			in.Email = &email
			existing, err := deps.UserRepo.FindByEmail(ctx, tenantID, email)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return nil, ErrEmailConflict
			}
		}
	}
	if len(in.Attributes) > 0 {
		defs, err := effectiveUserAttributeDefs(ctx, deps.AttrSchemaRepo, tenantID)
		if err != nil {
			return nil, err
		}
		if err := userdomain.ValidateAttributes(in.Attributes, defs); err != nil {
			return nil, errors.Join(ErrInvalidAttribute, err)
		}
	}
	now := idmusecases.NormalizedNow(in.Now)
	if err := idmusecases.CheckQuotaAndAudit(
		ctx, deps.QuotaRepo, deps.Emit, tenantID, tenancydomain.ResourceUsers, now,
	); err != nil {
		return nil, err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	user := &userdomain.User{
		ID: id, TenantID: tenantID, PreferredUsername: username,
		Name: in.Name, Email: in.Email, EmailVerified: in.EmailVerified,
		Roles: []string{}, Attributes: in.Attributes,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	if err := captureUserMutation(ctx, deps, nil, user, nil, now); err != nil {
		return nil, err
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.UserCreated{
		At: now, TenantID: tenantID, ActorUserID: "identity-broker", TargetUserID: user.ID,
	}); err != nil {
		return nil, err
	}
	notifyProvisioning(ctx, deps, tenantID, user.ID, userports.ProvisioningUserCreated, now)
	return user, nil
}
