package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func tenantContext() context.Context {
	id := tenancydomain.DefaultTenantID
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{
		ID: id, DisplayName: id, Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}, "https://idp.example/realms/"+id, "/realms/"+id)
}

type fakeMcpResourceServerRepo struct {
	byResource map[string]*oauthdomain.McpResourceServer
}

func newFakeMcpResourceServerRepo(servers ...*oauthdomain.McpResourceServer) *fakeMcpResourceServerRepo {
	r := &fakeMcpResourceServerRepo{byResource: map[string]*oauthdomain.McpResourceServer{}}
	for _, server := range servers {
		r.byResource[server.TenantID+"|"+server.Resource] = server
	}
	return r
}

func (r *fakeMcpResourceServerRepo) ListAll(context.Context, string) ([]*oauthdomain.McpResourceServer, error) {
	return nil, nil
}

func (r *fakeMcpResourceServerRepo) FindByID(context.Context, string, string) (*oauthdomain.McpResourceServer, error) {
	return nil, nil //nolint:nilnil // unused by these resource-indicator tests
}

func (r *fakeMcpResourceServerRepo) FindByResource(_ context.Context, tenantID, resource string) (*oauthdomain.McpResourceServer, error) {
	return r.byResource[tenantID+"|"+resource], nil
}

func (r *fakeMcpResourceServerRepo) FindByClientID(context.Context, string, string) (*oauthdomain.McpResourceServer, error) {
	return nil, nil //nolint:nilnil // 束縛のない登録を表す。
}

func (r *fakeMcpResourceServerRepo) Save(context.Context, *oauthdomain.McpResourceServer) error {
	return nil
}
func (r *fakeMcpResourceServerRepo) Delete(context.Context, string, string) error { return nil }
func (r *fakeMcpResourceServerRepo) SaveIfUnchanged(context.Context, *oauthdomain.McpResourceServer, time.Time) (bool, error) {
	return false, nil
}

var _ oauthports.McpResourceServerRepository = (*fakeMcpResourceServerRepo)(nil)

func assertOAuthError(t *testing.T, err error, code string) {
	t.Helper()
	var oauthErr *OAuthError
	if !errors.As(err, &oauthErr) || oauthErr.Code != code {
		t.Fatalf("error = %#v, want OAuth code %s", err, code)
	}
}
