package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// SessionStore persists domain.LoginSession. PostgreSQL is the single source
// of truth (wi-253 / ADR-126): Find and ListBySub only return active
// sessions so authentication resolution stays fail-closed, and Revoke
// tombstones a session instead of deleting it.
type SessionStore interface {
	Save(ctx context.Context, s *domain.LoginSession) error

	// Find resolves an active (unrevoked, unexpired) session for
	// authentication. A nil result must be treated as unauthenticated.
	Find(ctx context.Context, sessionID string) (*domain.LoginSession, error)

	// FindOwned resolves a session regardless of revoked/expired state,
	// scoped to its owner (userID). Returns nil if no such session exists
	// for that owner. Used by self-service revoke to distinguish "no such
	// session" from "already revoked" without leaking other users' sessions.
	FindOwned(ctx context.Context, sessionID, userID string) (*domain.LoginSession, error)

	// Revoke idempotently tombstones a session: the first call records
	// revoked_at / revoke_reason, later calls (with any reason) are a no-op.
	Revoke(ctx context.Context, sessionID string, reason spec.SessionEndReason, now time.Time) error

	// Touch coarsely records that a session was used for an authentication
	// decision, at domain.LoginSessionTouchInterval granularity.
	Touch(ctx context.Context, sessionID string, now time.Time) error

	// ListBySub は対象 sub の有効な (未失効・未期限切れ) LoginSession を新しい順で返す
	// (wi-20 スライス 2)。self / admin のセッション一覧に使う。
	ListBySub(ctx context.Context, sub string) ([]*domain.LoginSession, error)

	// DeleteAllForSub は ADR-036 の anonymize cascade から呼ばれる。
	// 対象 sub の LoginSession をすべて物理削除する (tombstone ではなく erasure)。
	DeleteAllForSub(ctx context.Context, sub string) error

	// DeleteExpiredBatch は expires_at が cutoff より前の行を最大 limit 件まで物理削除し、
	// 削除件数を返す。housekeeping cleanup に使う (wi-253 Plan §7)。
	DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error)
}
