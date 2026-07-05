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

	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	"github.com/ambi/idmagic/internal/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/internal/shared/adapters/http/server"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"

	"github.com/labstack/echo/v5"
)

type deviceFixture struct {
	e           *echo.Echo
	clientRepo  *memory.OAuth2ClientRepository
	deviceStore *memory.DeviceCodeStore
	authn       *fakeAuthnResolver
	userCode    string
	deviceCode  string
}

func tenantContext(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &spec.Tenant{
		ID: id, DisplayName: id, Status: spec.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}, "https://idp.example/realms/"+id, "/realms/"+id)
}

func newDeviceServer() deviceFixture {
	clientRepo := memory.NewClientRepository()
	// テスト用クライアントをシード
	clientRepo.Seed(&spec.OAuth2Client{
		TenantID:                spec.DefaultTenantID,
		ClientID:                "device-client",
		ClientType:              spec.ClientPublic,
		RedirectURIs:            []string{"https://device.example/cb"},
		GrantTypes:              []spec.GrantType{spec.GrantDeviceCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: spec.AuthMethodNone,
		Scope:                   "openid profile",
		FapiProfile:             spec.FapiNone,
		CreatedAt:               time.Now().UTC(),
	})

	deviceStore := memory.NewDeviceCodeStore()
	authn := &fakeAuthnResolver{
		ctx: &authdomain.AuthenticationContext{UserID: "user-1"},
	}

	e := echo.New()
	deps := httpadapter.Deps{
		Deps: support.Deps{
			Issuer:     "http://test",
			TenantRepo: memory.NewTenantRepository(),
		},
		ClientRepo:      clientRepo,
		DeviceCodeStore: deviceStore,
		AuthnResolver:   authn,
	}
	// default tenant をシード
	_ = deps.TenantRepo.Save(context.Background(), &spec.Tenant{
		ID:     spec.DefaultTenantID,
		Realm:  spec.DefaultRealm,
		Status: spec.TenantStatusActive,
	})

	httpadapter.Register(e, deps)

	return deviceFixture{
		e:           e,
		clientRepo:  clientRepo,
		deviceStore: deviceStore,
		authn:       authn,
	}
}

func TestDeviceAuthorizationAPI(t *testing.T) {
	fix := newDeviceServer()
	ctx := tenantContext(spec.DefaultTenantID)

	// 1. handleDeviceAuthorization (POST /device_authorization)
	t.Run("DeviceAuthorization_Succeeds", func(t *testing.T) {
		form := url.Values{
			"client_id": {"device-client"},
			"scope":     {"openid"},
		}
		req := httptest.NewRequest(http.MethodPost, "/device_authorization", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. body: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		userCode, _ := body["user_code"].(string)
		deviceCode, _ := body["device_code"].(string)

		if userCode == "" || deviceCode == "" {
			t.Fatalf("expected non-empty codes, got user_code=%q device_code=%q", userCode, deviceCode)
		}

		fix.userCode = userCode
		fix.deviceCode = deviceCode
	})

	t.Run("DeviceAuthorization_InvalidClient", func(t *testing.T) {
		form := url.Values{
			"client_id": {"client-none"},
		}
		req := httptest.NewRequest(http.MethodPost, "/device_authorization", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	// 2. handleDeviceContext (GET /api/auth/device)
	t.Run("DeviceContext_Succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/device?user_code="+fix.userCode, http.NoBody)
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["user_code"] != fix.userCode {
			t.Errorf("expected user_code %q, got %v", fix.userCode, body["user_code"])
		}
		if body["csrf_token"] == "" {
			t.Error("expected CSRF token")
		}
	})

	// 3. handleDeviceAPI (POST /api/auth/device)
	t.Run("DeviceAPI_Approve", func(t *testing.T) {
		payload := `{"user_code":"` + fix.userCode + `","action":"approve"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/device", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://test")
		req.Header.Set("X-Csrf-Token", "csrf-val")
		req.Header.Set("Cookie", "idmagic_csrf=csrf-val")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. body: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["next"] != "/status?state=approved" {
			t.Errorf("expected next redirect, got %v", body["next"])
		}

		// 承認されていることをインメモリで確認
		stored, _ := fix.deviceStore.FindByUserCode(ctx, domain.NormalizeUserCode(fix.userCode))
		if stored.State != spec.DeviceFlowApproved {
			t.Errorf("expected state DeviceFlowApproved, got %v", stored.State)
		}
	})

	t.Run("DeviceAPI_Deny", func(t *testing.T) {
		// 新しいコードをリクエスト
		form := url.Values{
			"client_id": {"device-client"},
			"scope":     {"openid"},
		}
		reqGen := httptest.NewRequest(http.MethodPost, "/device_authorization", strings.NewReader(form.Encode()))
		reqGen.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recGen := httptest.NewRecorder()
		fix.e.ServeHTTP(recGen, reqGen)
		var bodyGen map[string]any
		_ = json.Unmarshal(recGen.Body.Bytes(), &bodyGen)
		newUserCode := bodyGen["user_code"].(string)

		payload := `{"user_code":"` + newUserCode + `","action":"deny"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/device", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://test")
		req.Header.Set("X-Csrf-Token", "csrf-val")
		req.Header.Set("Cookie", "idmagic_csrf=csrf-val")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d. body: %s", rec.Code, rec.Body.String())
		}

		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["next"] != "/status?state=denied" {
			t.Errorf("expected next redirect, got %v", body["next"])
		}
	})

	t.Run("DeviceAPI_CSRFFail", func(t *testing.T) {
		payload := `{"user_code":"` + fix.userCode + `","action":"approve"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/device", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://test")
		// X-Csrf-Token 不一致
		req.Header.Set("X-Csrf-Token", "csrf-val-wrong")
		req.Header.Set("Cookie", "idmagic_csrf=csrf-val")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", rec.Code)
		}
	})

	t.Run("DeviceAPI_Unauthorized", func(t *testing.T) {
		// authnResolverをnilを返すように設定
		fix.authn.ctx = nil

		payload := `{"user_code":"` + fix.userCode + `","action":"approve"}`
		req := httptest.NewRequest(http.MethodPost, "/api/auth/device", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://test")
		req.Header.Set("X-Csrf-Token", "csrf-val")
		req.Header.Set("Cookie", "idmagic_csrf=csrf-val")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})
}
