// Package usecases owns the SharedSignals-internal interfaces
// CheckRevocationEpoch / AdvanceRevocationEpoch and the AgentRevocationReactor
// that reacts to IdManagement DomainEvents already flowing through the
// composition root's Emit pipeline (wi-58).
package usecases

import (
	"context"
	"errors"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
)

// RevocationDeps holds what AdvanceRevocationEpoch / CheckRevocationEpoch need.
type RevocationDeps struct {
	EpochRepo ssports.AgentRevocationEpochRepository
	Emit      func(spec.DomainEvent) error
}

func emit(sink func(spec.DomainEvent) error, event spec.DomainEvent) error {
	if sink == nil {
		return nil
	}
	return sink(event)
}

// AdvanceRevocationEpoch implements the SCL internal interface
// AdvanceRevocationEpoch (spec/contexts/sharedsignals.yaml): it fail-closed
// advances the revocation epoch of every agent in agentIDs to now, and emits
// RevocationEpochAdvanced + AgentAccessRevoked for each agent actually
// advanced. An agent whose epoch is already at or after now
// (ErrEpochNotAdvancing) is an idempotent no-op, not an error — the
// invariant it protects (no token issued before now stays valid) already
// holds via the existing, later-or-equal epoch.
func AdvanceRevocationEpoch(ctx context.Context, deps RevocationDeps, tenantID string, agentIDs []string, reason ssdomain.RevocationReason, sourceEventID *string, now time.Time) error {
	for _, agentID := range agentIDs {
		epoch := ssdomain.AgentRevocationEpoch{
			AgentID: agentID, TenantID: tenantID, Epoch: now, Reason: reason,
			AdvancedAt: now, SourceEventID: sourceEventID,
		}
		if err := epoch.Validate(); err != nil {
			return err
		}
		if err := deps.EpochRepo.Advance(ctx, epoch); err != nil {
			if errors.Is(err, ssdomain.ErrEpochNotAdvancing) {
				continue
			}
			return err
		}
		if err := emit(deps.Emit, &ssdomain.RevocationEpochAdvanced{At: now, TenantID: tenantID, AgentID: agentID, Reason: reason, Epoch: now}); err != nil {
			return err
		}
		if err := emit(deps.Emit, &ssdomain.AgentAccessRevoked{At: now, TenantID: tenantID, AgentID: agentID, Reason: reason}); err != nil {
			return err
		}
	}
	return nil
}

// CheckRevocationEpoch implements the SCL internal interface
// CheckRevocationEpoch: it returns the agent's current revocation epoch, or
// nil if the agent has never been revoked.
func CheckRevocationEpoch(ctx context.Context, deps RevocationDeps, tenantID, agentID string) (*ssdomain.AgentRevocationEpoch, error) {
	return deps.EpochRepo.FindByAgent(ctx, tenantID, agentID)
}

// AgentRevocationReactor reacts to IdManagement DomainEvents that IdManagement
// already emits unconditionally for audit (KillAgent/SetAgentDisabled/
// UnbindCredential/SetUserDisabled/SoftDeleteUser/DeleteUser), and
// fail-closed advances the affected agent(s)' revocation epoch (
// wi-58). Every path that emits those events builds it with
// NewAgentRevocationReactor and calls it after each Emit: the admin API through
// idmanagement/deps_http.Deps.ReactiveEmit, and the paths that stop a User
// outside the admin API through bootstrap's UserLifecycleCommands. IdManagement's own usecases
// need no SharedSignals dependency or extra call — the event they already
// emit is the trigger. This mirrors the shape [[wi-323]] plans to reuse for
// human User sessions (more event types, same reactor pattern).
type AgentRevocationReactor struct {
	EpochRepo ssports.AgentRevocationEpochRepository
	// AgentRepo resolves the agent set owned by a disabled/deleted user
	// (SharedSignals depends on IdManagement's AgentRepository through its
	// public ports package).
	AgentRepo agentports.AgentRepository
	// Emit records the derived RevocationEpochAdvanced / AgentAccessRevoked
	// events (best-effort audit trail is the caller's concern to compose;
	// see AdvanceRevocationEpoch's doc for its own error propagation).
	Emit func(spec.DomainEvent) error
}

// RevocationReactorDeps は、組み立ての地点が NewAgentRevocationReactor へ渡す依存である。
type RevocationReactorDeps struct {
	EpochRepo ssports.AgentRevocationEpochRepository
	AgentRepo agentports.AgentRepository
	Projector ProjectorDeps
	// Emit は派生イベント (RevocationEpochAdvanced、AgentAccessRevoked) を記録する配信点である。
	Emit func(spec.DomainEvent)
}

// NewAgentRevocationReactor は、IdManagement のイベントを発行するすべての経路が使う反応器を組み立てる。
// 派生イベントを記録した後、AgentAccessRevoked を外部への伝播へ投影する。投影はローカルの失効を
// 遅らせも失敗させもしない (REQ-SHAREDSIGNALS-007) ので、自前の 5 秒の期限で走らせ、失敗は記録して続ける。
func NewAgentRevocationReactor(deps RevocationReactorDeps) *AgentRevocationReactor {
	return &AgentRevocationReactor{
		EpochRepo: deps.EpochRepo,
		AgentRepo: deps.AgentRepo,
		Emit: func(event spec.DomainEvent) error {
			if deps.Emit != nil {
				deps.Emit(event)
			}
			if revoked, ok := event.(*ssdomain.AgentAccessRevoked); ok {
				projectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := ProjectAgentAccessRevoked(projectCtx, deps.Projector, revoked); err != nil {
					logging.Error(projectCtx, "sharedsignals: SET projection failed", "error", err, "agent_id", revoked.AgentID, "tenant_id", revoked.TenantID)
				}
			}
			return nil
		},
	}
}

// React implements the deps_http.EventReactor shape structurally
// (React(ctx, spec.DomainEvent) error) without importing idmanagement's
// deps_http package (dependency direction: SharedSignals depends on
// IdManagement, not the reverse). A nil EpochRepo (lightweight test wiring
// that never sets sharedsignals.Module) skips reaction, mirroring this
// codebase's other nil-skips-enforcement ports (e.g. QuotaRepo).
func (r *AgentRevocationReactor) React(ctx context.Context, event spec.DomainEvent) error {
	if r.EpochRepo == nil {
		return nil
	}
	deps := RevocationDeps{EpochRepo: r.EpochRepo, Emit: r.Emit}
	switch e := event.(type) {
	case *idmdomain.AgentKilled:
		return AdvanceRevocationEpoch(ctx, deps, e.TenantID, []string{e.AgentID}, ssdomain.RevocationReasonAgentKilled, nil, e.At)
	case *idmdomain.AgentDisabled:
		return AdvanceRevocationEpoch(ctx, deps, e.TenantID, []string{e.AgentID}, ssdomain.RevocationReasonAgentDisabled, nil, e.At)
	case *idmdomain.AgentCredentialUnbound:
		return AdvanceRevocationEpoch(ctx, deps, e.TenantID, []string{e.AgentID}, ssdomain.RevocationReasonAgentCredentialUnbound, nil, e.At)
	case *idmdomain.UserDisabled:
		return r.notifyOwner(ctx, deps, e.TenantID, e.TargetUserID, ssdomain.RevocationReasonOwnerDisabled, e.At)
	case *idmdomain.UserSoftDeleted:
		return r.notifyOwner(ctx, deps, e.TenantID, e.TargetUserID, ssdomain.RevocationReasonOwnerDisabled, e.At)
	case *idmdomain.UserDeleted:
		return r.notifyOwner(ctx, deps, e.TenantID, e.TargetUserID, ssdomain.RevocationReasonOwnerDeleted, e.At)
	}
	return nil
}

// notifyOwner resolves every agent owned by ownerUserID
// (agentdomain.Agent.OwnerUserID, the owner's Sub) and advances each one's
// revocation epoch. Owner offboarding does not carry an explicit agent set,
// so the reactor must resolve it itself.
func (r *AgentRevocationReactor) notifyOwner(ctx context.Context, deps RevocationDeps, tenantID, ownerUserID string, reason ssdomain.RevocationReason, now time.Time) error {
	if r.AgentRepo == nil {
		return nil
	}
	agents, err := r.AgentRepo.ListAll(ctx, tenantID)
	if err != nil {
		return err
	}
	var agentIDs []string
	for _, agent := range agents {
		if agent.OwnerUserID == ownerUserID {
			agentIDs = append(agentIDs, agent.ID)
		}
	}
	if len(agentIDs) == 0 {
		return nil
	}
	return AdvanceRevocationEpoch(ctx, deps, tenantID, agentIDs, reason, nil, now)
}
