package ports

import (
	"context"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

type SamlIdentityProviderProfileRepository interface {
	EnsureDefaultIDPProfile(ctx context.Context, tenantID string) (*samldomain.SamlIdentityProviderProfile, error)
	FindIDPProfileByID(ctx context.Context, tenantID, profileID string) (*samldomain.SamlIdentityProviderProfile, error)
	ListIDPProfilesByTenant(ctx context.Context, tenantID string) ([]*samldomain.SamlIdentityProviderProfile, error)
	SaveIDPProfile(ctx context.Context, profile *samldomain.SamlIdentityProviderProfile) error
	DeleteIDPProfile(ctx context.Context, tenantID, profileID string) error
}
