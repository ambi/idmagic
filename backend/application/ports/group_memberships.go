package ports

import "context"

// GroupMemberships は、User が所属する Group の ID を返す。
// Group の保存表現は Application モジュールの外に留める。
type GroupMemberships interface {
	GroupIDsOfUser(ctx context.Context, tenantID, userID string) ([]string, error)
}
