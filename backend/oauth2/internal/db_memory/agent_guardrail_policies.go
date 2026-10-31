package db_memory

import (
	"context"
	"slices"
	"sync"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// AgentGuardrailPolicyRepository はガードレールのメモリ実装である。
type AgentGuardrailPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]domain.AgentGuardrailPolicy // key: guardrailKey の結果
}

func NewAgentGuardrailPolicyRepository() *AgentGuardrailPolicyRepository {
	return &AgentGuardrailPolicyRepository{policies: map[string]domain.AgentGuardrailPolicy{}}
}

func (r *AgentGuardrailPolicyRepository) ListEffective(_ context.Context, tenantID string, agentIDs []string) ([]*domain.AgentGuardrailPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*domain.AgentGuardrailPolicy
	for _, scope := range append([]string{""}, slices.Sorted(slices.Values(agentIDs))...) {
		if policy, ok := r.policies[guardrailKey(tenantID, scope)]; ok {
			out = append(out, &policy)
		}
	}
	return out, nil
}

func (r *AgentGuardrailPolicyRepository) Find(_ context.Context, tenantID, agentID string) (*domain.AgentGuardrailPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	policy, ok := r.policies[guardrailKey(tenantID, agentID)]
	if !ok {
		return nil, nil
	}
	return &policy, nil
}

func (r *AgentGuardrailPolicyRepository) Save(_ context.Context, policy *domain.AgentGuardrailPolicy, expectedVersion string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := guardrailKey(policy.TenantID, policy.AgentID)
	current := ""
	if existing, ok := r.policies[key]; ok {
		current = existing.Version()
	}
	if current != expectedVersion {
		return ports.ErrAgentGuardrailConflict
	}
	r.policies[key] = *policy
	return nil
}

func (r *AgentGuardrailPolicyRepository) Delete(_ context.Context, tenantID, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.policies, guardrailKey(tenantID, agentID))
	return nil
}

func guardrailKey(tenantID, agentID string) string { return tenantID + "\x00" + agentID }
