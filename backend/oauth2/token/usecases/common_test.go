package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func tenantContext(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{
		ID: id, DisplayName: id, Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}, "https://idp.example/realms/"+id, "/realms/"+id)
}

type fakeMcpResourceServerRepo struct {
	byResource map[string]*domain.McpResourceServer
}

func newFakeMcpResourceServerRepo(servers ...*domain.McpResourceServer) *fakeMcpResourceServerRepo {
	r := &fakeMcpResourceServerRepo{byResource: map[string]*domain.McpResourceServer{}}
	for _, server := range servers {
		r.byResource[server.TenantID+"|"+server.Resource] = server
	}
	return r
}

func (r *fakeMcpResourceServerRepo) ListByTenant(context.Context, string) ([]*domain.McpResourceServer, error) {
	return nil, nil
}

func (r *fakeMcpResourceServerRepo) FindByID(context.Context, string, string) (*domain.McpResourceServer, error) {
	return nil, nil //nolint:nilnil // unused by these resource-indicator tests
}

func (r *fakeMcpResourceServerRepo) FindByResource(_ context.Context, tenantID, resource string) (*domain.McpResourceServer, error) {
	return r.byResource[tenantID+"|"+resource], nil
}

func (r *fakeMcpResourceServerRepo) Save(context.Context, *domain.McpResourceServer) error {
	return nil
}
func (r *fakeMcpResourceServerRepo) Delete(context.Context, string, string) error { return nil }

var _ ports.McpResourceServerRepository = (*fakeMcpResourceServerRepo)(nil)

func assertOAuthError(t *testing.T, err error, code string) {
	t.Helper()
	var oauthErr *OAuthError
	if !errors.As(err, &oauthErr) || oauthErr.Code != code {
		t.Fatalf("error = %#v, want OAuth code %s", err, code)
	}
}

func seedPaymentType(repo *oauth2memory.AuthorizationDetailTypeRepository) {
	repo.Seed(&domain.AuthorizationDetailType{
		TenantID: tenancydomain.DefaultTenantID, Type: "payment_initiation", State: domain.DetailTypeEnabled,
		Schema: domain.AuthorizationDetailsSchema{Rules: []domain.AuthorizationDetailFieldRule{
			{Name: "actions", Semantics: domain.DetailFieldSet, Required: true, Allowed: []string{"initiate", "status"}},
			{Name: "instructedAmount", Semantics: domain.DetailFieldAtMost, Required: true},
		}},
		DisplayTemplate: "最大 {instructedAmount} まで", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
}

func paymentDetail(amount float64, actions ...string) spec.AuthorizationDetail {
	return spec.AuthorizationDetail{
		Type: "payment_initiation", Actions: actions,
		Fields: map[string]any{"instructedAmount": amount},
	}
}
