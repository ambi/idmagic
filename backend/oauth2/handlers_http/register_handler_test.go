package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"

	"github.com/labstack/echo/v5"
)

type registerFixture struct {
	e          *echo.Echo
	clientRepo *oauth2memory.OAuth2ClientRepository
	events     *[]spec.DomainEvent
}

func newRegisterServer() registerFixture {
	clientRepo := oauth2memory.NewClientRepository()
	events := &[]spec.DomainEvent{}
	e := echo.New()
	deps := httpadapter.Deps{
		Issuer:     "http://test",
		TenantRepo: tenancymemory.NewTenantRepository(),
		Emit:       func(event spec.DomainEvent) { *events = append(*events, event) },
		OAuth2:     oauth2.Module{ClientRepo: clientRepo},
	}
	_ = deps.TenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID:     tenancydomain.DefaultTenantID,
		Realm:  tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	})

	httpadapter.Register(e, deps)

	return registerFixture{
		e:          e,
		clientRepo: clientRepo,
		events:     events,
	}
}

//spec:covers EX-OAUTH2-016-01: 動的クライアント登録は client_id を採番して client_secret と一緒に一度だけ返し、ClientRegistered を発行する。
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
		req := httptest.NewRequest(http.MethodPost, "/realms/default/register", strings.NewReader(payload))
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
		// 採番した client_id は保存層にも存在する。応答だけを読むと、値を作って
		// 返しただけで登録していない実装を通してしまう。
		clientID, _ := resp["client_id"].(string)
		stored, err := fix.clientRepo.FindByID(
			context.Background(), tenancydomain.DefaultTenantID, clientID)
		if err != nil || stored == nil {
			t.Fatalf("採番した client_id が保存されていない: stored=%v err=%v", stored, err)
		}
		assertRegisteredEvent(t, *fix.events, clientID)
	})

	t.Run("Register_InvalidJSON", func(t *testing.T) {
		payload := `{invalid-json}`
		req := httptest.NewRequest(http.MethodPost, "/realms/default/register", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	//spec:covers EX-OAUTH2-016-02: redirect_uri を持たない登録要求は拒否され、
	// クライアントは作成されない。
	//
	// 400 を書いてから保存も続ける実装はステータスだけを読むテストを通すので、
	// テナントのクライアント数が変わっていないことまで読み直す。
	t.Run("Register_ValidationError_NoRedirectURIs", func(t *testing.T) {
		before, err := fix.clientRepo.FindAll(context.Background(), tenancydomain.DefaultTenantID)
		if err != nil {
			t.Fatal(err)
		}
		// redirect_uris が無い場合
		payload := `{
			"client_name": "Dynamic Client No Redirect",
			"client_type": "confidential",
			"token_endpoint_auth_method": "client_secret_post",
			"grant_types": ["authorization_code"],
			"response_types": ["code"]
		}`
		req := httptest.NewRequest(http.MethodPost, "/realms/default/register", strings.NewReader(payload))
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
		if resp["client_id"] != nil {
			t.Fatalf("拒否された登録が client_id を返した: %+v", resp)
		}
		after, err := fix.clientRepo.FindAll(context.Background(), tenancydomain.DefaultTenantID)
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(before) {
			t.Fatalf("拒否された登録がクライアントを作った: before=%d after=%d", len(before), len(after))
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
		req := httptest.NewRequest(http.MethodPost, "/realms/default/register", strings.NewReader(payload))
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

// assertRegisteredEvent は、登録が採番した client_id を名指す ClientRegistered が
// 発行されたことを確かめる。型だけを読むと、別のクライアントの登録と区別できない。
func assertRegisteredEvent(t *testing.T, events []spec.DomainEvent, clientID string) {
	t.Helper()
	for _, event := range events {
		registered, ok := event.(*oauthdomain.ClientRegistered)
		if ok && registered.ClientID == clientID {
			return
		}
	}
	t.Fatalf("client_id=%q の ClientRegistered が発行されていない: %v", clientID, events)
}
