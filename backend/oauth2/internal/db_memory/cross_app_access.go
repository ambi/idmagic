package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
)

// CrossAppAccessPolicyRepository は ID-JAG の許可をメモリに保つ。
type CrossAppAccessPolicyRepository struct {
	mu     sync.RWMutex
	issue  []*oauthdomain.CrossAppAccessIssuePolicy
	redeem []*oauthdomain.CrossAppAccessRedeemPolicy
}

func NewCrossAppAccessPolicyRepository() *CrossAppAccessPolicyRepository {
	return &CrossAppAccessPolicyRepository{}
}

func (r *CrossAppAccessPolicyRepository) ListIssuePolicies(_ context.Context, tenantID string) ([]*oauthdomain.CrossAppAccessIssuePolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*oauthdomain.CrossAppAccessIssuePolicy{}
	for _, policy := range r.issue {
		if policy.TenantID == tenantID {
			cloned := *policy
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *oauthdomain.CrossAppAccessIssuePolicy) int {
		return strings.Compare(a.ClientID+"\x00"+a.Audience, b.ClientID+"\x00"+b.Audience)
	})
	return out, nil
}

func (r *CrossAppAccessPolicyRepository) FindIssuePolicy(_ context.Context, tenantID, id string) (*oauthdomain.CrossAppAccessIssuePolicy, error) {
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

func (r *CrossAppAccessPolicyRepository) CreateIssuePolicy(_ context.Context, policy *oauthdomain.CrossAppAccessIssuePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.issue {
		if existing.TenantID == policy.TenantID && existing.ClientID == policy.ClientID && existing.Audience == policy.Audience {
			return oauthports.ErrCrossAppAccessPolicyExists
		}
	}
	cloned := *policy
	r.issue = append(r.issue, &cloned)
	return nil
}

func (r *CrossAppAccessPolicyRepository) DeleteIssuePolicy(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issue = slices.DeleteFunc(r.issue, func(policy *oauthdomain.CrossAppAccessIssuePolicy) bool {
		return policy.TenantID == tenantID && policy.ID == id
	})
	return nil
}

func (r *CrossAppAccessPolicyRepository) ListRedeemPolicies(_ context.Context, tenantID string) ([]*oauthdomain.CrossAppAccessRedeemPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*oauthdomain.CrossAppAccessRedeemPolicy{}
	for _, policy := range r.redeem {
		if policy.TenantID == tenantID {
			cloned := *policy
			out = append(out, &cloned)
		}
	}
	slices.SortFunc(out, func(a, b *oauthdomain.CrossAppAccessRedeemPolicy) int {
		return strings.Compare(a.ProviderID+"\x00"+a.ClientID, b.ProviderID+"\x00"+b.ClientID)
	})
	return out, nil
}

func (r *CrossAppAccessPolicyRepository) FindRedeemPolicy(_ context.Context, tenantID, id string) (*oauthdomain.CrossAppAccessRedeemPolicy, error) {
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

func (r *CrossAppAccessPolicyRepository) FindRedeemPolicyFor(_ context.Context, tenantID, providerID, clientID string) (*oauthdomain.CrossAppAccessRedeemPolicy, error) {
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

func (r *CrossAppAccessPolicyRepository) CreateRedeemPolicy(_ context.Context, policy *oauthdomain.CrossAppAccessRedeemPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.redeem {
		if existing.TenantID == policy.TenantID && existing.ProviderID == policy.ProviderID && existing.ClientID == policy.ClientID {
			return oauthports.ErrCrossAppAccessPolicyExists
		}
	}
	cloned := *policy
	r.redeem = append(r.redeem, &cloned)
	return nil
}

func (r *CrossAppAccessPolicyRepository) DeleteRedeemPolicy(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.redeem = slices.DeleteFunc(r.redeem, func(policy *oauthdomain.CrossAppAccessRedeemPolicy) bool {
		return policy.TenantID == tenantID && policy.ID == id
	})
	return nil
}
