// Package testing_tenant は、テストがテナントを解決済みの文脈を作るための補助である。
//
// 本番ではテナントの解決はミドルウェアかジョブの実行が担い、テナントのない文脈から
// tenantports.TenantID を読むと panic する。ユースケースやリポジトリを直接呼ぶテストは、
// 解決を経ないので、ここで文脈へテナントを入れる。
package testing_tenant

import (
	"context"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

// Default は ctx へ default テナントを入れた文脈を返す。
func Default(ctx context.Context) context.Context {
	return tenantports.WithTenant(ctx, &domain.Tenant{ID: domain.DefaultTenantID}, "", "")
}

// ResolveDefault は、テナントを解決するミドルウェアの代わりに、要求の文脈へ default テナントを入れる。
// テナントの解決を経ずにハンドラーを登録するテストが使う。
func ResolveDefault(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		c.SetRequest(c.Request().WithContext(Default(c.Request().Context())))
		return next(c)
	}
}
