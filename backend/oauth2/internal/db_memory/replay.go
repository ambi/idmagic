package db_memory

import (
	"context"
	"errors"
	"sync"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// =====================================================================
// Replay stores (OAuth2: DPoP / client assertion jti)
// =====================================================================

type replayEntry struct {
	expiresAt time.Time
}

type replayStore struct {
	mu   sync.Mutex
	seen map[string]replayEntry
}

// errReplayTimeRequired は、予約の時刻を渡し忘れた呼び出しを期限内と誤らせずに止める。
var errReplayTimeRequired = errors.New("replay reservation requires the current time")

func newReplayStore() *replayStore { return &replayStore{seen: map[string]replayEntry{}} }

func (s *replayStore) recordIfNew(jti string, windowSeconds int, now time.Time) (bool, error) {
	if now.IsZero() {
		return false, errReplayTimeRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range s.seen {
		if !now.Before(e.expiresAt) {
			delete(s.seen, k)
		}
	}
	if _, ok := s.seen[jti]; ok {
		return false, nil
	}
	s.seen[jti] = replayEntry{expiresAt: now.Add(time.Duration(windowSeconds) * time.Second)}
	return true, nil
}

type DpopReplayStore struct{ inner *replayStore }

func NewDpopReplayStore() *DpopReplayStore { return &DpopReplayStore{inner: newReplayStore()} }

func (s *DpopReplayStore) RecordIfNew(_ context.Context, jti string, windowSeconds int, now time.Time) (bool, error) {
	return s.inner.recordIfNew(jti, windowSeconds, now)
}

type ClientAssertionReplayStore struct{ inner *replayStore }

func NewClientAssertionReplayStore() *ClientAssertionReplayStore {
	return &ClientAssertionReplayStore{inner: newReplayStore()}
}

func (s *ClientAssertionReplayStore) RecordIfNew(_ context.Context, jti string, windowSeconds int, now time.Time) (bool, error) {
	return s.inner.recordIfNew(jti, windowSeconds, now)
}

// TokenExchangeGrantStore は単一プロセスの交換の許可である。
type TokenExchangeGrantStore struct {
	mu     sync.Mutex
	grants map[string]tokenExchangeGrantEntry
}

type tokenExchangeGrantEntry struct {
	tenantID string
	grant    oauthdomain.TokenExchangeGrant
}

func NewTokenExchangeGrantStore() *TokenExchangeGrantStore {
	return &TokenExchangeGrantStore{grants: map[string]tokenExchangeGrantEntry{}}
}

func (s *TokenExchangeGrantStore) Save(ctx context.Context, grantHash string, grant oauthdomain.TokenExchangeGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grants[grantHash] = tokenExchangeGrantEntry{tenantID: tenantports.TenantID(ctx), grant: grant}
	return nil
}

func (s *TokenExchangeGrantStore) Claim(ctx context.Context, grantHash string, grant oauthdomain.TokenExchangeGrant, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.grants[grantHash]
	if !ok || entry.tenantID != tenantports.TenantID(ctx) || !entry.grant.ExpiresAt.After(now) ||
		entry.grant.ClientID != grant.ClientID || entry.grant.SubjectTokenID != grant.SubjectTokenID || entry.grant.Resource != grant.Resource {
		return false, nil
	}
	delete(s.grants, grantHash)
	return true, nil
}
