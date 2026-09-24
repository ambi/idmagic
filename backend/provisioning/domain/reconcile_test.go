package domain_test

import (
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

func reconcileConnection() domain.ProvisioningConnection {
	return domain.ProvisioningConnection{
		ApplicationID: "app-1", TenantID: "tenant-a", Status: domain.ConnectionActive, Health: domain.HealthOK,
		FeatureFlags: domain.ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true, DeactivateUsers: true, DeleteUsers: true},
		Scope:        domain.ScopeAssignedOnly,
		DeprovisionPolicy: domain.DeprovisionPolicy{
			OnUnassign: domain.DeprovisionDeactivate, OnDelete: domain.DeprovisionDeactivate,
		},
	}
}

// linkVersion はテストのリンクが最後に反映したバージョンである。User の Version をこれより大きくすると、
// 反映後に User が変わったことを表す。
const linkVersion = 20

func activeLink() domain.RemoteResourceLink {
	return domain.RemoteResourceLink{RemoteID: "remote-1", Active: true, LastSyncedVersion: linkVersion}
}

func inactiveLink() domain.RemoteResourceLink {
	return domain.RemoteResourceLink{RemoteID: "remote-1", Active: false, LastSyncedVersion: linkVersion}
}

func planOne(t *testing.T, in domain.ReconcileInput) []domain.ReconcileAction {
	t.Helper()
	if in.Limit == 0 {
		in.Limit = 100
	}
	return domain.PlanReconciliation(in)
}

func wantActions(t *testing.T, got []domain.ReconcileAction, want ...domain.ReconcileAction) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("PlanReconciliation() = %+v, want %+v", got, want)
	}
}

//spec:covers EX-PROVISIONING-019-01: スコープ内で有効な User にリンクがなければ、照合は create を作る。
func TestPlanReconciliation_CreatesAnInScopeActiveUserWithoutALink(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: true, Version: 10}},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationCreate})
}

// 下流に無い無効な User を、無効なまま作ることはしない。
func TestPlanReconciliation_DoesNotCreateAnInactiveUser(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: false, InScope: true, Version: 10}},
	})
	wantActions(t, got)
}

//spec:covers REQ-PLATFORM-003, EX-PLATFORM-003-03: 下流で有効なリンクを持つ User が無効になっていれば、バージョンを問わず deactivate を作る。
func TestPlanReconciliation_DeactivatesAUserDisabledWithoutCapture(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: false, InScope: true, Version: 10}},
		Links:      map[string]domain.RemoteResourceLink{"u1": activeLink()},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationDeactivate})
}

// 無効化を反映済みなら、同じ無効化を周期ごとに作り直さない。
func TestPlanReconciliation_LeavesAnAlreadyDeactivatedUserAlone(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: false, InScope: true, Version: 30}},
		Links:      map[string]domain.RemoteResourceLink{"u1": inactiveLink()},
	})
	wantActions(t, got)
}

// 無効化を反映した User が再び有効になれば、User のバージョンがリンクより古くても update で戻す。
func TestPlanReconciliation_ReactivatesAUserWhoseLinkIsInactive(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: true, Version: 10}},
		Links:      map[string]domain.RemoteResourceLink{"u1": inactiveLink()},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationUpdate})
}

func TestPlanReconciliation_UpdatesAUserChangedSinceTheLastSync(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: true, Version: 21}},
		Links:      map[string]domain.RemoteResourceLink{"u1": activeLink()},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationUpdate})
}

// バージョンが等しいときは反映済みである。境界を > と >= で取り違えると、周期ごとに update を作る。
func TestPlanReconciliation_LeavesAConvergedUserAlone(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: true, Version: 20}},
		Links:      map[string]domain.RemoteResourceLink{"u1": activeLink()},
	})
	wantActions(t, got)
}

//spec:covers EX-PROVISIONING-019-02: 割り当てを解除された User のリンクが下流で有効なら、on_unassign に従って deactivate を作る。
func TestPlanReconciliation_DeprovisionsAnUnassignedUserPerOnUnassign(t *testing.T) {
	got := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: false, Version: 10}},
		Links:      map[string]domain.RemoteResourceLink{"u1": activeLink()},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationDeactivate})
}

func TestPlanReconciliation_LeavesAnUnassignedUserWhosePolicyIsNone(t *testing.T) {
	conn := reconcileConnection()
	conn.DeprovisionPolicy.OnUnassign = domain.DeprovisionNone
	got := planOne(t, domain.ReconcileInput{
		Connection: conn,
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: false, Version: 10}},
		Links:      map[string]domain.RemoteResourceLink{"u1": activeLink()},
	})
	wantActions(t, got)
}

// on_unassign=delete はリンクの有効状態を問わず delete を作る。無効化済みの User も下流から消す。
func TestPlanReconciliation_DeletesAnUnassignedUserWhosePolicyIsDelete(t *testing.T) {
	conn := reconcileConnection()
	conn.DeprovisionPolicy.OnUnassign = domain.DeprovisionDelete
	got := planOne(t, domain.ReconcileInput{
		Connection: conn,
		Users:      []domain.ReconcileUser{{ID: "u1", Active: false, InScope: false, Version: 10}},
		Links:      map[string]domain.RemoteResourceLink{"u1": inactiveLink()},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationDelete})
}

// 削除済みの User は、スコープ内でも on_unassign ではなく on_delete に従う。
func TestPlanReconciliation_DeprovisionsADeletedUserPerOnDelete(t *testing.T) {
	conn := reconcileConnection()
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	conn.DeprovisionPolicy.OnUnassign = domain.DeprovisionNone
	got := planOne(t, domain.ReconcileInput{
		Connection: conn,
		Users:      []domain.ReconcileUser{{ID: "u1", Deleted: true, InScope: true, Version: 10}},
		Links:      map[string]domain.RemoteResourceLink{"u1": activeLink()},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationDelete})
}

//spec:covers EX-PROVISIONING-019-04: 猶予期間つきの削除は、delete を直ちに作らず予約を作る。
func TestPlanReconciliation_SchedulesTheDeletionOfADeletedUserWithAGracePeriod(t *testing.T) {
	conn := reconcileConnection()
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	conn.DeprovisionPolicy.GracePeriodDays = 7
	got := planOne(t, domain.ReconcileInput{
		Connection: conn,
		Users:      []domain.ReconcileUser{{ID: "u1", Deleted: true, Version: 10}},
		Links:      map[string]domain.RemoteResourceLink{"u1": activeLink()},
	})
	wantActions(t, got, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationDelete, Schedule: true})
}

func TestPlanReconciliation_HonorsTheFeatureFlags(t *testing.T) {
	conn := reconcileConnection()
	conn.FeatureFlags = domain.ProvisioningFeatureFlags{}
	got := planOne(t, domain.ReconcileInput{
		Connection: conn,
		Users: []domain.ReconcileUser{
			{ID: "create", Active: true, InScope: true, Version: 10},
			{ID: "update", Active: true, InScope: true, Version: 30},
			{ID: "deactivate", Active: false, InScope: true, Version: 10},
		},
		Links: map[string]domain.RemoteResourceLink{"update": activeLink(), "deactivate": activeLink()},
	})
	wantActions(t, got)
}

//spec:covers EX-PROVISIONING-019-03: pending または in_flight のプロビジョニングタスクがある User には、新しいプロビジョニングタスクを作らない。
func TestPlanReconciliation_SkipsAUserWithAnUnsettledTask(t *testing.T) {
	for _, status := range []domain.ProvisioningTaskStatus{domain.TaskPending, domain.TaskInFlight} {
		t.Run(string(status), func(t *testing.T) {
			got := planOne(t, domain.ReconcileInput{
				Connection: reconcileConnection(),
				Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: true, Version: 10}},
				Tasks:      map[string][]domain.ReconcileTask{"u1": {{Status: status, SourceVersion: 5}}},
			})
			wantActions(t, got)
		})
	}
}

// 前回の失敗から User が変わっていなければ、作り直しても同じく失敗するので作らない。
// User が失敗より後に変われば、作り直す。
func TestPlanReconciliation_RetriesADeadLetteredUserOnlyAfterTheUserChanges(t *testing.T) {
	deadLetter := map[string][]domain.ReconcileTask{"u1": {{Status: domain.TaskDeadLetter, SourceVersion: 15}}}
	unchanged := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: true, Version: 15}},
		Tasks:      deadLetter,
	})
	wantActions(t, unchanged)

	changed := planOne(t, domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users:      []domain.ReconcileUser{{ID: "u1", Active: true, InScope: true, Version: 16}},
		Tasks:      deadLetter,
	})
	wantActions(t, changed, domain.ReconcileAction{UserID: "u1", Operation: domain.OperationCreate})
}

// 上限を超えた分は次の照合が拾う。User の ID の順に並べるので、打ち切られる User は入力の順に依存しない。
func TestPlanReconciliation_StopsAtTheLimitInUserIDOrder(t *testing.T) {
	got := domain.PlanReconciliation(domain.ReconcileInput{
		Connection: reconcileConnection(),
		Users: []domain.ReconcileUser{
			{ID: "u3", Active: true, InScope: true, Version: 10},
			{ID: "u1", Active: true, InScope: true, Version: 10},
			{ID: "u2", Active: true, InScope: true, Version: 10},
		},
		Limit: 2,
	})
	wantActions(t, got,
		domain.ReconcileAction{UserID: "u1", Operation: domain.OperationCreate},
		domain.ReconcileAction{UserID: "u2", Operation: domain.OperationCreate},
	)
}
