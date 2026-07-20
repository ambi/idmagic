package handlers_http_test

// SCL シナリオ "prompt=none で session 無し" / "prompt=login" / "prompt=consent" /
// "max_age を超えた前回認証では再認証を要求する" を handler 層で検証する。
// AuthnResolver の差し替えだけで再認証フローを観測する単純構成。

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type fakeAuthnResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (f *fakeAuthnResolver) Resolve(_ context.Context, _ authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return f.ctx, nil
}

const (
	authClientID           = "auth-client"
	authClientSec          = "auth-client-secret"
	authRedirectURI        = "https://app.example.com/cb"
	authFirstPartyClientID = "auth-client-fp"
)

func newAuthorizeTestServer(t *testing.T, authn *authdomain.AuthenticationContext, consent *domain.Consent) (*echo.Echo, *[]spec.DomainEvent) {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := usermemory.NewUserRepository()
	consentRepo := oauth2memory.NewConsentRepository()
	secretHash := domain.HashClientSecret(authClientSec)
	now := time.Now().UTC()
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: authClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{authRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	// first-party クライアント (ADR-061): consent をスキップする検証用。
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: authFirstPartyClientID, ClientType: spec.ClientPublic,
		RedirectURIs:             []string{authRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodNone,
		Scope:                    "openid profile idmagic.admin",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		FirstParty:               true,
		CreatedAt:                now,
	})
	if authn != nil {
		userRepo.Seed(&userdomain.User{
			ID: authn.UserID, PreferredUsername: "alice",
			TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
		})
	}
	if consent != nil {
		_ = consentRepo.Save(context.Background(), tenancydomain.DefaultTenantID, consent)
	}
	e := echo.New()
	emitted := &[]spec.DomainEvent{}
	deps := httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://test",
			Emit:   func(e spec.DomainEvent) { *emitted = append(*emitted, e) },
		},
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, ConsentRepo: consentRepo,
			RequestStore: oauth2memory.NewAuthorizationRequestStore(), CodeStore: oauth2memory.NewAuthorizationCodeStore(), PARStore: oauth2memory.NewPARStore(),
		},
		UserRepo: userRepo,
	}
	if authn != nil {
		deps.AuthnResolver = &fakeAuthnResolver{ctx: authn}
	}
	httpadapter.Register(e, deps)
	return e, emitted
}

func authorizeQuery(extra url.Values) url.Values {
	q := url.Values{
		"client_id":             {authClientID},
		"redirect_uri":          {authRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile"},
		"code_challenge":        {"abcdef0123456789abcdef0123456789abcdef0123ab"},
		"code_challenge_method": {"S256"},
	}
	maps.Copy(q, extra)
	return q
}

func runAuthorize(t *testing.T, e *echo.Echo, q url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/authorize?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAuthorizePromptNoneWithoutSessionReturnsLoginRequired(t *testing.T) {
	e, _ := newAuthorizeTestServer(t, nil, nil)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"prompt": {"none"}}))
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "error=login_required") ||
		!strings.Contains(rec.Header().Get("Location"), "iss=") {
		t.Fatalf("status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAuthorizePromptLoginForcesReauthentication(t *testing.T) {
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
	}
	e, _ := newAuthorizeTestServer(t, authn, nil)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"prompt": {"login"}}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.HasSuffix(loc, "/login") {
		t.Fatalf("redirect Location=%q, want /login", loc)
	}
}

func TestAuthorizeMaxAgeBeyondLastAuthForcesReauthentication(t *testing.T) {
	// auth_time が 1 時間前、max_age=60 → NeedsReauthentication=true。
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: time.Now().Add(-time.Hour).Unix(), AMR: []string{"pwd"},
	}
	e, _ := newAuthorizeTestServer(t, authn, nil)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"max_age": {"60"}}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.HasSuffix(loc, "/login") {
		t.Fatalf("redirect Location=%q, want /login", loc)
	}
}

func TestAuthorizePromptConsentBypassesExistingConsent(t *testing.T) {
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	// 既存 Consent。prompt=consent が無ければ即 issueCode に進む。
	consent := &domain.Consent{
		UserID: "user_alice", ClientID: authClientID,
		Scopes:    []string{"openid", "profile"},
		State:     domain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	e, _ := newAuthorizeTestServer(t, authn, consent)
	rec := runAuthorize(t, e, authorizeQuery(url.Values{"prompt": {"consent"}}))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d body=%s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); !strings.HasSuffix(loc, "/consent") {
		t.Fatalf("redirect Location=%q, want /consent", loc)
	}
}

// ADR-061: first-party クライアントは consent 画面をスキップし、同意レコードが
// 無くても即 authorization code を発行する (redirect_uri へ code 付きで 303)。
func TestAuthorizeFirstPartyClientSkipsConsent(t *testing.T) {
	now := time.Now().UTC()
	authn := &authdomain.AuthenticationContext{
		UserID: "user_alice", AuthTime: now.Unix(), AMR: []string{"pwd"},
	}
	e, emitted := newAuthorizeTestServer(t, authn, nil)
	q := authorizeQuery(url.Values{})
	q.Set("client_id", authFirstPartyClientID)
	q.Set("scope", "openid profile idmagic.admin")
	rec := runAuthorize(t, e, q)
	// 認可コード発行は redirect_uri へ 302 (Found)。/consent への 303 ではない。
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d body=%s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if strings.Contains(loc, "/consent") {
		t.Fatalf("first-party client must skip consent, got Location=%q", loc)
	}
	if !strings.HasPrefix(loc, authRedirectURI) || !strings.Contains(loc, "code=") {
		t.Fatalf("expected redirect to %s with code, got Location=%q", authRedirectURI, loc)
	}
	found := false
	for _, e := range *emitted {
		if _, ok := e.(*domain.AuthorizationCodeIssued); ok {
			found = true
		}
	}
	if !found {
		t.Fatal("expected AuthorizationCodeIssued to be emitted")
	}
}
