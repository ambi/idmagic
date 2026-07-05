package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/internal/oauth2/domain"
	"github.com/ambi/idmagic/internal/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/internal/shared/adapters/http/server"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"

	"github.com/labstack/echo/v5"
)

type tokenFixture struct {
	e          *echo.Echo
	clientRepo *memory.OAuth2ClientRepository
}

func newTokenServer(t *testing.T) tokenFixture {
	t.Helper()
	clientRepo := memory.NewClientRepository()

	// confidential client
	secretHash := domain.HashClientSecret("secret-conf")
	clientRepo.Seed(&spec.OAuth2Client{
		TenantID:                spec.DefaultTenantID,
		ClientID:                "client-conf",
		ClientSecretHash:        &secretHash,
		ClientType:              spec.ClientConfidential,
		RedirectURIs:            []string{"https://app.example/cb"},
		GrantTypes:              []spec.GrantType{spec.GrantClientCredentials, spec.GrantRefreshToken},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: spec.AuthMethodClientSecretPost,
		Scope:                   "openid profile",
		FapiProfile:             spec.FapiNone,
		CreatedAt:               time.Now().UTC(),
	})

	// public client
	clientRepo.Seed(&spec.OAuth2Client{
		TenantID:                spec.DefaultTenantID,
		ClientID:                "client-pub",
		ClientType:              spec.ClientPublic,
		RedirectURIs:            []string{"https://app.example/cb"},
		GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: spec.AuthMethodNone,
		Scope:                   "openid profile",
		FapiProfile:             spec.FapiNone,
		CreatedAt:               time.Now().UTC(),
	})

	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := crypto.NewJWTSigner("http://test", keyStore)

	e := echo.New()
	deps := httpadapter.Deps{
		Deps: support.Deps{
			Issuer:     "http://test",
			TenantRepo: memory.NewTenantRepository(),
		},
		ClientRepo:          clientRepo,
		RefreshStore:        memory.NewRefreshTokenStore(),
		AccessTokenDenylist: memory.NewAccessTokenDenylist(),
		KeyStore:            keyStore,
		TokenIssuer:         tokenIssuer,
		TokenIntrospector:   tokenIssuer,
	}
	_ = deps.TenantRepo.Save(context.Background(), &spec.Tenant{
		ID:     spec.DefaultTenantID,
		Realm:  spec.DefaultRealm,
		Status: spec.TenantStatusActive,
	})

	httpadapter.Register(e, deps)

	return tokenFixture{
		e:          e,
		clientRepo: clientRepo,
	}
}

func TestTokenAPI(t *testing.T) {
	fix := newTokenServer(t)

	t.Run("Token_GrantTypeEmpty", func(t *testing.T) {
		form := url.Values{
			"client_id":     {"client-conf"},
			"client_secret": {"secret-conf"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] != "invalid_request" {
			t.Errorf("expected error invalid_request, got %v", resp["error"])
		}
	})

	t.Run("Token_UnsupportedGrantType", func(t *testing.T) {
		form := url.Values{
			"client_id":     {"client-conf"},
			"client_secret": {"secret-conf"},
			"grant_type":    {"bad-grant"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] != "unsupported_grant_type" {
			t.Errorf("expected error unsupported_grant_type, got %v", resp["error"])
		}
	})

	t.Run("Token_UnauthorizedClient", func(t *testing.T) {
		form := url.Values{
			"client_id":     {"client-conf"},
			"client_secret": {"secret-conf"},
			"grant_type":    {"authorization_code"}, // 宣言されていない
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] != "unauthorized_client" {
			t.Errorf("expected error unauthorized_client, got %v", resp["error"])
		}
	})

	t.Run("Token_ClientCredentials_Succeeds", func(t *testing.T) {
		form := url.Values{
			"client_id":     {"client-conf"},
			"client_secret": {"secret-conf"},
			"grant_type":    {"client_credentials"},
			"scope":         {"openid"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. body: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["access_token"] == nil {
			t.Error("expected access_token in response")
		}
		if resp["token_type"] != "Bearer" {
			t.Errorf("expected token_type Bearer, got %v", resp["token_type"])
		}
	})

	t.Run("Token_ClientCredentials_PublicClientForbidden", func(t *testing.T) {
		form := url.Values{
			"client_id":  {"client-pub"},
			"grant_type": {"client_credentials"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] != "unauthorized_client" {
			t.Errorf("expected error unauthorized_client, got %v", resp["error"])
		}
	})

	t.Run("Token_ClientCredentials_InvalidScope", func(t *testing.T) {
		form := url.Values{
			"client_id":     {"client-conf"},
			"client_secret": {"secret-conf"},
			"grant_type":    {"client_credentials"},
			"scope":         {"openid invalid-scope"},
		}
		req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] != "invalid_scope" {
			t.Errorf("expected error invalid_scope, got %v", resp["error"])
		}
	})

	t.Run("Revoke_Succeeds", func(t *testing.T) {
		form := url.Values{
			"client_id":     {"client-conf"},
			"client_secret": {"secret-conf"},
			"token":         {"dummy-token"},
		}
		req := httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}
