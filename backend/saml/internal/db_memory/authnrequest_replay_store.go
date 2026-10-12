package db_memory

import (
	"context"
	"sync"
	"time"
)

type authnRequestReplayStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func NewAuthnRequestReplayStore() *authnRequestReplayStore {
	return &authnRequestReplayStore{seen: make(map[string]time.Time)}
}

func (s *authnRequestReplayStore) RecordIfNew(_ context.Context, tenantID, entityID, requestID string, ttl time.Duration, now time.Time) (bool, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	key := tenantID + "\x00" + entityID + "\x00" + requestID
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, expiresAt := range s.seen {
		if !now.Before(expiresAt) {
			delete(s.seen, key)
		}
	}
	if _, exists := s.seen[key]; exists {
		return false, nil
	}
	s.seen[key] = now.Add(ttl)
	return true, nil
}
