// Package idmanagement は IdManagement bounded context の DI 組立を所有する
// (ADR-091, wi-178)。User/Group/Agent の永続化 port を Module 1 個に束ね、中央
// server/routes.go と bootstrap の Dependencies が受け渡す。idmanagement/adapters/http
// は oauth2/scim/authentication/tenancy 由来の port も必要とするため、HTTP route 登録は
// authentication.Module 同様に中央 routes.go が引き続き組み立てる (自己登録は行わない)。
package idmanagement

import (
	"github.com/ambi/idmagic/backend/idmanagement/ports"
)

// Module は identity-management context が所有する永続化 port の束。
// LifecycleWorkflow の port は IdGovernance context (backend/idgovernance) が所有する
// (wi-237, ADR-117)。User mutation から governance 側の run 生成へは
// ports.UserMutationCommitter 境界 port 経由で渡す。
type Module struct {
	UserRepo  ports.UserRepository
	GroupRepo ports.GroupRepository
	AgentRepo ports.AgentRepository
	// UserMutationCommitter は User mutation を確定させる境界 port。IdGovernance が
	// 実装を注入する。nil のとき admin usecase は UserRepo.Save に fallback する。
	UserMutationCommitter ports.UserMutationCommitter
}
