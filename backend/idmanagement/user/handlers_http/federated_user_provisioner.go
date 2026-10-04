package handlers_http

import (
	"context"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
)

// FederatedUserProvisioner は、フェデレーションの JIT が User を作る入口を、管理者の操作と
// 同じ依存から組み立てる。JIT の依存を別に組み立てると、動的グループの評価のような
// 作成の経路に共通する作用が、配線から黙って抜ける (REQ-IDMANAGEMENT-089)。
func FederatedUserProvisioner(d Deps) func(context.Context, userusecases.ProvisionFederatedUserInput) (*userdomain.User, error) {
	deps := adminUserDeps(d)
	return func(ctx context.Context, in userusecases.ProvisionFederatedUserInput) (*userdomain.User, error) {
		return userusecases.ProvisionFederatedUser(ctx, deps, in)
	}
}
