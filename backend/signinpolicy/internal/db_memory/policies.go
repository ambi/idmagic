package db_memory

import (
	"context"
	"slices"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	"github.com/ambi/idmagic/backend/signinpolicy/domain"
)

func cloneSignInPolicy(policy *domain.AppSignInPolicy) *domain.AppSignInPolicy {
	cloned := *policy
	cloned.Rules = slices.Clone(policy.Rules)
	return &cloned
}

// =====================================================================
// SignInPolicyRepository (wi-71)
// =====================================================================

type SignInPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*domain.AppSignInPolicy // key: sharedmem.TenantKey(tenant_id, application_id)
}

func NewSignInPolicyRepository() *SignInPolicyRepository {
	return &SignInPolicyRepository{policies: map[string]*domain.AppSignInPolicy{}}
}

func (r *SignInPolicyRepository) Get(_ context.Context, tenantID, applicationID string) (*domain.AppSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	policy := r.policies[sharedmem.ScopedKey(tenantID, applicationID, kernel.DefaultTenantID)]
	if policy == nil {
		return nil, nil
	}
	return cloneSignInPolicy(policy), nil
}

func (r *SignInPolicyRepository) ListAll(_ context.Context, tenantID string) ([]*domain.AppSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.AppSignInPolicy, 0)
	for _, policy := range r.policies {
		if policy.TenantID == tenantID {
			out = append(out, cloneSignInPolicy(policy))
		}
	}
	return out, nil
}

func (r *SignInPolicyRepository) Save(_ context.Context, policy *domain.AppSignInPolicy) error {
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

// =====================================================================
// DefaultSignInPolicyRepository (wi-115)
// =====================================================================

func cloneDefaultSignInPolicy(policy *domain.TenantDefaultSignInPolicy) *domain.TenantDefaultSignInPolicy {
	cloned := *policy
	cloned.Rules = slices.Clone(policy.Rules)
	return &cloned
}

type DefaultSignInPolicyRepository struct {
	mu       sync.RWMutex
	policies map[string]*domain.TenantDefaultSignInPolicy // key: tenant_id
}

func NewDefaultSignInPolicyRepository() *DefaultSignInPolicyRepository {
	return &DefaultSignInPolicyRepository{policies: map[string]*domain.TenantDefaultSignInPolicy{}}
}

func (r *DefaultSignInPolicyRepository) Get(_ context.Context, tenantID string) (*domain.TenantDefaultSignInPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	policy := r.policies[tenantID]
	if policy == nil {
		return nil, nil
	}
	return cloneDefaultSignInPolicy(policy), nil
}

func (r *DefaultSignInPolicyRepository) Save(_ context.Context, policy *domain.TenantDefaultSignInPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := cloneDefaultSignInPolicy(policy)
	if existing := r.policies[policy.TenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.policies[policy.TenantID] = cloned
	return nil
}
