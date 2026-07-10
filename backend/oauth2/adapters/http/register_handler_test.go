package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"

	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type registerFixture struct {
	e          *echo.Echo
	clientRepo *oauth2memory.OAuth2ClientRepository
}

func newRegisterServer() registerFixture {
	clientRepo := oauth2memory.NewClientRepository()
	e := echo.New()
	deps := httpadapter.Deps{
		Deps: support.Deps{
			Issuer:     "http://test",
			TenantRepo: memory.NewTenantRepository(),
		},
		ClientRepo: clientRepo,
	}
	_ = deps.TenantRepo.Save(context.Background(), &spec.Tenant{
		ID:     spec.DefaultTenantID,
		Realm:  spec.DefaultRealm,
		Status: spec.TenantStatusActive,
	})

	httpadapter.Register(e, deps)

	return registerFixture{
		e:          e,
		clientRepo: clientRepo,
	}
}

func TestRegisterClientAPI(t *testing.T) {
	fix := newRegisterServer()

	t.Run("Register_Succeeds", func(t *testing.T) {
		payload := `{
			"client_name": "Dynamic Client",
			"client_type": "confidential",
			"redirect_uris": ["https://app.example/cb"],
			"token_endpoint_auth_method": "client_secret_post",
			"grant_types": ["authorization_code"],
			"response_types": ["code"],
			"scope": "openid email"
		}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d. body: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp["client_id"] == nil || resp["client_secret"] == nil {
			t.Errorf("expected client_id and client_secret, got %+v", resp)
		}
		if resp["client_type"] != "confidential" {
			t.Errorf("expected client_type confidential, got %v", resp["client_type"])
		}
	})

	t.Run("Register_InvalidJSON", func(t *testing.T) {
		payload := `{invalid-json}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("Register_ValidationError_NoRedirectURIs", func(t *testing.T) {
		// redirect_uris が無い場合
		payload := `{
			"client_name": "Dynamic Client No Redirect",
			"client_type": "confidential",
			"token_endpoint_auth_method": "client_secret_post",
			"grant_types": ["authorization_code"],
			"response_types": ["code"]
		}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d. body: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] != "invalid_redirect_uri" {
			t.Errorf("expected error invalid_redirect_uri, got %v", resp["error"])
		}
	})

	t.Run("Register_ValidationError_BadJwksURI", func(t *testing.T) {
		// jwks_uri が https でない場合
		payload := `{
			"client_name": "Dynamic Client Bad JWKS URI",
			"client_type": "confidential",
			"redirect_uris": ["https://app.example/cb"],
			"token_endpoint_auth_method": "client_secret_post",
			"grant_types": ["authorization_code"],
			"response_types": ["code"],
			"jwks_uri": "http://insecure-jwks.example"
		}`
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d. body: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] != "invalid_client_metadata" {
			t.Errorf("expected error invalid_client_metadata, got %v", resp["error"])
		}
	})
}
