package ports_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/shared/tenantcontext"
	"github.com/ambi/idmagic/backend/tenancy/domain"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// WithTenant で格納したテナントが Tenant / TenantID から取り出せる。
func TestWithTenantAndAccessors(t *testing.T) {
	tenant := &domain.Tenant{ID: "acme"}
	ctx := tenantports.WithTenant(context.Background(), tenant, "https://acme.example.com/", "/realms/acme/")

	if got := tenantports.Tenant(ctx); got == nil || got.ID != "acme" {
		t.Fatalf("Tenant() = %v, want id=acme", got)
	}
	if got := tenantports.TenantID(ctx); got != "acme" {
		t.Errorf("TenantID() = %q, want acme", got)
	}
	// issuer は末尾スラッシュを除去して格納される。
	if got := tenantports.Issuer(ctx, "https://fallback"); got != "https://acme.example.com" {
		t.Errorf("Issuer() = %q, want trimmed issuer", got)
	}
	if got := tenantports.URLPrefix(ctx); got != "/realms/acme" {
		t.Errorf("URLPrefix() = %q, want /realms/acme", got)
	}
}

//spec:covers REQ-TENANCY-006: テナントを解決していない文脈では default テナントの ID を返さず panic する。
func TestTenantIDPanicsWithoutResolvedTenant(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"テナントがない": context.Background(),
		"ID が空":   tenantports.WithTenant(context.Background(), &domain.Tenant{ID: ""}, "", ""),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("TenantID() did not panic")
				}
			}()
			got := tenantports.TenantID(ctx)
			t.Errorf("TenantID() = %q, want panic", got)
		})
	}
}

// Tenant はテナントのない文脈で nil を返す。
func TestTenantNilWithoutResolvedTenant(t *testing.T) {
	if got := tenantports.Tenant(context.Background()); got != nil {
		t.Errorf("Tenant() = %v, want nil", got)
	}
}

// Issuer は context に issuer が無い場合 fallback を末尾スラッシュ除去して返す。
func TestIssuerFallback(t *testing.T) {
	ctx := context.Background()
	if got := tenantports.Issuer(ctx, "https://fallback.example.com/"); got != "https://fallback.example.com" {
		t.Errorf("Issuer() fallback = %q, want trimmed fallback", got)
	}
}

// URLPrefix は未設定なら空文字を返す。
func TestURLPrefixEmpty(t *testing.T) {
	if got := tenantports.URLPrefix(context.Background()); got != "" {
		t.Errorf("URLPrefix() = %q, want empty", got)
	}
}

//spec:covers REQ-TENANCY-006: ID だけの文脈でも対象テナントの ID を保ち、空 ID を default に補わない。
func TestTenantIDOnlyContext(t *testing.T) {
	ctx := tenantports.WithTenantID(context.Background(), "job-tenant")
	if got := tenantports.TenantID(ctx); got != "job-tenant" {
		t.Fatalf("TenantID() = %q, want job-tenant", got)
	}
	if got := tenantports.Tenant(ctx); got != nil {
		t.Fatalf("Tenant() = %v, want nil", got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("empty tenant ID did not panic")
		}
	}()
	tenantports.TenantID(tenantports.WithTenantID(ctx, ""))
}

func TestTenantIDOnlyContextReplacesResolvedSettings(t *testing.T) {
	resolved := tenantports.WithTenant(context.Background(), &domain.Tenant{ID: "http-tenant"}, "https://http.example", "/realms/http")
	ctx := tenantports.WithTenantID(resolved, "job-tenant")
	if got := tenantports.TenantID(ctx); got != "job-tenant" {
		t.Fatalf("TenantID() = %q, want job-tenant", got)
	}
	if got := tenantports.Tenant(ctx); got != nil {
		t.Fatalf("Tenant() = %v, want nil after replacement", got)
	}
	if got := tenantports.Issuer(ctx, "https://fallback/"); got != "https://fallback" {
		t.Fatalf("Issuer() = %q, want fallback", got)
	}
	if got := tenantports.URLPrefix(ctx); got != "" {
		t.Fatalf("URLPrefix() = %q, want empty", got)
	}
	ctx = tenantports.WithTenant(ctx, &domain.Tenant{ID: "next-http"}, "https://next.example", "/realms/next")
	if got := tenantports.TenantID(ctx); got != "next-http" {
		t.Fatalf("TenantID() = %q, want next-http", got)
	}
	if got := tenantports.Tenant(ctx); got == nil || got.ID != "next-http" {
		t.Fatalf("Tenant() = %v, want next-http settings", got)
	}
}

//spec:covers REQ-TENANCY-007: subdomain の Cookie 属性を共有側へ渡し、path と未解決の値に混ぜない。
func TestResolvedTenantPublishesHTTPLocation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		tenant       *domain.Tenant
		wantHostOnly bool
	}{
		{"subdomain", &domain.Tenant{ID: "sub", EndpointStyle: domain.TenantEndpointStyleSubdomain}, true},
		{"path", &domain.Tenant{ID: "path", EndpointStyle: domain.TenantEndpointStylePath}, false},
		{"missing", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tenantports.WithTenant(context.Background(), tc.tenant, "https://issuer/", "/prefix/")
			issuer, prefix, hostOnly := tenantcontext.HTTPLocation(ctx)
			if issuer != "https://issuer" || prefix != "/prefix" || hostOnly != tc.wantHostOnly {
				t.Fatalf("HTTPLocation() = (%q, %q, %v), want (https://issuer, /prefix, %v)", issuer, prefix, hostOnly, tc.wantHostOnly)
			}
		})
	}
}
