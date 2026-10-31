package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// AgentGuardrailPolicyRepository はガードレールを PostgreSQL に永続化する。バージョンの比較は一文の
// 条件付きの更新で行い、読んでから書く間の別の保存を上書きしない。
type AgentGuardrailPolicyRepository struct{ Pool sharedpg.DB }

func agentGuardrailFromRow(tenantID, scopeKey string, limits []byte, updatedAt time.Time) (*domain.AgentGuardrailPolicy, error) {
	policy := &domain.AgentGuardrailPolicy{TenantID: tenantID, AgentID: scopeKey, UpdatedAt: updatedAt}
	if err := json.Unmarshal(limits, &policy.AgentGuardrailLimits); err != nil {
		return nil, err
	}
	return policy, nil
}

func (r *AgentGuardrailPolicyRepository) ListEffective(ctx context.Context, tenantID string, agentIDs []string) ([]*domain.AgentGuardrailPolicy, error) {
	rows, err := New(r.Pool).ListEffectiveAgentGuardrailPolicies(ctx, ListEffectiveAgentGuardrailPoliciesParams{
		TenantID: tenantID, Column2: append([]string{""}, agentIDs...),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.AgentGuardrailPolicy, 0, len(rows))
	for _, row := range rows {
		policy, err := agentGuardrailFromRow(row.TenantID, row.ScopeKey, row.Limits, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, policy)
	}
	return out, nil
}

func (r *AgentGuardrailPolicyRepository) Find(ctx context.Context, tenantID, agentID string) (*domain.AgentGuardrailPolicy, error) {
	row, err := New(r.Pool).GetAgentGuardrailPolicy(ctx, GetAgentGuardrailPolicyParams{TenantID: tenantID, ScopeKey: agentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return agentGuardrailFromRow(row.TenantID, row.ScopeKey, row.Limits, row.UpdatedAt)
}

func (r *AgentGuardrailPolicyRepository) Save(ctx context.Context, policy *domain.AgentGuardrailPolicy, expectedVersion string) error {
	limits, err := json.Marshal(policy.AgentGuardrailLimits)
	if err != nil {
		return err
	}
	var affected int64
	if expectedVersion == "" {
		affected, err = New(r.Pool).InsertAgentGuardrailPolicy(ctx, InsertAgentGuardrailPolicyParams{
			TenantID: policy.TenantID, ScopeKey: policy.AgentID, AgentID: agentGuardrailAgentID(policy.AgentID),
			Limits: limits, Version: policy.Version(), UpdatedAt: policy.UpdatedAt,
		})
	} else {
		affected, err = New(r.Pool).UpdateAgentGuardrailPolicy(ctx, UpdateAgentGuardrailPolicyParams{
			TenantID: policy.TenantID, ScopeKey: policy.AgentID,
			Limits: limits, Version: policy.Version(), UpdatedAt: policy.UpdatedAt, ExpectedVersion: expectedVersion,
		})
	}
	if err != nil {
		return err
	}
	if affected == 0 {
		return ports.ErrAgentGuardrailConflict
	}
	return nil
}

func (r *AgentGuardrailPolicyRepository) Delete(ctx context.Context, tenantID, agentID string) error {
	return New(r.Pool).DeleteAgentGuardrailPolicy(ctx, DeleteAgentGuardrailPolicyParams{TenantID: tenantID, ScopeKey: agentID})
}

// agentGuardrailAgentID はテナントのデフォルトに NULL を、Agent に外部キーの値を返す。
func agentGuardrailAgentID(agentID string) pgtype.UUID {
	var id pgtype.UUID
	if agentID != "" {
		// 形式の誤りは外部キーと CHECK が拒否する。
		_ = id.Scan(agentID)
	}
	return id
}
