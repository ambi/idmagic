package handlers_http_test

// REQ-IDMANAGEMENT-024: 管理者はグループの連絡先メールとカスタム属性を、テナント定義の
// スキーマに従って設定できる。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func newAdminGroupHandlerWithAttrSchema(t *testing.T) (*echo.Echo, *groupmemory.TenantGroupAttributeSchemaRepository) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	schemaRepo := groupmemory.NewTenantGroupAttributeSchemaRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin", PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:          "http://idp.test",
			PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		}, UserRepo: userRepo, GroupRepo: groupRepo,
		Tenancy:       tenancy.Module{GroupAttrSchemaRepo: schemaRepo},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, schemaRepo
}

func TestAdminGroupAPICreateWithEmailAndAttributes(t *testing.T) {
	e, schemaRepo := newAdminGroupHandlerWithAttrSchema(t)
	csrf, cookie := adminCSRF(t, e)
	putAttrs := adminJSONRequest(t, e, http.MethodPut, "/api/admin/v1/tenant/group_attribute_schema", csrf, cookie, map[string]any{
		"attributes": []map[string]any{{"key": "cost_center", "type": "string"}},
	})
	if putAttrs.Code != http.StatusOK {
		t.Fatalf("schema put status=%d body=%s", putAttrs.Code, putAttrs.Body.String())
	}
	if got, err := schemaRepo.FindByTenant(httptest.NewRequest(http.MethodGet, "/", http.NoBody).Context(), tenancydomain.DefaultTenantID); err != nil || got == nil || len(got.Attributes) != 1 {
		t.Fatalf("schema not persisted: %v %+v", err, got)
	}

	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie, map[string]any{
		"name": "sales", "email": "sales@example.test",
		"attributes": map[string]any{"cost_center": map[string]any{"type": "string", "string": "CC-100"}},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		Email      string                     `json:"email"`
		Attributes map[string]json.RawMessage `json:"attributes"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Email != "sales@example.test" {
		t.Fatalf("email = %q", created.Email)
	}
	if _, ok := created.Attributes["cost_center"]; !ok {
		t.Fatalf("attributes = %+v", created.Attributes)
	}
}

func TestAdminGroupAPICreateRejectsInvalidEmail(t *testing.T) {
	e, _ := newAdminGroupHandlerWithAttrSchema(t)
	csrf, cookie := adminCSRF(t, e)
	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie, map[string]any{
		"name": "sales", "email": "not-an-email",
	})
	if create.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", create.Code, create.Body.String())
	}
}

func TestAdminGroupAPICreateRejectsUndefinedAttribute(t *testing.T) {
	e, _ := newAdminGroupHandlerWithAttrSchema(t)
	csrf, cookie := adminCSRF(t, e)
	create := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie, map[string]any{
		"name":       "sales",
		"attributes": map[string]any{"unknown": map[string]any{"type": "string", "string": "x"}},
	})
	if create.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", create.Code, create.Body.String())
	}
}
