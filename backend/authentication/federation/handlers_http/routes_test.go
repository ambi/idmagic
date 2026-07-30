package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/federation/handlers_http"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/federation/protocol_oidc"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
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

// newAdminServer wires the same routes as newServer plus a working admin auth stack
// (DemoHeaderResolver keyed by X-Demo-Sub, and a matching CSRF cookie/header pair) so admin
// CRUD/lifecycle/test handlers can be exercised end-to-end. oidcClient may be nil.
func newAdminServer(t *testing.T, oidcClient *oidcprotocol.Client) (*echo.Echo, federationmemory.Repositories) {
	t.Helper()
	repos := federationmemory.NewRepositories()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	if err := users.Save(context.Background(), &userdomain.User{
		ID: "admin-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	federationhttp.RegisterRoutes(e.Group(""), federationhttp.Deps{
		Broker: federationusecases.BrokerDeps{
			Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts,
			Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
				federationdomain.ProtocolOIDC: driverStub{},
			},
		},
		Auth: httpdeps.Deps{
			Deps: support.Deps{Issuer: "https://idp.test"},
			Authenticator: &support.Authenticator{
				UserRepo:      users,
				AuthnResolver: authusecases.DemoHeaderResolver{},
			},
		},
		OIDC: oidcClient,
	})
	return e, repos
}

const adminCSRFToken = "test-csrf-token"

func adminRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("X-Demo-Sub", "admin-1")
	request.Header.Set("Origin", "https://idp.test")
	request.Header.Set("X-Csrf-Token", adminCSRFToken)
	request.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: adminCSRFToken})
	return request
}

// RED (interface: CreateIdentityProviderConnection, ADR-149): a newly created connection
// starts Disabled, not the removed Draft status.
func TestCreateAdminInitialStatusIsDisabled(t *testing.T) {
	e, _ := newAdminServer(t, nil)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPost, "/api/admin/identity-providers", map[string]any{
		"display_name": "Google", "protocol": "oidc", "issuer": "https://accounts.example.com",
		"client_id": "client-1", "authorization_endpoint": "https://accounts.example.com/authorize",
		"token_endpoint": "https://accounts.example.com/token", "jwks_uri": "https://accounts.example.com/jwks",
		"claim_mapping":  map[string]string{"subject": "sub", "username": "email"},
		"linking_policy": "none",
	}))
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var created federationdomain.IdentityProviderConnection
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Status != federationdomain.ConnectionDisabled {
		t.Fatalf("status=%q, want disabled", created.Status)
	}
}

// RED (interface: UpdateIdentityProviderConnection, ADR-149 §編集時の自動デグレード):
// updating only display_name keeps an Active connection Active; changing the issuer degrades it.
func TestUpdateAdminDegradesOnlyOnTrustSourceChange(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	now := time.Now().UTC()
	connection := &federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token", JWKSURI: "https://accounts.example.com/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt:     now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{
		"display_name": "Google Workspace", "protocol": "oidc", "issuer": connection.Issuer,
		"client_id": connection.ClientID, "authorization_endpoint": connection.AuthorizationEndpoint,
		"token_endpoint": connection.TokenEndpoint, "jwks_uri": connection.JWKSURI,
		"claim_mapping": map[string]string{"subject": "sub", "username": "email"}, "linking_policy": "none",
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPut, "/api/admin/identity-providers/google", body))
	var updated federationdomain.IdentityProviderConnection
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatalf("status=%d body=%s err=%v", response.Code, response.Body.String(), err)
	}
	if updated.Status != federationdomain.ConnectionActive {
		t.Fatalf("status=%q after display_name-only change, want active preserved", updated.Status)
	}

	body["issuer"] = "https://other.example.com"
	response = httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPut, "/api/admin/identity-providers/google", body))
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatalf("status=%d body=%s err=%v", response.Code, response.Body.String(), err)
	}
	if updated.Status != federationdomain.ConnectionDisabled {
		t.Fatalf("status=%q after issuer change, want disabled", updated.Status)
	}
}

// RED (interface: DeleteIdentityProviderConnection, ADR-149): deletion no longer requires
// Disabled first; an Active connection can be deleted directly.
func TestDeleteAdminSucceedsForActiveConnection(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	now := time.Now().UTC()
	if err := repos.Connections.Save(context.Background(), &federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token", JWKSURI: "https://accounts.example.com/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt:     now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodDelete, "/api/admin/identity-providers/google", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s, want 204 without a status guard", response.Code, response.Body.String())
	}
}

// RED (interface: TestIdentityProviderConnection, ADR-150): the test action returns a
// structured success/failure result reflecting real reachability, not a canned "valid" string.
func TestTestAdminReportsStructuredReachabilityResult(t *testing.T) {
	oidcClient := oidcprotocol.Client{HTTPClient: &http.Client{Transport: fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == "https://accounts.example.com/jwks" {
			return nil, context.DeadlineExceeded
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
	})}}
	e, repos := newAdminServer(t, &oidcClient)
	now := time.Now().UTC()
	if err := repos.Connections.Save(context.Background(), &federationdomain.IdentityProviderConnection{
		ID: "google", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Google",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionDisabled,
		Issuer: "https://accounts.example.com", ClientID: "client-1",
		AuthorizationEndpoint: "https://accounts.example.com/authorize",
		TokenEndpoint:         "https://accounts.example.com/token", JWKSURI: "https://accounts.example.com/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt:     now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPost, "/api/admin/identity-providers/google/test", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Result struct {
			Success  bool     `json:"success"`
			Failures []string `json:"failures"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Result.Success || len(body.Result.Failures) == 0 {
		t.Fatalf("result=%+v, want success=false with a JWKS unreachability failure", body.Result)
	}
}

type fakeRoundTripper func(*http.Request) (*http.Response, error)

func (f fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

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
