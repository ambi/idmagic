package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/saml/domain"
)

type SamlIdentityProviderProfileRepository interface {
	EnsureDefaultIDPProfile(ctx context.Context, tenantID string) (*domain.SamlIdentityProviderProfile, error)
	FindIDPProfileByID(ctx context.Context, tenantID, profileID string) (*domain.SamlIdentityProviderProfile, error)
	ListIDPProfilesByTenant(ctx context.Context, tenantID string) ([]*domain.SamlIdentityProviderProfile, error)
	SaveIDPProfile(ctx context.Context, profile *domain.SamlIdentityProviderProfile) error
	DeleteIDPProfile(ctx context.Context, tenantID, profileID string) error
}
