package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
)

// RecoveryCodeRepository は backup recovery code set の永続化 (wi-26 / ADR-087)。
// 平文は保存せず code_hash (SHA-256 hex) のみを持つ。1 ユーザーの有効 set を 1 単位で扱う。
type RecoveryCodeRepository interface {
	ListBySub(ctx context.Context, sub string) ([]*domain.RecoveryCode, error)
	// ReplaceAll は対象 sub の既存 code を全削除してから新しい set を保存する (再生成)。
	ReplaceAll(ctx context.Context, sub string, codes []*domain.RecoveryCode) error
	// MarkConsumed は未使用の code_hash を使用済み (consumed_at) にする。該当が無ければ
	// (false, nil) を返す (未知 / 使用済み / 別 sub)。
	MarkConsumed(ctx context.Context, sub, codeHash string, now time.Time) (bool, error)
	// DeleteAllForSub は失効および ADR-036 の anonymize cascade から呼ばれる。
	DeleteAllForSub(ctx context.Context, sub string) error
}
