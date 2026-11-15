package saml

import (
	"context"

	"github.com/ambi/idmagic/backend/saml/domain"
	"github.com/ambi/idmagic/backend/saml/internal/usecases"
)

func (m Module) CreateSeedServiceProvider(ctx context.Context, sp *domain.SamlServiceProvider) error {
	return (usecases.ServiceProviderAdministration{SPRepo: m.SPRepo}).CreateSeed(ctx, sp)
}
