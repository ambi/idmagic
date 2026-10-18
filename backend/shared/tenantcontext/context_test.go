package tenantcontext_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/shared/tenantcontext"
)

type httpScope struct{ id string }

func (s httpScope) TenantID() string { return s.id }

func (httpScope) HTTPLocation() (string, string, bool) {
	return "https://tenant.example", "/realms/tenant", true
}

func TestScopeCarriesIDAndOptionalLocation(t *testing.T) {
	ctx := tenantcontext.WithScope(context.Background(), httpScope{id: "http"})
	if got := tenantcontext.TenantID(ctx); got != "http" {
		t.Fatalf("TenantID() = %q, want http", got)
	}
	issuer, prefix, secure := tenantcontext.HTTPLocation(ctx)
	if issuer != "https://tenant.example" || prefix != "/realms/tenant" || !secure {
		t.Fatalf("HTTPLocation() = (%q, %q, %v)", issuer, prefix, secure)
	}
	if got := tenantcontext.Issuer(ctx, "https://fallback/"); got != issuer {
		t.Fatalf("Issuer() = %q, want %q", got, issuer)
	}
	if got := tenantcontext.URLPrefix(ctx); got != prefix {
		t.Fatalf("URLPrefix() = %q, want %q", got, prefix)
	}
	ctx = tenantcontext.WithID(ctx, "job")
	if got := tenantcontext.TenantID(ctx); got != "job" {
		t.Fatalf("TenantID() = %q, want job", got)
	}
	issuer, prefix, secure = tenantcontext.HTTPLocation(ctx)
	if issuer != "" || prefix != "" || secure {
		t.Fatalf("ID-only context retained HTTP location: (%q, %q, %v)", issuer, prefix, secure)
	}
	if got := tenantcontext.Issuer(ctx, "https://fallback/"); got != "https://fallback" {
		t.Fatalf("Issuer() = %q, want trimmed fallback", got)
	}
}

//spec:covers REQ-TENANCY-006: 未解決または空 ID の文脈からは default テナントへ退避しない。
func TestTenantIDRejectsMissingOrEmptyScope(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"missing":        context.Background(),
		"empty identity": tenantcontext.WithID(context.Background(), ""),
		"empty resolved": tenantcontext.WithScope(context.Background(), httpScope{}),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("TenantID() did not panic")
				}
			}()
			tenantcontext.TenantID(ctx)
		})
	}
}
