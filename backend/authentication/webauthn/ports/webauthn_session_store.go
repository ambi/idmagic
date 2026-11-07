package ports

import (
	"context"
	"encoding/json"
	"time"
)

// WebAuthnSession は、開始から完了までの間に保持する WebAuthn のセレモニーの状態である。
// Payload は WebAuthn の実装が直列化した JSON で、ストアは中身を解釈せずにそのまま保存する。
type WebAuthnSession struct {
	Payload json.RawMessage
}

// WebAuthnSessionStore は WebAuthn ceremony の challenge を短命に
// 保持する ephemeral store (wi-26)。登録は sub、ログインは pending login session id
// をキーにする。Take は一度きり (取得と同時に削除) の消費で replay を防ぐ。
type WebAuthnSessionStore interface {
	Save(ctx context.Context, key string, session WebAuthnSession, expiresAt time.Time) error
	// Take は key に対応するセッションを取り出し、同時に削除する。無ければ (nil, nil)。
	Take(ctx context.Context, key string) (*WebAuthnSession, error)
}
