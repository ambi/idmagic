package usecases

import (
	"context"
	"errors"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ErrAgentOwnerInactive は、所有者の User が Active でない Agent を再有効化しようとした場合に返る。
var ErrAgentOwnerInactive = errors.New("agent owner is not active")

// AgentOwnerDeps は、所有者の停止を Agent へ伝えるために要る依存である。User のユースケースが、
// Agent の管理に要るほかの依存（OAuth2 のクライアントや使用量）を持たずに呼べるよう分けている。
type AgentOwnerDeps struct {
	AgentRepo agentports.AgentRepository
	Emit      func(spec.DomainEvent) error
}

// DisableAgentsOwnedBy は、ownerUserID が所有する Active の Agent をすべて Disabled にし、
// Agent ごとに AgentDisabled を発行する。Disabled と Killed の Agent は変えない。
// 所有者の User を止めたユースケースが、その確定の後に呼ぶ。
func DisableAgentsOwnedBy(ctx context.Context, deps AgentOwnerDeps, tenantID, ownerUserID, actorUserID string, now time.Time) error {
	agents, err := deps.AgentRepo.ListAll(ctx, tenantID)
	if err != nil {
		return err
	}
	now = idmusecases.NormalizedNow(now)
	for _, agent := range agents {
		if agent.OwnerUserID != ownerUserID || agent.Status != idmdomain.AgentStatusActive {
			continue
		}
		updated := *agent
		updated.Status = idmdomain.AgentStatusDisabled
		updated.DisabledAt = &now
		updated.UpdatedAt = now
		if err := deps.AgentRepo.Save(ctx, &updated); err != nil {
			return err
		}
		if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentDisabled{At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: agent.ID}); err != nil {
			return err
		}
	}
	return nil
}
