package usecases

import (
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func parDepsWithResourceServer(servers ...*domain.McpResourceServer) (PARDeps, *oauth2memory.OAuth2ClientRepository) {
	clientRepo := oauth2memory.NewClientRepository()
	deps := PARDeps{
		ClientRepo:            clientRepo,
		Store:                 oauth2memory.NewPARStore(),
		AuthzDetailTypeRepo:   oauth2memory.NewAuthorizationDetailTypeRepository(),
		McpResourceServerRepo: newFakeMcpResourceServerRepo(servers...),
	}
	return deps, clientRepo
}

func TestPushAuthorizationRequest_unregisteredResource_rejectedFailClosed(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	deps, clientRepo := parDepsWithResourceServer()
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "client-1",
		RedirectURIs: []string{"https://example.com/cb"},
		GrantTypes:   []spec.GrantType{spec.GrantAuthorizationCode},
	})

	_, err := PushAuthorizationRequest(ctx, deps, PARInput{
		ClientID:   "client-1",
		Parameters: map[string]string{"response_type": "code"},
		Resource:   []string{"https://mcp.example.com/unknown"},
	}, time.Now().UTC())
	assertOAuthError(t, err, "invalid_target")
}

func TestPushAuthorizationRequest_registeredResource_accepted(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	rs := &domain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"mcp.read"}, State: domain.McpResourceServerActive,
	}
	deps, clientRepo := parDepsWithResourceServer(rs)
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "client-1",
		RedirectURIs: []string{"https://example.com/cb"},
		GrantTypes:   []spec.GrantType{spec.GrantAuthorizationCode},
	})

	res, err := PushAuthorizationRequest(ctx, deps, PARInput{
		ClientID:   "client-1",
		Parameters: map[string]string{"response_type": "code", "scope": "mcp.read"},
		Resource:   []string{"https://mcp.example.com/tools"},
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.RequestURI == "" {
		t.Fatal("expected request_uri to be issued")
	}
}
