package db_memory

import (
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/apitoken/domain"
)

type Repository struct {
	mu    sync.RWMutex
	byID  map[string]*domain.ApiToken
	byJTI map[string]string
}

func NewRepository() *Repository {
	return &Repository{byID: map[string]*domain.ApiToken{}, byJTI: map[string]string{}}
}

func cloneToken(token *domain.ApiToken) *domain.ApiToken {
	if token == nil {
		return nil
	}
	clone := *token
	clone.Scopes = append(domain.Scopes(nil), token.Scopes...)
	if token.ExpiresAt != nil {
		value := *token.ExpiresAt
		clone.ExpiresAt = &value
	}
	if token.RevokedAt != nil {
		value := *token.RevokedAt
		clone.RevokedAt = &value
	}
	return &clone
}

func (r *Repository) Save(_ context.Context, token *domain.ApiToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if previous := r.byID[token.ID]; previous != nil {
		delete(r.byJTI, previous.JTI)
	}
	r.byID[token.ID] = cloneToken(token)
	r.byJTI[token.JTI] = token.ID
	return nil
}

func (r *Repository) FindByJTI(_ context.Context, tenantID, jti string) (*domain.ApiToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	token := r.byID[r.byJTI[jti]]
	if token == nil || token.TenantID != tenantID {
		return nil, nil
	}
	return cloneToken(token), nil
}

func (r *Repository) List(_ context.Context, tenantID string) ([]*domain.ApiToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*domain.ApiToken, 0)
	for _, token := range r.byID {
		if token.TenantID == tenantID {
			result = append(result, cloneToken(token))
		}
	}
	return result, nil
}

func (r *Repository) Revoke(_ context.Context, tenantID, id string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if token := r.byID[id]; token != nil && token.TenantID == tenantID && token.RevokedAt == nil {
		token.RevokedAt = &at
	}
	return nil
}

func (r *Repository) RevokeByJTI(_ context.Context, tenantID, jti string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if token := r.byID[r.byJTI[jti]]; token != nil && token.TenantID == tenantID && token.RevokedAt == nil {
		token.RevokedAt = &at
	}
	return nil
}
