package db_memory

import (
	"context"
	"sync"
	"time"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// =====================================================================
// EmailChangeTokenStore (IdManagement/User)
// =====================================================================

type EmailChangeTokenStore struct {
	mu      sync.Mutex
	records map[string]userports.EmailChangeTokenRecord
}

func NewEmailChangeTokenStore() *EmailChangeTokenStore {
	return &EmailChangeTokenStore{records: map[string]userports.EmailChangeTokenRecord{}}
}

func (s *EmailChangeTokenStore) Save(
	_ context.Context,
	record userports.EmailChangeTokenRecord,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, existing := range s.records {
		if existing.Sub == record.Sub {
			delete(s.records, hash)
		}
	}
	s.records[record.TokenHash] = record
	return nil
}

func (s *EmailChangeTokenStore) Consume(
	_ context.Context,
	tokenHash string,
	now time.Time,
) (*userports.EmailChangeTokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[tokenHash]
	if !ok {
		return nil, nil
	}
	delete(s.records, tokenHash)
	if !now.Before(record.ExpiresAt) {
		return nil, nil
	}
	return &record, nil
}
