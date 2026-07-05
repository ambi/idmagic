package tenancy_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"
)

// WithTenant で格納したテナントが Tenant / TenantID から取り出せる。
func TestWithTenantAndAccessors(t *testing.T) {
	tenant := &spec.Tenant{ID: "acme"}
	ctx := tenancy.WithTenant(context.Background(), tenant, "https://acme.example.com/", "/realms/acme/")

	if got := tenancy.Tenant(ctx); got == nil || got.ID != "acme" {
		t.Fatalf("Tenant() = %v, want id=acme", got)
	}
	if got := tenancy.TenantID(ctx); got != "acme" {
		t.Errorf("TenantID() = %q, want acme", got)
	}
	// issuer は末尾スラッシュを除去して格納される。
	if got := tenancy.Issuer(ctx, "https://fallback"); got != "https://acme.example.com" {
		t.Errorf("Issuer() = %q, want trimmed issuer", got)
	}
	if got := tenancy.URLPrefix(ctx); got != "/realms/acme" {
		t.Errorf("URLPrefix() = %q, want /realms/acme", got)
	}
}

// テナント未設定の context では TenantID は DefaultTenantID を返す。
func TestTenantIDDefaults(t *testing.T) {
	ctx := context.Background()
	if got := tenancy.TenantID(ctx); got != spec.DefaultTenantID {
		t.Errorf("TenantID() = %q, want %q", got, spec.DefaultTenantID)
	}
	if got := tenancy.Tenant(ctx); got != nil {
		t.Errorf("Tenant() = %v, want nil", got)
	}
	// ID が空のテナントも default にフォールバックする。
	ctx = tenancy.WithTenant(ctx, &spec.Tenant{ID: ""}, "", "")
	if got := tenancy.TenantID(ctx); got != spec.DefaultTenantID {
		t.Errorf("TenantID() with empty id = %q, want %q", got, spec.DefaultTenantID)
	}
}

// Issuer は context に issuer が無い場合 fallback を末尾スラッシュ除去して返す。
func TestIssuerFallback(t *testing.T) {
	ctx := context.Background()
	if got := tenancy.Issuer(ctx, "https://fallback.example.com/"); got != "https://fallback.example.com" {
		t.Errorf("Issuer() fallback = %q, want trimmed fallback", got)
	}
}

// URLPrefix は未設定なら空文字を返す。
func TestURLPrefixEmpty(t *testing.T) {
	if got := tenancy.URLPrefix(context.Background()); got != "" {
		t.Errorf("URLPrefix() = %q, want empty", got)
	}
}
