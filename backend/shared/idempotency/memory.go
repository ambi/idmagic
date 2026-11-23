package idempotency

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

type memoryKey struct{ tenant, principal, hash string }

type MemoryStore struct {
	mu      sync.Mutex
	records map[memoryKey]Record
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{records: make(map[memoryKey]Record)} }

func (s *MemoryStore) Claim(_ context.Context, candidate Record, now time.Time) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey{candidate.TenantID, candidate.PrincipalID, candidate.KeyHash}
	if existing, ok := s.records[key]; ok && existing.ExpiresAt.After(now) {
		return clone(existing), false, nil
	}
	s.records[key] = clone(candidate)
	return clone(candidate), true, nil
}

func (s *MemoryStore) Complete(_ context.Context, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey{record.TenantID, record.PrincipalID, record.KeyHash}
	if current, ok := s.records[key]; !ok || current.ID != record.ID || len(current.Ciphertext) != 0 {
		return errors.New("idempotency reservation no longer belongs to this request")
	}
	s.records[key] = clone(record)
	return nil
}

func (s *MemoryStore) PurgeExpired(_ context.Context, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, record := range s.records {
		if !record.ExpiresAt.After(now) {
			delete(s.records, key)
		}
	}
	return nil
}

func (s *MemoryStore) ListPending(_ context.Context, tenantID string, version int32, limit int, excluded []int32) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var rows []Record
	for _, record := range s.records {
		if record.TenantID == tenantID && record.KeyVersion != 0 && record.KeyVersion != version && !slices.Contains(excluded, record.KeyVersion) && len(rows) < limit {
			rows = append(rows, clone(record))
		}
	}
	return rows, nil
}

func (s *MemoryStore) ReplaceCipher(_ context.Context, record Record, oldVersion int32) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := memoryKey{record.TenantID, record.PrincipalID, record.KeyHash}
	if current, ok := s.records[key]; ok && current.ID == record.ID && current.KeyVersion == oldVersion {
		s.records[key] = clone(record)
		return true, nil
	}
	return false, nil
}

func (s *MemoryStore) PendingCount(_ context.Context, tenantID string, version int32) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, record := range s.records {
		if record.TenantID == tenantID && record.KeyVersion != 0 && record.KeyVersion != version {
			count++
		}
	}
	return count, nil
}

func clone(record Record) Record { record.Ciphertext = slices.Clone(record.Ciphertext); return record }
