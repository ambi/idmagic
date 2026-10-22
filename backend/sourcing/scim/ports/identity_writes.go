package ports

import (
	"context"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// IdentityWrites は投影済みの属性を所有者の規則で保存する。
// 作成は SCIM の対応表も一緒に確定し、呼び出し側で補償しない。
type IdentityWrites interface {
	CreateUser(ctx context.Context, scimID string, user *userdomain.User) (*userdomain.User, error)
	UpdateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error)
	CreateGroup(ctx context.Context, scimID string, group *groupdomain.Group, members []string) (*groupdomain.Group, error)
	UpdateGroup(ctx context.Context, group *groupdomain.Group) (*groupdomain.Group, error)
	SetGroupMembership(ctx context.Context, tenantID, groupID, userID string, present bool, now time.Time) (bool, error)
	DeleteGroup(ctx context.Context, tenantID, groupID string, now time.Time) error
}
