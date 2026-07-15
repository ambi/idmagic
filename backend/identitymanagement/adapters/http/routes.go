// Package http: identity management bounded context の HTTP アダプタ。
//
// ユーザー・グループ・エージェントの管理 API と、エンドユーザー自身の
// profile / email / data export の self-service API を所有する。
package http

import (
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	scimports "github.com/ambi/idmagic/backend/scim/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// Deps は identity management HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	UserRepo                 idmports.UserRepository
	GroupRepo                idmports.GroupRepository
	AgentRepo                idmports.AgentRepository
	ClientRepo               oauthports.OAuth2ClientRepository
	ScimRepo                 scimports.ScimRepository
	AttrSchemaRepo           tenantports.TenantUserAttributeSchemaRepository
	ConsentRepo              oauthports.ConsentRepository
	RefreshStore             oauthports.RefreshTokenStore
	DeviceCodeStore          oauthports.DeviceCodeStore
	MfaFactorRepo            authnports.MfaFactorRepository
	PasswordHasher           authnports.PasswordHasher
	PasswordHistoryRepo      authnports.PasswordHistoryRepository
	EmailChangeTokenStore    authnports.EmailChangeTokenStore
	EmailSender              authnports.EmailSender
	JobRepo                  jobsports.JobRepository
	LifecycleWorkflowRepo    idmports.LifecycleWorkflowRepository
	LifecycleWorkflowRunRepo idmports.LifecycleWorkflowRunRepository
	UserWorkflowCapture      idmports.UserWorkflowCapture
}

func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/account/summary", d.handleGetAccountSummary)
	g.GET("/api/account/profile", d.handleGetAccountProfile)
	g.PATCH("/api/account/profile", d.handleUpdateAccountProfile)
	g.POST("/api/account/email/change_request", d.handleRequestEmailChange)
	g.GET("/api/account/email/verify_context", d.handleEmailVerifyContext)
	g.POST("/api/account/email/verify", d.handleConfirmEmailChange)
	g.GET("/api/account/data_export", d.handleExportAccountData)
	g.GET("/api/admin/users", d.handleListAdminUsers)
	g.GET("/api/admin/users/:sub", d.handleGetAdminUser)
	g.POST("/api/admin/users", d.handleCreateAdminUser)
	g.POST("/api/admin/users/imports", d.handleImportAdminUsers)
	g.GET("/api/admin/users/imports/:job_id", d.handleGetAdminUserImport)
	g.PATCH("/api/admin/users/:sub", d.handleUpdateAdminUser)
	g.POST("/api/admin/users/:sub/disable", d.handleDisableAdminUser)
	g.POST("/api/admin/users/:sub/enable", d.handleEnableAdminUser)
	g.DELETE("/api/admin/users/:sub", d.handleDeleteAdminUser)
	g.POST("/api/admin/users/:sub/restore", d.handleRestoreAdminUser)
	g.POST("/api/admin/users/:sub/required_actions", d.handleSetUserRequiredAction)
	g.DELETE("/api/admin/users/:sub/required_actions/:action", d.handleClearUserRequiredAction)
	g.GET("/api/admin/users/:sub/groups", d.handleListUserGroups)
	g.GET("/api/admin/groups", d.handleListGroups)
	g.GET("/api/admin/groups/:group_id", d.handleGetGroup)
	g.POST("/api/admin/groups", d.handleCreateGroup)
	g.PATCH("/api/admin/groups/:group_id", d.handleUpdateGroup)
	g.DELETE("/api/admin/groups/:group_id", d.handleDeleteGroup)
	g.PUT("/api/admin/groups/:group_id/dynamic-rule", d.handleUpdateDynamicGroupRule)
	g.POST("/api/admin/groups/:group_id/dynamic-rule/preview", d.handlePreviewDynamicGroupRule)
	g.POST("/api/admin/groups/:group_id/dynamic-rule/enable", d.handleEnableDynamicGroupRule)
	g.POST("/api/admin/groups/:group_id/dynamic-rule/disable", d.handleDisableDynamicGroupRule)
	g.POST("/api/admin/groups/:group_id/members/:user_sub", d.handleAddGroupMember)
	g.DELETE("/api/admin/groups/:group_id/members/:user_sub", d.handleRemoveGroupMember)
	g.GET("/api/admin/agents", d.handleListAgents)
	g.GET("/api/admin/agents/:agent_id", d.handleGetAgent)
	g.POST("/api/admin/agents", d.handleRegisterAgent)
	g.PATCH("/api/admin/agents/:agent_id", d.handleUpdateAgent)
	g.POST("/api/admin/agents/:agent_id/disable", d.handleDisableAgent)
	g.POST("/api/admin/agents/:agent_id/enable", d.handleEnableAgent)
	g.POST("/api/admin/agents/:agent_id/kill", d.handleKillAgent)
	g.DELETE("/api/admin/agents/:agent_id", d.handleDeleteAgent)
	g.POST("/api/admin/agents/:agent_id/credentials", d.handleBindAgentCredential)
	g.DELETE("/api/admin/agents/:agent_id/credentials/:client_id", d.handleUnbindAgentCredential)
	g.GET("/api/admin/lifecycle_workflows", d.handleListLifecycleWorkflows)
	g.GET("/api/admin/lifecycle_workflows/:workflow_id", d.handleGetLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows", d.handleCreateLifecycleWorkflow)
	g.PUT("/api/admin/lifecycle_workflows/:workflow_id", d.handleUpdateLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows/:workflow_id/enable", d.handleEnableLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows/:workflow_id/disable", d.handleDisableLifecycleWorkflow)
	g.DELETE("/api/admin/lifecycle_workflows/:workflow_id", d.handleArchiveLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows/:workflow_id/dry_run", d.handleDryRunLifecycleWorkflow)
	g.GET("/api/admin/lifecycle_workflows/:workflow_id/runs", d.handleListLifecycleWorkflowRuns)
	g.GET("/api/admin/lifecycle_workflow_runs/:run_id", d.handleGetLifecycleWorkflowRun)
	g.POST("/api/admin/lifecycle_workflow_runs/:run_id/retry", d.handleRetryLifecycleWorkflowRun)
}

func (d Deps) ConsentDeps() oauthusecases.ConsentDeps {
	return oauthusecases.ConsentDeps{ConsentRepo: d.ConsentRepo, Emit: d.Emit}
}

// legacyEmit adapts the fire-and-forget support.Deps.Emit to the
// error-returning signature usecases in this context require (wi-184 T003).
// It is the default for handlers not yet migrated to the transaction
// runner; migrated handlers (admin_user_handler.go Create/Update/
// SetDisabled) override deps.Emit with a transaction-bound one instead.
func (d Deps) legacyEmit() func(spec.DomainEvent) error {
	return func(event spec.DomainEvent) error {
		if d.Emit != nil {
			d.Emit(event)
		}
		return nil
	}
}
