package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/authentication/domain"
)

type SessionStore interface {
	Save(ctx context.Context, s *domain.LoginSession) error
	Find(ctx context.Context, sessionID string) (*domain.LoginSession, error)
	Delete(ctx context.Context, sessionID string) error
	// ListBySub は対象 sub の有効な (未期限切れ・認証完了済み) LoginSession を返す
	// (wi-20 スライス 2)。self / admin のセッション一覧に使う。順序は問わない。
	ListBySub(ctx context.Context, sub string) ([]*domain.LoginSession, error)
	// DeleteAllForSub は ADR-036 の anonymize cascade から呼ばれる。
	// 対象 sub の LoginSession をすべて物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
