package usecases

import (
	"context"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// CreateSourceUser は、取り込み元の属性から、パスワードを持たない User を作る。
// 一意性、属性、使用量、変更の捕捉と下流通知はほかの作成経路と共有する。
func CreateSourceUser(ctx context.Context, deps AdminUserDeps, source *userdomain.User) (*userdomain.User, error) {
	return createUser(ctx, deps, newUser{
		User: userdomain.User{
			TenantID: source.TenantID, PreferredUsername: source.PreferredUsername,
			Name: source.Name, GivenName: source.GivenName, FamilyName: source.FamilyName,
			Email: source.Email, EmailVerified: source.EmailVerified, Attributes: source.Attributes,
			Lifecycle: userdomain.UserLifecycle{Status: source.Lifecycle.Status}, Roles: []string{},
		},
		ActorUserID: "scim",
	}, source.CreatedAt)
}
