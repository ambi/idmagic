package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

// =====================================================================
// AgentRepository (ADR-048)
// =====================================================================

type AgentRepository struct {
	mu       sync.RWMutex
	agents   map[string]*idmdomain.Agent                    // key: sharedmem.TenantKey(tenant_id, id)
	bindings map[string][]*idmdomain.AgentCredentialBinding // key: sharedmem.TenantKey(tenant_id, agent_id)
}

func NewAgentRepository() *AgentRepository {
	return &AgentRepository{
		agents:   map[string]*idmdomain.Agent{},
		bindings: map[string][]*idmdomain.AgentCredentialBinding{},
	}
}

func cloneAgent(agent *idmdomain.Agent) *idmdomain.Agent {
	cloned := *agent
	cloned.Roles = slices.Clone(agent.Roles)
	return &cloned
}

func (r *AgentRepository) ListByTenant(_ context.Context, tenantID string) ([]*idmdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*idmdomain.Agent, 0)
	for _, agent := range r.agents {
		if agent.TenantID == tenantID {
			out = append(out, cloneAgent(agent))
		}
	}
	slices.SortFunc(out, func(a, b *idmdomain.Agent) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *AgentRepository) FindByID(_ context.Context, tenantID, id string) (*idmdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent := r.agents[sharedmem.TenantKey(tenantID, id)]
	if agent == nil {
		return nil, nil
	}
	return cloneAgent(agent), nil
}

func (r *AgentRepository) Save(_ context.Context, agent *idmdomain.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[sharedmem.TenantKey(agent.TenantID, agent.ID)] = cloneAgent(agent)
	return nil
}

func (r *AgentRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, sharedmem.TenantKey(tenantID, id))
	delete(r.bindings, sharedmem.TenantKey(tenantID, id))
	return nil
}

func (r *AgentRepository) ListBindings(_ context.Context, tenantID, agentID string) ([]*idmdomain.AgentCredentialBinding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored := r.bindings[sharedmem.TenantKey(tenantID, agentID)]
	out := make([]*idmdomain.AgentCredentialBinding, 0, len(stored))
	for _, binding := range stored {
		cloned := *binding
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *idmdomain.AgentCredentialBinding) int { return strings.Compare(a.ClientID, b.ClientID) })
	return out, nil
}

func (r *AgentRepository) AddBinding(_ context.Context, binding *idmdomain.AgentCredentialBinding) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var tenantID string
	for _, a := range r.agents {
		if a.ID == binding.AgentID {
			tenantID = a.TenantID
			break
		}
	}
	if tenantID == "" {
		return false, nil
	}
	for key, bindings := range r.bindings {
		agent := r.agents[key]
		if agent == nil || agent.TenantID != tenantID {
			continue
		}
		if slices.ContainsFunc(bindings, func(b *idmdomain.AgentCredentialBinding) bool { return b.ClientID == binding.ClientID }) {
			return false, nil
		}
	}
	key := sharedmem.TenantKey(tenantID, binding.AgentID)
	for _, existing := range r.bindings[key] {
		if existing.ClientID == binding.ClientID {
			return false, nil
		}
	}
	cloned := *binding
	r.bindings[key] = append(r.bindings[key], &cloned)
	return true, nil
}

func (r *AgentRepository) RemoveBinding(_ context.Context, tenantID, agentID, clientID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.TenantKey(tenantID, agentID)
	bindings := r.bindings[key]
	for i, existing := range bindings {
		if existing.ClientID == clientID {
			r.bindings[key] = slices.Delete(bindings, i, i+1)
			return true, nil
		}
	}
	return false, nil
}

func (r *AgentRepository) FindByClientID(_ context.Context, tenantID, clientID string) (*idmdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, bindings := range r.bindings {
		if !slices.ContainsFunc(bindings, func(b *idmdomain.AgentCredentialBinding) bool { return b.ClientID == clientID }) {
			continue
		}
		agent := r.agents[key]
		if agent != nil && agent.TenantID == tenantID {
			return cloneAgent(agent), nil
		}
	}
	return nil, nil
}
