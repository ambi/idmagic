package usecases_test

import (
	"context"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

type reconcileFixture struct {
	t              *testing.T
	deps           usecases.ReconcileDeps
	connRepo       *memory.ProvisioningConnectionRepository
	taskRepo       *memory.ProvisioningTaskRepository
	linkRepo       *memory.RemoteResourceLinkRepository
	userRepo       *usermemory.UserRepository
	assignmentRepo *appmemory.ApplicationAssignmentRepository
}

func newReconcileFixture(t *testing.T, conn *domain.ProvisioningConnection) *reconcileFixture {
	t.Helper()
	f := &reconcileFixture{
		t: t, connRepo: memory.NewProvisioningConnectionRepository(), taskRepo: memory.NewProvisioningTaskRepository(),
		linkRepo: memory.NewRemoteResourceLinkRepository(), userRepo: usermemory.NewUserRepository(),
		assignmentRepo: appmemory.NewApplicationAssignmentRepository(),
	}
	f.deps = usecases.ReconcileDeps{
		ConnectionRepo: f.connRepo, TaskRepo: f.taskRepo, LinkRepo: f.linkRepo, UserRepo: f.userRepo, AssignmentRepo: f.assignmentRepo,
	}
	if err := f.connRepo.Register(context.Background(), conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return f
}

func (f *reconcileFixture) saveUser(tenantID, id string, status idmdomain.UserStatus) {
	f.t.Helper()
	user := &userdomain.User{
		ID: id, TenantID: tenantID, PreferredUsername: id, Lifecycle: userdomain.UserLifecycle{Status: status},
		CreatedAt: time.Now().Add(-time.Hour), UpdatedAt: time.Now().Add(-time.Hour),
	}
	if err := f.userRepo.Save(context.Background(), user); err != nil {
		f.t.Fatalf("Save(user) error = %v", err)
	}
}

func (f *reconcileFixture) linkActiveUser(connectionID, userID string) {
	f.t.Helper()
	link := domain.NewRemoteResourceLink(connectionID, testTenantID, domain.SourceTypeUser, userID)
	_ = link.ApplySync(time.Now().UnixNano(), "remote-"+userID, userID, nil, time.Now())
	link.Active = true
	if err := f.linkRepo.Upsert(context.Background(), link); err != nil {
		f.t.Fatalf("Upsert(link) error = %v", err)
	}
}

func (f *reconcileFixture) reconcile(now time.Time) int {
	f.t.Helper()
	created, err := usecases.ReconcileConnections(context.Background(), f.deps, 100, now)
	if err != nil {
		f.t.Fatalf("ReconcileConnections() error = %v", err)
	}
	return created
}

func (f *reconcileFixture) tasks(connectionID string) []*domain.ProvisioningTask {
	f.t.Helper()
	tasks, err := f.taskRepo.ListByConnection(context.Background(), testTenantID, connectionID, nil, 100)
	if err != nil {
		f.t.Fatalf("ListByConnection() error = %v", err)
	}
	return tasks
}

// 猶予期間つきの削除は、削除のプロビジョニングタスクではなく予約として保存し、期限の到来後に実体化する。
func TestReconcileConnections_SchedulesTheDeletionOfADeletedUserWithAGracePeriod(t *testing.T) {
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	conn.DeprovisionPolicy.GracePeriodDays = 7
	f := newReconcileFixture(t, conn)
	f.saveUser(testTenantID, "u-deleted", idmdomain.UserStatusDeleted)
	f.linkActiveUser("app-1", "u-deleted")
	now := time.Now().UTC()

	if created := f.reconcile(now); created != 1 {
		t.Fatalf("created = %d, want 1 reservation", created)
	}
	if tasks := f.tasks("app-1"); len(tasks) != 0 {
		t.Fatalf("tasks = %+v, want none before the grace period elapses", tasks)
	}
	due, err := f.taskRepo.ListDueDeprovisions(context.Background(), now.Add(8*24*time.Hour), 10)
	if err != nil || len(due) != 1 || due[0].UserID != "u-deleted" {
		t.Fatalf("due reservations = %+v, err=%v, want one for u-deleted", due, err)
	}
	if created := f.reconcile(now.Add(time.Minute)); created != 0 {
		t.Fatalf("second reconcile created = %d, want 0 (the reservation already exists)", created)
	}
}

// 削除済みの User の読み取りはテナントを取らないので、リンクが別テナントの User を指していれば手を出さない。
func TestReconcileConnections_LeavesALinkToAnotherTenantsUserAlone(t *testing.T) {
	f := newReconcileFixture(t, activeConnection("app-1", domain.ScopeAllUsers))
	f.saveUser("tenant-b", "u-foreign", idmdomain.UserStatusDeleted)
	f.linkActiveUser("app-1", "u-foreign")

	if created := f.reconcile(time.Now().UTC()); created != 0 {
		t.Fatalf("created = %d, want 0 for another tenant's user: %+v", created, f.tasks("app-1"))
	}
}

// assigned_only の接続では、割り当てのある User だけがスコープに入る。
func TestReconcileConnections_ScopesAnAssignedOnlyConnectionByAssignments(t *testing.T) {
	f := newReconcileFixture(t, activeConnection("app-1", domain.ScopeAssignedOnly))
	f.saveUser(testTenantID, "u-assigned", idmdomain.UserStatusActive)
	f.saveUser(testTenantID, "u-unassigned", idmdomain.UserStatusActive)
	if err := f.assignmentRepo.Save(context.Background(), &appdomain.ApplicationAssignment{
		TenantID: testTenantID, ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "u-assigned",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Save(assignment) error = %v", err)
	}

	if created := f.reconcile(time.Now().UTC()); created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}
	tasks := f.tasks("app-1")
	if len(tasks) != 1 || tasks[0].SourceID != "u-assigned" || tasks[0].Operation != domain.OperationCreate {
		t.Fatalf("tasks = %+v, want one create for u-assigned", tasks)
	}
}

// 隔離中と無効の接続は照合しない。照合が隔離を迂回して下流へ書き込むと、隔離の意味がなくなる。
func TestReconcileConnections_SkipsQuarantinedAndDisabledConnections(t *testing.T) {
	quarantined := activeConnection("app-q", domain.ScopeAllUsers)
	quarantined.Health = domain.HealthQuarantined
	now := time.Now().UTC()
	quarantined.QuarantinedAt = &now
	f := newReconcileFixture(t, quarantined)
	disabled := activeConnection("app-d", domain.ScopeAllUsers)
	disabled.Status = domain.ConnectionDisabled
	if err := f.connRepo.Register(context.Background(), disabled, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	f.saveUser(testTenantID, "u1", idmdomain.UserStatusActive)

	if created := f.reconcile(now); created != 0 {
		t.Fatalf("created = %d, want 0 for quarantined and disabled connections", created)
	}
}
