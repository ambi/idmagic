package db_memory

import (
	"context"
	"sync"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	"github.com/ambi/idmagic/backend/tenancy"
)

// =====================================================================
// SessionStore (Authentication) — PostgreSQL 実装と同じ contract を持つ in-memory 版
// (wi-253)。有効性判定は domain.LoginSession.Active に委ね、失効は tombstone
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

// inTenant はリクエストのテナントに属する行だけを対象とする。PostgreSQL 実装は
// すべての lookup と更新を tenant_id で絞るので、この判定を持たない in-memory 版は
// 同じ contract を名乗れない。判定の両側とも空のテナントは既定テナントへ寄せるため、
// テナントを立てない呼び出し元から見た振る舞いは変わらない。
func inTenant(ctx context.Context, sess *authdomain.LoginSession) bool {
	tenantID := sess.TenantID
	sharedmem.DefaultTenant(&tenantID)
	return tenantID == tenancy.TenantID(ctx)
}

// Find は有効な (未失効・未期限切れ) セッションだけを返す fail-closed な解決用 lookup。
func (s *SessionStore) Find(ctx context.Context, id string) (*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok || !inTenant(ctx, sess) || !sess.Active(s.now()) {
		return nil, nil
	}
	return sess, nil
}

// FindOwned は失効・期限切れを含めて対象の所有者確認に使う lookup。
func (s *SessionStore) FindOwned(ctx context.Context, id, userID string) (*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok || !inTenant(ctx, sess) || sess.UserID != userID {
		return nil, nil
	}
	return sess, nil
}

func (s *SessionStore) Revoke(ctx context.Context, id string, reason spec.SessionEndReason, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok && inTenant(ctx, sess) {
		sess.Revoke(reason, now)
	}
	return nil
}

func (s *SessionStore) Touch(ctx context.Context, id string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[id]; ok && inTenant(ctx, sess) {
		sess.Touch(now)
	}
	return nil
}

func (s *SessionStore) ListBySub(ctx context.Context, sub string) ([]*authdomain.LoginSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	out := []*authdomain.LoginSession{}
	for _, sess := range s.sessions {
		if inTenant(ctx, sess) && sess.UserID == sub && !sess.AuthenticationPending && sess.Active(now) {
			out = append(out, sess)
		}
	}
	return out, nil
}

func (s *SessionStore) DeleteAllForSub(ctx context.Context, sub string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.sessions {
		if inTenant(ctx, sess) && sess.UserID == sub {
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
