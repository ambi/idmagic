// Package idmanagement は IdManagement モジュールの組み立てを所有する。User、Group、Agent の
// 保存先を Module に束ね、経路の登録、ジョブのハンドラーの登録、管理 API の外から User を止める
// 操作を、組み立て地点が渡すほかのモジュールの依存とともに組み立てる。
package idmanagement

import (
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// Module は identity-management context が所有する永続化 port の束。
// LifecycleWorkflow の port は IdGovernance context (backend/idgovernance) が所有する
// (wi-237)。User mutation から governance 側の run 生成へは
// userports.UserMutationCommitter 境界 port 経由で渡す。
type Module struct {
	UserRepo              userports.UserRepository
	GroupRepo             groupports.GroupRepository
	AgentRepo             agentports.AgentRepository
	EmailChangeTokenStore userports.EmailChangeTokenStore
	CSVArtifacts          idmports.CSVArtifactStore
	UserImportCommitter   userports.UserImportRowCommitter
	GroupImportCommitter  groupports.GroupImportRowCommitter
	// GroupMembershipImportCommitter は membership CSV の 1 行を確定する境界 port
	// (wi-351)。Group 本体の import committer とは変更集合が別なので分けている。
	GroupMembershipImportCommitter groupports.GroupMembershipImportRowCommitter
	// UserMutationCommitter は User mutation を確定させる境界 port。IdGovernance が
	// 実装を注入する。nil のとき admin usecase は UserRepo.Save に fallback する。
	UserMutationCommitter userports.UserMutationCommitter
	// ProvisioningNotifier は outbound Provisioning (wi-45) の境界 port。
	// nil のとき outbound provisioning は未配線として何もしない。
	ProvisioningNotifier userports.ProvisioningNotifier
	// GroupProvisioningNotifier は Group 変更の境界 port (wi-441)。User 側と分けて
	// いるのは、IdManagement が持つ引き金の語彙が Group と User で別だからである。
	// プロビジョニングタスクを生むかどうかは接続の push_groups が決めるので、常に配線してよい。
	GroupProvisioningNotifier groupports.ProvisioningNotifier
	// AttrSchemaRepo と GroupAttrSchemaRepo は、IdManagement が実装する Tenancy の属性スキーマの
	// 保存先である。組み立て地点が Tenancy の Module へ渡す。
	AttrSchemaRepo      tenantports.TenantUserAttributeSchemaRepository
	GroupAttrSchemaRepo tenantports.TenantGroupAttributeSchemaRepository
}
