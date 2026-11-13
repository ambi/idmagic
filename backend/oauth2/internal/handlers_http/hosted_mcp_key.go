package handlers_http

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
)

func (d Deps) mcpGatewayKey(ctx context.Context, serverID string) (*signingdomain.SigningKey, error) {
	ctx = signingports.WithKeyScope(signingports.WithKeyUsage(ctx, signingdomain.KeyUsageMcpGatewayDPoP), serverID)
	if d.KeyStore == nil || !d.KeyStore.Healthy(ctx) {
		return nil, errors.New("gateway key provider is unavailable")
	}
	key, err := d.KeyStore.GetActiveKey(ctx)
	if err != nil {
		return nil, err
	}
	if key == nil || key.Usage != signingdomain.KeyUsageMcpGatewayDPoP || key.ScopeID != serverID {
		return nil, errors.New("gateway key has the wrong purpose or scope")
	}
	if _, err := tokens_jose.ValidatePublicSigningJWK(key.PublicJWK); err != nil {
		return nil, err
	}
	return key, nil
}
