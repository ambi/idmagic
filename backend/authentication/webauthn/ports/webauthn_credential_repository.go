package ports

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/webauthn/domain"
)

// ErrWebAuthnCredentialExists は、保存しようとした credential ID が、持ち主を問わず既に
// 保存されている場合に Save が返す。既存の記録は変えない。
var ErrWebAuthnCredentialExists = errors.New("webauthn credential id already registered")

// WebAuthnCredentialRepository は登録済み WebAuthn / Passkey credential の永続化 (wi-26 /
// )。1 ユーザーが複数持てるため credential_id で一意識別する。
type WebAuthnCredentialRepository interface {
	ListBySub(ctx context.Context, sub string) ([]*domain.WebAuthnCredential, error)
	FindByCredentialID(ctx context.Context, credentialID string) (*domain.WebAuthnCredential, error)
	Save(ctx context.Context, credential *domain.WebAuthnCredential) error
	// UpdateSignCount は assertion 成功時に署名カウンタと最終利用時刻を更新する。
	UpdateSignCount(ctx context.Context, credentialID string, signCount uint32, lastUsedAt time.Time) error
	Delete(ctx context.Context, sub, credentialID string) error
	// DeleteAllForSub は Purge の匿名化 cascade と、管理者による認証要素のリセットから呼ばれる。
	DeleteAllForSub(ctx context.Context, sub string) error
}
