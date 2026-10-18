package ports

import (
	"context"
	"time"
)

type PasswordHistoryEntry struct {
	Encoded   string
	CreatedAt time.Time
}

type PasswordHistoryWriter interface {
	Add(ctx context.Context, sub, encoded string, now time.Time) error
}

type PasswordHistoryRepository interface {
	PasswordHistoryWriter
	Recent(ctx context.Context, sub string, depth int) ([]PasswordHistoryEntry, error)
	// DeleteAllForSub は anonymize cascade から呼ばれる。
	// 対象 sub のパスワード履歴をすべて物理削除する。
	DeleteAllForSub(ctx context.Context, sub string) error
}
