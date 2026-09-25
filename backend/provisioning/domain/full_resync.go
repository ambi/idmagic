package domain

import "time"

// FullResyncStatus は FullResync の状態である（docs/domain/provisioning/internals.md §Full Resync の完了追跡）。
type FullResyncStatus string

const (
	FullResyncStatusRunning   FullResyncStatus = "running"
	FullResyncStatusCompleted FullResyncStatus = "completed"
)

// FullResync は StartFullResync の一回分であり、開始時に確定した対象数のプロビジョニングタスクの決着を待つ。
type FullResync struct {
	ID             string
	TenantID       string
	ConnectionID   string
	Status         FullResyncStatus
	TotalTasks     int
	SucceededCount int
	FailedCount    int
	StartedAt      time.Time
	CompletedAt    *time.Time
}

// FullResyncTally は FullResync へ関連付いた終端のプロビジョニングタスクを数えた結果である。
// pending と in_flight は数えない。
type FullResyncTally struct {
	Succeeded int
	Failed    int
}

// NewFullResync は totalTasks 件の対象を待つ running の FullResync を作る。
func NewFullResync(id, tenantID, connectionID string, totalTasks int, now time.Time) *FullResync {
	return &FullResync{
		ID: id, TenantID: tenantID, ConnectionID: connectionID, Status: FullResyncStatusRunning,
		TotalTasks: totalTasks, StartedAt: now.UTC(),
	}
}

// Settle は tally から完了を判定する。完了するなら completed にした写しと true を返し、r 自体は変えない。
// 関連付けは一件ずつで対象数を超えないので、終端の件数が対象数に達したことは、全件が作られて終端になったことを表す。
// StartFullResync がプロビジョニングタスクを作り終える前に先のプロビジョニングタスクが終わっても完了しない。
func (r FullResync) Settle(tally FullResyncTally, now time.Time) (FullResync, bool) {
	if r.Status != FullResyncStatusRunning || tally.Succeeded+tally.Failed != r.TotalTasks {
		return r, false
	}
	completedAt := now.UTC()
	r.Status = FullResyncStatusCompleted
	r.SucceededCount, r.FailedCount = tally.Succeeded, tally.Failed
	r.CompletedAt = &completedAt
	return r, true
}

// CompletedEvent は完了した r の FullResyncCompleted を返す。
func (r FullResync) CompletedEvent() *FullResyncCompleted {
	var at time.Time
	if r.CompletedAt != nil {
		at = *r.CompletedAt
	}
	return &FullResyncCompleted{
		At: at, TenantID: r.TenantID, ApplicationID: r.ConnectionID,
		TotalSubjects: r.TotalTasks, SucceededCount: r.SucceededCount, FailedCount: r.FailedCount,
	}
}
