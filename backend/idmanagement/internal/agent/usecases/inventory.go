package usecases

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var (
	// ErrInvalidDormantDays は dormant_days が 1 から MaxDormantDays の範囲にない。
	ErrInvalidDormantDays = errors.New("dormant_days must be between 1 and 365")
	// ErrInvalidInventoryFinding は既知でない所見による絞り込みである。
	ErrInvalidInventoryFinding = errors.New("finding must be owner_missing, dormant, or unbound")
	// errInventorySourceMissing は所見の材料を読む先が配線されていない。所見を欠いた一覧を
	// 正常として返さないため、配線の欠落も失敗にする。
	errInventorySourceMissing = errors.New("agent inventory requires user, client, activity, and observation sources")
)

// InventoryFilter は棚卸しの条件である。Finding が空なら所見で絞り込まない。
type InventoryFilter struct {
	Finding     agentdomain.InventoryFinding
	DormantDays int
}

// InventoryEntry は棚卸しの Agent 一件と、その所見である。
type InventoryEntry struct {
	Facts    agentdomain.InventoryFacts
	Findings []agentdomain.InventoryFinding
	// ConfirmedObservations は管理者がこの Agent との一致を確認した観測である。
	ConfirmedObservations []*agentdomain.AgentObservation
}

// ListAgentInventory は、テナントのすべての Agent について所有者、束縛先、最終のトークン発行を
// 読み、所見を付けて名前の昇順 (同じなら id の順) に返す (REQ-IDMANAGEMENT-094)。
// 棚卸しは状態を持たず、読んだ事実から毎回組み立てる。どの読み取りの失敗も、所見を欠いた
// 一覧を返さずに error にする。
func ListAgentInventory(ctx context.Context, deps AdminAgentDeps, filter InventoryFilter, now time.Time) ([]InventoryEntry, error) {
	if filter.DormantDays == 0 {
		filter.DormantDays = agentdomain.DefaultDormantDays
	}
	if filter.DormantDays < 1 || filter.DormantDays > agentdomain.MaxDormantDays {
		return nil, ErrInvalidDormantDays
	}
	if filter.Finding != "" && !filter.Finding.Valid() {
		return nil, ErrInvalidInventoryFinding
	}
	if deps.UserRepo == nil || deps.ClientRepo == nil || deps.Activity == nil || deps.Observations == nil {
		return nil, errInventorySourceMissing
	}
	tenantID := tenantports.TenantID(ctx)
	agents, err := deps.AgentRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(agents, func(a, b *agentdomain.Agent) int {
		return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	agentIDs := make([]string, len(agents))
	for i, agent := range agents {
		agentIDs[i] = agent.ID
	}
	lastIssued, err := deps.Activity.LastTokenIssuedAt(ctx, tenantID, agentIDs)
	if err != nil {
		return nil, err
	}
	confirmed, err := deps.Observations.ListConfirmed(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	observationsByAgent := map[string][]*agentdomain.AgentObservation{}
	for _, observation := range confirmed {
		observationsByAgent[*observation.ConfirmedAgentID] = append(observationsByAgent[*observation.ConfirmedAgentID], observation)
	}
	owners := map[string]agentdomain.OwnerStatus{}
	entries := make([]InventoryEntry, 0, len(agents))
	for _, agent := range agents {
		clientIDs, err := agentClientIDs(ctx, deps, tenantID, agent.ID)
		if err != nil {
			return nil, err
		}
		missing, err := missingClientIDs(ctx, deps, tenantID, clientIDs)
		if err != nil {
			return nil, err
		}
		ownerStatus, ok := owners[agent.OwnerUserID]
		if !ok {
			if ownerStatus, err = readOwnerStatus(ctx, deps, tenantID, agent.OwnerUserID); err != nil {
				return nil, err
			}
			owners[agent.OwnerUserID] = ownerStatus
		}
		facts := agentdomain.InventoryFacts{
			Agent: agent, OwnerStatus: ownerStatus, ClientIDs: clientIDs, MissingClientIDs: missing,
		}
		if issuedAt, ok := lastIssued[agent.ID]; ok {
			facts.LastTokenIssuedAt = &issuedAt
		}
		findings := facts.Findings(now, filter.DormantDays)
		if filter.Finding != "" && !slices.Contains(findings, filter.Finding) {
			continue
		}
		entries = append(entries, InventoryEntry{Facts: facts, Findings: findings, ConfirmedObservations: observationsByAgent[agent.ID]})
	}
	return entries, nil
}

func missingClientIDs(ctx context.Context, deps AdminAgentDeps, tenantID string, clientIDs []string) ([]string, error) {
	missing := []string{}
	for _, clientID := range clientIDs {
		exists, _, err := deps.ClientRepo.LookupBindableClient(ctx, tenantID, clientID)
		if err != nil {
			return nil, err
		}
		if !exists {
			missing = append(missing, clientID)
		}
	}
	return missing, nil
}

// readOwnerStatus は所有者の状態を読む。別のテナントの User と削除済みの User は、同じテナントに
// 存在しない所有者として扱う。
func readOwnerStatus(ctx context.Context, deps AdminAgentDeps, tenantID, ownerUserID string) (agentdomain.OwnerStatus, error) {
	owner, err := deps.UserRepo.FindBySub(ctx, ownerUserID)
	if err != nil {
		return "", err
	}
	if owner == nil || owner.TenantID != tenantID || owner.IsDeleted() {
		return agentdomain.OwnerStatusMissing, nil
	}
	switch owner.Lifecycle.EffectiveStatus() {
	case idmdomain.UserStatusActive:
		return agentdomain.OwnerStatusActive, nil
	case idmdomain.UserStatusPendingDeletion:
		return agentdomain.OwnerStatusPendingDeletion, nil
	case idmdomain.UserStatusDisabled:
		return agentdomain.OwnerStatusDisabled, nil
	}
	return agentdomain.OwnerStatusMissing, nil
}
