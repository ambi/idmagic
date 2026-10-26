package ports

import (
	"context"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

// GroupRepository は tenant-scoped な Group 集約とそのメンバーシップを永続化する。
// すべての操作はテナント境界に閉じ、cross-tenant 参照は use case 側で
// reject する。
type GroupRepository interface {
	ListAll(ctx context.Context, tenantID string) ([]*groupdomain.Group, error)
	// ListPage returns up to limit groups for tenantID ordered by (name, id)
	// ascending, strictly after the keyset (afterName, afterID). Pass "", ""
	// for the first page. Backs ListGroups keyset pagination (wi-159).
	ListPage(ctx context.Context, tenantID, afterName, afterID string, limit int) ([]*groupdomain.Group, error)
	ListPageBefore(ctx context.Context, tenantID, beforeName, beforeID string, limit int) ([]*groupdomain.Group, error)
	Count(ctx context.Context, tenantID string) (int64, error)
	FindByID(ctx context.Context, tenantID, id string) (*groupdomain.Group, error)
	Save(ctx context.Context, group *groupdomain.Group) error
	// SetRoleDelegation だけが既存の承認を変更する。Save は承認を保ち、上流管理の印を解除しない。
	SetRoleDelegation(ctx context.Context, tenantID, groupID string, delegation *groupdomain.GroupRoleDelegation, now time.Time) error
	Delete(ctx context.Context, tenantID, id string) error

	ListMembersByGroup(ctx context.Context, tenantID, groupID string) ([]*groupdomain.GroupMember, error)
	// ListGroupsByUser は指定 User が所属するグループを返す。認可経路 (effective
	// roles の解決) と admin UI の両方から呼ばれる。Roles は現在の式と承認を照合した
	// 付与可能なロールであり、管理用の設定値は FindByID で読む。
	ListGroupsByUser(ctx context.Context, tenantID, userID string) ([]*groupdomain.Group, error)
	CountMembers(ctx context.Context, tenantID, groupID string) (int, error)
	// AddMember は membership を追加し、新規追加なら true を返す。既に所属済みなら
	// false を返し no-op とする (冪等)。
	AddMember(ctx context.Context, member *groupdomain.GroupMember) (bool, error)
	// RemoveMember は membership を削除し、削除されたなら true を返す。非所属なら
	// false を返し no-op とする (冪等)。
	RemoveMember(ctx context.Context, tenantID, groupID, userID string) (bool, error)

	FindDynamicRule(ctx context.Context, tenantID, groupID string) (*groupdomain.DynamicGroupRule, error)
	ListDynamicRules(ctx context.Context, tenantID string) ([]*groupdomain.DynamicGroupRule, error)
	SaveDynamicRule(ctx context.Context, rule *groupdomain.DynamicGroupRule) error
}
