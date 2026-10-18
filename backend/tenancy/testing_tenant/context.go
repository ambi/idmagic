// Package testing_tenant は、テストが対象のテナントを文脈へ載せるための補助である。
//
// 本番ではテナントの解決はミドルウェアかジョブの実行が担い、テナントのない文脈から
// tenantports.TenantID を読むと panic する。ユースケースやリポジトリを直接呼ぶテストは、
// 解決を経ないので、ここで文脈へテナントを入れる。
package testing_tenant

import (
	"context"
	"strings"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

// Default は ctx へ default テナントの ID だけを入れた文脈を返す。
func Default(ctx context.Context) context.Context {
	return tenantports.WithTenantID(ctx, domain.DefaultTenantID)
}

// ResolveDefault は、テナントを解決するミドルウェアの代わりに、要求の文脈へ default テナントを入れる。
// テナントの解決を経ずにハンドラーを登録するテストが使う。
func ResolveDefault(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		c.SetRequest(c.Request().WithContext(Default(c.Request().Context())))
		return next(c)
	}
}

// ResolveDefaultAt はルーターを組み立てない HTTP テストに default の正規位置を渡す。
func ResolveDefaultAt(issuer string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			const prefix = "/realms/default"
			tenant := &domain.Tenant{ID: domain.DefaultTenantID, Realm: domain.DefaultRealm, Status: domain.TenantStatusActive, EndpointStyle: domain.TenantEndpointStylePath}
			ctx := tenantports.WithTenant(c.Request().Context(), tenant, strings.TrimSuffix(issuer, "/")+prefix, prefix)
			c.SetRequest(c.Request().WithContext(ctx))
			c.Response().Header().Add("Vary", "Host")
			return next(c)
		}
	}
}
