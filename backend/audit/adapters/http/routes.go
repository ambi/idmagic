// Package http: audit コンテキストの HTTP アダプタ。
//
// 管理者向けの監査イベント検索 / 参照 / エクスポート API
// (ListAdminAuditEvents / GetAdminAuditEvent / ExportAdminAuditEvents) を所有する。
// 共有基盤 support.Deps を受け取り router から登録される。
package http

import (
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

// Deps は Audit HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	AuditEventRepo  auditports.AuditEventRepository
	TenantSaltStore auditports.TenantSaltStore
}

// RegisterRoutes はテナント解決済みグループに audit コンテキストのエンドポイントを
// 登録する。パス・メソッド・middleware は分割前と一致する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/audit_events", d.handleListAdminAuditEvents)
	g.GET("/api/admin/audit_events/export", d.handleExportAdminAuditEvents)
	g.GET("/api/admin/audit_events/:id", d.handleGetAdminAuditEvent)
}
