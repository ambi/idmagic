package handlers_http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/federation/handlers_http"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type driverStub struct{}

func (driverStub) Start(
	_ federationdomain.IdentityProviderConnection,
	_ federationdomain.FederatedLoginAttempt,
	_ string,
	_ time.Time,
) (string, error) {
	return "https://idp.example/authorize", nil
}

func (driverStub) Complete(
	context.Context,
	federationdomain.IdentityProviderConnection,
	federationdomain.FederatedLoginAttempt,
	string,
	string,
	time.Time,
) (federationdomain.NormalizedClaims, error) {
	return federationdomain.NormalizedClaims{}, nil
}

func TestPublicProviderDiscoveryOmitsTrustAndSecret(t *testing.T) {
	e, repos := newServer(t)
	now := time.Now().UTC()
	if err := repos.Connections.Save(context.Background(), &federationdomain.IdentityProviderConnection{
		ID: "oidc", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client", SecretReference: "env:SECRET",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/federation/providers", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if !strings.Contains(body, `"display_name":"Workforce"`) ||
		strings.Contains(body, "SECRET") || strings.Contains(body, "token_endpoint") {
		t.Fatalf("unsafe discovery response=%s", body)
	}
}

func TestStartFederatedLoginRedirectsOnlyThroughSavedProvider(t *testing.T) {
	e, repos := newServer(t)
	now := time.Now().UTC()
	connection := &federationdomain.IdentityProviderConnection{
		ID: "oidc", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/auth/federation/start?provider_id=oidc", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "https://idp.example/authorize" {
		t.Fatalf("status=%d location=%q body=%s", response.Code, response.Header().Get("Location"), response.Body.String())
	}
}

func newServer(t *testing.T) (*echo.Echo, federationmemory.Repositories) {
	t.Helper()
	repos := federationmemory.NewRepositories()
	e := echo.New()
	federationhttp.RegisterRoutes(e.Group(""), federationhttp.Deps{
		Broker: federationusecases.BrokerDeps{
			Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts,
			Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
				federationdomain.ProtocolOIDC: driverStub{},
			},
		},
	})
	return e, repos
}
