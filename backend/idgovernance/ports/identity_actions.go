package ports

import (
	"context"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// GroupMembership は管理操作と同じイベントと通知を伴って所属を変える。
// changed は保存先で実際に変更した場合だけ true となる。
type GroupMembership interface {
	SetGroupMembership(ctx context.Context, tenantID, groupID, userID string, present bool, now time.Time) (changed bool, err error)
}

// UserRequiredActions は必須操作を付与または解除し、対応するイベントを発行する。
// この変更から新しい WorkflowRun は計画しない。
type UserRequiredActions interface {
	SetUserRequiredAction(ctx context.Context, tenantID, userID string, action idmdomain.RequiredAction, required bool, now time.Time) (changed bool, err error)
}
