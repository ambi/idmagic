package domain

import (
	"fmt"
	"slices"
	"strings"
)

// ReconcileUser はインクリメンタル同期が 1 人の User について読む事実である。
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

// ReconcileTask はインクリメンタル同期が User ごとに読む、確定していないか失敗したプロビジョニングタスクである。
type ReconcileTask struct {
	Status        ProvisioningTaskStatus
	SourceVersion int64
}

// ReconcileInput は 1 接続のインクリメンタル同期の入力である。Links と Tasks は User の ID をキーとする。
type ReconcileInput struct {
	Connection ProvisioningConnection
	Users      []ReconcileUser
	Links      map[string]RemoteResourceLink
	Tasks      map[string][]ReconcileTask
	Limit      int
}

// ReconcileAction はインクリメンタル同期が作るもの 1 件である。Schedule が true なら、プロビジョニングタスクではなく
// 猶予期間つき削除の予約を作る。
type ReconcileAction struct {
	UserID    string
	Operation ProvisioningOperation
	Schedule  bool
}

// ReconcilePlan は 1 接続のインクリメンタル同期が作るものである。
type ReconcilePlan struct {
	Actions []ReconcileAction
	// QuarantineReason が空でなければ、計画した deprovision が誤削除ガードの閾値を超えている。
	// そのとき Actions は空であり、呼び出し側は何も作らずに接続を隔離する。
	QuarantineReason string
}

// PlanReconciliation は、あるべき状態と下流へ反映済みの状態の差分から、作るべきものを返す。
// 結果は User の ID の順に並べて Limit 件で打ち切るので、残りは次のインクリメンタル同期が拾う。
func PlanReconciliation(in ReconcileInput) ReconcilePlan {
	users := slices.Clone(in.Users)
	slices.SortFunc(users, func(a, b ReconcileUser) int { return strings.Compare(a.ID, b.ID) })

	var actions []ReconcileAction
	// 誤削除ガードは上限で打ち切る前の件数で判定するので、上限に達した後も deprovision だけは数え続ける。
	deprovisions := 0
	for _, user := range users {
		if awaitsSettledTask(user, in.Tasks[user.ID]) {
			continue
		}
		var link *RemoteResourceLink
		if l, ok := in.Links[user.ID]; ok {
			link = &l
		}
		action, ok := planUser(in.Connection, user, link)
		if !ok {
			continue
		}
		if action.Operation == OperationDeactivate || action.Operation == OperationDelete {
			deprovisions++
		}
		if len(actions) < in.Limit {
			actions = append(actions, action)
		}
	}
	if in.Connection.DeprovisionPolicy.exceedsAccidentalDeletionGuard(deprovisions, len(in.Links)) {
		return ReconcilePlan{QuarantineReason: fmt.Sprintf(
			"accidental deletion guard: %d deprovisions of %d linked users exceed the threshold", deprovisions, len(in.Links))}
	}
	return ReconcilePlan{Actions: actions}
}

// exceedsAccidentalDeletionGuard は、1 回のインクリメンタル同期の deprovision が件数または割合の閾値を超えるかを返す。
// 割合の分母は下流へ反映済みの User のリンク数である。等しい値は超過ではない。
func (p DeprovisionPolicy) exceedsAccidentalDeletionGuard(deprovisions, linkedUsers int) bool {
	if p.AccidentalDeletionCountThreshold != nil && deprovisions > *p.AccidentalDeletionCountThreshold {
		return true
	}
	return p.AccidentalDeletionPercentThreshold != nil && deprovisions*100 > *p.AccidentalDeletionPercentThreshold*linkedUsers
}

// awaitsSettledTask は、既存のプロビジョニングタスクの決着を待つべき User かを返す。
// 未確定のタスクは、イベント同期や前回のインクリメンタル同期が同じ変更をすでに扱っている。
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
