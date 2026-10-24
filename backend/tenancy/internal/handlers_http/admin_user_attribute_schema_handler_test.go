package handlers_http_test

// SCL scenario "テナント内 admin は所属テナントの custom 属性定義を読み・更新できる"
// を /api/admin/v1/tenant/user-attribute-schema 経由で検証する (wi-19)。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	userschemaports "github.com/ambi/idmagic/backend/idmanagement/user/ports"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	memory "github.com/ambi/idmagic/backend/tenancy/internal/db_memory"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type userAttributeSchemaResponse struct {
	TenantID   string                        `json:"tenant_id"`
	Attributes []userdomain.UserAttributeDef `json:"attributes"`
	Builtin    []userdomain.UserAttributeDef `json:"builtin"`
	CreatedAt  time.Time                     `json:"created_at"`
	UpdatedAt  time.Time                     `json:"updated_at"`
}

func newUserAttributeSchemaServer(
	t *testing.T, actor *userdomain.User, tenants ...*domain.Tenant,
) (*echo.Echo, userschemaports.TenantUserAttributeSchemaRepository, *[]spec.DomainEvent) {
	t.Helper()
	var seeds []*userdomain.User
	if actor != nil {
		seeds = append(seeds, actor)
	}
	return newUserAttributeSchemaServerWithUsers(t, actor, seeds, tenants...)
}

// newUserAttributeSchemaServerWithUsers は、組み立て地点と同じく User とスキーマの保存先を一つの
// IdManagement のモジュールから作り、users を保存したサーバーを返す。
func newUserAttributeSchemaServerWithUsers(
	t *testing.T, actor *userdomain.User, users []*userdomain.User, tenants ...*domain.Tenant,
) (*echo.Echo, userschemaports.TenantUserAttributeSchemaRepository, *[]spec.DomainEvent) {
	t.Helper()
	identity := idmanagement.NewMemoryModule()
	userRepo := identity.UserRepo
	for _, u := range users {
		if err := userRepo.Save(context.Background(), u); err != nil {
			t.Fatal(err)
		}
	}
	tenantRepo := memory.NewTenantRepository()
	for _, tenant := range tenants {
		if err := tenantRepo.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	schemaRepo := identity.AttrSchemaRepo
	resolver := &fakeAuthnResolver{}
	if actor != nil {
		resolver.ctx = &authdomain.AuthenticationContext{
			UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}
	}
	events := make([]spec.DomainEvent, 0)
	emit := func(e spec.DomainEvent) { events = append(events, e) }
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenantRepo,
		Emit:       emit, UserRepo: userRepo,
		AttrSchemaRepo: schemaRepo,
		AuthnResolver:  resolver,
	})
	return e, schemaRepo, &events
}

func putUserAttributeSchema(t *testing.T, e *echo.Echo, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	tenant := tenantPrefix(path)
	csrf, cookie := passwordResetContextCSRF(t, e, tenant+"/api/auth/password-reset-context")
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

//verify:operation GetTenantUserAttributeSchema: 200 JSON は自身の tenant_id、空のカスタム属性と組み込み属性の一覧を返す。
func TestUserAttributeSchemaGetReturnsBuiltinForUndefinedTenant(t *testing.T) {
	e, _, _ := newUserAttributeSchemaServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/tenant/user-attribute-schema", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body userAttributeSchemaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TenantID != "acme" || len(body.Attributes) != 0 {
		t.Fatalf("expected empty custom attributes, got %+v", body)
	}
	if len(body.Builtin) == 0 {
		t.Fatalf("expected builtin catalog to be returned")
	}
}

//verify:operation GetTenantUserAttributeSchema: 非管理者には 403 access_denied の Problem Details を返す。
func TestUserAttributeSchemaGetRejectsNonAdmin(t *testing.T) {
	e, _, _ := newUserAttributeSchemaServer(t, settingsActor("alice", "acme", nil), activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/tenant/user-attribute-schema", http.NoBody))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertTenancyProblem(t, rec, http.StatusForbidden, "access_denied")
}

//spec:covers EX-TENANCY-002-01: admin が editable_by_user=true の custom 属性を追加すると更新後のスキーマへ保存される。
//verify:operation UpdateTenantUserAttributeSchema: PUT は 200 のテナント、カスタム属性、組み込み属性、作成と更新時刻を返し、保存とイベントを確認する。
func TestUserAttributeSchemaPutPersistsAndEmitsEvent(t *testing.T) {
	e, schemaRepo, events := newUserAttributeSchemaServer(
		t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"),
	)
	rec := putUserAttributeSchema(t, e, "/realms/acme/api/admin/v1/tenant/user-attribute-schema", map[string]any{
		"attributes": []map[string]any{
			{"key": "region", "type": "string", "visibility": "claim_exposed", "claim_name": "region", "editable_by_user": true},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body userAttributeSchemaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TenantID != "acme" || len(body.Attributes) != 1 || body.Attributes[0].Key != "region" || body.Attributes[0].Type != "string" || !body.Attributes[0].EditableByUser || len(body.Builtin) == 0 || body.CreatedAt.IsZero() || body.UpdatedAt.IsZero() {
		t.Fatalf("updated schema=%+v", body)
	}
	stored, err := schemaRepo.FindByTenant(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || len(stored.Attributes) != 1 || stored.Attributes[0].Key != "region" || !stored.Attributes[0].EditableByUser {
		t.Fatalf("schema not persisted: %#v", stored)
	}
	found := false
	for _, ev := range *events {
		if ev.EventType() == "TenantUserAttributeSchemaUpdated" {
			found = true
		}
	}
	if !found {
		t.Fatalf("TenantUserAttributeSchemaUpdated not emitted: %+v", *events)
	}
}

//verify:operation UpdateTenantUserAttributeSchema: 組み込み属性との衝突は 422 invalid_attribute_schema の Problem Details で拒否する。
func TestUserAttributeSchemaPutRejectsBuiltinCollision(t *testing.T) {
	e, _, _ := newUserAttributeSchemaServer(
		t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"),
	)
	rec := putUserAttributeSchema(t, e, "/realms/acme/api/admin/v1/tenant/user-attribute-schema", map[string]any{
		"attributes": []map[string]any{
			{"key": "nickname", "type": "string", "visibility": "claim_exposed"},
		},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "urn:idmagic:error:invalid_attribute_schema") {
		t.Fatalf("unexpected body=%s", rec.Body.String())
	}
	assertTenancyProblem(t, rec, http.StatusUnprocessableEntity, "invalid_attribute_schema")
}

func memberWithBadge(id, tenantID, badge string) *userdomain.User {
	u := settingsActor(id, tenantID, nil)
	value := badge
	u.Attributes = map[string]userdomain.AttributeValue{"badge": {Type: "string", String: &value}}
	return u
}

//spec:covers REQ-TENANCY-002: 削除されていない二人の User が同じ値を持つ属性を unique に変える保存を 409 と attribute_values_not_unique で拒否してスキーマを変えず、値が重ならない属性は unique にできること。
//verify:operation UpdateTenantUserAttributeSchema: 値が重なる属性を unique にする保存は 409 attribute_values_not_unique の Problem Details で拒否し、スキーマを変えない。
func TestUserAttributeSchemaPutRejectsUniqueOverSharedValues(t *testing.T) {
	actor := settingsActor("admin", "acme", []string{"admin"})
	e, schemaRepo, events := newUserAttributeSchemaServerWithUsers(t, actor, []*userdomain.User{
		actor, memberWithBadge("alice", "acme", "B-1"), memberWithBadge("bob", "acme", "B-1"),
	}, activeTenant("acme", "Acme"))
	const path = "/realms/acme/api/admin/v1/tenant/user-attribute-schema"
	badge := func(unique bool) map[string]any {
		return map[string]any{"attributes": []map[string]any{
			{"key": "badge", "type": "string", "visibility": "admin_readable", "unique": unique},
		}}
	}
	if rec := putUserAttributeSchema(t, e, path, badge(false)); rec.Code != http.StatusOK {
		t.Fatalf("define badge status=%d body=%s", rec.Code, rec.Body.String())
	}
	*events = nil

	rec := putUserAttributeSchema(t, e, path, badge(true))
	assertTenancyProblem(t, rec, http.StatusConflict, "attribute_values_not_unique")
	stored, err := schemaRepo.FindByTenant(context.Background(), "acme")
	if err != nil || stored == nil || stored.Attributes[0].Unique {
		t.Fatalf("refused schema was stored: (%+v, %v)", stored, err)
	}
	if len(*events) != 0 {
		t.Fatalf("refused save emitted %+v", *events)
	}

	other := map[string]any{"attributes": []map[string]any{
		{"key": "badge", "type": "string", "visibility": "admin_readable"},
		{"key": "employee_no", "type": "string", "visibility": "admin_readable", "unique": true},
	}}
	ok := putUserAttributeSchema(t, e, path, other)
	if ok.Code != http.StatusOK || !strings.Contains(ok.Body.String(), `"unique":true`) {
		t.Fatalf("unique without shared values status=%d body=%s", ok.Code, ok.Body.String())
	}
}
