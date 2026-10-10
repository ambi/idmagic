package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

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
	return createUser(ctx, deps, newUser{
		User: userdomain.User{
			TenantID: tenantports.TenantID(ctx), PreferredUsername: username,
			Name: in.Name, Email: normalizeEmail(in.Email), EmailVerified: in.EmailVerified,
			Roles: []string{}, Attributes: in.Attributes,
		},
		ActorUserID: "identity-broker",
	}, idmusecases.NormalizedNow(in.Now))
}
