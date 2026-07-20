// Package idmanagement は IdManagement bounded context の DI 組立を所有する
// (ADR-091, wi-178)。User/Group/Agent の永続化 port を Module 1 個に束ね、中央
// server/routes.go と bootstrap の Dependencies が受け渡す。idmanagement/adapters/http
// は oauth2/scim/authentication/tenancy 由来の port も必要とするため、HTTP route 登録は
// authentication.Module 同様に中央 routes.go が引き続き組み立てる (自己登録は行わない)。
package idmanagement

import (
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// Module は identity-management context が所有する永続化 port の束。
// LifecycleWorkflow の port は IdGovernance context (backend/idgovernance) が所有する
// (wi-237, ADR-117)。User mutation から governance 側の run 生成へは
// userports.UserMutationCommitter 境界 port 経由で渡す。
type Module struct {
	UserRepo              userports.UserRepository
	GroupRepo             groupports.GroupRepository
	AgentRepo             agentports.AgentRepository
	EmailChangeTokenStore userports.EmailChangeTokenStore
	// UserMutationCommitter は User mutation を確定させる境界 port。IdGovernance が
	// 実装を注入する。nil のとき admin usecase は UserRepo.Save に fallback する。
	UserMutationCommitter userports.UserMutationCommitter
	// ProvisioningNotifier は outbound Provisioning (wi-45, ADR-128) の境界 port。
	// nil のとき outbound provisioning は未配線として何もしない。
	ProvisioningNotifier userports.ProvisioningNotifier
}
