package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
)

// WebAuthnCredentialRepository は登録済み WebAuthn / Passkey credential の永続化 (wi-26 /
// ADR-087)。1 ユーザーが複数持てるため credential_id で一意識別する。
type WebAuthnCredentialRepository interface {
	ListBySub(ctx context.Context, sub string) ([]*domain.WebAuthnCredential, error)
	FindByCredentialID(ctx context.Context, credentialID string) (*domain.WebAuthnCredential, error)
	Save(ctx context.Context, credential *domain.WebAuthnCredential) error
	// UpdateSignCount は assertion 成功時に署名カウンタと最終利用時刻を更新する。
	UpdateSignCount(ctx context.Context, credentialID string, signCount uint32, lastUsedAt time.Time) error
	Delete(ctx context.Context, sub, credentialID string) error
	// DeleteAllForSub は ADR-036 の anonymize cascade から呼ばれる。
	DeleteAllForSub(ctx context.Context, sub string) error
}
