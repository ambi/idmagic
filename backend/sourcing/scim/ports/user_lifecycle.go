package ports

import (
	"context"
	"time"
)

// UserLifecycle は、SCIM の取り込みが User を止める、再開する、削除を予約するときに通す
// IdManagement の操作である。SCIM は User の状態を直接保存しない。止めることに伴う
// イベント、端末の失効、下流への通知、所有する Agent の無効化を IdManagement が行う。
type UserLifecycle interface {
	// SetUserDisabled は User を無効化または再有効化し、User を変えたかを返す。
	SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (changed bool, err error)
	// ScheduleUserDeletion は User の削除を予約する。
	ScheduleUserDeletion(ctx context.Context, tenantID, userID string, now time.Time) error
}
