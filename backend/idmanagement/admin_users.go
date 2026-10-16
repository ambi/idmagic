package idmanagement

import (
	"context"
	"time"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// AdminUsers は、管理 API と同じ規則で User を作成、変更、停止、削除、復元する操作である。
// IdManagement の外の結合テストが、User の変更に対するほかのモジュールの反応を確かめるときに使う。
type AdminUsers struct{ deps userusecases.AdminUserDeps }

// CreateUserInput、UpdateUserInput、SoftDeleteUserInput、DeleteUserInput は AdminUsers の操作の入力である。
type (
	CreateUserInput     = userusecases.CreateUserInput
	UpdateUserInput     = userusecases.UpdateUserInput
	SoftDeleteUserInput = userusecases.SoftDeleteUserInput
	DeleteUserInput     = userusecases.DeleteUserInput
)

// AdminUsers は、d とこの Module の保存先と通知で User を変える操作を返す。管理 API と同じく、
// User の変更は Module の UserMutationCommitter で確定する。
func (m Module) AdminUsers(d LifecycleDeps, emit func(spec.DomainEvent) error) AdminUsers {
	deps := m.adminUserDeps(d, emit)
	deps.UserMutationCommitter = m.UserMutationCommitter
	return AdminUsers{deps: deps}
}

// CreateUser は User を作成する。
func (a AdminUsers) CreateUser(ctx context.Context, in CreateUserInput) (*userdomain.User, error) {
	return userusecases.CreateUser(ctx, a.deps, in)
}

// UpdateUser は User を変更する。
func (a AdminUsers) UpdateUser(ctx context.Context, in UpdateUserInput) (*userdomain.User, error) {
	return userusecases.UpdateUser(ctx, a.deps, in)
}

// SetUserDisabled は User を停止または再開する。
func (a AdminUsers) SetUserDisabled(
	ctx context.Context, actorUserID, targetUserID string, disabled bool, now time.Time,
) (*userdomain.User, error) {
	return userusecases.SetUserDisabled(ctx, a.deps, actorUserID, targetUserID, disabled, now)
}

// SoftDeleteUser は User の削除を予約する。
func (a AdminUsers) SoftDeleteUser(ctx context.Context, in SoftDeleteUserInput) error {
	return userusecases.SoftDeleteUser(ctx, a.deps, in)
}

// RestoreUser は削除を予約した User を元に戻す。
func (a AdminUsers) RestoreUser(ctx context.Context, actorUserID, targetUserID string, now time.Time) (*userdomain.User, error) {
	return userusecases.RestoreUser(ctx, a.deps, actorUserID, targetUserID, now)
}

// DeleteUser は User を完全に削除する。
func (a AdminUsers) DeleteUser(ctx context.Context, in DeleteUserInput) error {
	return userusecases.DeleteUser(ctx, a.deps, in)
}
