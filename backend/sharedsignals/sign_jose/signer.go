// Package sign_jose implements ports.SecurityEventTokenSigner: PS256 JWT
// signing for outbound Security Event Tokens, reusing SigningKeys' key
// rotation/JWKS instead of introducing separate SET key material (
// decisions 3/7). This is an adapters-layer package (wraps
// shared/security/tokens_jose, a Clean Architecture "adapters" module) so
// sharedsignals/usecases can depend on the ports.SecurityEventTokenSigner
// port without crossing the use_cases -> adapters layering rule.
package sign_jose

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type Signer struct {
	KeyStore signingports.KeyStore
}

var _ ssports.SecurityEventTokenSigner = (*Signer)(nil)

func (s *Signer) Sign(ctx context.Context, tenantID string, claims map[string]any) (string, error) {
	ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: tenantID}, "", "")
	key, err := s.KeyStore.GetActiveKey(ctx)
	if err != nil {
		return "", err
	}
	return tokens_jose.SignPS256(key, map[string]string{"typ": "secevent+jwt"}, claims)
}
