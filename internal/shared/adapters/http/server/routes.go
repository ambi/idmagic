// Package server: Echo v5 を用いた HTTP アダプタの router。
//
// 依存集約 (support.Deps) とテナント解決 middleware は support パッケージが持ち、
// 各エンドポイントのハンドラは責務ごとに *_handler.go へ分割している。
// このファイルではルーティング登録 (Register) のみを定義する。
package server

import (
	apphttp "idmagic/internal/application/adapters/http"
	authhttp "idmagic/internal/authentication/adapters/http"
	idmhttp "idmagic/internal/identitymanagement/adapters/http"
	oauth2http "idmagic/internal/oauth2/adapters/http"
	samlhttp "idmagic/internal/saml/adapters/http"
	"idmagic/internal/scim"
	"idmagic/internal/shared/adapters/http/support"
	"idmagic/internal/shared/spec"
	tenancyhttp "idmagic/internal/tenancy/adapters/http"
	wsfederationhttp "idmagic/internal/wsfederation/adapters/http"

	"github.com/labstack/echo/v5"
)

// Deps は support.Deps を埋め込む薄いラッパ。ハンドラを所有コンテキストの
// メソッドとして保持するためのキャリアで、固有のフィールドは持たない。
type Deps struct {
	*support.Deps
}

func Register(e *echo.Echo, cd support.Deps) {
	d := Deps{&cd}
	registerTenantRoutes(e.Group("", d.ResolveDefaultTenant), d)
	registerTenantRoutes(e.Group("/realms/:tenant_id", d.ResolvePathTenant), d)
	// テナント CRUD は system_admin 専用のテナント横断操作 (各 handler が
	// requireSystemAdmin でゲート)。`/api/admin/tenants` として、control-plane
	// グループ (/realms/default 配下、ADR-032 の cookie path 整合) と bare group の
	// 両方に登録する。システムコンソール (/system, bare base path, Bearer 認証) は
	// bare の `/api/admin/tenants` を叩く。どちらも default tenant を解決し同じ
	// handler に入る。
	controlPlane := e.Group("/realms/"+spec.DefaultTenantID, d.ResolveControlPlaneTenant)
	tenancyhttp.RegisterControlPlaneRoutes(controlPlane, d.Deps)
	tenancyhttp.RegisterControlPlaneRoutes(e.Group("", d.ResolveDefaultTenant), d.Deps)
	e.GET("/health", d.handleHealth)
	e.GET("/livez", d.handleLivez)
	e.GET("/readyz", d.handleReadyz)
	e.GET("/startupz", d.handleStartupz)
}

func registerTenantRoutes(g *echo.Group, d Deps) {
	oauth2http.RegisterRoutes(g, d.Deps)
	authhttp.RegisterRoutes(g, d.Deps)
	idmhttp.RegisterRoutes(g, d.Deps)
	tenancyhttp.RegisterRoutes(g, d.Deps)
	wsfederationhttp.RegisterRoutes(g, d.Deps)
	samlhttp.RegisterRoutes(g, d.Deps)
	apphttp.RegisterRoutes(g, d.Deps)

	scimUsecases := scim.NewUsecases(d.ScimRepo, d.UserRepo, d.GroupRepo, d.Emit)
	scim.RegisterRoutes(g, d.Deps, scimUsecases)
}
