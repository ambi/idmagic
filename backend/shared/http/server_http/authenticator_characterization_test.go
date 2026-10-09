package server_http_test

// 管理 API と Bearer の認証で、規範 ID を引くテストが固定していない分岐を、組み立てた
// router の HTTP 応答として固定する特性化テストである。
//
// `server_http.Register` を境界にするのは、認証の部品をどこへ置いても、この入口と
// `Deps` の形は変わらないからである。部品の型を直接組み立てるテストは、部品を動かすと
// 組み立てから書き直すことになり、変化を検出する役に立たない。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken"
	apitokenmemory "github.com/ambi/idmagic/backend/apitoken/db_memory"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	"github.com/ambi/idmagic/backend/authentication"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	consentmemory "github.com/ambi/idmagic/backend/oauth2/consent/db_memory"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/consent/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	characterizationIssuer = "https://idp.example"
	// 管理 API の経路のうち、RequireAdmin の後に同意の保存先を引くもの。保存先に
	// 同意がなければ 404 consent_not_found を返すので、認可を通ったかを応答で読める。
	adminConsentPath = "/realms/default/api/admin/v1/consents/alice/client-1"
	// RequireControlPlaneUser を使う制御面の経路。
	controlPlanePath = "/realms/default/api/admin/v1/tenants"
	// ResolveAdminActor を使う署名鍵の一覧。
	adminKeysPath = "/realms/default/api/admin/v1/keys"
	// RequireAdministrator を使う API トークンの一覧。
	apiTokensPath = "/realms/default/api/admin/v1/api-tokens"
)

var errCharacterizedStore = errors.New("characterized store failure")

// faultyUsers は FindBySub の n 回目以降を失敗または不在にする。認証の経路が同じ利用者を
// 何回引くかは、どの照合で失敗が表に出るかを決めるので、回数で指定する。
type faultyUsers struct {
	*usermemory.UserRepository
	calls       int
	failFrom    int
	missingFrom int
}

func (r *faultyUsers) FindBySub(ctx context.Context, sub string) (*userdomain.User, error) {
	r.calls++
	if r.failFrom > 0 && r.calls >= r.failFrom {
		return nil, errCharacterizedStore
	}
	if r.missingFrom > 0 && r.calls >= r.missingFrom {
		return nil, nil //nolint:nilnil // 利用者の保存先は不在を (nil, nil) で表す。
	}
	return r.UserRepository.FindBySub(ctx, sub)
}

type faultyGroups struct {
	*groupmemory.GroupRepository
	listErr error
}

func (r *faultyGroups) ListGroupsByUser(ctx context.Context, tenantID, userID string) ([]*groupdomain.Group, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.GroupRepository.ListGroupsByUser(ctx, tenantID, userID)
}

type faultyConsents struct {
	*consentmemory.ConsentRepository
	findErr error
}

func (r *faultyConsents) Find(ctx context.Context, tenantID, sub, clientID string) (*consentdomain.Consent, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.ConsentRepository.Find(ctx, tenantID, sub, clientID)
}

type scriptedIntrospector struct {
	result *oauthports.IntrospectionResult
	err    error
}

func (s scriptedIntrospector) IntrospectAccessToken(context.Context, string) (*oauthports.IntrospectionResult, error) {
	return s.result, s.err
}

type failingDenylist struct{}

func (failingDenylist) Add(context.Context, string, time.Time) error { return nil }

func (failingDenylist) IsRevoked(context.Context, string) (bool, error) {
	return false, errCharacterizedStore
}

type authCharacterization struct {
	deps         httpadapter.Deps
	users        *faultyUsers
	groups       *faultyGroups
	consents     *faultyConsents
	sessions     *sessionusecases.SessionManager
	sessionStore *sessionmemory.SessionStore
	tenants      *tenancymemory.TenantRepository
	apiTokens    *apitokenusecases.Service
	echo         *echo.Echo
}

func newAuthCharacterization(t *testing.T) *authCharacterization {
	t.Helper()
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensjose.NewJWTSigner(characterizationIssuer, keyStore)
	apiTokenRepo := apitokenmemory.NewRepository()
	sessionStore := sessionmemory.NewSessionStore()
	sessions := sessionusecases.NewSessionManager(sessionStore)
	h := &authCharacterization{
		users:        &faultyUsers{UserRepository: usermemory.NewUserRepository()},
		groups:       &faultyGroups{GroupRepository: groupmemory.NewGroupRepository()},
		consents:     &faultyConsents{ConsentRepository: consentmemory.NewConsentRepository()},
		sessions:     sessions,
		sessionStore: sessionStore,
		tenants:      tenants,
		apiTokens: apitokenusecases.New(apiTokenRepo,
			apitokenusecases.WithTokenIssuer(signer), apitokenusecases.WithTokenIntrospector(signer)),
	}
	h.deps = httpadapter.Deps{
		Issuer: characterizationIssuer, Contract: spec.CurrentRuntimeContract(),
		TenantRepo:     tenants,
		IdManagement:   idmanagement.Module{UserRepo: h.users, GroupRepo: h.groups},
		Authentication: authentication.Module{SessionManager: sessions, AuthnResolver: sessions},
		SigningKeys:    signingkeys.Module{KeyStore: keyStore},
		OAuth2: oauth2.Module{
			TokenIssuer: signer, TokenIntrospector: signer, ConsentRepo: h.consents,
		},
		ApiTokens: apitoken.Module{Repo: apiTokenRepo, TokenIssuer: signer, TokenIntrospector: signer},
	}
	return h
}

func (h *authCharacterization) seedUser(t *testing.T, id string, active bool, roles ...string) {
	t.Helper()
	now := time.Now().UTC()
	status := idmdomain.UserStatusActive
	if !active {
		status = idmdomain.UserStatusDisabled
	}
	h.users.Seed(&userdomain.User{
		ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: id, PasswordHash: "unused",
		Roles: roles, Lifecycle: userdomain.UserLifecycle{Status: status, StatusChangedAt: &now},
		CreatedAt: now, UpdatedAt: now,
	})
}

func (h *authCharacterization) grantGroupRole(t *testing.T, userID, role string) {
	t.Helper()
	now := time.Now().UTC()
	group := &groupdomain.Group{
		ID: "group-" + role, TenantID: tenancydomain.DefaultTenantID, Name: role,
		Roles: []string{role}, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.groups.Save(context.Background(), group); err != nil {
		t.Fatal(err)
	}
	if _, err := h.groups.AddMember(context.Background(), &groupdomain.GroupMember{
		GroupID: group.ID, UserID: userID, Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
}

func (h *authCharacterization) realmContext(t *testing.T) context.Context {
	t.Helper()
	tenant, err := h.tenants.FindByRealm(context.Background(), tenancydomain.DefaultRealm)
	if err != nil || tenant == nil {
		t.Fatalf("default realm: tenant=%v err=%v", tenant, err)
	}
	return tenancy.WithTenant(context.Background(), tenant, characterizationIssuer+"/realms/default", "/realms/default")
}

func (h *authCharacterization) sessionCookie(t *testing.T, userID string) *http.Cookie {
	t.Helper()
	authn, err := h.sessions.Create(h.realmContext(t), userID, []string{"pwd"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: sessionusecases.SessionCookie, Value: authn.SessionID}
}

type characterizedResponse struct {
	status    int
	problem   string
	challenge string
}

func (h *authCharacterization) get(t *testing.T, path string, decorate func(*http.Request)) characterizedResponse {
	t.Helper()
	if h.echo == nil {
		h.echo = echo.New()
		httpadapter.Register(h.echo, h.deps)
	}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, characterizationIssuer+path, http.NoBody)
	if decorate != nil {
		decorate(request)
	}
	recorder := httptest.NewRecorder()
	h.echo.ServeHTTP(recorder, request)
	var body struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return characterizedResponse{
		status: recorder.Code, problem: body.Type, challenge: recorder.Header().Get("WWW-Authenticate"),
	}
}

func withCookie(cookie *http.Cookie) func(*http.Request) {
	return func(request *http.Request) { request.AddCookie(cookie) }
}

func withBearer(token string) func(*http.Request) {
	return func(request *http.Request) { request.Header.Set("Authorization", "Bearer "+token) }
}

// 観測した応答。consentLookedUp は認可を通って同意の保存先まで届いたことを表す。
var (
	authenticationRequired = characterizedResponse{status: 401, problem: "urn:idmagic:error:authentication_required"}
	accessDenied           = characterizedResponse{status: 403, problem: "urn:idmagic:error:access_denied"}
	consentLookedUp        = characterizedResponse{status: 404, problem: "urn:idmagic:error:consent_not_found"}
	internalServerError    = characterizedResponse{status: 500, problem: "urn:idmagic:error:internal_server_error"}
	invalidToken           = characterizedResponse{
		status: 401, problem: "urn:idmagic:error:invalid_token",
		challenge: `Bearer error="invalid_token", resource_metadata="https://idp.example/realms/default/.well-known/oauth-protected-resource"`,
	}
	listed = characterizedResponse{status: 200}
)

func assertCharacterized(t *testing.T, got, want characterizedResponse) {
	t.Helper()
	if got != want {
		t.Fatalf("response=%+v, want %+v", got, want)
	}
}

func TestCharacterizeAdminAccessThroughTheSessionCookie(t *testing.T) {
	t.Run("no credential", func(t *testing.T) {
		h := newAuthCharacterization(t)
		assertCharacterized(t, h.get(t, adminConsentPath, nil), authenticationRequired)
	})
	t.Run("user without the admin role", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "user")
		assertCharacterized(t, h.get(t, adminConsentPath, withCookie(h.sessionCookie(t, "alice"))),
			accessDenied)
	})
	t.Run("admin role granted through a group", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "user")
		h.grantGroupRole(t, "alice", "admin")
		assertCharacterized(t, h.get(t, adminConsentPath, withCookie(h.sessionCookie(t, "alice"))),
			consentLookedUp)
	})
	t.Run("group lookup failure falls back to the direct roles", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "user")
		h.grantGroupRole(t, "alice", "admin")
		h.groups.listErr = errCharacterizedStore
		assertCharacterized(t, h.get(t, adminConsentPath, withCookie(h.sessionCookie(t, "alice"))),
			accessDenied)
	})
	t.Run("inactive user ends the session", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", false, "admin")
		cookie := h.sessionCookie(t, "alice")
		assertCharacterized(t, h.get(t, adminConsentPath, withCookie(cookie)), authenticationRequired)
		session, err := h.sessionStore.Find(h.realmContext(t), cookie.Value)
		if err != nil || session != nil {
			t.Fatalf("session=%+v err=%v, want the session ended", session, err)
		}
	})
	t.Run("first user lookup fails", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		h.users.failFrom = 1
		assertCharacterized(t, h.get(t, adminConsentPath, withCookie(h.sessionCookie(t, "alice"))),
			internalServerError)
	})
	t.Run("second user lookup fails", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		h.users.failFrom = 2
		assertCharacterized(t, h.get(t, adminConsentPath, withCookie(h.sessionCookie(t, "alice"))),
			internalServerError)
	})
	t.Run("consent lookup failure after authorization", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		h.consents.findErr = errCharacterizedStore
		assertCharacterized(t, h.get(t, adminConsentPath, withCookie(h.sessionCookie(t, "alice"))),
			internalServerError)
	})
}

func TestCharacterizeAdminAccessThroughABearerToken(t *testing.T) {
	adminToken := &oauthports.IntrospectionResult{
		Active: true, JTI: "jti-1", ClientID: "client-1", Sub: "alice", Scope: "idmagic.admin",
		Iat: time.Now().Unix(),
	}
	t.Run("no introspector", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		h.deps.OAuth2.TokenIntrospector = nil
		h.deps.ApiTokens.TokenIntrospector = nil
		assertCharacterized(t, h.get(t, adminConsentPath, withBearer("opaque")), authenticationRequired)
	})
	t.Run("introspection fails", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.deps.OAuth2.TokenIntrospector = scriptedIntrospector{err: errCharacterizedStore}
		assertCharacterized(t, h.get(t, adminConsentPath, withBearer("opaque")),
			internalServerError)
	})
	t.Run("inactive token", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.deps.OAuth2.TokenIntrospector = scriptedIntrospector{result: &oauthports.IntrospectionResult{}}
		assertCharacterized(t, h.get(t, adminConsentPath, withBearer("opaque")), invalidToken)
	})
	t.Run("revocation lookup fails", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		h.deps.OAuth2.TokenIntrospector = scriptedIntrospector{result: adminToken}
		h.deps.OAuth2.AccessTokenDenylist = failingDenylist{}
		assertCharacterized(t, h.get(t, adminConsentPath, withBearer("opaque")),
			internalServerError)
	})
	t.Run("admin token", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		h.deps.OAuth2.TokenIntrospector = scriptedIntrospector{result: adminToken}
		assertCharacterized(t, h.get(t, adminConsentPath, withBearer("opaque")),
			consentLookedUp)
	})
	// 管理発行のトークンは、イントロスペクションの主体とクライアントが API トークンの
	// 記録と一致したときだけ通る。一致する場合を対照に置き、拒否が照合によることを示す。
	managedToken := func(t *testing.T, h *authCharacterization, sub string) string {
		t.Helper()
		literal, metadata, err := h.apiTokens.Issue(h.realmContext(t), tenancydomain.DefaultTenantID,
			"alice", "characterization", []string{"consents:read"}, 1, "")
		if err != nil {
			t.Fatal(err)
		}
		h.deps.OAuth2.TokenIntrospector = scriptedIntrospector{result: &oauthports.IntrospectionResult{
			Active: true, ClientID: metadata.ClientID, Sub: sub, Managed: true, Iat: time.Now().Unix(),
		}}
		return literal
	}
	t.Run("managed token issued to the subject", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		literal := managedToken(t, h, "alice")
		assertCharacterized(t, h.get(t, adminConsentPath, withBearer(literal)), consentLookedUp)
	})
	t.Run("managed token issued to another subject", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "admin")
		literal := managedToken(t, h, "mallory")
		assertCharacterized(t, h.get(t, adminConsentPath, withBearer(literal)), invalidToken)
	})
	t.Run("managed token on the control plane", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "alice", true, "system_admin")
		literal := managedToken(t, h, "alice")
		assertCharacterized(t, h.get(t, controlPlanePath, withBearer(literal)), invalidToken)
	})
}

func TestCharacterizeControlPlaneAccess(t *testing.T) {
	t.Run("first user lookup fails", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "root", true, "system_admin")
		h.users.failFrom = 1
		assertCharacterized(t, h.get(t, controlPlanePath, withCookie(h.sessionCookie(t, "root"))),
			internalServerError)
	})
	t.Run("second user lookup fails", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "root", true, "system_admin")
		h.users.failFrom = 2
		assertCharacterized(t, h.get(t, controlPlanePath, withCookie(h.sessionCookie(t, "root"))),
			internalServerError)
	})
	t.Run("user disappears before the second lookup", func(t *testing.T) {
		h := newAuthCharacterization(t)
		h.seedUser(t, "root", true, "system_admin")
		h.users.missingFrom = 2
		assertCharacterized(t, h.get(t, controlPlanePath, withCookie(h.sessionCookie(t, "root"))),
			accessDenied)
	})
}

func TestCharacterizeAdminActorResolution(t *testing.T) {
	for _, path := range []string{adminKeysPath, apiTokensPath} {
		t.Run(path+" first user lookup fails", func(t *testing.T) {
			h := newAuthCharacterization(t)
			h.seedUser(t, "alice", true, "admin")
			h.users.failFrom = 1
			assertCharacterized(t, h.get(t, path, withCookie(h.sessionCookie(t, "alice"))),
				internalServerError)
		})
		t.Run(path+" second user lookup fails", func(t *testing.T) {
			h := newAuthCharacterization(t)
			h.seedUser(t, "alice", true, "admin")
			h.users.failFrom = 2
			assertCharacterized(t, h.get(t, path, withCookie(h.sessionCookie(t, "alice"))),
				internalServerError)
		})
		t.Run(path+" user disappears before the second lookup", func(t *testing.T) {
			h := newAuthCharacterization(t)
			h.seedUser(t, "alice", true, "admin")
			h.users.missingFrom = 2
			assertCharacterized(t, h.get(t, path, withCookie(h.sessionCookie(t, "alice"))),
				accessDenied)
		})
		t.Run(path+" admin", func(t *testing.T) {
			h := newAuthCharacterization(t)
			h.seedUser(t, "alice", true, "admin")
			assertCharacterized(t, h.get(t, path, withCookie(h.sessionCookie(t, "alice"))),
				listed)
		})
	}
}
