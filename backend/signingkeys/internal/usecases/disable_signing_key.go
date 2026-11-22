package usecases

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
)

type DisableSigningKeyDeps struct {
	KeyStore signingports.KeyStore
	Emit     func(spec.DomainEvent)
}

func DisableSigningKey(ctx context.Context, deps DisableSigningKeyDeps, kid string) (*signingdomain.SigningKey, error) {
	key, changed, err := deps.KeyStore.Disable(ctx, kid)
	if err != nil {
		return nil, err
	}
	if changed && deps.Emit != nil {
		deps.Emit(&signingdomain.SigningKeyDisabled{At: *key.ArchivedAt, TenantID: key.TenantID, Kid: key.Kid})
	}
	return key, nil
}
