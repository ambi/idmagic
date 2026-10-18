package idgovernance

import (
	"context"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	"github.com/ambi/idmagic/backend/idgovernance/internal/db_memory"
	"github.com/ambi/idmagic/backend/idgovernance/internal/db_postgres"
	"github.com/ambi/idmagic/backend/idgovernance/internal/handlers_http"
	"github.com/ambi/idmagic/backend/idgovernance/internal/ports"
	"github.com/ambi/idmagic/backend/idgovernance/internal/usecases"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
)

func NewMemoryModule(users userports.UserRepository) Module {
	workflows := db_memory.NewLifecycleWorkflowRepository()
	runs := db_memory.NewLifecycleWorkflowRunRepository()
	capture := &db_memory.UserWorkflowCapture{Users: users, Runs: runs}
	return Module{
		LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs, UserWorkflowCapture: capture,
		UserMutationCommitter: usecases.UserMutationCommitter{WorkflowRepo: workflows, Capture: capture, UserRepo: users, RunRepo: runs},
	}
}

func NewPostgresModule(pool postgres.DB, users userports.UserRepository, usersInTx func(pgx.Tx) userports.UserWriter) Module {
	workflows := &db_postgres.LifecycleWorkflowRepository{Pool: pool}
	runs := &db_postgres.LifecycleWorkflowRunRepository{Pool: pool}
	capture := db_postgres.NewUserWorkflowCapture(pool, usersInTx)
	return Module{
		LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs, UserWorkflowCapture: capture,
		UserMutationCommitter: usecases.UserMutationCommitter{WorkflowRepo: workflows, Capture: capture, UserRepo: users, RunRepo: runs},
	}
}

type RouteDeps struct {
	support.Deps
	*support.Authenticator
	JobEnqueuer     jobsports.Enqueuer
	UserRepo        userports.UserRepository
	GroupRepo       groupports.GroupRepository
	ApplicationRepo appports.ApplicationRepository
	AssignmentRepo  appports.AssignmentRepository
	Notifier        notificationports.Notifier
	AttrSchemaRepo  userports.TenantUserAttributeSchemaRepository
}

func (m Module) RegisterRoutes(g *echo.Group, d RouteDeps) {
	handlers_http.RegisterRoutes(g, handlers_http.Deps{
		Deps: d.Deps, Authenticator: d.Authenticator,
		LifecycleWorkflowRepo: m.LifecycleWorkflowRepo, LifecycleWorkflowRunRepo: m.LifecycleWorkflowRunRepo,
		JobEnqueuer: d.JobEnqueuer, UserRepo: d.UserRepo, GroupRepo: d.GroupRepo,
		ApplicationRepo: d.ApplicationRepo, AssignmentRepo: d.AssignmentRepo, Notifier: d.Notifier, AttrSchemaRepo: d.AttrSchemaRepo,
	})
}

type ExecutorDeps struct {
	UserRepo               userports.UserRepository
	GroupRepo              groupports.GroupRepository
	ApplicationRepo        appports.ApplicationRepository
	AssignmentRepo         appports.AssignmentRepository
	ApplicationAssignments ports.ApplicationAssignments
	UserLifecycle          ports.UserLifecycle
	GroupMembership        ports.GroupMembership
	UserRequiredActions    ports.UserRequiredActions
	Notifier               notificationports.Notifier
	Emit                   func(spec.DomainEvent) error
}

const LifecycleWorkflowRunJobKind = usecases.LifecycleWorkflowRunJobKind

func (m Module) Executor(d ExecutorDeps) jobsports.Handler {
	return usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{
		RunRepo: m.LifecycleWorkflowRunRepo, WorkflowRepo: m.LifecycleWorkflowRepo,
		UserRepo: d.UserRepo, GroupRepo: d.GroupRepo, ApplicationRepo: d.ApplicationRepo, AssignmentRepo: d.AssignmentRepo,
		ApplicationAssignments: d.ApplicationAssignments, UserLifecycle: d.UserLifecycle,
		GroupMembership: d.GroupMembership, UserRequiredActions: d.UserRequiredActions, Notifier: d.Notifier, Emit: d.Emit,
	})
}

func (m Module) DispatchQueuedRuns(ctx context.Context, jobs jobsports.Enqueuer, limit int, now time.Time) error {
	return usecases.DispatchQueuedLifecycleWorkflowRuns(ctx, usecases.LifecycleWorkflowDispatcherDeps{RunRepo: m.LifecycleWorkflowRunRepo, Jobs: jobs}, limit, now)
}
