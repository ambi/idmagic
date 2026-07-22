package db_memory

import (
	"context"
	"sync"

	"github.com/ambi/idmagic/backend/apitoken/domain"
)

type Repository struct {
	mu     sync.RWMutex
	byID   map[string]*domain.ApiToken
	byHash map[string]string
}

func NewRepository() *Repository {
	return &Repository{byID: map[string]*domain.ApiToken{}, byHash: map[string]string{}}
}

func cloneToken(token *domain.ApiToken) *domain.ApiToken {
	if token == nil {
		return nil
	}
	clone := *token
	clone.Scopes = append(domain.Scopes(nil), token.Scopes...)
	if token.ExpiresAt != nil {
		expiresAt := *token.ExpiresAt
		clone.ExpiresAt = &expiresAt
	}
	return &clone
}

func (r *Repository) Save(_ context.Context, token *domain.ApiToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if previous := r.byID[token.ID]; previous != nil {
		delete(r.byHash, previous.TokenHash)
	}
	r.byID[token.ID] = cloneToken(token)
	r.byHash[token.TokenHash] = token.ID
	return nil
}

func (r *Repository) FindByHash(_ context.Context, tokenHash string) (*domain.ApiToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneToken(r.byID[r.byHash[tokenHash]]), nil
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

func (r *Repository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	token := r.byID[id]
	if token != nil && token.TenantID == tenantID {
		delete(r.byHash, token.TokenHash)
		delete(r.byID, id)
	}
	return nil
}
