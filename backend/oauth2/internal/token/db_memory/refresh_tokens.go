package db_memory

import (
	"context"
	"errors"
	"slices"
	"sort"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// =====================================================================
// RefreshTokenStore (OAuth2, ファミリーローテーション対応)
// =====================================================================

type RefreshTokenStore struct {
	mu     sync.Mutex
	byHash map[string]*domain.RefreshTokenRecord
	byID   map[string]*domain.RefreshTokenRecord
}

func NewRefreshTokenStore() *RefreshTokenStore {
	return &RefreshTokenStore{
		byHash: map[string]*domain.RefreshTokenRecord{},
		byID:   map[string]*domain.RefreshTokenRecord{},
	}
}

func (s *RefreshTokenStore) DeleteForClient(_ context.Context, tenantID, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, token := range s.byHash {
		if token.ClientID == clientID && token.TenantID == tenantID {
			delete(s.byHash, hash)
			delete(s.byID, token.ID)
		}
	}
	return nil
}

func (s *RefreshTokenStore) FindByHash(_ context.Context, hash string) (*domain.RefreshTokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneRefreshToken(s.byHash[hash]), nil
}

func (s *RefreshTokenStore) Save(_ context.Context, rec *domain.RefreshTokenRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sharedmem.DefaultScope(&rec.TenantID, kernel.DefaultTenantID)
	stored := cloneRefreshToken(rec)
	s.byHash[stored.Hash] = stored
	s.byID[stored.ID] = stored
	return nil
}

func (s *RefreshTokenStore) Rotate(_ context.Context, parentID string, newRec *domain.RefreshTokenRecord) (*domain.RefreshTokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	parent, ok := s.byID[parentID]
	if !ok {
		return nil, errors.New("parent refresh token not found")
	}
	if parent.Rotated || parent.Revoked {
		return nil, nil
	}
	parent.Rotated = true
	sharedmem.DefaultScope(&newRec.TenantID, kernel.DefaultTenantID)
	stored := cloneRefreshToken(newRec)
	s.byHash[stored.Hash] = stored
	s.byID[stored.ID] = stored
	return cloneRefreshToken(stored), nil
}

func (s *RefreshTokenStore) RevokeFamily(_ context.Context, familyID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var revokedIDs []string
	for _, rec := range s.byID {
		if rec.FamilyID == familyID && !rec.Revoked {
			rec.Revoked = true
			revokedIDs = append(revokedIDs, rec.ID)
		}
	}
	sort.Strings(revokedIDs)
	return revokedIDs, nil
}

func (s *RefreshTokenStore) RevokeBySid(_ context.Context, sid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.byID {
		if rec.Sid != nil && *rec.Sid == sid {
			rec.Revoked = true
		}
	}
	return nil
}

func (s *RefreshTokenStore) DeleteAllForSub(_ context.Context, sub string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, rec := range s.byID {
		if rec.UserID == sub {
			delete(s.byID, id)
			delete(s.byHash, rec.Hash)
		}
	}
	return nil
}

func cloneRefreshToken(in *domain.RefreshTokenRecord) *domain.RefreshTokenRecord {
	if in == nil {
		return nil
	}
	out := *in
	out.Scopes = append([]string(nil), in.Scopes...)
	out.AMR = slices.Clone(in.AMR)
	if in.SenderConstraint != nil {
		senderConstraint := *in.SenderConstraint
		out.SenderConstraint = &senderConstraint
	}
	return &out
}
