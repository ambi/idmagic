package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

type RefreshTokenStore interface {
	FindByHash(ctx context.Context, hash string) (*domain.RefreshTokenRecord, error)
	Save(ctx context.Context, rec *domain.RefreshTokenRecord) error
	// Rotate は parentId を rotated にしつつ新レコードを atomic に保存。
	Rotate(ctx context.Context, parentID string, newRec *domain.RefreshTokenRecord) (*domain.RefreshTokenRecord, error)
	RevokeFamily(ctx context.Context, familyID string) error
	// DeleteAllForSub は ADR-036 の anonymize cascade から呼ばれる。
	// 対象 sub の RefreshToken を物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
