package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/signingkeys/ports"
)

type ArchiveExpiredSigningKeysDeps struct {
	KeyStore ports.KeyStore
	Emit     func(spec.DomainEvent)
}

// ArchiveExpiredSigningKeys removes grace-expired keys from JWKS and records their lifecycle event.
func ArchiveExpiredSigningKeys(ctx context.Context, deps ArchiveExpiredSigningKeysDeps, now time.Time) ([]*signingdomain.SigningKey, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	keys, err := deps.KeyStore.ArchiveExpired(ctx, now)
	if err != nil {
		return nil, err
	}
	if deps.Emit != nil {
		for _, key := range keys {
			deps.Emit(&signingdomain.SigningKeyArchived{At: now, TenantID: key.TenantID, Kid: key.Kid, RetiredAt: key.RetiredAt, ExpiresAt: key.ExpiresAt, DisposedAt: now})
		}
	}
	return keys, nil
}
