package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type fixedAuthnResolver struct {
	sub string
}

func (r *fixedAuthnResolver) Resolve(
	context.Context,
	authdomain.Headers,
) (*authdomain.AuthenticationContext, error) {
	return &authdomain.AuthenticationContext{UserID: r.sub, AuthTime: time.Now().Unix()}, nil
}

func TestRealmDiscoveryUsesTenantIssuer(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
		ID: "acme", Realm: "acme", DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{Issuer: "https://idp.example", SCL: spec.MustLoadSCL(), TenantRepo: tenants}})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/.well-known/openid-configuration", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if got := doc["issuer"]; got != "https://idp.example/realms/acme" {
		t.Fatalf("issuer = %v", got)
	}
	if got := doc["authorization_endpoint"]; got != "https://idp.example/realms/acme/authorize" {
		t.Fatalf("authorization_endpoint = %v", got)
	}
}

func TestBareRouteUsesDefaultAndDisabledTenantIsRejected(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	now := time.Now().UTC()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
		{ID: "acme", Realm: "acme", DisplayName: "Acme", Status: tenancydomain.TenantStatusDisabled, CreatedAt: now, DisabledAt: &now},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{Issuer: "https://idp.example", SCL: spec.MustLoadSCL(), TenantRepo: tenants}})

	bare := httptest.NewRecorder()
	e.ServeHTTP(bare, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", http.NoBody))
	if bare.Code != http.StatusOK {
		t.Fatalf("bare status = %d, body = %s", bare.Code, bare.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(bare.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if got := doc["issuer"]; got != "https://idp.example/realms/default" {
		t.Fatalf("bare issuer = %v", got)
	}

	disabled := httptest.NewRecorder()
	e.ServeHTTP(disabled, httptest.NewRequest(http.MethodGet, "/realms/acme/authorize", http.NoBody))
	if disabled.Code != http.StatusBadRequest {
		t.Fatalf("disabled status = %d, body = %s", disabled.Code, disabled.Body.String())
	}
}

func TestTenantAdminRequiresSystemAdmin(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default",
		Status: tenancydomain.TenantStatusActive, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	users := idmmemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&idmdomain.User{
		ID: "ops", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "ops",
		PasswordHash: "hash", Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&idmdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "hash", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	resolver := &fixedAuthnResolver{sub: "ops"}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{TenantRepo: tenants}, UserRepo: users, AuthnResolver: resolver})

	allowed := httptest.NewRecorder()
	e.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/tenants", http.NoBody))
	if allowed.Code != http.StatusOK {
		t.Fatalf("system_admin status = %d, body = %s", allowed.Code, allowed.Body.String())
	}

	resolver.sub = "admin"
	denied := httptest.NewRecorder()
	e.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/tenants", http.NoBody))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("admin status = %d, body = %s", denied.Code, denied.Body.String())
	}
}

// 別テナントのセッションがリクエストに紛れ込んだ場合、resolveAuthentication が
// 未認証として弾くこと (cookie path 分離が破られたケースの defense-in-depth)。
func TestCrossTenantSessionRejectsSystemAdmin(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	now := time.Now().UTC()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
		{ID: "acme", Realm: "acme", DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	users := idmmemory.NewUserRepository()
	users.Seed(&idmdomain.User{
		ID: "acme-admin", TenantID: "acme", PreferredUsername: "acme-admin",
		PasswordHash: "hash", Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	})
	resolver := &fixedAuthnResolver{sub: "acme-admin"}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{TenantRepo: tenants}, UserRepo: users, AuthnResolver: resolver})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/tenants", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cross-tenant session status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
