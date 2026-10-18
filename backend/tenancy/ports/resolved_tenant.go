package ports

import (
	"context"
	"strings"

	"github.com/ambi/idmagic/backend/shared/tenantcontext"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

type resolvedTenant struct {
	tenant *domain.Tenant
	issuer string
	prefix string
}

func (r resolvedTenant) TenantID() string {
	if r.tenant == nil {
		return ""
	}
	return r.tenant.ID
}

func (r resolvedTenant) HTTPLocation() (string, string, bool) {
	return r.issuer, r.prefix, r.tenant != nil && r.tenant.EffectiveEndpointStyle() == domain.TenantEndpointStyleSubdomain
}

func WithTenant(ctx context.Context, tenant *domain.Tenant, issuer, urlPrefix string) context.Context {
	return tenantcontext.WithScope(ctx, resolvedTenant{tenant: tenant, issuer: strings.TrimSuffix(issuer, "/"), prefix: strings.TrimSuffix(urlPrefix, "/")})
}

// WithTenantID は設定を解決しない背景処理の対象 ID だけを運ぶ。
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return tenantcontext.WithID(ctx, tenantID)
}

func Tenant(ctx context.Context) *domain.Tenant {
	resolved, _ := tenantcontext.Current(ctx).(resolvedTenant)
	return resolved.tenant
}

// TenantID は解決済みまたは ID だけの文脈から対象のテナントの ID を返す。
// テナントのない文脈がテナントに属する処理へ届くのは配線の誤りなので、default テナントへ
// 落とさずに panic する。落とすと、その処理は default テナントのデータを読み書きする。
func TenantID(ctx context.Context) string {
	return tenantcontext.TenantID(ctx)
}

func Issuer(ctx context.Context, fallback string) string {
	return tenantcontext.Issuer(ctx, fallback)
}

// URLPrefix は middleware が解決した URL prefix (`/realms/{id}` または空文字) を返す。
// cookie path や redirect URL の組み立てに使う。
func URLPrefix(ctx context.Context) string {
	return tenantcontext.URLPrefix(ctx)
}
