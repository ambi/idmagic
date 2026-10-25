package usecases

import (
	"context"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// CreateSourceUser は、取り込み元の属性から、パスワードを持たない User を作る。
// 一意性、属性、使用量、変更の捕捉と下流通知はほかの作成経路と共有する。
// actor は取り込み元の名前であり、イベントの actor に記録する。
func CreateSourceUser(ctx context.Context, deps AdminUserDeps, actor string, source *userdomain.User) (*userdomain.User, error) {
	return createUser(ctx, deps, newUser{
		User: userdomain.User{
			TenantID: source.TenantID, PreferredUsername: source.PreferredUsername,
			Name: source.Name, GivenName: source.GivenName, FamilyName: source.FamilyName,
			Email: source.Email, EmailVerified: source.EmailVerified, Attributes: source.Attributes,
			Lifecycle: userdomain.UserLifecycle{Status: source.Lifecycle.Status}, Roles: []string{},
		},
		ActorUserID:  actor,
		FromUpstream: true,
	}, source.CreatedAt)
}

// UpdateSourceUser は、ローカル専用属性の追加、変更と削除を拒否してから、共通の更新を呼ぶ。
func UpdateSourceUser(ctx context.Context, deps AdminUserDeps, actor string, source *userdomain.User) (*userdomain.User, error) {
	return updateUser(ctx, deps, UpdateUserInput{
		ActorUserID: actor, Sub: source.ID, PreferredUsername: &source.PreferredUsername,
		Name: source.Name, GivenName: source.GivenName, FamilyName: source.FamilyName,
		Email: source.Email, EmailVerified: &source.EmailVerified, Attributes: &source.Attributes, Now: source.UpdatedAt,
	}, true)
}
