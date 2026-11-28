package db_memory

import (
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

// =====================================================================
// PARStore (OAuth2)
// =====================================================================

type PARStore struct {
	mu      sync.Mutex
	records map[string]*oauthdomain.PARRecord
}

func NewPARStore() *PARStore {
	return &PARStore{records: map[string]*oauthdomain.PARRecord{}}
}

func (s *PARStore) Save(_ context.Context, rec *oauthdomain.PARRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sharedmem.DefaultScope(&rec.TenantID, kernel.DefaultTenantID)
	s.records[rec.RequestURI] = rec
	return nil
}

func (s *PARStore) Find(_ context.Context, requestURI string) (*oauthdomain.PARRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.records[requestURI], nil
}

func (s *PARStore) Consume(_ context.Context, requestURI string, now time.Time) (*oauthdomain.PARRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[requestURI]
	if !ok || rec.Used || !now.Before(rec.ExpiresAt) {
		return nil, nil
	}
	rec.Used = true
	return rec, nil
}
