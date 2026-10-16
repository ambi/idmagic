package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	httpdeps "github.com/ambi/idmagic/backend/authentication/internal/deps_http"
	federationmemory "github.com/ambi/idmagic/backend/authentication/internal/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/internal/federation/handlers_http"
	federationusecases "github.com/ambi/idmagic/backend/authentication/internal/federation/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/internal/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	"github.com/labstack/echo/v5"
)

type completingFederationDriver struct {
	state       string
	callbackURL string
	response    string
}

func (d *completingFederationDriver) Start(_ federationdomain.IdentityProviderConnection, attempt federationdomain.FederatedLoginAttempt, callbackURL string, _ time.Time) (string, error) {
	d.state = attempt.State
	d.callbackURL = callbackURL
	return "https://idp.example/authorize", nil
}

func (d *completingFederationDriver) Complete(_ context.Context, _ federationdomain.IdentityProviderConnection, _ federationdomain.FederatedLoginAttempt, response, _ string, _ time.Time) (federationdomain.NormalizedClaims, error) {
	d.response = response
	return federationdomain.NormalizedClaims{Subject: "external-user", Username: "alice@example.com"}, nil
}

//spec:covers REQ-AUTHENTICATION-001, EX-AUTHENTICATION-001-01: start が単回限りの試行として state を保存して上流へ遷移し、callback が既存の関連付けから同じローカル User を解決して `federated` の AMR を持つ LoginSession を発行することを固定する (初回の JIT 作成は TestCompleteJITRequiresPolicyAndProvisioner が持つ)。
//verify:operation StartFederatedLogin: OIDC・SAML の開始と callback の 303、session、認証情報および不正な callback の拒否
//verify:operation CompleteFederatedLogin: OIDC・SAML の開始と callback の 303、session、認証情報および不正な callback の拒否
//verify:operation CompleteSamlFederatedLogin: OIDC・SAML の開始と callback の 303、session、認証情報および不正な callback の拒否
func TestFederatedLoginPrimaryUseCase_REQ_AUTHENTICATION_001(t *testing.T) {
	for _, protocol := range []federationdomain.Protocol{federationdomain.ProtocolOIDC, federationdomain.ProtocolSAML} {
		t.Run(string(protocol), func(t *testing.T) {
			now := time.Now().UTC()
			repos := federationmemory.NewRepositories()
			users := idmanagement.NewMemoryModule().UserRepo
			_ = users.Save(context.Background(), &userdomain.User{
				ID: "user-alice", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice@example.com", PasswordHash: "unused",
				Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
			})
			connection := &federationdomain.IdentityProviderConnection{
				ID: "provider", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce",
				Protocol: protocol, Status: federationdomain.ConnectionActive,
				Issuer: "https://idp.example", ClientID: "client", AuthorizationEndpoint: "https://idp.example/auth",
				TokenEndpoint: "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
				SAMLEntityID: "https://idp.example", SAMLSSOURL: "https://idp.example/sso", SAMLSigningCertificates: []string{"protocol-driver-fixture"},
				ClaimMapping: federationdomain.ClaimMapping{Subject: "sub", Username: "email"}, LinkingPolicy: federationdomain.LinkingNone,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := repos.Connections.Save(testing_tenant.Default(context.Background()), connection); err != nil {
				t.Fatal(err)
			}
			if err := repos.Identities.Create(testing_tenant.Default(context.Background()), &federationdomain.FederatedIdentity{
				TenantID: tenancydomain.DefaultTenantID, ProviderID: connection.ID, ExternalSubject: "external-user",
				LocalUserID: "user-alice", LinkedAt: now,
			}); err != nil {
				t.Fatal(err)
			}
			driver := &completingFederationDriver{}
			sessions := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
			e := echo.New()
			federationhttp.RegisterRoutes(e.Group("", testing_tenant.ResolveDefault), federationhttp.Deps{
				Broker: federationusecases.BrokerDeps{
					Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts,
					Users: users, Sessions: sessions,
					Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{protocol: driver},
				},
				Auth: httpdeps.Deps{Deps: support.Deps{Issuer: "http://idp.test"}},
			})

			refusedStart := httptest.NewRecorder()
			e.ServeHTTP(refusedStart, httptest.NewRequest(http.MethodGet, "/api/auth/federation/start?provider_id=unknown", http.NoBody))
			federationBrowserProblem(t, refusedStart)
			if driver.state != "" {
				t.Fatal("refused start reached the upstream driver")
			}
			start := httptest.NewRequest(http.MethodGet, "/api/auth/federation/start?provider_id=provider", http.NoBody)
			started := httptest.NewRecorder()
			e.ServeHTTP(started, start)
			if started.Code != http.StatusSeeOther || driver.state == "" || started.Header().Get("Location") != "https://idp.example/authorize" || started.Body.Len() != 0 {
				t.Fatalf("start status=%d state=%q body=%s", started.Code, driver.state, started.Body.String())
			}
			callback := httptest.NewRequest(http.MethodGet, "/api/auth/federation/oidc/callback?state="+url.QueryEscape(driver.state)+"&code=verified", http.NoBody)
			callbackPath := "/api/auth/federation/oidc/callback"
			if protocol == federationdomain.ProtocolSAML {
				callbackPath = "/api/auth/federation/saml/callback"
				form := url.Values{"RelayState": {driver.state}, "SAMLResponse": {"verified"}}
				callback = httptest.NewRequest(http.MethodPost, callbackPath, strings.NewReader(form.Encode()))
				callback.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			if driver.callbackURL != "http://idp.test"+callbackPath {
				t.Fatalf("upstream callback=%q", driver.callbackURL)
			}
			refusedCallback := httptest.NewRecorder()
			e.ServeHTTP(refusedCallback, httptest.NewRequest(callback.Method, callbackPath, http.NoBody))
			federationBrowserProblem(t, refusedCallback)
			if driver.response != "" {
				t.Fatal("incomplete callback reached the upstream driver")
			}
			completed := httptest.NewRecorder()
			e.ServeHTTP(completed, callback)
			if completed.Code != http.StatusSeeOther || completed.Header().Get("Location") != "/authorize/resume" || completed.Body.Len() != 0 || driver.response != "verified" {
				t.Fatalf("callback status=%d location=%q body=%s", completed.Code, completed.Header().Get("Location"), completed.Body.String())
			}
			cookies := completed.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Value == "" {
				t.Fatalf("session cookies=%+v", cookies)
			}
			headers := http.Header{"Cookie": []string{cookies[0].String()}}
			ctx := tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
			authn, err := sessions.Resolve(ctx, authdomain.HTTPHeadersAdapter{H: headers})
			if err != nil || authn == nil || authn.UserID != "user-alice" {
				t.Fatalf("authn=%+v err=%v", authn, err)
			}
			//spec:covers RFC8176-AMR-VOCABULARY: 連合ログインが発行する `federated` が語彙の内側にあることを
			// 固定する。語彙の検査は保存の直前に掛かるので、`federated` が語彙から落ちれば callback
			// はここまで到達しない。宣言と実装が食い違っていた時期の帰結がこれであり、語彙を
			// 締める変更が正規の経路を止めていないことの対照でもある (wi-508)。
			if !slices.Equal(authn.AMR, []string{authdomain.AMRFederated}) {
				t.Fatalf("amr=%v, want [%s]", authn.AMR, authdomain.AMRFederated)
			}
			// 上流の IdP が何を検証したかはブローカーに分からないので、acr は mfa へ上がらない。
			if authn.ACR != "urn:idmagic:acr:pwd" {
				t.Fatalf("acr=%q, want urn:idmagic:acr:pwd", authn.ACR)
			}
		})
	}
}

func federationBrowserProblem(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusBadRequest || rec.Header().Get("Content-Type") != "application/problem+json" || rec.Header().Get("Location") != "" || rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("refusal status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["type"] != "urn:idmagic:error:invalid_request" || body["status"] != float64(http.StatusBadRequest) || body["detail"] == "" {
		t.Fatalf("problem=%v err=%v", body, err)
	}
}
