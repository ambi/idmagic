package usecases

// 本人が Agent へ与える代行の付与の操作 (REQ-OAUTH2-057、REQ-OAUTH2-058、REQ-OAUTH2-059)。
// 付与は同意と同じ保管先の別の行で、撤回、期限、クォータの規則を同意と共有する。

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// MaxAgentDelegationLifetime は付与の期限の既定かつ上限で、同意の有効期間と同じである。
const MaxAgentDelegationLifetime = 365 * 24 * time.Hour

var (
	// ErrDelegationAgentNotFound は、付与の相手の Agent が所属テナントに存在しないか
	// Active でないことを表す。
	ErrDelegationAgentNotFound = errors.New("agent not found")
	// ErrInvalidDelegationGrant は、付与の入力が要件を満たさないことを表す。
	ErrInvalidDelegationGrant = errors.New("invalid agent delegation grant")
)

// AgentDelegationDeps は、付与の相手の Agent を確かめる操作の依存である。
type AgentDelegationDeps struct {
	ConsentDeps
	Agents agentports.AgentRepository
}

// AgentDelegationGrant は本人の付与の入力である。ExpiresAt が nil なら上限の期限にする。
type AgentDelegationGrant struct {
	Sub       string
	AgentID   string
	Scopes    []string
	ExpiresAt *time.Time
}

// GrantAgentDelegation は Agent への代行を付与し、既にあれば置き換える。
func GrantAgentDelegation(ctx context.Context, deps AgentDelegationDeps, in AgentDelegationGrant, now time.Time) (*domain.Consent, error) {
	scopes := normalizeDelegationScopes(in.Scopes)
	latest := now.Add(MaxAgentDelegationLifetime)
	expiresAt := latest
	if in.ExpiresAt != nil {
		expiresAt = *in.ExpiresAt
	}
	if len(scopes) == 0 || !expiresAt.After(now) || expiresAt.After(latest) {
		return nil, ErrInvalidDelegationGrant
	}
	tenantID := tenantports.TenantID(ctx)
	if deps.Agents == nil {
		return nil, ErrDelegationAgentNotFound
	}
	agent, err := deps.Agents.FindByID(ctx, tenantID, in.AgentID)
	if err != nil {
		return nil, err
	}
	if agent == nil || !agent.IsActive() {
		return nil, ErrDelegationAgentNotFound
	}
	existing, err := deps.ConsentRepo.FindAgentDelegation(ctx, tenantID, in.Sub, in.AgentID)
	if err != nil {
		return nil, err
	}
	// 同意の付与と同じく、期限切れの行は枠を持ったままなので足さない。
	if deps.QuotaRepo != nil && (existing == nil || existing.State == domain.ConsentRevoked) {
		if err := deps.QuotaRepo.CheckAndIncrement(ctx, tenantID, tenancydomain.ResourceConsents, 1); err != nil {
			if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
				emit(deps.Emit, &tenancydomain.QuotaExceeded{At: now, TenantID: tenantID, Resource: qErr.Resource, HardLimit: true})
			}
			return nil, err
		}
	}
	grant := &domain.Consent{
		UserID: in.Sub, AgentID: in.AgentID, Scopes: scopes,
		State: domain.ConsentGranted, GrantedAt: now, ExpiresAt: expiresAt,
	}
	if err := deps.ConsentRepo.SaveAgentDelegation(ctx, tenantID, grant); err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.AgentDelegationGranted{
		At: now, TenantID: tenantID, UserID: in.Sub, AgentID: in.AgentID, Scopes: scopes, ExpiresAt: expiresAt,
	})
	return grant, nil
}

// ListGrantedAgentDelegations は sub の Granted で期限内の付与だけを返す。
func ListGrantedAgentDelegations(ctx context.Context, deps ConsentDeps, sub string, now time.Time) ([]*domain.Consent, error) {
	all, err := deps.ConsentRepo.ListAgentDelegations(ctx, tenantports.TenantID(ctx), sub)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(all, func(grant *domain.Consent) bool {
		return grant.State != domain.ConsentGranted || !now.Before(grant.ExpiresAt)
	}), nil
}

// RevokeAgentDelegation は sub の付与を撤回する。発行済みトークンへの伝播は、
// イントロスペクションが付与を読むことで成り立つ (REQ-OAUTH2-061)。
func RevokeAgentDelegation(ctx context.Context, deps ConsentDeps, sub, agentID string, now time.Time) error {
	tenantID := tenantports.TenantID(ctx)
	grant, err := deps.ConsentRepo.FindAgentDelegation(ctx, tenantID, sub, agentID)
	if err != nil {
		return err
	}
	if grant == nil {
		return ErrConsentNotFound
	}
	if err := deps.ConsentRepo.RevokeAgentDelegation(ctx, tenantID, sub, agentID); err != nil {
		return err
	}
	if deps.QuotaRepo != nil && grant.State == domain.ConsentGranted {
		if err := deps.QuotaRepo.Decrement(ctx, tenantID, tenancydomain.ResourceConsents, 1); err != nil {
			return err
		}
	}
	emit(deps.Emit, &domain.AgentDelegationRevoked{At: now, TenantID: tenantID, UserID: sub, AgentID: agentID})
	return nil
}

func normalizeDelegationScopes(scopes []string) []string {
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope != "" && !slices.Contains(out, scope) {
			out = append(out, scope)
		}
	}
	return out
}
