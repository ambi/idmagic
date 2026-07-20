package memory

import (
	"context"
	"sync"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
)

// =====================================================================
// PasswordResetTokenStore (Authentication)
// =====================================================================

type PasswordResetTokenStore struct {
	mu      sync.Mutex
	records map[string]passwordports.PasswordResetTokenRecord
}

func NewPasswordResetTokenStore() *PasswordResetTokenStore {
	return &PasswordResetTokenStore{records: map[string]passwordports.PasswordResetTokenRecord{}}
}

func (s *PasswordResetTokenStore) Save(
	_ context.Context,
	record passwordports.PasswordResetTokenRecord,
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

func (s *PasswordResetTokenStore) Consume(
	_ context.Context,
	tokenHash string,
	now time.Time,
) (*passwordports.PasswordResetTokenRecord, error) {
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
