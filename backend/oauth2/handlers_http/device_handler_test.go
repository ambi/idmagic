package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

type deviceFixture struct {
	e           *echo.Echo
	clientRepo  *oauth2memory.OAuth2ClientRepository
	deviceStore *oauth2memory.DeviceCodeStore
	authn       *fakeAuthnResolver
	userCode    string
	deviceCode  string
}

func tenantContext(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{
		ID: id, DisplayName: id, Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}, "https://idp.example/realms/"+id, "/realms/"+id)
}

func newDeviceServer(t *testing.T) deviceFixture {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	// テスト用クライアントをシード
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID:                tenancydomain.DefaultTenantID,
		ClientID:                "device-client",
		ClientType:              spec.ClientPublic,
		RedirectURIs:            []string{"https://device.example/cb"},
		GrantTypes:              []spec.GrantType{spec.GrantDeviceCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodNone,
		Scope:                   "openid profile",
		FapiProfile:             domain.FapiNone,
		CreatedAt:               time.Now().UTC(),
	})

	deviceStore := oauth2memory.NewDeviceCodeStore()
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	authn := &fakeAuthnResolver{
		ctx: &authdomain.AuthenticationContext{UserID: "user-1"},
	}

	e := echo.New()
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	tokenIssuer := tokens_jose.NewJWTSigner("http://test", keyStore)
	deps := httpadapter.Deps{
		Issuer:     "http://test",
		TenantRepo: tenancymemory.NewTenantRepository(),
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, DeviceCodeStore: deviceStore, RefreshStore: oauth2memory.NewRefreshTokenStore(),
		},
		AuthnResolver: authn,
		KeyStore:      keyStore,
		TokenIssuer:   tokenIssuer,
		UserRepo:      userRepo,
	}
	// default tenant をシード
	_ = deps.TenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID:     tenancydomain.DefaultTenantID,
		Realm:  tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
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
	fix := newDeviceServer(t)
	ctx := tenantContext(tenancydomain.DefaultTenantID)

	// 1. handleDeviceAuthorization (POST /device_authorization)
	t.Run("DeviceAuthorization_Succeeds", func(t *testing.T) {
		form := url.Values{
			"client_id": {"device-client"},
			"scope":     {"openid"},
		}
		req := httptest.NewRequest(http.MethodPost, "/realms/default/device_authorization", strings.NewReader(form.Encode()))
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
		req := httptest.NewRequest(http.MethodPost, "/realms/default/device_authorization", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		fix.e.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	// 2. handleDeviceContext (GET /api/auth/device)
	t.Run("DeviceContext_Succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/device?user_code="+fix.userCode, http.NoBody)
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
		req := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/device", strings.NewReader(payload))
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

		// REQ-OAUTH2-027: 承認後の device_code を正式な token endpoint で交換し、最終成果の資格情報を得る。
		tokenForm := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {fix.deviceCode},
			"client_id":   {"device-client"},
		}
		tokenReq := httptest.NewRequest(http.MethodPost, "/realms/default/token", strings.NewReader(tokenForm.Encode()))
		tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		tokenRec := httptest.NewRecorder()
		fix.e.ServeHTTP(tokenRec, tokenReq)
		if tokenRec.Code != http.StatusOK {
			t.Fatalf("approved device_code exchange status=%d body=%s", tokenRec.Code, tokenRec.Body.String())
		}
		var tokenBody map[string]any
		if err := json.Unmarshal(tokenRec.Body.Bytes(), &tokenBody); err != nil {
			t.Fatal(err)
		}
		if tokenBody["access_token"] == "" {
			t.Fatalf("approved device_code exchange returned no access token: %s", tokenRec.Body.String())
		}
	})

	t.Run("DeviceAPI_Deny", func(t *testing.T) {
		// 新しいコードをリクエスト
		form := url.Values{
			"client_id": {"device-client"},
			"scope":     {"openid"},
		}
		reqGen := httptest.NewRequest(http.MethodPost, "/realms/default/device_authorization", strings.NewReader(form.Encode()))
		reqGen.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recGen := httptest.NewRecorder()
		fix.e.ServeHTTP(recGen, reqGen)
		var bodyGen map[string]any
		_ = json.Unmarshal(recGen.Body.Bytes(), &bodyGen)
		newUserCode := bodyGen["user_code"].(string)

		payload := `{"user_code":"` + newUserCode + `","action":"deny"}`
		req := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/device", strings.NewReader(payload))
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

		// 拒否された device_code を /token で交換すると access_denied が返る。
		// RFC 6749 §5.2 はトークンエンドポイントのエラー応答を 400 と定め、
		// invalid_client だけに 401 を許す。RFC 8628 §3.5 の access_denied も
		// これに従うので、403 で返る経路は存在しない。契約の Token が 403 を
		// 宣言していないことは、この実測に対応している。
		deniedDeviceCode := bodyGen["device_code"].(string)
		tokenForm := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deniedDeviceCode},
			"client_id":   {"device-client"},
		}
		tokenReq := httptest.NewRequest(http.MethodPost, "/realms/default/token", strings.NewReader(tokenForm.Encode()))
		tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		tokenRec := httptest.NewRecorder()
		fix.e.ServeHTTP(tokenRec, tokenReq)

		if tokenRec.Code != http.StatusBadRequest {
			t.Fatalf("denied device_code exchange status=%d body=%s, want 400", tokenRec.Code, tokenRec.Body.String())
		}
		var tokenBody map[string]any
		if err := json.Unmarshal(tokenRec.Body.Bytes(), &tokenBody); err != nil {
			t.Fatalf("decode token error body: %v (body=%s)", err, tokenRec.Body.String())
		}
		if tokenBody["error"] != "access_denied" {
			t.Fatalf("token error=%v, want access_denied (body=%s)", tokenBody["error"], tokenRec.Body.String())
		}
		// 拒否が何も通していないこと。アクセストークンは 1 つも返っていない。
		if _, issued := tokenBody["access_token"]; issued {
			t.Fatalf("denied device_code exchange returned a token: %s", tokenRec.Body.String())
		}
	})

	t.Run("DeviceAPI_CSRFFail", func(t *testing.T) {
		payload := `{"user_code":"` + fix.userCode + `","action":"approve"}`
		req := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/device", strings.NewReader(payload))
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
		req := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/device", strings.NewReader(payload))
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
