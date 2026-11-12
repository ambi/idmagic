package db_memory

import (
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

type PasswordChallengeStore struct {
	mu    sync.Mutex
	items map[key]domain.PasswordChallenge
}

func NewPasswordChallengeStore() *PasswordChallengeStore {
	return &PasswordChallengeStore{items: map[key]domain.PasswordChallenge{}}
}

func (s *PasswordChallengeStore) Enqueue(_ context.Context, item domain.PasswordChallenge, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, v := range s.items {
		if v.TenantID == item.TenantID && v.ConnectorID == item.ConnectorID && now.Before(v.ExpiresAt) && v.Result == "" {
			count++
		}
	}
	if count >= domain.MaxPasswordChallenges {
		return ports.ErrDelegationUnavailable
	}
	if _, exists := s.items[key{item.TenantID, item.ID}]; exists {
		return ports.ErrDelegationUnavailable
	}
	s.items[key{item.TenantID, item.ID}] = item
	return nil
}

func (s *PasswordChallengeStore) Claim(_ context.Context, tenantID, connectorID string, revision int64, now time.Time) (*domain.PasswordChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, item := range s.items {
		if item.TenantID != tenantID || item.ConnectorID != connectorID || item.Revision != revision || !now.Before(item.ExpiresAt) || item.Delivered {
			continue
		}
		stored := item
		stored.Delivered = true
		stored.EncryptedRequest = ""
		s.items[k] = stored
		item.Delivered = true
		return &item, nil
	}
	return nil, nil
}

func (s *PasswordChallengeStore) Complete(_ context.Context, tenantID, connectorID, id string, revision int64, result domain.PasswordResult, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key{tenantID, id}
	item, ok := s.items[k]
	if !ok || item.ConnectorID != connectorID || item.Revision != revision || !item.Delivered || !now.Before(item.ExpiresAt) || !result.Valid() || (item.Result != "" && item.Result != result) {
		return ports.ErrDelegationUnavailable
	}
	item.Result = result
	s.items[k] = item
	return nil
}

func (s *PasswordChallengeStore) Find(_ context.Context, tenantID, id string) (*domain.PasswordChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key{tenantID, id}]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (s *PasswordChallengeStore) Delete(_ context.Context, tenantID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key{tenantID, id})
	return nil
}

func (s *PasswordChallengeStore) Sweep(_ context.Context, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, item := range s.items {
		if !now.Before(item.ExpiresAt) {
			delete(s.items, k)
		}
	}
	return nil
}
