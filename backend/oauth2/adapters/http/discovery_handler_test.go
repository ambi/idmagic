package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

func TestDiscoveryRoutesIncludeRFC8414Alias(t *testing.T) {
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{Deps: support.Deps{Issuer: "https://idp.example", SCL: spec.MustLoadSCL()}})

	for _, path := range []string{
		"/.well-known/openid-configuration",
		"/.well-known/oauth-authorization-server",
	} {
		req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"acr_values_supported"`)) {
			t.Fatalf("%s omitted acr_values_supported", path)
		}
		// RFC 9207 §3。authorization_response_iss_parameter_supported を必ず広告する。
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"authorization_response_iss_parameter_supported":true`)) {
			t.Fatalf("%s omitted authorization_response_iss_parameter_supported", path)
		}
	}
}

// SCL scenario "テナントごとに別の署名鍵で発行する" / TenantJwksIsolation 不変条件。
// /realms/{tenant_id}/jwks は当該テナントの鍵だけを返す。
func TestPerTenantJwksIsolation(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	for _, id := range []string{"tenant-a", "tenant-b"} {
		if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
			ID: id, Realm: id, DisplayName: id, Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
	}
	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	// 各テナントで token 発行相当 (GetActiveKey) を起こし鍵を作る。
	ctxA := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
	ctxB := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
	keyA, err := keyStore.GetActiveKey(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	keyB, err := keyStore.GetActiveKey(ctxB)
	if err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "https://idp.example", SCL: spec.MustLoadSCL(),
			TenantRepo: tenants,
		}, KeyStore: keyStore,
	})

	kidsA := jwksKids(t, e, "/realms/tenant-a/jwks")
	if len(kidsA) != 1 || kidsA[keyA.Kid] != true {
		t.Fatalf("tenant-a jwks must contain only its own kid: %v", kidsA)
	}
	if kidsA[keyB.Kid] {
		t.Fatalf("tenant-a jwks leaked tenant-b kid %s", keyB.Kid)
	}
	kidsB := jwksKids(t, e, "/realms/tenant-b/jwks")
	if kidsB[keyA.Kid] {
		t.Fatalf("tenant-b jwks leaked tenant-a kid %s", keyA.Kid)
	}
}

func jwksKids(t *testing.T, e *echo.Echo, path string) map[string]bool {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
	}
	var body struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	kids := map[string]bool{}
	for _, k := range body.Keys {
		if kid, ok := k["kid"].(string); ok {
			kids[kid] = true
		}
	}
	return kids
}
