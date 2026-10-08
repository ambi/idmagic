package tenancy_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"
)

// WithTenant で格納したテナントが Tenant / TenantID から取り出せる。
func TestWithTenantAndAccessors(t *testing.T) {
	tenant := &domain.Tenant{ID: "acme"}
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

//spec:covers REQ-TENANCY-006: テナントを解決していない文脈では default テナントの ID を返さず panic する。
func TestTenantIDPanicsWithoutResolvedTenant(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"テナントがない": context.Background(),
		"ID が空":   tenancy.WithTenant(context.Background(), &domain.Tenant{ID: ""}, "", ""),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("TenantID() did not panic")
				}
			}()
			got := tenancy.TenantID(ctx)
			t.Errorf("TenantID() = %q, want panic", got)
		})
	}
}

// Tenant はテナントのない文脈で nil を返す。
func TestTenantNilWithoutResolvedTenant(t *testing.T) {
	if got := tenancy.Tenant(context.Background()); got != nil {
		t.Errorf("Tenant() = %v, want nil", got)
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
