package server_http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
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
	e.ServeHTTP(bare, httptest.NewRequest(http.MethodGet, "/realms/default/.well-known/openid-configuration", http.NoBody))
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
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "ops", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "ops",
		PasswordHash: "hash", Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "hash", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	resolver := &fixedAuthnResolver{sub: "ops"}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{TenantRepo: tenants}, UserRepo: users, AuthnResolver: resolver})

	allowed := httptest.NewRecorder()
	e.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/tenants", http.NoBody))
	if allowed.Code != http.StatusOK {
		t.Fatalf("system_admin status = %d, body = %s", allowed.Code, allowed.Body.String())
	}

	resolver.sub = "admin"
	denied := httptest.NewRecorder()
	e.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/tenants", http.NoBody))
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
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "acme-admin", TenantID: "acme", PreferredUsername: "acme-admin",
		PasswordHash: "hash", Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	})
	resolver := &fixedAuthnResolver{sub: "acme-admin"}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{TenantRepo: tenants}, UserRepo: users, AuthnResolver: resolver})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/tenants", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cross-tenant session status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

// control-plane (テナント CRUD) は他の全ボンデッドコンテキストと同じ tenantGroup
// (/realms/:tenant_id) に登録されており、default 以外の realm への隔離は
// requireSystemAdmin (user.TenantID == DefaultTenantID) のみが担う。これは
// 「別グループとして登録していないので届かない」という構造的な保証から
// 「ハンドラーが拒否する」という実行時の保証への切り替えを直接検証する
// 回帰ガード。
func TestControlPlaneRoutesUnreachableFromOtherRealm(t *testing.T) {
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
	users := usermemory.NewUserRepository()
	// system_admin ロールを acme テナント所属ユーザーに付与しても、
	// requireSystemAdmin は user.TenantID == DefaultTenantID を要求するため
	// 拒否されるはず (テナント境界がロール名より優先される)。
	users.Seed(&userdomain.User{
		ID: "acme-admin", TenantID: "acme", PreferredUsername: "acme-admin",
		PasswordHash: "hash", Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	})
	resolver := &fixedAuthnResolver{sub: "acme-admin"}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{TenantRepo: tenants}, UserRepo: users, AuthnResolver: resolver})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/tenants", http.NoBody))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("acme realm control-plane status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

// tenancy/handlers_http の control-plane ルートは、外側 (/realms/:tenant_id, リクエスト
// 自身の realm) と内側 (/api/admin/v1/tenants/:target_tenant_id, CRUD 対象) の 2 つの
// パスパラメータを持つ。両方を同じ名前 (tenant_id) にすると、echo の
// Context.Param が外側の値を返してしまい、ハンドラーが誤ったテナントを操作する
// (wi: echo v5.3.1 対応時に発見)。このテストはパラメータが正しく分離されている
// ことを直接確認する。
func TestControlPlaneGetTenantResolvesPathParamNotRequestRealm(t *testing.T) {
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
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "ops", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "ops",
		PasswordHash: "hash", Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	})
	resolver := &fixedAuthnResolver{sub: "ops"}
	e := echo.New()
	Register(e, Deps{Deps: support.Deps{TenantRepo: tenants}, UserRepo: users, AuthnResolver: resolver})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/tenants/acme", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Realm string `json:"realm"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Realm != "acme" {
		t.Fatalf("realm = %q, want %q (request realm %q leaked into CRUD target param)", got.Realm, "acme", "default")
	}
}
