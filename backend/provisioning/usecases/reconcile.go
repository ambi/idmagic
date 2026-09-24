package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ReconcileDeps は ReconcileConnections の依存である。
type ReconcileDeps struct {
	ConnectionRepo ports.ProvisioningConnectionRepository
	TaskRepo       ports.ProvisioningTaskRepository
	LinkRepo       ports.RemoteResourceLinkRepository
	// AssignmentRepo は assigned_only の接続のスコープを決める。nil なら、どの User も割り当てられていないとみなす。
	AssignmentRepo appports.AssignmentRepository
	UserRepo       userports.UserRepository
}

// ReconcileConnections は全テナントの有効な接続を照合し、作ったプロビジョニングタスクと予約の数を返す。
// 1 接続の失敗はほかの接続の照合を止めず、まとめて返す。limit は 1 接続につき 1 回で作る数の上限である。
func ReconcileConnections(ctx context.Context, deps ReconcileDeps, limit int, now time.Time) (int, error) {
	tenants, err := deps.ConnectionRepo.ListTenantsWithActiveConnections(ctx)
	if err != nil {
		return 0, err
	}
	created := 0
	var errs []error
	for _, tenantID := range tenants {
		connections, err := deps.ConnectionRepo.ListAll(ctx, tenantID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, conn := range connections {
			if conn.Status != domain.ConnectionActive || conn.Health == domain.HealthQuarantined {
				continue
			}
			n, err := reconcileConnection(ctx, deps, *conn, limit, now)
			created += n
			if err != nil {
				errs = append(errs, fmt.Errorf("provisioning: reconcile connection %s: %w", conn.ApplicationID, err))
			}
		}
	}
	return created, errors.Join(errs...)
}

func reconcileConnection(ctx context.Context, deps ReconcileDeps, conn domain.ProvisioningConnection, limit int, now time.Time) (int, error) {
	in, err := readReconcileInput(ctx, deps, conn)
	if err != nil {
		return 0, err
	}
	in.Limit = limit
	created := 0
	for _, action := range domain.PlanReconciliation(in) {
		ok, err := saveReconcileAction(ctx, deps, conn, action, now)
		if err != nil {
			return created, err
		}
		if ok {
			created++
		}
	}
	return created, nil
}

// readReconcileInput は、接続のスコープ内の User と、リンクだけが残る User を読む。
func readReconcileInput(ctx context.Context, deps ReconcileDeps, conn domain.ProvisioningConnection) (domain.ReconcileInput, error) {
	tenantID := conn.TenantID
	links, err := deps.LinkRepo.ListByConnection(ctx, tenantID, conn.ApplicationID, domain.SourceTypeUser)
	if err != nil {
		return domain.ReconcileInput{}, err
	}
	tasks, err := deps.TaskRepo.ListUnsettledByConnection(ctx, tenantID, conn.ApplicationID, domain.SourceTypeUser)
	if err != nil {
		return domain.ReconcileInput{}, err
	}
	assigned, err := assignedUserIDs(ctx, deps, conn)
	if err != nil {
		return domain.ReconcileInput{}, err
	}
	users, err := deps.UserRepo.FindAll(ctx, tenantID)
	if err != nil {
		return domain.ReconcileInput{}, err
	}

	in := domain.ReconcileInput{
		Connection: conn,
		Links:      make(map[string]domain.RemoteResourceLink, len(links)),
		Tasks:      map[string][]domain.ReconcileTask{},
	}
	seen := make(map[string]bool, len(users))
	for _, user := range users {
		seen[user.ID] = true
		in.Users = append(in.Users, domain.ReconcileUser{
			ID: user.ID, Active: user.IsActive(), Version: user.UpdatedAt.UnixNano(),
			InScope: conn.Scope == domain.ScopeAllUsers || assigned[user.ID],
		})
	}
	for _, link := range links {
		in.Links[link.SourceID] = *link
		if seen[link.SourceID] {
			continue
		}
		user, err := deps.UserRepo.FindBySubIncludingDeleted(ctx, link.SourceID)
		if err != nil {
			return domain.ReconcileInput{}, err
		}
		if user != nil && user.TenantID != tenantID {
			// FindBySubIncludingDeleted はテナントを取らない。別テナントの User を消す判断はしない。
			continue
		}
		in.Users = append(in.Users, domain.ReconcileUser{ID: link.SourceID, Deleted: true, Version: deletedVersion(user)})
	}
	for _, task := range tasks {
		in.Tasks[task.SourceID] = append(in.Tasks[task.SourceID], domain.ReconcileTask{Status: task.Status, SourceVersion: task.SourceVersion})
	}
	return in, nil
}

func assignedUserIDs(ctx context.Context, deps ReconcileDeps, conn domain.ProvisioningConnection) (map[string]bool, error) {
	assigned := map[string]bool{}
	if conn.Scope == domain.ScopeAllUsers || deps.AssignmentRepo == nil {
		return assigned, nil
	}
	assignments, err := deps.AssignmentRepo.ListByApplication(ctx, conn.TenantID, conn.ApplicationID)
	if err != nil {
		return nil, err
	}
	for _, a := range assignments {
		if a.SubjectType == appdomain.AssignmentSubjectUser {
			assigned[a.SubjectID] = true
		}
	}
	return assigned, nil
}

// deletedVersion は削除済みの User のバージョンを返す。行ごと消えた User は、どの dead_letter よりも古いとみなす。
func deletedVersion(user *userdomain.User) int64 {
	if user == nil {
		return 0
	}
	return user.UpdatedAt.UnixNano()
}

// saveReconcileAction は照合の結果 1 件を保存し、新しく作ったかを返す。
func saveReconcileAction(ctx context.Context, deps ReconcileDeps, conn domain.ProvisioningConnection, action domain.ReconcileAction, now time.Time) (bool, error) {
	id, err := spec.NewUUIDv4()
	if err != nil {
		return false, err
	}
	version := now.UnixNano()
	if action.Schedule {
		reservation := domain.NewScheduledDeprovision(id, conn.TenantID, conn.ApplicationID, action.UserID, version, now, conn.DeprovisionPolicy.GracePeriodDays)
		return deps.TaskRepo.ScheduleDeprovision(ctx, reservation)
	}
	return deps.TaskRepo.Save(ctx, &domain.ProvisioningTask{
		ID: id, TenantID: conn.TenantID, ConnectionID: conn.ApplicationID, SourceType: domain.SourceTypeUser, SourceID: action.UserID,
		SourceVersion: version, Operation: action.Operation, Status: domain.TaskPending, CreatedAt: now, UpdatedAt: now,
	})
}
