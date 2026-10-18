package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// AgentRepository
// =====================================================================

type AgentRepository struct {
	mu       sync.RWMutex
	agents   map[string]*agentdomain.Agent                    // key: sharedmem.TenantKey(tenant_id, id)
	bindings map[string][]*agentdomain.AgentCredentialBinding // key: sharedmem.TenantKey(tenant_id, agent_id)
}

func NewAgentRepository() *AgentRepository {
	return &AgentRepository{
		agents:   map[string]*agentdomain.Agent{},
		bindings: map[string][]*agentdomain.AgentCredentialBinding{},
	}
}

func cloneAgent(agent *agentdomain.Agent) *agentdomain.Agent {
	cloned := *agent
	cloned.Roles = slices.Clone(agent.Roles)
	return &cloned
}

func (r *AgentRepository) ListAll(_ context.Context, tenantID string) ([]*agentdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*agentdomain.Agent, 0)
	for _, agent := range r.agents {
		if agent.TenantID == tenantID {
			out = append(out, cloneAgent(agent))
		}
	}
	slices.SortFunc(out, func(a, b *agentdomain.Agent) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// ListPage implements ports.AgentRepository.ListPage (wi-159): keyset
// pagination ordered by (Name, ID) ascending, strictly after the given keyset.
func (r *AgentRepository) ListPage(_ context.Context, tenantID, afterName, afterID string, limit int) ([]*agentdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*agentdomain.Agent, 0)
	for _, agent := range r.agents {
		if agent.TenantID == tenantID {
			out = append(out, cloneAgent(agent))
		}
	}
	key := func(a *agentdomain.Agent) (string, string) { return a.Name, a.ID }
	return sharedmem.KeysetPage(out, key, false, afterName, afterID, limit), nil
}

func (r *AgentRepository) ListPageBefore(_ context.Context, tenantID, beforeName, beforeID string, limit int) ([]*agentdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*agentdomain.Agent, 0)
	for _, agent := range r.agents {
		if agent.TenantID == tenantID {
			out = append(out, cloneAgent(agent))
		}
	}
	key := func(a *agentdomain.Agent) (string, string) { return a.Name, a.ID }
	return sharedmem.KeysetPageBefore(out, key, false, beforeName, beforeID, limit), nil
}

func (r *AgentRepository) Count(_ context.Context, tenantID string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var count int64
	for _, agent := range r.agents {
		if agent.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (r *AgentRepository) FindByID(_ context.Context, tenantID, id string) (*agentdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent := r.agents[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]
	if agent == nil {
		return nil, nil
	}
	return cloneAgent(agent), nil
}

func (r *AgentRepository) Save(_ context.Context, agent *agentdomain.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[sharedmem.ScopedKey(agent.TenantID, agent.ID, kernel.DefaultTenantID)] = cloneAgent(agent)
	return nil
}

func (r *AgentRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID))
	delete(r.bindings, sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID))
	return nil
}

func (r *AgentRepository) ListBindings(_ context.Context, tenantID, agentID string) ([]*agentdomain.AgentCredentialBinding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stored := r.bindings[sharedmem.ScopedKey(tenantID, agentID, kernel.DefaultTenantID)]
	out := make([]*agentdomain.AgentCredentialBinding, 0, len(stored))
	for _, binding := range stored {
		cloned := *binding
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *agentdomain.AgentCredentialBinding) int { return strings.Compare(a.ClientID, b.ClientID) })
	return out, nil
}

func (r *AgentRepository) AddBinding(_ context.Context, binding *agentdomain.AgentCredentialBinding) (bool, error) {
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
		if slices.ContainsFunc(bindings, func(b *agentdomain.AgentCredentialBinding) bool { return b.ClientID == binding.ClientID }) {
			return false, nil
		}
	}
	key := sharedmem.ScopedKey(tenantID, binding.AgentID, kernel.DefaultTenantID)
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
	key := sharedmem.ScopedKey(tenantID, agentID, kernel.DefaultTenantID)
	bindings := r.bindings[key]
	for i, existing := range bindings {
		if existing.ClientID == clientID {
			r.bindings[key] = slices.Delete(bindings, i, i+1)
			return true, nil
		}
	}
	return false, nil
}

func (r *AgentRepository) FindByClientID(_ context.Context, tenantID, clientID string) (*agentdomain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for key, bindings := range r.bindings {
		if !slices.ContainsFunc(bindings, func(b *agentdomain.AgentCredentialBinding) bool { return b.ClientID == clientID }) {
			continue
		}
		agent := r.agents[key]
		if agent != nil && agent.TenantID == tenantID {
			return cloneAgent(agent), nil
		}
	}
	return nil, nil
}
