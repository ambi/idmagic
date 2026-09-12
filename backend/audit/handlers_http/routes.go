// Package http: audit コンテキストの HTTP アダプタ。
//
// 管理者向けの監査イベント検索 / 参照 / エクスポート API
// (ListAdminAuditEvents / GetAdminAuditEvent / ExportAdminAuditEvents) と、その制御面の
// 双子 (ListSystemAuditEvents / GetSystemAuditEvent / ExportSystemAuditEvents) を所有する。
// 共有基盤 support.Deps を受け取り router から登録される。
package handlers_http

import (
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// Deps は Audit HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	AuditEventRepo  auditports.AuditEventRepository
	TenantSaltStore auditports.TenantSaltStore
	// UserRepo は username -> user_id 解決 (wi-147) に使う。実アカウントが常に確定する
	// イベントの検索は、payload に username/hash を持たせず検索時にここで解決する。
	UserRepo userports.UserRepository
}

// RegisterRoutes はテナント解決済みグループに audit コンテキストのエンドポイントを
// 登録する。
//
// テナント管理経路とシステム経路は別のパスに分かれ、どちらを呼んだかがテナントの範囲を
// 決める。同じハンドラーへ合流させて範囲をロールから推定する形にはしない。
// 検索選択肢だけは 1 本を両方が呼ぶ。返すのは検索軸の語彙で、どのテナントの記録も
// 含まないためである。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/v1/audit_events", d.handleListAdminAuditEvents)
	g.GET("/api/admin/v1/audit_events/export", d.handleExportAdminAuditEvents)
	g.GET("/api/admin/v1/audit_events/search_options", d.handleAdminAuditEventSearchOptions)
	g.GET("/api/admin/v1/audit_events/:id", d.handleGetAdminAuditEvent)

	g.GET("/api/admin/v1/system/audit_events", d.handleListSystemAuditEvents)
	g.GET("/api/admin/v1/system/audit_events/export", d.handleExportSystemAuditEvents)
	g.GET("/api/admin/v1/system/audit_events/:id", d.handleGetSystemAuditEvent)
}
