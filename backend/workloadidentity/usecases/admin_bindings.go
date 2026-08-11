// 管理者向け AgentWorkloadBinding ライフサイクル操作。SCL WorkloadIdentity
// bounded context が所有する admin インターフェース群: ListAgentWorkloadBindings /
// CreateAgentWorkloadBinding / DisableAgentWorkloadBinding /
// EnableAgentWorkloadBinding / DeleteAgentWorkloadBinding。
package usecases

import (
	"context"
	"strings"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// AdminAgentWorkloadBindingDeps extends AdminWorkloadIdentityDeps with the
// AgentRepository needed to validate that agent_id stays tenant-local
// (WorkloadIdentityReferencesStayTenantLocal).
type AdminAgentWorkloadBindingDeps struct {
	AdminWorkloadIdentityDeps
	AgentRepo agentports.AgentRepository
}

func ListAgentWorkloadBindings(ctx context.Context, deps AdminAgentWorkloadBindingDeps, trustBundleID string) ([]*domain.AgentWorkloadBinding, error) {
	tenantID := tenancy.TenantID(ctx)
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenantID, trustBundleID)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, ErrTrustBundleNotFound
	}
	return deps.BindingRepo.ListByTrustBundle(ctx, tenantID, trustBundleID)
}

type CreateAgentWorkloadBindingInput struct {
	SubjectPattern string
	AgentID        string
}

// CreateAgentWorkloadBinding は WorkloadTrustBundle 配下に binding を作成する。
// agent_id は同一テナントの既存 Agent でなければならず、subject_pattern は同一
// trust_bundle_id 内で完全重複を許さない (ambiguous match の runtime 判定は
// domain.MatchAgent が別途担う)。
func CreateAgentWorkloadBinding(ctx context.Context, deps AdminAgentWorkloadBindingDeps, trustBundleID string, in CreateAgentWorkloadBindingInput, now time.Time) (*domain.AgentWorkloadBinding, error) {
	tenantID := tenancy.TenantID(ctx)
	bundle, err := deps.TrustBundleRepo.FindByID(ctx, tenantID, trustBundleID)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, ErrTrustBundleNotFound
	}
	pattern := strings.TrimSpace(in.SubjectPattern)
	if pattern == "" {
		return nil, ErrBindingSubjectPatternEmpty
	}
	agentID := strings.TrimSpace(in.AgentID)
	if agentID == "" {
		return nil, ErrBindingAgentNotFound
	}
	if deps.AgentRepo == nil {
		return nil, ErrBindingAgentNotFound
	}
	agent, err := deps.AgentRepo.FindByID(ctx, tenantID, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrBindingAgentNotFound
	}
	existing, err := deps.BindingRepo.ListByTrustBundle(ctx, tenantID, trustBundleID)
	if err != nil {
		return nil, err
	}
	for _, b := range existing {
		if b.SubjectPattern == pattern {
			return nil, ErrBindingSubjectPatternExists
		}
	}
	id, err := domain.NewAgentWorkloadBindingID()
	if err != nil {
		return nil, err
	}
	binding := &domain.AgentWorkloadBinding{
		ID: id, TenantID: tenantID, TrustBundleID: trustBundleID, SubjectPattern: pattern,
		AgentID: agentID, Status: domain.AgentWorkloadBindingStatusEnabled, CreatedAt: now,
	}
	if err := binding.Validate(); err != nil {
		return nil, err
	}
	if err := deps.BindingRepo.Save(ctx, binding); err != nil {
		return nil, err
	}
	emit(deps.Emit, &domain.AgentWorkloadBindingCreated{
		At: now, TenantID: tenantID, TrustBundleID: trustBundleID, BindingID: binding.ID, AgentID: agentID,
	})
	return binding, nil
}

func DisableAgentWorkloadBinding(ctx context.Context, deps AdminAgentWorkloadBindingDeps, id string, now time.Time) (*domain.AgentWorkloadBinding, error) {
	return setBindingStatus(ctx, deps, id, domain.AgentWorkloadBindingStatusDisabled, now)
}

func EnableAgentWorkloadBinding(ctx context.Context, deps AdminAgentWorkloadBindingDeps, id string, now time.Time) (*domain.AgentWorkloadBinding, error) {
	return setBindingStatus(ctx, deps, id, domain.AgentWorkloadBindingStatusEnabled, now)
}

func setBindingStatus(ctx context.Context, deps AdminAgentWorkloadBindingDeps, id string, status domain.AgentWorkloadBindingStatus, now time.Time) (*domain.AgentWorkloadBinding, error) {
	tenantID := tenancy.TenantID(ctx)
	binding, err := deps.BindingRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if binding == nil {
		return nil, ErrBindingNotFound
	}
	if binding.Status == status {
		return binding, nil
	}
	updated := *binding
	updated.Status = status
	updated.UpdatedAt = &now
	if status == domain.AgentWorkloadBindingStatusDisabled {
		updated.DisabledAt = &now
	} else {
		updated.DisabledAt = nil
	}
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.BindingRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if status == domain.AgentWorkloadBindingStatusDisabled {
		emit(deps.Emit, &domain.AgentWorkloadBindingDisabled{At: now, TenantID: tenantID, BindingID: binding.ID})
	} else {
		emit(deps.Emit, &domain.AgentWorkloadBindingEnabled{At: now, TenantID: tenantID, BindingID: binding.ID})
	}
	return &updated, nil
}

func DeleteAgentWorkloadBinding(ctx context.Context, deps AdminAgentWorkloadBindingDeps, id string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	binding, err := deps.BindingRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if binding == nil {
		return ErrBindingNotFound
	}
	if err := deps.BindingRepo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	emit(deps.Emit, &domain.AgentWorkloadBindingDeleted{At: now, TenantID: tenantID, BindingID: id})
	return nil
}
