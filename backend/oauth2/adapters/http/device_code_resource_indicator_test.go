package http_test

// RFC 8707 resource indicator を device_code グラントの /token エンドポイントへ
// 配線したことを確認する (wi-264)。usecases 層の詳細な振る舞いは
// device_flow_resource_indicator_test.go で検証済みのため、ここでは HTTP 層の
// 配線 (resource フォームパラメータ → usecase → レスポンス) のみを確認する。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/adapters/crypto"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2/device/usecases"
	"github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// newApprovedDeviceCodeServer は device_code grant で /token を叩ける状態
// (user_code 承認済み) までセットアップした完全な HTTP サーバーを返す。
func newApprovedDeviceCodeServer(t *testing.T, mcpResourceServerRepo *oauth2memory.McpResourceServerRepository) (*echo.Echo, string) {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	clientRepo.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: "device-client-2",
		ClientType:              spec.ClientPublic,
		GrantTypes:              []spec.GrantType{spec.GrantDeviceCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodNone,
		Scope:                   "openid",
		FapiProfile:             domain.FapiNone,
		CreatedAt:               time.Now().UTC(),
	})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
		PasswordHash: "hash", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	deviceStore := oauth2memory.NewDeviceCodeStore()
	refreshStore := oauth2memory.NewRefreshTokenStore()

	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := crypto.NewJWTSigner("http://test", keyStore)

	e := echo.New()
	tenantRepo := tenancymemory.NewTenantRepository()
	deps := httpadapter.Deps{
		Deps: support.Deps{Issuer: "http://test", TenantRepo: tenantRepo},
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, DeviceCodeStore: deviceStore, RefreshStore: refreshStore,
			McpResourceServerRepo: mcpResourceServerRepo,
		},
		UserRepo: userRepo, KeyStore: keyStore, TokenIssuer: tokenIssuer, TokenIntrospector: tokenIssuer,
	}
	if err := tenantRepo.Save(t.Context(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}
	httpadapter.Register(e, deps)

	// device_authorization + 承認 (usecase を直接呼び出し、HTTP フォームは token 交換だけに絞る)。
	authOut, err := usecases.RequestDeviceAuthorization(t.Context(), usecases.DeviceAuthorizationDeps{
		ClientRepo: clientRepo, DeviceCodeStore: deviceStore, BaseVerification: "http://test/device",
	}, usecases.DeviceAuthorizationInput{ClientID: "device-client-2", Scope: "openid"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("RequestDeviceAuthorization: %v", err)
	}
	if err := usecases.ApproveUserCode(t.Context(), usecases.VerifyUserCodeDeps{DeviceCodeStore: deviceStore}, authOut.UserCode, "user-1", time.Now().UTC()); err != nil {
		t.Fatalf("ApproveUserCode: %v", err)
	}
	return e, authOut.DeviceCode
}

func TestTokenDeviceCode_unregisteredResource_rejectedAsInvalidTarget(t *testing.T) {
	e, deviceCode := newApprovedDeviceCodeServer(t, oauth2memory.NewMcpResourceServerRepository())
	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"client_id":   {"device-client-2"},
		"device_code": {deviceCode},
		"resource":    {"https://mcp.example.com/unknown"},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != "invalid_target" {
		t.Fatalf("expected invalid_target, got %v", resp["error"])
	}
}

func TestTokenDeviceCode_registeredResource_boundAudience(t *testing.T) {
	repo := oauth2memory.NewMcpResourceServerRepository()
	repo.Seed(&domain.McpResourceServer{
		ResourceServerID: "rs-1", Resource: "https://mcp.example.com/tools",
		Name: "Tools", Scopes: []string{"openid"}, State: domain.McpResourceServerActive,
	})
	e, deviceCode := newApprovedDeviceCodeServer(t, repo)
	form := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"client_id":   {"device-client-2"},
		"device_code": {deviceCode},
		"resource":    {"https://mcp.example.com/tools"},
	}
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	accessToken, _ := resp["access_token"].(string)
	if accessToken == "" {
		t.Fatal("expected access_token in response")
	}

	introForm := url.Values{"token": {accessToken}, "client_id": {"device-client-2"}}
	introReq := httptest.NewRequest(http.MethodPost, "/introspect", strings.NewReader(introForm.Encode()))
	introReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	introRec := httptest.NewRecorder()
	e.ServeHTTP(introRec, introReq)
	var introResp map[string]any
	_ = json.Unmarshal(introRec.Body.Bytes(), &introResp)
	if !audienceContains(introResp["aud"], "https://mcp.example.com/tools") {
		t.Fatalf("expected aud bound to resource, got %v", introResp["aud"])
	}
}
