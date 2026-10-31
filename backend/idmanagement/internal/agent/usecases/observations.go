package usecases

import (
	"context"
	"errors"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var (
	// ErrObservationBatchInvalid はバッチの件数が範囲にない。
	ErrObservationBatchInvalid = errors.New("agent observation batch must hold 1 to 100 items")
	// ErrObservationNotFound は観測が要求先のテナントに存在しない。
	ErrObservationNotFound = errors.New("agent observation not found")
	// ErrObservationAlreadyDecided は観測にすでに別の決定がある。
	ErrObservationAlreadyDecided = errors.New("agent observation already decided")
	// ErrInvalidObservationDecision は既知でない決定による絞り込みである。
	ErrInvalidObservationDecision = errors.New("decision must be pending, confirmed, or excluded")
	// errObservationSourceMissing は候補の材料を読む先が配線されていない。候補を欠いた一覧を
	// 正常として返さないため、配線の欠落も失敗にする。
	errObservationSourceMissing = errors.New("agent observations require workload bindings")
)

// ObservationView は観測と、その時点の候補である。
type ObservationView struct {
	Observation *agentdomain.AgentObservation
	Match       agentdomain.ObservationMatch
	AgentNames  map[string]string
	Stale       bool
}

// ImportObservations はバッチを未確認の報告として取り込む (REQ-IDMANAGEMENT-096、097)。全件を検証
// してから保存し、Agent、束縛、ロールには触れない (REQ-IDMANAGEMENT-101)。
func ImportObservations(
	ctx context.Context, deps AdminAgentDeps, actorUserID, sourceTokenID string, batch []*agentdomain.AgentObservation, now time.Time,
) (agentports.ObservationImportCounts, error) {
	if len(batch) == 0 || len(batch) > agentdomain.MaxObservationBatch {
		return agentports.ObservationImportCounts{}, ErrObservationBatchInvalid
	}
	now = idmusecases.NormalizedNow(now)
	tenantID := tenantports.TenantID(ctx)
	for _, item := range batch {
		item.Normalize()
		if err := item.Validate(now); err != nil {
			return agentports.ObservationImportCounts{}, err
		}
		id, err := agentdomain.NewAgentObservationID()
		if err != nil {
			return agentports.ObservationImportCounts{}, err
		}
		item.ID, item.TenantID, item.SourceTokenID, item.CreatedAt = id, tenantID, sourceTokenID, now
		item.ObservedAt = item.ObservedAt.UTC()
	}
	counts, err := deps.Observations.Import(ctx, tenantID, batch, agentdomain.MaxTenantObservations)
	if err != nil {
		return agentports.ObservationImportCounts{}, err
	}
	if counts.Created+counts.Updated == 0 {
		return counts, nil
	}
	return counts, idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentObservationsImported{
		At: now, TenantID: tenantID, ActorUserID: actorUserID, SourceTokenID: sourceTokenID,
		Created: counts.Created, Updated: counts.Updated, Unchanged: counts.Unchanged,
	})
}

// ViewObservations は観測ごとに候補を計算する (REQ-IDMANAGEMENT-098)。候補は保存せず、どの材料の
// 読み取りの失敗も、候補を欠いた一覧を返さずに error にする。
func ViewObservations(ctx context.Context, deps AdminAgentDeps, observations []*agentdomain.AgentObservation, now time.Time) ([]ObservationView, error) {
	views := make([]ObservationView, len(observations))
	for i, observation := range observations {
		view, err := viewObservation(ctx, deps, observation, now)
		if err != nil {
			return nil, err
		}
		views[i] = view
	}
	return views, nil
}

func viewObservation(ctx context.Context, deps AdminAgentDeps, observation *agentdomain.AgentObservation, now time.Time) (ObservationView, error) {
	evidence, err := observationEvidence(ctx, deps, observation)
	if err != nil {
		return ObservationView{}, err
	}
	match := agentdomain.MatchObservation(*observation, evidence)
	names := map[string]string{}
	candidates := match.Candidates[:0]
	for _, candidate := range match.Candidates {
		agent, err := deps.AgentRepo.FindByID(ctx, observation.TenantID, candidate.AgentID)
		if err != nil {
			return ObservationView{}, err
		}
		if agent == nil {
			continue
		}
		names[agent.ID] = agent.Name
		candidates = append(candidates, candidate)
	}
	match.Candidates = candidates
	if observation.Decision() == agentdomain.ObservationPending && len(candidates) == 0 {
		match.State = agentdomain.ObservationStateUnregistered
	}
	return ObservationView{Observation: observation, Match: match, AgentNames: names, Stale: observation.Stale(now)}, nil
}

func observationEvidence(ctx context.Context, deps AdminAgentDeps, observation *agentdomain.AgentObservation) ([]agentdomain.MatchEvidence, error) {
	if deps.WorkloadSubjects == nil {
		return nil, errObservationSourceMissing
	}
	agentIDs, err := deps.WorkloadSubjects.AgentsForSubject(ctx, observation.TenantID, observation.Issuer, observation.Subject)
	if err != nil {
		return nil, err
	}
	evidence := make([]agentdomain.MatchEvidence, 0, len(agentIDs)+1)
	for _, agentID := range agentIDs {
		evidence = append(evidence, agentdomain.MatchEvidence{AgentID: agentID, Reason: agentdomain.MatchReasonWorkloadBinding})
	}
	if observation.ExternalRegistrationID != nil {
		agent, err := deps.AgentRepo.FindByClientID(ctx, observation.TenantID, *observation.ExternalRegistrationID)
		if err != nil {
			return nil, err
		}
		if agent != nil {
			evidence = append(evidence, agentdomain.MatchEvidence{AgentID: agent.ID, Reason: agentdomain.MatchReasonClientBinding})
		}
	}
	return evidence, nil
}

// ConfirmAgentMatch は未決定の観測を同じテナントの Agent に結ぶ (REQ-IDMANAGEMENT-099)。確定は
// 条件付きの更新で行い、並行する先の決定を上書きしない。Agent の状態は変えない
// (REQ-IDMANAGEMENT-101)。
func ConfirmAgentMatch(ctx context.Context, deps AdminAgentDeps, actorUserID, observationID, agentID string, now time.Time) (ObservationView, error) {
	now = idmusecases.NormalizedNow(now)
	tenantID := tenantports.TenantID(ctx)
	observation, err := findObservation(ctx, deps, tenantID, observationID)
	if err != nil {
		return ObservationView{}, err
	}
	if observation.ConfirmedAgentID != nil && *observation.ConfirmedAgentID == agentID {
		return viewObservation(ctx, deps, observation, now)
	}
	if observation.Decision() != agentdomain.ObservationPending {
		return ObservationView{}, ErrObservationAlreadyDecided
	}
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, agentID)
	if err != nil {
		return ObservationView{}, err
	}
	if agent == nil {
		return ObservationView{}, ErrAgentNotFound
	}
	evidence, err := observationEvidence(ctx, deps, observation)
	if err != nil {
		return ObservationView{}, err
	}
	changed, err := deps.Observations.Confirm(ctx, tenantID, observationID, agentID, actorUserID, now)
	if errors.Is(err, agentports.ErrObservationAgentMissing) {
		return ObservationView{}, ErrAgentNotFound
	}
	if err != nil {
		return ObservationView{}, err
	}
	current, err := findObservation(ctx, deps, tenantID, observationID)
	if err != nil {
		return ObservationView{}, err
	}
	if !changed {
		if current.ConfirmedAgentID != nil && *current.ConfirmedAgentID == agentID {
			return viewObservation(ctx, deps, current, now)
		}
		return ObservationView{}, ErrObservationAlreadyDecided
	}
	reasons := []string{}
	for _, item := range evidence {
		if item.AgentID == agentID {
			reasons = append(reasons, string(item.Reason))
		}
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentObservationConfirmed{
		At: now, TenantID: tenantID, ActorUserID: actorUserID, ObservationID: observationID, AgentID: agentID, Reasons: reasons,
	}); err != nil {
		return ObservationView{}, err
	}
	return viewObservation(ctx, deps, current, now)
}

// ExcludeObservation は未決定の観測を除外する (REQ-IDMANAGEMENT-100)。
func ExcludeObservation(ctx context.Context, deps AdminAgentDeps, actorUserID, observationID string, now time.Time) (ObservationView, error) {
	now = idmusecases.NormalizedNow(now)
	tenantID := tenantports.TenantID(ctx)
	if _, err := findObservation(ctx, deps, tenantID, observationID); err != nil {
		return ObservationView{}, err
	}
	changed, err := deps.Observations.Exclude(ctx, tenantID, observationID, actorUserID, now)
	if err != nil {
		return ObservationView{}, err
	}
	current, err := findObservation(ctx, deps, tenantID, observationID)
	if err != nil {
		return ObservationView{}, err
	}
	if !changed {
		if current.Excluded {
			return viewObservation(ctx, deps, current, now)
		}
		return ObservationView{}, ErrObservationAlreadyDecided
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentObservationExcluded{
		At: now, TenantID: tenantID, ActorUserID: actorUserID, ObservationID: observationID,
	}); err != nil {
		return ObservationView{}, err
	}
	return viewObservation(ctx, deps, current, now)
}

// ReopenObservation は確認または除外した観測を未決定へ戻す (REQ-IDMANAGEMENT-100)。権限は増減しない。
func ReopenObservation(ctx context.Context, deps AdminAgentDeps, actorUserID, observationID string, now time.Time) (ObservationView, error) {
	now = idmusecases.NormalizedNow(now)
	tenantID := tenantports.TenantID(ctx)
	previous, err := deps.Observations.Reopen(ctx, tenantID, observationID)
	if err != nil {
		return ObservationView{}, err
	}
	if previous == nil {
		return ObservationView{}, ErrObservationNotFound
	}
	if previous.Decision() != agentdomain.ObservationPending {
		if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.AgentObservationReopened{
			At: now, TenantID: tenantID, ActorUserID: actorUserID, ObservationID: observationID, PreviousAgentID: previous.ConfirmedAgentID,
		}); err != nil {
			return ObservationView{}, err
		}
	}
	current, err := findObservation(ctx, deps, tenantID, observationID)
	if err != nil {
		return ObservationView{}, err
	}
	return viewObservation(ctx, deps, current, now)
}

func findObservation(ctx context.Context, deps AdminAgentDeps, tenantID, id string) (*agentdomain.AgentObservation, error) {
	observation, err := deps.Observations.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if observation == nil {
		return nil, ErrObservationNotFound
	}
	return observation, nil
}

// ParseObservationDecision は一覧の絞り込みを検証する。空は絞り込まない。
func ParseObservationDecision(raw string) (agentdomain.ObservationDecisionKind, error) {
	decision := agentdomain.ObservationDecisionKind(raw)
	if raw != "" && !decision.Valid() {
		return "", ErrInvalidObservationDecision
	}
	return decision, nil
}

// PurgeExpiredObservations は保持期限を過ぎた、確認していない観測を消す (REQ-IDMANAGEMENT-102)。
func PurgeExpiredObservations(ctx context.Context, repo agentports.AgentObservationRepository, now time.Time) (int64, error) {
	return repo.PurgeExpired(ctx, now.Add(-agentdomain.ObservationRetention))
}
