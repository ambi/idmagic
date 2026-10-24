package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/labstack/echo/v5"
)

//spec:covers REQ-TENANCY-002, REQ-TENANCY-020: 属性定義の全項目を保存して取得でき、別テナントへの GET と PUT は状態とイベントを変えない。
func TestAttributeSchemaRoutesPreserveEveryFieldAndTenantIsolation(t *testing.T) {
	for _, kind := range []string{"user", "group"} {
		t.Run(kind, func(t *testing.T) {
			actor := settingsActor("admin", "acme", []string{"admin"})
			var e *echo.Echo
			var events *[]spec.DomainEvent
			var read func(string) any
			var attributes []map[string]any
			var eventType string
			if kind == "user" {
				eventType = "TenantUserAttributeSchemaUpdated"
				server, repo, emitted := newUserAttributeSchemaServer(t, actor, activeTenant("acme", "Acme"), activeTenant("beta", "Beta"))
				e, events = server, emitted
				read = func(tenant string) any {
					schema, err := repo.FindByTenant(context.Background(), tenant)
					if err != nil {
						t.Fatal(err)
					}
					if schema == nil {
						return nil
					}
					return schema.Attributes
				}
				attributes = []map[string]any{
					{"key": "staff_code", "label": "社員番号", "type": "string", "multi_valued": false, "required": true, "editable_by_user": true, "unique": true, "claim_name": "staff_code", "oidc_scope": "profile", "visibility": "claim_exposed", "pii": true},
					{"key": "regions", "label": "地域", "type": "string", "multi_valued": true, "required": false, "editable_by_user": false, "unique": false, "visibility": "private", "pii": false},
				}
			} else {
				eventType = "TenantGroupAttributeSchemaUpdated"
				server, repo, emitted := newGroupAttributeSchemaServer(t, actor, activeTenant("acme", "Acme"), activeTenant("beta", "Beta"))
				e, events = server, emitted
				read = func(tenant string) any {
					schema, err := repo.FindByTenant(context.Background(), tenant)
					if err != nil {
						t.Fatal(err)
					}
					if schema == nil {
						return nil
					}
					return schema.Attributes
				}
				attributes = []map[string]any{
					{"key": "cost_codes", "label": "原価コード", "type": "number", "multi_valued": true, "required": true},
				}
			}
			path := "/api/admin/v1/tenant/" + kind + "-attribute-schema"
			assertAttributes := func(value any) {
				t.Helper()
				encoded, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				var actual []map[string]any
				if err := json.Unmarshal(encoded, &actual); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(actual, attributes) {
					t.Fatalf("attributes = %#v, want %#v", actual, attributes)
				}
			}
			put := putUserAttributeSchema(t, e, "/realms/acme"+path, map[string]any{"attributes": attributes})
			if put.Code != http.StatusOK {
				t.Fatalf("PUT status = %d, body = %s", put.Code, put.Body.String())
			}
			get := httptest.NewRecorder()
			e.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/realms/acme"+path, http.NoBody))
			for _, response := range []*httptest.ResponseRecorder{put, get} {
				if response.Code != http.StatusOK {
					t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
				}
				var body struct {
					TenantID   string          `json:"tenant_id"`
					Attributes json.RawMessage `json:"attributes"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.TenantID != "acme" {
					t.Fatalf("tenant = %q", body.TenantID)
				}
				assertAttributes(body.Attributes)
			}
			assertAttributes(read("acme"))
			if len(*events) != 1 || (*events)[0].EventType() != eventType {
				t.Fatalf("events = %#v", *events)
			}
			*events = nil
			invalid := putUserAttributeSchema(t, e, "/realms/acme"+path, map[string]any{"attributes": []map[string]any{
				{"key": "invalid", "type": "future_type", "visibility": "admin_readable"},
			}})
			assertTenancyProblem(t, invalid, http.StatusUnprocessableEntity, "invalid_attribute_schema")
			assertAttributes(read("acme"))
			if len(*events) != 0 {
				t.Fatalf("invalid definitions emitted %#v", *events)
			}
			getOther := httptest.NewRecorder()
			e.ServeHTTP(getOther, httptest.NewRequest(http.MethodGet, "/realms/beta"+path, http.NoBody))
			assertTenancyProblem(t, getOther, http.StatusUnauthorized, "authentication_required")
			putOther := putUserAttributeSchema(t, e, "/realms/beta"+path, map[string]any{"attributes": attributes})
			assertTenancyProblem(t, putOther, http.StatusUnauthorized, "authentication_required")
			if read("beta") != nil {
				t.Fatal("cross-tenant request saved a schema")
			}
			assertAttributes(read("acme"))
			if len(*events) != 0 {
				t.Fatalf("refused requests emitted %#v", *events)
			}
		})
	}
}
