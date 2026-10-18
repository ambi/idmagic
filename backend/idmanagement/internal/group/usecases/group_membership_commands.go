package usecases

import (
	"context"
	"time"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// GroupMembershipCommands は外部モジュールの所属変更を管理操作と同じ境界で行う。
type GroupMembershipCommands struct {
	Deps  AdminGroupDeps
	Actor string
}

func (c GroupMembershipCommands) SetGroupMembership(ctx context.Context, tenantID, groupID, userID string, present bool, now time.Time) (bool, error) {
	ctx = tenantports.WithTenantID(ctx, tenantID)
	return setGroupMembership(ctx, c.Deps, c.Actor, groupID, userID, present, now)
}
