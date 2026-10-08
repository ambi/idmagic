package tenancy

import (
	"context"
	"strings"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

type contextKey string

const (
	tenantKey    contextKey = "tenant"
	issuerKey    contextKey = "tenant-issuer"
	urlPrefixKey contextKey = "tenant-url-prefix"
)

func WithTenant(ctx context.Context, tenant *domain.Tenant, issuer, urlPrefix string) context.Context {
	ctx = context.WithValue(ctx, tenantKey, tenant)
	ctx = context.WithValue(ctx, issuerKey, strings.TrimSuffix(issuer, "/"))
	return context.WithValue(ctx, urlPrefixKey, strings.TrimSuffix(urlPrefix, "/"))
}

func Tenant(ctx context.Context) *domain.Tenant {
	tenant, _ := ctx.Value(tenantKey).(*domain.Tenant)
	return tenant
}

// TenantID は文脈で解決済みのテナントの ID を返す。
// テナントのない文脈がテナントに属する処理へ届くのは配線の誤りなので、default テナントへ
// 落とさずに panic する。落とすと、その処理は default テナントのデータを読み書きする。
func TenantID(ctx context.Context) string {
	if tenant := Tenant(ctx); tenant != nil && tenant.ID != "" {
		return tenant.ID
	}
	panic("tenancy: no resolved tenant in context")
}

func Issuer(ctx context.Context, fallback string) string {
	issuer, _ := ctx.Value(issuerKey).(string)
	if issuer != "" {
		return issuer
	}
	return strings.TrimSuffix(fallback, "/")
}

// URLPrefix は middleware が解決した URL prefix (`/realms/{id}` または空文字) を返す。
// cookie path や redirect URL の組み立てに使う。
func URLPrefix(ctx context.Context) string {
	prefix, _ := ctx.Value(urlPrefixKey).(string)
	return prefix
}
