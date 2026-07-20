package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func newAdminAuthzTypeHandler() *echo.Echo {
	users := usermemory.NewUserRepository()
	types := oauth2memory.NewAuthorizationDetailTypeRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "regular", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://idp.test",

			Emit: func(spec.DomainEvent) {},
		}, UserRepo: users, OAuth2: oauth2.Module{AuthzDetailTypeRepo: types},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e
}

func validTypePayload() map[string]any {
	return map[string]any{
		"type":             "data_access",
		"description":      "文書アクセス",
		"display_template": "{datatypes} を {actions}",
		"schema": map[string]any{
			"rules": []map[string]any{
				{"name": "actions", "semantics": "set", "required": true, "allowed": []string{"read", "write"}},
				{"name": "datatypes", "semantics": "set", "required": true},
			},
		},
	}
}

func TestAdminCreatesAndListsAuthorizationDetailType(t *testing.T) {
	e := newAdminAuthzTypeHandler()
	csrf, cookie := adminCSRF(t, e)

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/authorization-detail-types", csrf, cookie, validTypePayload())
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/authorization-detail-types", http.NoBody)
	listReq.Header.Set("X-Demo-Sub", "admin")
	listRes := httptest.NewRecorder()
	e.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status=%d", listRes.Code)
	}
	var list struct {
		Types []map[string]any `json:"types"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Types) != 1 || list.Types[0]["type"] != "data_access" {
		t.Fatalf("unexpected list: %+v", list.Types)
	}
}

func TestAdminRejectsInvalidTypeSchema(t *testing.T) {
	e := newAdminAuthzTypeHandler()
	csrf, cookie := adminCSRF(t, e)
	payload := validTypePayload()
	// 空の rules はスキーマ違反 (Min(1)) で fail-closed。
	payload["schema"] = map[string]any{"rules": []map[string]any{}}
	res := adminJSONRequest(t, e, http.MethodPost, "/api/admin/authorization-detail-types", csrf, cookie, payload)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty schema, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminAuthorizationDetailTypeRequiresAdmin(t *testing.T) {
	e := newAdminAuthzTypeHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/authorization-detail-types", http.NoBody)
	req.Header.Set("X-Demo-Sub", "regular")
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("non-admin must not list types, got %d", res.Code)
	}
}
