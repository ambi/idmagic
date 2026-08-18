package handlers_http_test

// REQ-TENANCY-020: 管理者はテナント固有のグループ属性スキーマを定義できる。
// /api/admin/v1/tenant/group_attribute_schema 経由で検証する (wi-315)。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/handlers_http"

	"github.com/labstack/echo/v5"
)

func newGroupAttributeSchemaServer(
	t *testing.T, actor *userdomain.User, tenants ...*domain.Tenant,
) (*echo.Echo, *groupmemory.TenantGroupAttributeSchemaRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	if actor != nil {
		userRepo.Seed(actor)
	}
	tenantRepo := memory.NewTenantRepository()
	for _, tenant := range tenants {
		if err := tenantRepo.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	schemaRepo := groupmemory.NewTenantGroupAttributeSchemaRepository()
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
		Deps: support.Deps{
			Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
			TenantRepo: tenantRepo,
			Emit:       emit,
		}, UserRepo: userRepo,
		Tenancy:       tenancy.Module{GroupAttrSchemaRepo: schemaRepo},
		AuthnResolver: resolver,
	})
	return e, schemaRepo, &events
}

func TestGroupAttributeSchemaGetReturnsEmptyForUndefinedTenant(t *testing.T) {
	e, _, _ := newGroupAttributeSchemaServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/tenant/group_attribute_schema", http.NoBody))
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

func TestGroupAttributeSchemaGetRejectsNonAdmin(t *testing.T) {
	e, _, _ := newGroupAttributeSchemaServer(t, settingsActor("alice", "acme", nil), activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/tenant/group_attribute_schema", http.NoBody))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGroupAttributeSchemaPutPersistsAndEmitsEvent(t *testing.T) {
	e, schemaRepo, events := newGroupAttributeSchemaServer(
		t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"),
	)
	rec := putUserAttributeSchema(t, e, "/realms/acme/api/admin/v1/tenant/group_attribute_schema", map[string]any{
		"attributes": []map[string]any{
			{"key": "cost_center", "type": "string"},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
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

func TestGroupAttributeSchemaPutRejectsDuplicateKey(t *testing.T) {
	e, _, _ := newGroupAttributeSchemaServer(
		t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"),
	)
	rec := putUserAttributeSchema(t, e, "/realms/acme/api/admin/v1/tenant/group_attribute_schema", map[string]any{
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
}
