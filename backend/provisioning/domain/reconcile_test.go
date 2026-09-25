package domain_test

import (
	"fmt"
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
	return domain.PlanReconciliation(in).Actions
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
	}).Actions
	wantActions(t, got,
		domain.ReconcileAction{UserID: "u1", Operation: domain.OperationCreate},
		domain.ReconcileAction{UserID: "u2", Operation: domain.OperationCreate},
	)
}

// guardedConnection は誤削除ガードの閾値を設定した接続を返す。nil はその閾値を設定しないことを表す。
func guardedConnection(count, percent *int) domain.ProvisioningConnection {
	conn := reconcileConnection()
	conn.DeprovisionPolicy.AccidentalDeletionCountThreshold = count
	conn.DeprovisionPolicy.AccidentalDeletionPercentThreshold = percent
	return conn
}

// linkedUsers は反映済みのリンクを持つ n 人の User と、そのリンクを返す。inScope が false なら、全員がスコープ外になっている。
func linkedUsers(n int, inScope bool) ([]domain.ReconcileUser, map[string]domain.RemoteResourceLink) {
	users := make([]domain.ReconcileUser, 0, n)
	links := make(map[string]domain.RemoteResourceLink, n)
	for i := range n {
		id := fmt.Sprintf("u%02d", i)
		users = append(users, domain.ReconcileUser{ID: id, Active: true, InScope: inScope, Version: linkVersion})
		links[id] = activeLink()
	}
	return users, links
}

func wantQuarantine(t *testing.T, plan domain.ReconcilePlan) {
	t.Helper()
	if plan.QuarantineReason == "" || len(plan.Actions) != 0 {
		t.Fatalf("PlanReconciliation() = %+v, want no action and a quarantine reason", plan)
	}
}

func wantNoQuarantine(t *testing.T, plan domain.ReconcilePlan, actions int) {
	t.Helper()
	if plan.QuarantineReason != "" || len(plan.Actions) != actions {
		t.Fatalf("PlanReconciliation() = %+v, want %d actions and no quarantine", plan, actions)
	}
}

//spec:covers REQ-PROVISIONING-011: 計画した deprovision が件数の閾値を超えたら、照合は何も作らず隔離の理由を返す。1 回あたりの上限で打ち切る前の件数を比べるので、上限より多い deprovision も見逃さない。
func TestPlanReconciliation_QuarantinesInsteadOfDeprovisioningOverTheGuard(t *testing.T) {
	users, links := linkedUsers(3, false)
	plan := domain.PlanReconciliation(domain.ReconcileInput{
		Connection: guardedConnection(new(2), nil), Users: users, Links: links, Limit: 1,
	})
	wantQuarantine(t, plan)
}

// 閾値と等しい件数は超過ではない。ここを誤ると、正当な一括の deprovision が止まる。
func TestPlanReconciliation_DeprovisionsAtTheCountThreshold(t *testing.T) {
	users, links := linkedUsers(2, false)
	plan := domain.PlanReconciliation(domain.ReconcileInput{
		Connection: guardedConnection(new(2), nil), Users: users, Links: links, Limit: 100,
	})
	wantNoQuarantine(t, plan, 2)
}

//spec:covers EX-PROVISIONING-011-03: 件数の閾値が無くても、反映済みの 10 人のうち 6 人の deprovision は割合の閾値 50 を超えるので、照合は何も作らず隔離の理由を返す。5 人なら超えない。
func TestPlanReconciliation_QuarantinesOverThePercentThreshold(t *testing.T) {
	inScope, inScopeLinks := linkedUsers(10, true)
	for _, deprovisioned := range []struct {
		count      int
		quarantine bool
	}{{6, true}, {5, false}} {
		users := slices.Clone(inScope)
		for i := range deprovisioned.count {
			users[i].InScope = false
		}
		plan := domain.PlanReconciliation(domain.ReconcileInput{
			Connection: guardedConnection(nil, new(50)), Users: users, Links: inScopeLinks, Limit: 100,
		})
		if deprovisioned.quarantine {
			wantQuarantine(t, plan)
		} else {
			wantNoQuarantine(t, plan, deprovisioned.count)
		}
	}
}

// 猶予期間つき削除の予約も、いずれ下流の削除になるので数える。
func TestPlanReconciliation_CountsScheduledDeletionsTowardTheGuard(t *testing.T) {
	conn := guardedConnection(new(1), nil)
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	conn.DeprovisionPolicy.GracePeriodDays = 7
	users, links := linkedUsers(2, true)
	for i := range users {
		users[i].Deleted = true
	}
	plan := domain.PlanReconciliation(domain.ReconcileInput{Connection: conn, Users: users, Links: links, Limit: 100})
	wantQuarantine(t, plan)
}

// 作成と更新は deprovision ではないので、閾値を超える数を作っても隔離しない。
func TestPlanReconciliation_DoesNotCountProvisioningTowardTheGuard(t *testing.T) {
	users := []domain.ReconcileUser{
		{ID: "u1", Active: true, InScope: true, Version: 10},
		{ID: "u2", Active: true, InScope: true, Version: 10},
		{ID: "u3", Active: true, InScope: true, Version: linkVersion + 1},
	}
	links := map[string]domain.RemoteResourceLink{"u3": activeLink()}
	plan := domain.PlanReconciliation(domain.ReconcileInput{
		Connection: guardedConnection(new(0), nil), Users: users, Links: links, Limit: 100,
	})
	wantNoQuarantine(t, plan, 3)
}
