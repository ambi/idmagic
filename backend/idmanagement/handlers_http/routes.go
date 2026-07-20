// Package http: identity management bounded context の HTTP アダプタ。
//
// Deps の定義・route 登録の集約点。ユーザー・グループ・エージェントそれぞれの
// ハンドラ実装は user/handlers_http・group/handlers_http・agent/handlers_http に
// feature 垂直スライスとして分割されている (ADR-130 Phase 2)。ハンドラは
// Deps のメソッドではなくフリー関数 (func HandleX(d Deps, c *echo.Context) error)
// として実装されており、Deps 型自体の分割（cross-feature port の重複）を避けている。
package handlers_http

import (
	agenthttp "github.com/ambi/idmagic/backend/idmanagement/agent/handlers_http"
	httpdeps "github.com/ambi/idmagic/backend/idmanagement/deps_http"
	grouphttp "github.com/ambi/idmagic/backend/idmanagement/group/handlers_http"
	userhttp "github.com/ambi/idmagic/backend/idmanagement/user/handlers_http"

	"github.com/labstack/echo/v5"
)

// Deps は identity management HTTP ハンドラが必要とする依存。実体は httpdeps.Deps
// (型 alias)。外部呼び出し元は従来どおり idmhttp.Deps{...} をフラットな field で
// 構築できる。
type Deps = httpdeps.Deps

func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/account/summary", func(c *echo.Context) error { return userhttp.HandleGetAccountSummary(d, c) })
	g.GET("/api/account/profile", func(c *echo.Context) error { return userhttp.HandleGetAccountProfile(d, c) })
	g.PATCH("/api/account/profile", func(c *echo.Context) error { return userhttp.HandleUpdateAccountProfile(d, c) })
	g.POST("/api/account/email/change_request", func(c *echo.Context) error { return userhttp.HandleRequestEmailChange(d, c) })
	g.GET("/api/account/email/verify_context", func(c *echo.Context) error { return userhttp.HandleEmailVerifyContext(d, c) })
	g.POST("/api/account/email/verify", func(c *echo.Context) error { return userhttp.HandleConfirmEmailChange(d, c) })
	g.GET("/api/account/data_export", func(c *echo.Context) error { return userhttp.HandleExportAccountData(d, c) })
	g.GET("/api/admin/users", func(c *echo.Context) error { return userhttp.HandleListAdminUsers(d, c) })
	g.GET("/api/admin/users/:sub", func(c *echo.Context) error { return userhttp.HandleGetAdminUser(d, c) })
	g.POST("/api/admin/users", func(c *echo.Context) error { return userhttp.HandleCreateAdminUser(d, c) })
	g.POST("/api/admin/users/imports", func(c *echo.Context) error { return userhttp.HandleImportAdminUsers(d, c) })
	g.GET("/api/admin/users/imports/:job_id", func(c *echo.Context) error { return userhttp.HandleGetAdminUserImport(d, c) })
	g.PATCH("/api/admin/users/:sub", func(c *echo.Context) error { return userhttp.HandleUpdateAdminUser(d, c) })
	g.POST("/api/admin/users/:sub/disable", func(c *echo.Context) error { return userhttp.HandleDisableAdminUser(d, c) })
	g.POST("/api/admin/users/:sub/enable", func(c *echo.Context) error { return userhttp.HandleEnableAdminUser(d, c) })
	g.DELETE("/api/admin/users/:sub", func(c *echo.Context) error { return userhttp.HandleDeleteAdminUser(d, c) })
	g.POST("/api/admin/users/:sub/restore", func(c *echo.Context) error { return userhttp.HandleRestoreAdminUser(d, c) })
	g.POST("/api/admin/users/:sub/required_actions", func(c *echo.Context) error { return userhttp.HandleSetUserRequiredAction(d, c) })
	g.DELETE("/api/admin/users/:sub/required_actions/:action", func(c *echo.Context) error { return userhttp.HandleClearUserRequiredAction(d, c) })
	g.GET("/api/admin/users/:sub/groups", func(c *echo.Context) error { return grouphttp.HandleListUserGroups(d, c) })
	g.GET("/api/admin/groups", func(c *echo.Context) error { return grouphttp.HandleListGroups(d, c) })
	g.GET("/api/admin/groups/:group_id", func(c *echo.Context) error { return grouphttp.HandleGetGroup(d, c) })
	g.POST("/api/admin/groups", func(c *echo.Context) error { return grouphttp.HandleCreateGroup(d, c) })
	g.PATCH("/api/admin/groups/:group_id", func(c *echo.Context) error { return grouphttp.HandleUpdateGroup(d, c) })
	g.DELETE("/api/admin/groups/:group_id", func(c *echo.Context) error { return grouphttp.HandleDeleteGroup(d, c) })
	g.PUT("/api/admin/groups/:group_id/dynamic-rule", func(c *echo.Context) error { return grouphttp.HandleUpdateDynamicGroupRule(d, c) })
	g.POST("/api/admin/groups/:group_id/dynamic-rule/preview", func(c *echo.Context) error { return grouphttp.HandlePreviewDynamicGroupRule(d, c) })
	g.POST("/api/admin/groups/:group_id/dynamic-rule/enable", func(c *echo.Context) error { return grouphttp.HandleEnableDynamicGroupRule(d, c) })
	g.POST("/api/admin/groups/:group_id/dynamic-rule/disable", func(c *echo.Context) error { return grouphttp.HandleDisableDynamicGroupRule(d, c) })
	g.POST("/api/admin/groups/:group_id/members/:user_sub", func(c *echo.Context) error { return grouphttp.HandleAddGroupMember(d, c) })
	g.DELETE("/api/admin/groups/:group_id/members/:user_sub", func(c *echo.Context) error { return grouphttp.HandleRemoveGroupMember(d, c) })
	g.GET("/api/admin/agents", func(c *echo.Context) error { return agenthttp.HandleListAgents(d, c) })
	g.GET("/api/admin/agents/:agent_id", func(c *echo.Context) error { return agenthttp.HandleGetAgent(d, c) })
	g.POST("/api/admin/agents", func(c *echo.Context) error { return agenthttp.HandleRegisterAgent(d, c) })
	g.PATCH("/api/admin/agents/:agent_id", func(c *echo.Context) error { return agenthttp.HandleUpdateAgent(d, c) })
	g.POST("/api/admin/agents/:agent_id/disable", func(c *echo.Context) error { return agenthttp.HandleDisableAgent(d, c) })
	g.POST("/api/admin/agents/:agent_id/enable", func(c *echo.Context) error { return agenthttp.HandleEnableAgent(d, c) })
	g.POST("/api/admin/agents/:agent_id/kill", func(c *echo.Context) error { return agenthttp.HandleKillAgent(d, c) })
	g.DELETE("/api/admin/agents/:agent_id", func(c *echo.Context) error { return agenthttp.HandleDeleteAgent(d, c) })
	g.POST("/api/admin/agents/:agent_id/credentials", func(c *echo.Context) error { return agenthttp.HandleBindAgentCredential(d, c) })
	g.DELETE("/api/admin/agents/:agent_id/credentials/:client_id", func(c *echo.Context) error { return agenthttp.HandleUnbindAgentCredential(d, c) })
}
