package wsfederation

import (
	"context"

	"github.com/ambi/idmagic/backend/wsfederation/domain"
	"github.com/ambi/idmagic/backend/wsfederation/internal/usecases"
)

func (m Module) CreateSeedRelyingParty(ctx context.Context, rp *domain.WsFedRelyingParty) error {
	return (usecases.RelyingPartyAdministration{RPRepo: m.RPRepo}).CreateSeed(ctx, rp)
}
