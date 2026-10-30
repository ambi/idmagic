package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// ConsentRepository (OAuth2)

type memConsent struct {
	domain.Consent
	TenantID string
}

type ConsentRepository struct {
	mu       sync.RWMutex
	consents map[string]*memConsent
	// delegations は Agent への代行の付与である。クライアントの同意の操作に現れない。
	delegations map[string]*memConsent
}

func NewConsentRepository() *ConsentRepository {
	return &ConsentRepository{consents: map[string]*memConsent{}, delegations: map[string]*memConsent{}}
}

func (r *ConsentRepository) Find(_ context.Context, tenantID, sub, clientID string) (*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mc := r.consents[consentKey(tenantID, sub, clientID)]
	if mc == nil {
		return nil, nil
	}
	cloned := mc.Consent
	cloned.Scopes = slices.Clone(mc.Scopes)
	return &cloned, nil
}

func (r *ConsentRepository) FindAll(_ context.Context, tenantID string) ([]*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Consent, 0)
	for _, mc := range r.consents {
		if mc.TenantID != tenantID {
			continue
		}
		cloned := mc.Consent
		cloned.Scopes = slices.Clone(mc.Scopes)
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *domain.Consent) int {
		if a.UserID != b.UserID {
			return strings.Compare(a.UserID, b.UserID)
		}
		return strings.Compare(a.ClientID, b.ClientID)
	})
	return out, nil
}

// ListPage implements ports.ConsentRepository.ListPage
// (wi-159): keyset pagination ordered by (UserID, ClientID)
// ascending, strictly after the given keyset.
func (r *ConsentRepository) ListPage(_ context.Context, tenantID, afterUserID, afterClientID string, limit int) ([]*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Consent, 0)
	for _, mc := range r.consents {
		if mc.TenantID != tenantID {
			continue
		}
		cloned := mc.Consent
		cloned.Scopes = slices.Clone(mc.Scopes)
		out = append(out, &cloned)
	}
	key := func(c *domain.Consent) (string, string) { return c.UserID, c.ClientID }
	return sharedmem.KeysetPage(out, key, false, afterUserID, afterClientID, limit), nil
}

func (r *ConsentRepository) ListPageBefore(_ context.Context, tenantID, beforeUserID, beforeClientID string, limit int) ([]*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Consent, 0)
	for _, mc := range r.consents {
		if mc.TenantID != tenantID {
			continue
		}
		cloned := mc.Consent
		cloned.Scopes = slices.Clone(mc.Scopes)
		out = append(out, &cloned)
	}
	key := func(c *domain.Consent) (string, string) { return c.UserID, c.ClientID }
	return sharedmem.KeysetPageBefore(out, key, false, beforeUserID, beforeClientID, limit), nil
}

func (r *ConsentRepository) Save(_ context.Context, tenantID string, c *domain.Consent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *c
	cloned.Scopes = slices.Clone(c.Scopes)
	r.consents[consentKey(tenantID, cloned.UserID, cloned.ClientID)] = &memConsent{
		Consent:  cloned,
		TenantID: tenantID,
	}
	return nil
}

func (r *ConsentRepository) Revoke(_ context.Context, tenantID, sub, clientID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	mc, ok := r.consents[consentKey(tenantID, sub, clientID)]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	mc.State = domain.ConsentRevoked
	mc.RevokedAt = &now
	return nil
}

func (r *ConsentRepository) DeleteAllForSub(_ context.Context, sub string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, consent := range r.consents {
		if consent.UserID == sub {
			delete(r.consents, key)
		}
	}
	for key, grant := range r.delegations {
		if grant.UserID == sub {
			delete(r.delegations, key)
		}
	}
	return nil
}

func consentKey(tenantID, sub, clientID string) string {
	return sharedmem.ScopedKey(tenantID, sub+"|"+clientID, kernel.DefaultTenantID)
}

func (r *ConsentRepository) FindAgentDelegation(_ context.Context, tenantID, sub, agentID string) (*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mc := r.delegations[consentKey(tenantID, sub, agentID)]
	if mc == nil {
		return nil, nil
	}
	return cloneConsent(mc), nil
}

func (r *ConsentRepository) ListAgentDelegations(_ context.Context, tenantID, sub string) ([]*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Consent, 0)
	for _, mc := range r.delegations {
		if mc.TenantID == tenantID && mc.UserID == sub {
			out = append(out, cloneConsent(mc))
		}
	}
	slices.SortFunc(out, func(a, b *domain.Consent) int { return strings.Compare(a.AgentID, b.AgentID) })
	return out, nil
}

func (r *ConsentRepository) tenantDelegations(tenantID string) []*domain.Consent {
	out := make([]*domain.Consent, 0)
	for _, mc := range r.delegations {
		if mc.TenantID == tenantID {
			out = append(out, cloneConsent(mc))
		}
	}
	return out
}

func (r *ConsentRepository) ListAgentDelegationPage(_ context.Context, tenantID, afterUserID, afterAgentID string, limit int) ([]*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := func(c *domain.Consent) (string, string) { return c.UserID, c.AgentID }
	return sharedmem.KeysetPage(r.tenantDelegations(tenantID), key, false, afterUserID, afterAgentID, limit), nil
}

func (r *ConsentRepository) ListAgentDelegationPageBefore(_ context.Context, tenantID, beforeUserID, beforeAgentID string, limit int) ([]*domain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := func(c *domain.Consent) (string, string) { return c.UserID, c.AgentID }
	return sharedmem.KeysetPageBefore(r.tenantDelegations(tenantID), key, false, beforeUserID, beforeAgentID, limit), nil
}

func (r *ConsentRepository) SaveAgentDelegation(_ context.Context, tenantID string, c *domain.Consent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := consentKey(tenantID, c.UserID, c.AgentID)
	saved := &memConsent{Consent: *c, TenantID: tenantID}
	saved.Scopes = slices.Clone(c.Scopes)
	saved.State = domain.ConsentGranted
	saved.RevokedAt = nil
	saved.LastUsedAt = nil
	if existing := r.delegations[key]; existing != nil {
		saved.LastUsedAt = existing.LastUsedAt
	}
	r.delegations[key] = saved
	return nil
}

func (r *ConsentRepository) RevokeAgentDelegation(_ context.Context, tenantID, sub, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	mc, ok := r.delegations[consentKey(tenantID, sub, agentID)]
	if !ok || mc.RevokedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	mc.State = domain.ConsentRevoked
	mc.RevokedAt = &now
	return nil
}

func (r *ConsentRepository) TouchAgentDelegation(_ context.Context, tenantID, sub, agentID string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if mc, ok := r.delegations[consentKey(tenantID, sub, agentID)]; ok {
		mc.LastUsedAt = &at
	}
	return nil
}

func cloneConsent(mc *memConsent) *domain.Consent {
	cloned := mc.Consent
	cloned.Scopes = slices.Clone(mc.Scopes)
	return &cloned
}
