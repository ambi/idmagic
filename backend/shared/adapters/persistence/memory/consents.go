package memory

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

// =====================================================================
// ConsentRepository (OAuth2)
// =====================================================================

type memConsent struct {
	oauthdomain.Consent
	TenantID string
}

type ConsentRepository struct {
	mu       sync.RWMutex
	consents map[string]*memConsent
}

func NewConsentRepository() *ConsentRepository {
	return &ConsentRepository{consents: map[string]*memConsent{}}
}

func (r *ConsentRepository) Find(_ context.Context, tenantID, sub, clientID string) (*oauthdomain.Consent, error) {
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

func (r *ConsentRepository) FindAll(_ context.Context, tenantID string) ([]*oauthdomain.Consent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*oauthdomain.Consent, 0)
	for _, mc := range r.consents {
		if mc.TenantID != tenantID {
			continue
		}
		cloned := mc.Consent
		cloned.Scopes = slices.Clone(mc.Scopes)
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *oauthdomain.Consent) int {
		if a.UserID != b.UserID {
			return strings.Compare(a.UserID, b.UserID)
		}
		return strings.Compare(a.ClientID, b.ClientID)
	})
	return out, nil
}

func (r *ConsentRepository) Save(_ context.Context, tenantID string, c *oauthdomain.Consent) error {
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
	mc.State = oauthdomain.ConsentRevoked
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
	return nil
}

func consentKey(tenantID, sub, clientID string) string {
	return TenantKey(tenantID, sub+"|"+clientID)
}
