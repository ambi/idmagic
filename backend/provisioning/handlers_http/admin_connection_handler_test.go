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

// REQ-PROVISIONING-002: 管理 API で登録した下流接続をテストし、検出した能力を同じ接続の管理状態から読み直す。
func TestAdminProvisioningConnectionLifecycle(t *testing.T) {
	now := time.Now().UTC()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
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
	request := func(method, path string, body any) *httptest.ResponseRecorder {
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
		req.Header.Set("X-Demo-Sub", "admin")
		if method != http.MethodGet {
			req.Header.Set("Origin", "http://idp.test")
			req.Header.Set("X-Csrf-Token", csrf)
			req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: csrf})
		}
		response := httptest.NewRecorder()
		e.ServeHTTP(response, req)
		return response
	}

	created := request(http.MethodPost, "/api/admin/v1/applications/app-1/provisioning", map[string]any{
		"base_url":   "https://downstream.example/scim/v2",
		"credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "secret"},
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	tested := request(http.MethodPost, "/api/admin/v1/applications/app-1/provisioning/test", nil)
	if tested.Code != http.StatusOK || !bytes.Contains(tested.Body.Bytes(), []byte(`"reachable":true`)) {
		t.Fatalf("test status=%d body=%s", tested.Code, tested.Body.String())
	}
	got := request(http.MethodGet, "/api/admin/v1/applications/app-1/provisioning", nil)
	if got.Code != http.StatusOK || !bytes.Contains(got.Body.Bytes(), []byte(`"supports_patch":true`)) {
		t.Fatalf("get status=%d body=%s", got.Code, got.Body.String())
	}
}
