package wsfederation

import (
	"context"

	wsfeddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	wsfedusecases "github.com/ambi/idmagic/backend/wsfederation/internal/usecases"
)

func (m Module) CreateSeedRelyingParty(ctx context.Context, rp *wsfeddomain.WsFedRelyingParty) error {
	return (wsfedusecases.RelyingPartyAdministration{RPRepo: m.RPRepo}).CreateSeed(ctx, rp)
}
