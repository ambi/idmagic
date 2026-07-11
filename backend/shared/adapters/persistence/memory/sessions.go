package memory

import (
	"context"
	"sync"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
)

// =====================================================================
// SessionStore (Authentication)
// =====================================================================

type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]*authdomain.LoginSession
	// Clock は期限切れ判定に使う時計。nil なら time.Now。決定的な時刻でセッション失効を
	// 制御したいテストが差し替える (本番は実時計のまま)。
	Clock func() time.Time
}

func NewSessionStore() *SessionStore {
	return &SessionStore{sessions: map[string]*authdomain.LoginSession{}}
}

func (s *SessionStore) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now()
}

func (s *SessionStore) Save(_ context.Context, sess *authdomain.LoginSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	DefaultTenant(&sess.TenantID)
	s.sessions[sess.ID] = sess
	return nil
}

func (s *SessionStore) Find(_ context.Context, id string) (*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, nil
	}
	if s.now().After(sess.ExpiresAt) {
		delete(s.sessions, id)
		return nil, nil
	}
	return sess, nil
}

func (s *SessionStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *SessionStore) ListBySub(_ context.Context, sub string) ([]*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	out := []*authdomain.LoginSession{}
	for id, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, id)
			continue
		}
		if sess.UserID == sub && !sess.AuthenticationPending {
			out = append(out, sess)
		}
	}
	return out, nil
}

func (s *SessionStore) DeleteAllForSub(_ context.Context, sub string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if sess.UserID == sub {
			delete(s.sessions, id)
		}
	}
	return nil
}
