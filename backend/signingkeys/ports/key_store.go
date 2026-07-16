package ports

import (
	"context"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
)

// SigningKey は本実装では RSA を想定。alg=PS256 のみ。
// 公開鍵 JWK は JWKS 配布用。鍵はテナントに帰属する (TenantID)。
// VaultTransit provider では PrivateKey は nil で、署名は provider が担う。
// KeyStore はテナント帰属の署名鍵を扱う。tenant scope は ctx (tenancy.TenantID)
// から解決し、列挙・検索・回転・署名鍵選択はすべて ctx のテナントに閉じる。
type KeyStore interface {
	GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error)
	GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error)
	FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error)
	Rotate(ctx context.Context) (*signingdomain.SigningKey, error)
	// Disable は ctx テナントの鍵 1 件を無効化 (JWKS から除去) する。
	Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error)
	// Provider はこの KeyStore の KeyProvider を返す (ヘルス表示用)。
	Provider() signingdomain.KeyProvider
	// Healthy は ctx テナントの provider が到達可能かを返す。false のとき
	// fail-closed で新規署名を止める。
	Healthy(ctx context.Context) bool
}
