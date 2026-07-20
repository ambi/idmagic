package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func tenantContext(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{
		ID: id, DisplayName: id, Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}, "https://idp.example/realms/"+id, "/realms/"+id)
}

func assertOAuthError(t *testing.T, err error, code string) {
	t.Helper()
	var oauthErr *OAuthError
	if !errors.As(err, &oauthErr) || oauthErr.Code != code {
		t.Fatalf("error = %#v, want OAuth code %s", err, code)
	}
}

type fakeTokenIssuer struct {
	lastAccessTokenInput ports.AccessTokenInput
}

func (f *fakeTokenIssuer) SignAccessToken(_ context.Context, in ports.AccessTokenInput) (string, string, error) {
	f.lastAccessTokenInput = in
	return "access-token", "jti-1", nil
}

func (*fakeTokenIssuer) SignIDToken(context.Context, ports.IDTokenInput) (string, error) {
	return "id-token", nil
}
func (*fakeTokenIssuer) AccessTokenTTLSeconds() int { return 600 }
func (*fakeTokenIssuer) IDTokenTTLSeconds() int     { return 3600 }
