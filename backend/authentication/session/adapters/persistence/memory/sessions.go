package memory

import (
	"context"
	"sync"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// =====================================================================
// SessionStore (Authentication) — PostgreSQL 実装と同じ contract を持つ in-memory 版
// (wi-253 / ADR-126)。有効性判定は domain.LoginSession.Active に委ね、失効は tombstone
// (Revoke) で物理削除しない。
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
	sharedmem.DefaultTenant(&sess.TenantID)
	s.sessions[sess.ID] = sess
	return nil
}

// Find は有効な (未失効・未期限切れ) セッションだけを返す fail-closed な解決用 lookup。
func (s *SessionStore) Find(_ context.Context, id string) (*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok || !sess.Active(s.now()) {
		return nil, nil
	}
	return sess, nil
}

// FindOwned は失効・期限切れを含めて対象の所有者確認に使う lookup。
func (s *SessionStore) FindOwned(_ context.Context, id, userID string) (*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok || sess.UserID != userID {
		return nil, nil
	}
	return sess, nil
}

func (s *SessionStore) Revoke(_ context.Context, id string, reason spec.SessionEndReason, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		sess.Revoke(reason, now)
	}
	return nil
}

func (s *SessionStore) Touch(_ context.Context, id string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok {
		sess.Touch(now)
	}
	return nil
}

func (s *SessionStore) ListBySub(_ context.Context, sub string) ([]*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	out := []*authdomain.LoginSession{}
	for _, sess := range s.sessions {
		if sess.UserID == sub && !sess.AuthenticationPending && sess.Active(now) {
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

// DeleteExpiredBatch は expires_at が cutoff より前の行を最大 limit 件まで物理削除する
// housekeeping cleanup 用の primitive (wi-253 Plan §7)。
func (s *SessionStore) DeleteExpiredBatch(_ context.Context, cutoff time.Time, limit int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := 0
	for id, sess := range s.sessions {
		if deleted >= limit {
			break
		}
		if sess.ExpiresAt.Before(cutoff) {
			delete(s.sessions, id)
			deleted++
		}
	}
	return deleted, nil
}
