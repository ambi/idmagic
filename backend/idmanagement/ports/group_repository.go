package ports

import (
	"context"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

// GroupRepository は tenant-scoped な Group 集約とそのメンバーシップを永続化する
// (ADR-038)。すべての操作はテナント境界に閉じ、cross-tenant 参照は use case 側で
// reject する。
type GroupRepository interface {
	ListByTenant(ctx context.Context, tenantID string) ([]*idmdomain.Group, error)
	FindByID(ctx context.Context, tenantID, id string) (*idmdomain.Group, error)
	Save(ctx context.Context, group *idmdomain.Group) error
	Delete(ctx context.Context, tenantID, id string) error

	ListMembersByGroup(ctx context.Context, tenantID, groupID string) ([]*idmdomain.GroupMember, error)
	// ListGroupsByUser は指定 User が所属するグループを返す。認可経路 (effective
	// roles の解決) と admin UI の両方から呼ばれる。
	ListGroupsByUser(ctx context.Context, tenantID, userID string) ([]*idmdomain.Group, error)
	CountMembers(ctx context.Context, tenantID, groupID string) (int, error)
	// AddMember は membership を追加し、新規追加なら true を返す。既に所属済みなら
	// false を返し no-op とする (冪等)。
	AddMember(ctx context.Context, member *idmdomain.GroupMember) (bool, error)
	// RemoveMember は membership を削除し、削除されたなら true を返す。非所属なら
	// false を返し no-op とする (冪等)。
	RemoveMember(ctx context.Context, tenantID, groupID, userID string) (bool, error)

	FindDynamicRule(ctx context.Context, tenantID, groupID string) (*idmdomain.DynamicGroupRule, error)
	ListDynamicRules(ctx context.Context, tenantID string) ([]*idmdomain.DynamicGroupRule, error)
	SaveDynamicRule(ctx context.Context, rule *idmdomain.DynamicGroupRule) error
}
