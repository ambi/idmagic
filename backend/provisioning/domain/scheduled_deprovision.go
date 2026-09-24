package domain

import "time"

// ScheduledDeprovisionStatus は猶予期間つき削除の予約の状態を表す。
type ScheduledDeprovisionStatus string

const (
	// ScheduledDeprovisionScheduled は期限の到来を待っている予約である。取り消せるのはこの状態だけである。
	ScheduledDeprovisionScheduled ScheduledDeprovisionStatus = "scheduled"
	// ScheduledDeprovisionMaterialized は delete のプロビジョニングタスクへ変えた予約である。
	ScheduledDeprovisionMaterialized ScheduledDeprovisionStatus = "materialized"
	// ScheduledDeprovisionCancelled は猶予期間中に対象が適用範囲へ戻ったため取り消した予約である。
	ScheduledDeprovisionCancelled ScheduledDeprovisionStatus = "cancelled"
)

// ScheduledDeprovision は、DeprovisionPolicy.grace_period_days を持つ接続で User の削除を
// 受け取ったときに、delete の ProvisioningTask の代わりに保存する予約である。
// 期限まではプロビジョニングタスクの行を作らないため、worker が期限前に下流へ送ることはない。
type ScheduledDeprovision struct {
	ID           string
	TenantID     string
	ConnectionID string
	UserID       string
	// SourceVersion は削除イベントのバージョンであり、期限到来時に作るプロビジョニングタスクの source_version になる。
	// 取消はこれより新しいバージョンの割り当てだけが行う。
	SourceVersion int64
	DueAt         time.Time
	Status        ScheduledDeprovisionStatus
	// TaskID は materialized で作ったプロビジョニングタスクを指す。
	TaskID    *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewScheduledDeprovision は deletedAt から graceDays 日後を期限とする予約を作る。
func NewScheduledDeprovision(id, tenantID, connectionID, userID string, version int64, deletedAt time.Time, graceDays int) *ScheduledDeprovision {
	return &ScheduledDeprovision{
		ID: id, TenantID: tenantID, ConnectionID: connectionID, UserID: userID, SourceVersion: version,
		DueAt:  deletedAt.Add(time.Duration(graceDays) * 24 * time.Hour),
		Status: ScheduledDeprovisionScheduled, CreatedAt: deletedAt, UpdatedAt: deletedAt,
	}
}

// IsDue は予約が now の時点でプロビジョニングタスクへ変えるべきかを返す。
func (s ScheduledDeprovision) IsDue(now time.Time) bool {
	return s.Status == ScheduledDeprovisionScheduled && !now.Before(s.DueAt)
}

// Task は期限に達した予約から作る delete のプロビジョニングタスクを返す。
func (s ScheduledDeprovision) Task(id string, now time.Time) *ProvisioningTask {
	return &ProvisioningTask{
		ID: id, TenantID: s.TenantID, ConnectionID: s.ConnectionID, SourceType: SourceTypeUser, SourceID: s.UserID,
		SourceVersion: s.SourceVersion, Operation: OperationDelete, Status: TaskPending, CreatedAt: now, UpdatedAt: now,
	}
}
