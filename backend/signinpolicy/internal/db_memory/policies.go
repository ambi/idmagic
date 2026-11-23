package db_memory

import (
	"context"
	"slices"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
)

func cloneSignInPolicy(policy *policydomain.AppSignInPolicy) *policydomain.AppSignInPolicy {
	cloned := *policy
	cloned.Rules = slices.Clone(policy.Rules)
	return &cloned
}

// =====================================================================
// SignInPolicyRepository (wi-71)
// =====================================================================

type SignInPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*policydomain.AppSignInPolicy // key: sharedmem.TenantKey(tenant_id, application_id)
}

func NewSignInPolicyRepository() *SignInPolicyRepository {
	return &SignInPolicyRepository{policies: map[string]*policydomain.AppSignInPolicy{}}
}

func (r *SignInPolicyRepository) Get(_ context.Context, tenantID, applicationID string) (*policydomain.AppSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	policy := r.policies[sharedmem.ScopedKey(tenantID, applicationID, kernel.DefaultTenantID)]
	if policy == nil {
		return nil, nil
	}
	return cloneSignInPolicy(policy), nil
}

func (r *SignInPolicyRepository) ListAll(_ context.Context, tenantID string) ([]*policydomain.AppSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*policydomain.AppSignInPolicy, 0)
	for _, policy := range r.policies {
		if policy.TenantID == tenantID {
			out = append(out, cloneSignInPolicy(policy))
		}
	}
	return out, nil
}

func (r *SignInPolicyRepository) Save(_ context.Context, policy *policydomain.AppSignInPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(policy.TenantID, policy.ApplicationID, kernel.DefaultTenantID)
	cloned := cloneSignInPolicy(policy)
	if existing := r.policies[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.policies[key] = cloned
	return nil
}

func (r *SignInPolicyRepository) Delete(_ context.Context, tenantID, applicationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.policies, sharedmem.ScopedKey(tenantID, applicationID, kernel.DefaultTenantID))
	return nil
}

func (r *SignInPolicyRepository) Update(_ context.Context, tenantID, applicationID string, change func(*policydomain.AppSignInPolicy) (*policydomain.AppSignInPolicy, error)) (*policydomain.AppSignInPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(tenantID, applicationID, kernel.DefaultTenantID)
	var current *policydomain.AppSignInPolicy
	if stored := r.policies[key]; stored != nil {
		current = cloneSignInPolicy(stored)
	}
	updated, err := change(current)
	if err != nil {
		return nil, err
	}
	r.policies[key] = cloneSignInPolicy(updated)
	return cloneSignInPolicy(updated), nil
}

// =====================================================================
// DefaultSignInPolicyRepository (wi-115)
// =====================================================================

func cloneDefaultSignInPolicy(policy *policydomain.TenantDefaultSignInPolicy) *policydomain.TenantDefaultSignInPolicy {
	cloned := *policy
	cloned.Rules = slices.Clone(policy.Rules)
	return &cloned
}

type DefaultSignInPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*policydomain.TenantDefaultSignInPolicy // key: tenant_id
}

func NewDefaultSignInPolicyRepository() *DefaultSignInPolicyRepository {
	return &DefaultSignInPolicyRepository{policies: map[string]*policydomain.TenantDefaultSignInPolicy{}}
}

func (r *DefaultSignInPolicyRepository) Get(_ context.Context, tenantID string) (*policydomain.TenantDefaultSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	policy := r.policies[tenantID]
	if policy == nil {
		return nil, nil
	}
	return cloneDefaultSignInPolicy(policy), nil
}

func (r *DefaultSignInPolicyRepository) Save(_ context.Context, policy *policydomain.TenantDefaultSignInPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := cloneDefaultSignInPolicy(policy)
	if existing := r.policies[policy.TenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.policies[policy.TenantID] = cloned
	return nil
}

func (r *DefaultSignInPolicyRepository) Update(_ context.Context, tenantID string, change func(*policydomain.TenantDefaultSignInPolicy) (*policydomain.TenantDefaultSignInPolicy, error)) (*policydomain.TenantDefaultSignInPolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var current *policydomain.TenantDefaultSignInPolicy
	if stored := r.policies[tenantID]; stored != nil {
		current = cloneDefaultSignInPolicy(stored)
	}
	updated, err := change(current)
	if err != nil {
		return nil, err
	}
	r.policies[tenantID] = cloneDefaultSignInPolicy(updated)
	return cloneDefaultSignInPolicy(updated), nil
}
