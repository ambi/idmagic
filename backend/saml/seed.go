package saml

import (
	"context"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	samlusecases "github.com/ambi/idmagic/backend/saml/internal/usecases"
)

func (m Module) CreateSeedServiceProvider(ctx context.Context, sp *samldomain.SamlServiceProvider) error {
	return (samlusecases.ServiceProviderAdministration{SPRepo: m.SPRepo}).CreateSeed(ctx, sp)
}
