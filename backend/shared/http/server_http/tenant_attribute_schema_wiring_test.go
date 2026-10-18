package server_http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

//spec:covers REQ-TENANCY-002, REQ-TENANCY-020: 正式なルートが IdManagement の保存先を使い、同じテナントの User と Group のカスタム定義を返す。
func TestTenantAttributeSchemaRoutesReadIdentityOwnedSchemas(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	tenants := tenancy.NewMemoryModule()
	if err := tenants.EnsureDefault(ctx, now); err != nil {
		t.Fatal(err)
	}
	identity := idmanagement.NewMemoryModule()
	if err := identity.UserRepo.Save(ctx, controlPlaneTestUser("admin", tenancydomain.DefaultTenantID, "admin")); err != nil {
		t.Fatal(err)
	}
	if err := identity.AttrSchemaRepo.Save(ctx, &userdomain.TenantUserAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
		Attributes: []userdomain.UserAttributeDef{{Key: "region", Type: idmdomain.AttributeTypeString, Visibility: idmdomain.AttrVisibilityAdminReadable}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := identity.GroupAttrSchemaRepo.Save(ctx, &groupdomain.TenantGroupAttributeSchema{
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
		Attributes: []groupdomain.GroupAttributeDef{{Key: "cost_center", Type: idmdomain.AttributeTypeNumber}},
	}); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants.TenantRepo, IdManagement: identity,
		AuthnResolver: &fixedAuthnResolver{sub: "admin"},
	})
	for _, tc := range []struct{ path, key, attributeType string }{
		{path: "user-attribute-schema", key: "region", attributeType: "string"},
		{path: "group-attribute-schema", key: "cost_center", attributeType: "number"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/tenant/"+tc.path, http.NoBody)
			response := httptest.NewRecorder()
			e.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			var schema struct {
				TenantID   string `json:"tenant_id"`
				Attributes []struct {
					Key  string `json:"key"`
					Type string `json:"type"`
				} `json:"attributes"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &schema); err != nil {
				t.Fatal(err)
			}
			if schema.TenantID != tenancydomain.DefaultTenantID || len(schema.Attributes) != 1 || schema.Attributes[0].Key != tc.key || schema.Attributes[0].Type != tc.attributeType {
				t.Fatalf("route did not read the identity-owned schema: %s", response.Body.String())
			}
		})
	}
}
