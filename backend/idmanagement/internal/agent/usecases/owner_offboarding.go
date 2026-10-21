package usecases

import (
	"context"
	"errors"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
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
		// 一覧を読んだ後の管理者の更新や停止を書き戻さないよう、読み直した最新の Agent が
		// まだこの所有者の Active であるときだけ無効化する。
		disabled := false
		if _, err := deps.AgentRepo.Update(ctx, tenantID, agent.ID, func(latest *agentdomain.Agent) (bool, error) {
			if latest.OwnerUserID != ownerUserID || latest.Status != idmdomain.AgentStatusActive {
				return false, nil
			}
			latest.Status = idmdomain.AgentStatusDisabled
			latest.DisabledAt = &now
			latest.UpdatedAt = now
			disabled = true
			return true, nil
		}); err != nil {
			return err
		}
		if !disabled {
			continue
		}
		if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentDisabled{At: now, TenantID: tenantID, ActorUserID: actorUserID, AgentID: agent.ID}); err != nil {
			return err
		}
	}
	return nil
}
