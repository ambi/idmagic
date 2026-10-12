package provisioning

import (
	"context"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/provisioning/internal/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/internal/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/provisioning/internal/source_idmanagement"
	"github.com/ambi/idmagic/backend/provisioning/internal/usecases"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

func NewMemoryModule() Module {
	return Module{ConnectionRepo: db_memory.NewProvisioningConnectionRepository(), RemoteLinkRepo: db_memory.NewRemoteResourceLinkRepository(), TaskRepo: db_memory.NewProvisioningTaskRepository()}
}

func NewPostgresModule(pool postgres.DB) Module {
	return Module{ConnectionRepo: &db_postgres.ProvisioningConnectionRepository{Pool: pool}, RemoteLinkRepo: &db_postgres.RemoteResourceLinkRepository{Pool: pool}, TaskRepo: &db_postgres.ProvisioningTaskRepository{Pool: pool}}
}

// TaskHandler は User と Group の属性、所属の両方を同じ記録の正から読み取る。
func (m Module) TaskHandler(users userports.UserRepository, groups groupports.GroupRepository, newClient func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error), emit func(spec.DomainEvent), notifier notificationports.Notifier) jobsports.Handler {
	attributes := source_idmanagement.CombinedAttributeSource{
		User:  &source_idmanagement.UserAttributeSource{UserRepo: users},
		Group: &source_idmanagement.GroupAttributeSource{GroupRepo: groups},
	}
	members := &source_idmanagement.GroupMemberSource{GroupRepo: groups}
	return Handler(m.JobHandlerDeps(attributes, members, newClient, emit, notifier))
}

func (m Module) Dispatcher(jobs jobsports.Enqueuer, emit func(spec.DomainEvent)) func(context.Context, int, time.Time) (int, error) {
	deps := m.DispatcherDeps(jobs, emit)
	return func(ctx context.Context, limit int, now time.Time) (int, error) {
		return usecases.DispatchPendingTasks(ctx, deps, limit, now)
	}
}

func (m Module) Reconciler(users userports.UserRepository, assignments appports.AssignmentRepository, emit func(spec.DomainEvent), notifier notificationports.Notifier) func(context.Context, int, time.Time) (int, error) {
	deps := m.ReconcileDeps(users, assignments, emit, notifier)
	return func(ctx context.Context, limit int, now time.Time) (int, error) {
		return usecases.ReconcileConnections(ctx, deps, limit, now)
	}
}
