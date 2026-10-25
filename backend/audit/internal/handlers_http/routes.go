// Package http: audit コンテキストの HTTP アダプタ。
//
// 管理者向けの監査イベント検索 / 参照 / エクスポート API
// (ListAdminAuditEvents / GetAdminAuditEvent / ExportAdminAuditEvents) と、その制御面の
// 双子 (ListSystemAuditEvents / GetSystemAuditEvent / ExportSystemAuditEvents) を所有する。
// 共有基盤 support.Deps を受け取り router から登録される。
package handlers_http

import (
	"github.com/ambi/idmagic/backend/audit/domain"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// Deps は Audit HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	AuditEventRepo  auditports.AuditEventRepository
	TenantSaltStore auditports.TenantSaltStore
	// LookupUsername は username -> user_id 解決に使う。実アカウントが常に確定する
	// イベントの検索は、payload に username/hash を持たせず検索時にここで解決する。
	LookupUsername auditports.UsernameLookup
	Exports        usecases.ExportDeps
	ExportPolicy   domain.ExportPolicy
}

// RegisterRoutes はテナント解決済みグループに audit コンテキストのエンドポイントを
// 登録する。
//
// テナント管理経路とシステム経路は別のパスに分かれ、どちらを呼んだかがテナントの範囲を
// 決める。同じハンドラーへ合流させて範囲をロールから推定する形にはしない。
// 検索選択肢だけは 1 本を両方が呼ぶ。返すのは検索軸の語彙で、どのテナントの記録も
// 含まないためである。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.POST("/api/admin/v1/audit-events/exports", d.startAdminExport)
	g.GET("/api/admin/v1/audit-events/exports", d.listAdminExports)
	g.GET("/api/admin/v1/audit-events/exports/:export_id", d.getAdminExport)
	g.POST("/api/admin/v1/audit-events/exports/:export_id/cancel", d.cancelAdminExport)
	g.GET("/api/admin/v1/audit-events/exports/:export_id/download", d.downloadAdminExport)
	g.POST("/api/admin/v1/system/audit-events/exports", d.startSystemExport)
	g.GET("/api/admin/v1/system/audit-events/exports", d.listSystemExports)
	g.GET("/api/admin/v1/system/audit-events/exports/:export_id", d.getSystemExport)
	g.POST("/api/admin/v1/system/audit-events/exports/:export_id/cancel", d.cancelSystemExport)
	g.GET("/api/admin/v1/system/audit-events/exports/:export_id/download", d.downloadSystemExport)
	g.GET("/api/admin/v1/audit-events", d.handleListAdminAuditEvents)
	g.GET("/api/admin/v1/audit-events/export", d.handleExportAdminAuditEvents)
	g.GET("/api/admin/v1/audit-events/search-options", d.handleAdminAuditEventSearchOptions)
	g.GET("/api/admin/v1/audit-events/:id", d.handleGetAdminAuditEvent)

	g.GET("/api/admin/v1/system/audit-events", d.handleListSystemAuditEvents)
	g.GET("/api/admin/v1/system/audit-events/export", d.handleExportSystemAuditEvents)
	g.GET("/api/admin/v1/system/audit-events/:id", d.handleGetSystemAuditEvent)
}
