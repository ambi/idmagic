// Package http: identity governance bounded context の HTTP アダプタ。
//
// LifecycleWorkflow (JML 自動化) の管理 API (/api/admin/lifecycle_workflows) を
// 所有する。管理者認証・CSRF・エラー整形は shared/adapters/http/support を再利用し、
// dry-run が参照する User/Group/Application は record context の port を注入で受け取る。
package http

import (
	appports "github.com/ambi/idmagic/backend/application/ports"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification"

	"github.com/labstack/echo/v5"
)

// Deps は identity governance HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	LifecycleWorkflowRepo    igports.LifecycleWorkflowRepository
	LifecycleWorkflowRunRepo igports.LifecycleWorkflowRunRepository
	JobRepo                  jobsports.JobRepository
	// UserRepo/GroupRepo and ApplicationRepo/AssignmentRepo/EmailSender are only
	// used by dry-run (DryRunLifecycleWorkflow, wi-222) to evaluate actions'
	// current state without mutating record contexts.
	UserRepo        userports.UserRepository
	GroupRepo       groupports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AssignmentRepo  appports.AssignmentRepository
	EmailSender     sharednotification.EmailSender
}

// RegisterRoutes wires the 11 lifecycle workflow admin endpoints.
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/lifecycle_workflows", d.handleListLifecycleWorkflows)
	g.GET("/api/admin/lifecycle_workflows/:workflow_id", d.handleGetLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows", d.handleCreateLifecycleWorkflow)
	g.PUT("/api/admin/lifecycle_workflows/:workflow_id", d.handleUpdateLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows/:workflow_id/enable", d.handleEnableLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows/:workflow_id/disable", d.handleDisableLifecycleWorkflow)
	g.DELETE("/api/admin/lifecycle_workflows/:workflow_id", d.handleDeleteLifecycleWorkflow)
	g.POST("/api/admin/lifecycle_workflows/:workflow_id/dry_run", d.handleDryRunLifecycleWorkflow)
	g.GET("/api/admin/lifecycle_workflows/:workflow_id/runs", d.handleListLifecycleWorkflowRuns)
	g.GET("/api/admin/lifecycle_workflow_runs/:run_id", d.handleGetLifecycleWorkflowRun)
	g.POST("/api/admin/lifecycle_workflow_runs/:run_id/retry", d.handleRetryLifecycleWorkflowRun)
}
