// Package idgovernance は IdGovernance bounded context の DI 組立を所有する
// (wi-237, ADR-117)。LifecycleWorkflow (JML 自動化) の定義・実行・executor と、
// User mutation を観測して run を生成する境界 port の実装を束ねる。HTTP route 登録は
// 他 context 同様に中央 server/routes.go が組み立てる (自己登録は行わない)。
package idgovernance

import (
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// Module は identity-governance context が所有する永続化 port と境界実装の束。
type Module struct {
	LifecycleWorkflowRepo    igports.LifecycleWorkflowRepository
	LifecycleWorkflowRunRepo igports.LifecycleWorkflowRunRepository
	UserWorkflowCapture      igports.UserWorkflowCapture
	// UserMutationCommitter は IdManagement の境界 port を実装し、composition root が
	// IdManagement.Module.UserMutationCommitter へ注入する。
	UserMutationCommitter userports.UserMutationCommitter
}
