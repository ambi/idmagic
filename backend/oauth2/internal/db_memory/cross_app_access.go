package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// CrossAppAccessPolicyRepository は ID-JAG の許可をメモリに保つ。
type CrossAppAccessPolicyRepository struct {
	mu     sync.RWMutex
	issue  []*domain.CrossAppAccessIssuePolicy
	redeem []*domain.CrossAppAccessRedeemPolicy
}

func NewCrossAppAccessPolicyRepository() *CrossAppAccessPolicyRepository {
	return &CrossAppAccessPolicyRepository{}
}

func (r *CrossAppAccessPolicyRepository) ListIssuePolicies(_ context.Context, tenantID string) ([]*domain.CrossAppAccessIssuePolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.CrossAppAccessIssuePolicy{}
	for _, policy := range r.issue {
		if policy.TenantID == tenantID {
			cloned := *policy
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *domain.CrossAppAccessIssuePolicy) int {
		return strings.Compare(a.ClientID+"\x00"+a.Audience, b.ClientID+"\x00"+b.Audience)
	})
	return out, nil
}

func (r *CrossAppAccessPolicyRepository) FindIssuePolicy(_ context.Context, tenantID, id string) (*domain.CrossAppAccessIssuePolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, policy := range r.issue {
		if policy.TenantID == tenantID && policy.ID == id {
			cloned := *policy
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *CrossAppAccessPolicyRepository) CreateIssuePolicy(_ context.Context, policy *domain.CrossAppAccessIssuePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.issue {
		if existing.TenantID == policy.TenantID && existing.ClientID == policy.ClientID && existing.Audience == policy.Audience {
			return ports.ErrCrossAppAccessPolicyExists
		}
	}
	cloned := *policy
	r.issue = append(r.issue, &cloned)
	return nil
}

func (r *CrossAppAccessPolicyRepository) DeleteIssuePolicy(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issue = slices.DeleteFunc(r.issue, func(policy *domain.CrossAppAccessIssuePolicy) bool {
		return policy.TenantID == tenantID && policy.ID == id
	})
	return nil
}

func (r *CrossAppAccessPolicyRepository) ListRedeemPolicies(_ context.Context, tenantID string) ([]*domain.CrossAppAccessRedeemPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.CrossAppAccessRedeemPolicy{}
	for _, policy := range r.redeem {
		if policy.TenantID == tenantID {
			cloned := *policy
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *domain.CrossAppAccessRedeemPolicy) int {
		return strings.Compare(a.ProviderID+"\x00"+a.ClientID, b.ProviderID+"\x00"+b.ClientID)
	})
	return out, nil
}

func (r *CrossAppAccessPolicyRepository) FindRedeemPolicy(_ context.Context, tenantID, id string) (*domain.CrossAppAccessRedeemPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, policy := range r.redeem {
		if policy.TenantID == tenantID && policy.ID == id {
			cloned := *policy
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *CrossAppAccessPolicyRepository) FindRedeemPolicyFor(_ context.Context, tenantID, providerID, clientID string) (*domain.CrossAppAccessRedeemPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, policy := range r.redeem {
		if policy.TenantID == tenantID && policy.ProviderID == providerID && policy.ClientID == clientID {
			cloned := *policy
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *CrossAppAccessPolicyRepository) CreateRedeemPolicy(_ context.Context, policy *domain.CrossAppAccessRedeemPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.redeem {
		if existing.TenantID == policy.TenantID && existing.ProviderID == policy.ProviderID && existing.ClientID == policy.ClientID {
			return ports.ErrCrossAppAccessPolicyExists
		}
	}
	cloned := *policy
	r.redeem = append(r.redeem, &cloned)
	return nil
}

func (r *CrossAppAccessPolicyRepository) DeleteRedeemPolicy(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.redeem = slices.DeleteFunc(r.redeem, func(policy *domain.CrossAppAccessRedeemPolicy) bool {
		return policy.TenantID == tenantID && policy.ID == id
	})
	return nil
}
