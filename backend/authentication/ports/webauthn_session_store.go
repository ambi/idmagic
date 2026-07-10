package ports

import (
	"context"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnSessionStore は WebAuthn ceremony の challenge (go-webauthn SessionData) を短命に
// 保持する ephemeral store (wi-26 / ADR-087)。登録は sub、ログインは pending login session id
// をキーにする。Take は一度きり (取得と同時に削除) の消費で replay を防ぐ。
type WebAuthnSessionStore interface {
	Save(ctx context.Context, key string, data gowebauthn.SessionData, expiresAt time.Time) error
	// Take は key に対応する SessionData を取り出し、同時に削除する。無ければ (nil, nil)。
	Take(ctx context.Context, key string) (*gowebauthn.SessionData, error)
}
