package idmanagement

import (
	"context"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	agentusecases "github.com/ambi/idmagic/backend/idmanagement/internal/agent/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// AgentLifecycleCommands は Agent の状態変更と必須の失効処理を同じ入口から行う。
type AgentLifecycleCommands struct {
	deps  agentusecases.AdminAgentDeps
	actor string
}

func (m Module) AgentLifecycleCommands(emit func(spec.DomainEvent) error, revocation userports.AccessRevocationReactor, actor string) AgentLifecycleCommands {
	return AgentLifecycleCommands{deps: agentusecases.AdminAgentDeps{AgentRepo: m.AgentRepo, UserRepo: m.UserRepo, Emit: emit, Revocation: requireRevocation(revocation)}, actor: actor}
}

// DisableReviewedAgent は、すでに失効した Agent を再開せず、管理操作と同じ失効の経路を通る。
func (c AgentLifecycleCommands) DisableReviewedAgent(ctx context.Context, tenant, id string, now time.Time) error {
	a, err := c.deps.AgentRepo.FindByID(ctx, tenant, id)
	if err != nil || a == nil {
		return err
	}
	if a.Status == idmdomain.AgentStatusKilled {
		return nil
	}
	// 状態の保存後に失効リアクターが失敗することがある。Disabled でも同じ公開操作を
	// 再実行し、状態だけを根拠に失効済みとは扱わない。
	_, err = agentusecases.SetAgentDisabled(tenantports.WithTenantID(ctx, tenant), c.deps, c.actor, id, true, now)
	return err
}
