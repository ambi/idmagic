package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	provisioningmemory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	provisioninghttp "github.com/ambi/idmagic/backend/provisioning/handlers_http"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type discoveryTarget struct{}

func (discoveryTarget) Discover(context.Context) (domain.ProvisioningCapabilities, error) {
	return domain.ProvisioningCapabilities{SupportsPatch: true, SupportsFilter: true}, nil
}

func (discoveryTarget) CreateUser(context.Context, []domain.AttributeMappingRule, map[string]any) (string, *string, error) {
	return "", nil, nil
}

func (discoveryTarget) UpdateUser(context.Context, string, []domain.AttributeMappingRule, map[string]any, bool) (*string, error) {
	// 差分なしを表す nil を返す。適用先の表示名が変わらなかったことは失敗ではない。
	return nil, nil //nolint:nilnil // ports 契約: 変化なしは nil で表し、adapter の失敗としない。
}
func (discoveryTarget) DeleteUser(context.Context, string) error { return nil }
func (discoveryTarget) SearchUserByAttribute(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}

func (discoveryTarget) CreateGroup(context.Context, []domain.AttributeMappingRule, map[string]any) (string, *string, error) {
	return "", nil, nil
}

func (discoveryTarget) UpdateGroup(context.Context, string, []domain.AttributeMappingRule, map[string]any, bool) (*string, error) {
	// UpdateUser と同じ契約。変化なしは nil で表す。
	return nil, nil //nolint:nilnil // ports 契約: 変化なしは nil で表し、adapter の失敗としない。
}
func (discoveryTarget) DeleteGroup(context.Context, string) error { return nil }
func (discoveryTarget) SearchGroupByAttribute(context.Context, string, string) (string, bool, error) {
	return "", false, nil
}

func (discoveryTarget) PatchGroupMembers(context.Context, string, string, []string) error { return nil }

// provisioningRequest は sub の利用者として管理 API へ要求を送る。
type provisioningRequest func(sub, method, path string, body any) *httptest.ResponseRecorder

// newProvisioningAdminServer は、admin のロールを持つ "admin" と持たない "member" がいる
// テナントに、Provisioning の管理 API を組み立てる。
func newProvisioningAdminServer(t *testing.T) provisioningRequest {
	t.Helper()
	now := time.Now().UTC()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "member", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "member", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	connections := provisioningmemory.NewProvisioningConnectionRepository()
	e := echo.New()
	e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
	provisioninghttp.RegisterRoutes(e.Group(""), provisioninghttp.Deps{
		Issuer: "http://idp.test",
		Authenticator: &support.Authenticator{
			UserRepo: users, AuthnResolver: authusecases.DemoHeaderResolver{},
		},
		ConnectionRepo: connections,
		NewTargetClient: func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error) {
			return discoveryTarget{}, nil
		},
	})

	csrf := "csrf-token"
	return func(sub, method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var payload []byte
		if body != nil {
			var err error
			payload, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Demo-Sub", sub)
		if method != http.MethodGet {
			req.Header.Set("Origin", "http://idp.test")
			req.Header.Set("X-Csrf-Token", csrf)
			req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: csrf})
		}
		response := httptest.NewRecorder()
		e.ServeHTTP(response, req)
		return response
	}
}

const provisioningConnectionPath = "/api/admin/v1/applications/app-1/provisioning"

func registerProvisioningConnection(t *testing.T, request provisioningRequest) {
	t.Helper()
	created := request("admin", http.MethodPost, provisioningConnectionPath, map[string]any{
		"base_url":   "https://downstream.example/scim/v2",
		"credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "secret"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
}

//spec:covers REQ-PROVISIONING-002, EX-PROVISIONING-002-01: 管理 API で登録した下流接続をテストし、検出した能力を同じ接続の管理状態から読み直す。
func TestAdminProvisioningConnectionLifecycle(t *testing.T) {
	request := newProvisioningAdminServer(t)
	registerProvisioningConnection(t, request)
	tested := request("admin", http.MethodPost, provisioningConnectionPath+"/test", nil)
	if tested.Code != http.StatusOK || !bytes.Contains(tested.Body.Bytes(), []byte(`"reachable":true`)) {
		t.Fatalf("test status=%d body=%s", tested.Code, tested.Body.String())
	}
	got := request("admin", http.MethodGet, provisioningConnectionPath, nil)
	if got.Code != http.StatusOK || !bytes.Contains(got.Body.Bytes(), []byte(`"supports_patch":true`)) {
		t.Fatalf("get status=%d body=%s", got.Code, got.Body.String())
	}
}

// 参照も削除も access_denied で拒否し、拒否した削除は接続を消さない。
//
//spec:covers EX-PROVISIONING-001-04: admin のロールを持たない利用者の接続の管理は 403 の access_denied で拒否され、接続は残る
func TestAdminProvisioningConnectionRefusesNonAdmin(t *testing.T) {
	request := newProvisioningAdminServer(t)
	registerProvisioningConnection(t, request)
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		refused := request("member", method, provisioningConnectionPath, nil)
		if refused.Code != http.StatusForbidden || !bytes.Contains(refused.Body.Bytes(), []byte("urn:idmagic:error:access_denied")) {
			t.Fatalf("%s status=%d body=%s, want 403 access_denied", method, refused.Code, refused.Body.String())
		}
	}
	if kept := request("admin", http.MethodGet, provisioningConnectionPath, nil); kept.Code != http.StatusOK {
		t.Fatalf("拒否した削除で接続が消えた: status=%d body=%s", kept.Code, kept.Body.String())
	}
}
