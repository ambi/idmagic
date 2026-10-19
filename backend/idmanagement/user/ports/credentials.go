package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// UserCredentialPurger は、完全削除の認証情報の区間を担当する。途中失敗の後も再実行でき、
// 個々の保存先の確定を一括のトランザクションへ変えない。セッションの削除は別の区間である。
type UserCredentialPurger interface {
	PurgeUserCredentials(ctx context.Context, sub string) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

// PasswordHistoryWriter は、通常の作成と CSV が必要とする初回の履歴の書き込みだけを要求する。
// CSV の PostgreSQL アダプターには、行と同じ Tx に束縛した実装を渡す。
type PasswordHistoryWriter interface {
	Add(ctx context.Context, sub, encoded string, now time.Time) error
}

// PasswordPolicyRejection は、所有者が決めた違反の表示コードを返す。規則の計算と列挙は所有者に残す。
type PasswordPolicyRejection interface {
	error
	PasswordPolicyViolations() []string
}

// UserDeviceRevoker は、無効化に伴う端末の失効と、呼び出し側が発行するイベントを返す。
type UserDeviceRevoker interface {
	RevokeAllForUser(ctx context.Context, tenantID, userID string, reason spec.TrustedDeviceRevokeReason, now time.Time) ([]spec.DomainEvent, error)
}
