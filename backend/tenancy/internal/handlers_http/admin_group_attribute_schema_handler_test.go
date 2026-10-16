package handlers_http_test

//spec:covers REQ-TENANCY-020: 管理者はテナント固有のグループ属性スキーマを定義できる。
// /api/admin/v1/tenant/group-attribute-schema 経由で検証する (wi-315)。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	memory "github.com/ambi/idmagic/backend/tenancy/internal/db_memory"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/internal/handlers_http"

	"github.com/labstack/echo/v5"
)

func newGroupAttributeSchemaServer(
	t *testing.T, actor *userdomain.User, tenants ...*domain.Tenant,
) (*echo.Echo, tenantports.TenantGroupAttributeSchemaRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := idmanagement.NewMemoryModule().UserRepo
	if actor != nil {
		_ = userRepo.Save(context.Background(), actor)
	}
	tenantRepo := memory.NewTenantRepository()
	for _, tenant := range tenants {
		if err := tenantRepo.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	schemaRepo := idmanagement.NewMemoryModule().GroupAttrSchemaRepo
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
		Tenancy:       tenancy.Module{GroupAttrSchemaRepo: schemaRepo},
		AuthnResolver: resolver,
	})
	return e, schemaRepo, &events
}

//verify:operation GetTenantGroupAttributeSchema: 200 JSON は自身の tenant_id と空の属性一覧を返す。
func TestGroupAttributeSchemaGetReturnsEmptyForUndefinedTenant(t *testing.T) {
	e, _, _ := newGroupAttributeSchemaServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/tenant/group-attribute-schema", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body tenancyhttp.GroupAttributeSchemaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TenantID != "acme" || len(body.Attributes) != 0 {
		t.Fatalf("expected empty attributes, got %+v", body)
	}
}

//verify:operation GetTenantGroupAttributeSchema: 非管理者には 403 access_denied の Problem Details を返す。
func TestGroupAttributeSchemaGetRejectsNonAdmin(t *testing.T) {
	e, _, _ := newGroupAttributeSchemaServer(t, settingsActor("alice", "acme", nil), activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/tenant/group-attribute-schema", http.NoBody))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	assertTenancyProblem(t, rec, http.StatusForbidden, "access_denied")
}

//spec:covers EX-TENANCY-020-01: cost_center 属性を保存し、更新後のスキーマと TenantGroupAttributeSchemaUpdated を観測する。
//verify:operation UpdateTenantGroupAttributeSchema: PUT は 200 のテナント、カスタム属性、作成と更新時刻を返し、保存とイベントを確認する。
func TestGroupAttributeSchemaPutPersistsAndEmitsEvent(t *testing.T) {
	e, schemaRepo, events := newGroupAttributeSchemaServer(
		t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"),
	)
	rec := putUserAttributeSchema(t, e, "/realms/acme/api/admin/v1/tenant/group-attribute-schema", map[string]any{
		"attributes": []map[string]any{
			{"key": "cost_center", "type": "string"},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body tenancyhttp.GroupAttributeSchemaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TenantID != "acme" || len(body.Attributes) != 1 || body.Attributes[0].Key != "cost_center" || body.Attributes[0].Type != "string" || body.CreatedAt.IsZero() || body.UpdatedAt.IsZero() {
		t.Fatalf("updated schema=%+v", body)
	}
	stored, err := schemaRepo.FindByTenant(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || len(stored.Attributes) != 1 || stored.Attributes[0].Key != "cost_center" {
		t.Fatalf("schema not persisted: %#v", stored)
	}
	found := false
	for _, ev := range *events {
		if ev.EventType() == "TenantGroupAttributeSchemaUpdated" {
			found = true
		}
	}
	if !found {
		t.Fatalf("TenantGroupAttributeSchemaUpdated not emitted: %+v", *events)
	}
}

//spec:covers EX-TENANCY-020-02: 同じ key を重複して追加すると InvalidGroupAttributeSchemaError で拒否する。
//verify:operation UpdateTenantGroupAttributeSchema: 重複キーは 422 invalid_attribute_schema の Problem Details で拒否する。
func TestGroupAttributeSchemaPutRejectsDuplicateKey(t *testing.T) {
	e, _, _ := newGroupAttributeSchemaServer(
		t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"),
	)
	rec := putUserAttributeSchema(t, e, "/realms/acme/api/admin/v1/tenant/group-attribute-schema", map[string]any{
		"attributes": []map[string]any{
			{"key": "cost_center", "type": "string"},
			{"key": "cost_center", "type": "number"},
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
