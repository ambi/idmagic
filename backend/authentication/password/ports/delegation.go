package ports

import (
	"context"
	"errors"
)

// PasswordVerification は、外部の権威の選択と照合の結果を認証の意味へ写したものである。
type PasswordVerification struct {
	Delegated   bool
	Accepted    bool
	Unavailable bool
}

var ErrPasswordDelegated = errors.New("password is managed by the directory")

// PasswordDelegation は取り込み元の対応と配送を認証から隠す。書き込みは、委譲の切り替えと
// 共通の排他の中で検査して実行し、検査直後の有効化による資格情報の追加を防ぐ。
type PasswordDelegation interface {
	VerifyPassword(ctx context.Context, userID, password string) (PasswordVerification, error)
	WithLocalPasswordWrite(ctx context.Context, userID string, apply func() error) error
}
