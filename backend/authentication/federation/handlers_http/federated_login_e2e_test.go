package handlers_http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/federation/handlers_http"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type completingFederationDriver struct {
	state string
}

func (d *completingFederationDriver) Start(_ federationdomain.IdentityProviderConnection, attempt federationdomain.FederatedLoginAttempt, _ string, _ time.Time) (string, error) {
	d.state = attempt.State
	return "https://idp.example/authorize", nil
}

func (*completingFederationDriver) Complete(context.Context, federationdomain.IdentityProviderConnection, federationdomain.FederatedLoginAttempt, string, string, time.Time) (federationdomain.NormalizedClaims, error) {
	return federationdomain.NormalizedClaims{Subject: "external-user", Username: "alice@example.com"}, nil
}

// REQ-AUTHENTICATION-001: 外部 IdP の callback を正式入口から完了し、既存の相関先 User に対するログイン session を発行する。
func TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001(t *testing.T) {
	now := time.Now().UTC()
	repos := federationmemory.NewRepositories()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "user-alice", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice@example.com", PasswordHash: "unused",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	})
	connection := &federationdomain.IdentityProviderConnection{
		ID: "provider", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client", AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint: "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping: federationdomain.ClaimMapping{Subject: "sub", Username: "email"}, LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	if err := repos.Identities.Create(context.Background(), &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: connection.ID, ExternalSubject: "external-user",
		LocalUserID: "user-alice", LinkedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	driver := &completingFederationDriver{}
	sessions := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	e := echo.New()
	federationhttp.RegisterRoutes(e.Group(""), federationhttp.Deps{
		Broker: federationusecases.BrokerDeps{
			Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts,
			Users: users, Sessions: sessions,
			Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{federationdomain.ProtocolOIDC: driver},
		},
		Auth: httpdeps.Deps{Deps: support.Deps{Issuer: "http://idp.test"}},
	})

	start := httptest.NewRequest(http.MethodGet, "/api/auth/federation/start?provider_id=provider", http.NoBody)
	started := httptest.NewRecorder()
	e.ServeHTTP(started, start)
	if started.Code != http.StatusSeeOther || driver.state == "" {
		t.Fatalf("start status=%d state=%q body=%s", started.Code, driver.state, started.Body.String())
	}
	callback := httptest.NewRequest(http.MethodGet, "/api/auth/federation/oidc/callback?state="+url.QueryEscape(driver.state)+"&code=verified", http.NoBody)
	completed := httptest.NewRecorder()
	e.ServeHTTP(completed, callback)
	if completed.Code != http.StatusSeeOther || completed.Header().Get("Location") != "/authorize/resume" {
		t.Fatalf("callback status=%d location=%q body=%s", completed.Code, completed.Header().Get("Location"), completed.Body.String())
	}
	cookies := completed.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value == "" {
		t.Fatalf("session cookies=%+v", cookies)
	}
	headers := http.Header{"Cookie": []string{cookies[0].String()}}
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	authn, err := sessions.Resolve(ctx, authdomain.HTTPHeadersAdapter{H: headers})
	if err != nil || authn == nil || authn.UserID != "user-alice" {
		t.Fatalf("authn=%+v err=%v", authn, err)
	}
}
