package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// UserLifecycleCommands は、管理 API の外のモジュール（LifecycleWorkflow と SCIM の取り込み）が
// User を止めるための操作である。User の状態を直接保存させず、管理 API と同じユースケースを通して、
// イベント、記憶済みの端末の失効、下流への通知、所有する Agent の無効化を伴わせる。
// 組み立ての地点が、呼ぶ側のモジュールのポートへ渡す。
//
// Deps には UserMutationCommitter を渡さない。ほかのモジュールの手順による User の変更から、
// さらに LifecycleWorkflow の実行を作らないためである。
type UserLifecycleCommands struct {
	Deps  AdminUserDeps
	Actor string
}

// SetUserDisabled は User を無効化または再有効化し、User を変えたかを返す。削除予約中の User は
// 無効化も再有効化もしない (REQ-IDMANAGEMENT-046) ので、変更なしとして返す。
func (c UserLifecycleCommands) SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (bool, error) {
	if _, err := SetUserDisabled(c.tenantContext(ctx, tenantID), c.deps(), c.Actor, userID, disabled, now); err != nil {
		if errors.Is(err, ErrUserPendingDeletion) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ScheduleUserDeletion は User の削除を予約する。
func (c UserLifecycleCommands) ScheduleUserDeletion(ctx context.Context, tenantID, userID string, now time.Time) error {
	return SoftDeleteUser(c.tenantContext(ctx, tenantID), c.deps(), SoftDeleteUserInput{ActorUserID: c.Actor, Sub: userID, Now: now})
}

func (c UserLifecycleCommands) deps() AdminUserDeps {
	deps := c.Deps
	deps.UserMutationCommitter = nil
	return deps
}

func (c UserLifecycleCommands) tenantContext(ctx context.Context, tenantID string) context.Context {
	return tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: tenantID}, "", "")
}
