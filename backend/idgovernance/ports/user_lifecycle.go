package ports

import (
	"context"
	"time"
)

// UserLifecycle は、LifecycleWorkflow が User を無効化または再有効化する操作である。
// User の状態を直接保存せず、IdManagement の操作として行い、イベント、記憶済みの端末の失効、
// 下流への通知、所有する Agent の無効化を伴わせる。changed は User を変えたかを返す。
type UserLifecycle interface {
	SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (changed bool, err error)
}
