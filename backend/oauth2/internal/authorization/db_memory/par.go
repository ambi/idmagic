package db_memory

import (
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// PARStore (OAuth2)
// =====================================================================

type PARStore struct {
	mu      sync.Mutex
	records map[string]*domain.PARRecord
}

func NewPARStore() *PARStore {
	return &PARStore{records: map[string]*domain.PARRecord{}}
}

func (s *PARStore) Save(_ context.Context, rec *domain.PARRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sharedmem.DefaultTenant(&rec.TenantID)
	s.records[rec.RequestURI] = rec
	return nil
}

func (s *PARStore) Find(_ context.Context, requestURI string) (*domain.PARRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.records[requestURI], nil
}

func (s *PARStore) Consume(_ context.Context, requestURI string) (*domain.PARRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[requestURI]
	if !ok || rec.Used || time.Now().After(rec.ExpiresAt) {
		return nil, nil
	}
	rec.Used = true
	return rec, nil
}
