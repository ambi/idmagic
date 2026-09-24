package domain

import (
	"slices"
	"strings"
)

// ReconcileUser は照合が 1 人の User について読む事実である。
type ReconcileUser struct {
	ID string
	// Deleted は User が削除済みであることを表す。削除済みの User は、リンクを持つものだけが現れる。
	Deleted bool
	// Active は User のステータスが active であることを表す。
	Active bool
	// InScope は User が接続のスコープ（all_users または割り当て）に入っていることを表す。
	InScope bool
	// Version は User の updated_at の UnixNano である。
	Version int64
}

// ReconcileTask は照合が User ごとに読む、確定していないか失敗したプロビジョニングタスクである。
type ReconcileTask struct {
	Status        ProvisioningTaskStatus
	SourceVersion int64
}

// ReconcileInput は 1 接続の照合の入力である。Links と Tasks は User の ID をキーとする。
type ReconcileInput struct {
	Connection ProvisioningConnection
	Users      []ReconcileUser
	Links      map[string]RemoteResourceLink
	Tasks      map[string][]ReconcileTask
	Limit      int
}

// ReconcileAction は照合が作るもの 1 件である。Schedule が true なら、プロビジョニングタスクではなく
// 猶予期間つき削除の予約を作る。
type ReconcileAction struct {
	UserID    string
	Operation ProvisioningOperation
	Schedule  bool
}

// PlanReconciliation は、あるべき状態と下流へ反映済みの状態の差分から、作るべきものを返す。
// 結果は User の ID の順に並べて Limit 件で打ち切るので、残りは次の照合が拾う。
func PlanReconciliation(in ReconcileInput) []ReconcileAction {
	users := slices.Clone(in.Users)
	slices.SortFunc(users, func(a, b ReconcileUser) int { return strings.Compare(a.ID, b.ID) })

	var actions []ReconcileAction
	for _, user := range users {
		if len(actions) >= in.Limit {
			break
		}
		if awaitsSettledTask(user, in.Tasks[user.ID]) {
			continue
		}
		var link *RemoteResourceLink
		if l, ok := in.Links[user.ID]; ok {
			link = &l
		}
		if action, ok := planUser(in.Connection, user, link); ok {
			actions = append(actions, action)
		}
	}
	return actions
}

// awaitsSettledTask は、既存のプロビジョニングタスクの決着を待つべき User かを返す。
// 未確定のタスクは、書き込み時の捕捉や前回の照合が同じ変更をすでに扱っている。
// dead_letter のタスクより User が新しくなければ、作り直しても同じく失敗する。
func awaitsSettledTask(user ReconcileUser, tasks []ReconcileTask) bool {
	for _, task := range tasks {
		switch task.Status {
		case TaskPending, TaskInFlight:
			return true
		case TaskDeadLetter:
			if task.SourceVersion >= user.Version {
				return true
			}
		case TaskSucceeded:
		}
	}
	return false
}

func planUser(conn ProvisioningConnection, user ReconcileUser, link *RemoteResourceLink) (ReconcileAction, bool) {
	flags := conn.FeatureFlags
	act := func(op ProvisioningOperation, enabled bool) (ReconcileAction, bool) {
		return ReconcileAction{UserID: user.ID, Operation: op}, enabled
	}
	switch {
	case user.Deleted || !user.InScope:
		if link == nil {
			return ReconcileAction{}, false
		}
		return planDeprovision(conn, user, *link)
	case link == nil:
		return act(OperationCreate, user.Active && flags.CreateUsers)
	case user.Active && !link.Active:
		return act(OperationUpdate, flags.UpdateUsers)
	case !user.Active && link.Active:
		return act(OperationDeactivate, flags.DeactivateUsers)
	case user.Active && user.Version > link.LastSyncedVersion:
		return act(OperationUpdate, flags.UpdateUsers)
	default:
		return ReconcileAction{}, false
	}
}

// planDeprovision は、スコープ外になったか削除された User のリンクを、接続の DeprovisionPolicy に従って片付ける。
func planDeprovision(conn ProvisioningConnection, user ReconcileUser, link RemoteResourceLink) (ReconcileAction, bool) {
	policy := conn.DeprovisionPolicy.OnUnassign
	if user.Deleted {
		policy = conn.DeprovisionPolicy.OnDelete
	}
	op, enabled := DeprovisionOperation(policy, conn.FeatureFlags)
	switch {
	case !enabled:
		return ReconcileAction{}, false
	case op == OperationDeactivate && !link.Active:
		return ReconcileAction{}, false
	case op == OperationDelete && user.Deleted && conn.DeprovisionPolicy.GracePeriodDays > 0:
		return ReconcileAction{UserID: user.ID, Operation: op, Schedule: true}, true
	default:
		return ReconcileAction{UserID: user.ID, Operation: op}, true
	}
}

// DeprovisionOperation は、割り当て解除または削除に対する方針を下流への操作へ変換する。
// enabled が false なら、方針が none であるか、その操作の機能フラグが無効である。
func DeprovisionOperation(action ProvisioningDeprovisionAction, flags ProvisioningFeatureFlags) (op ProvisioningOperation, enabled bool) {
	switch action {
	case DeprovisionDeactivate:
		return OperationDeactivate, flags.DeactivateUsers
	case DeprovisionDelete:
		return OperationDelete, flags.DeleteUsers
	default:
		return "", false
	}
}
