package ports

import (
	"context"
	"crypto"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// SigningKey は本実装では RSA を想定。alg=PS256 のみ。
// 公開鍵 JWK は JWKS 配布用。鍵はテナントに帰属する (TenantID)。
// VaultTransit provider では PrivateKey は nil で、署名は provider が担う。
type SigningKey struct {
	TenantID   string
	Kid        string
	Alg        spec.SignatureAlgorithm
	Provider   spec.KeyProvider
	Usage      spec.KeyUsage
	PrivateKey crypto.PrivateKey
	PublicKey  crypto.PublicKey
	PublicJWK  map[string]any
	Active     bool
	CreatedAt  time.Time
}

// TenantKeyHealth は system_admin 向けのテナント 1 件分の署名鍵ヘルス。
type TenantKeyHealth struct {
	TenantID     string
	Provider     spec.KeyProvider
	Usage        spec.KeyUsage
	ActiveKid    string
	JWKSKeyCount int
	Healthy      bool
}

// KeyStore はテナント帰属の署名鍵を扱う。tenant scope は ctx (tenancy.TenantID)
// から解決し、列挙・検索・回転・署名鍵選択はすべて ctx のテナントに閉じる。
type KeyStore interface {
	GetActiveKey(ctx context.Context) (*SigningKey, error)
	GetAllKeys(ctx context.Context) ([]*SigningKey, error)
	FindByKID(ctx context.Context, kid string) (*SigningKey, error)
	Rotate(ctx context.Context) (*SigningKey, error)
	// Disable は ctx テナントの鍵 1 件を無効化 (JWKS から除去) する。
	Disable(ctx context.Context, kid string) (*SigningKey, error)
	// Provider はこの KeyStore の KeyProvider を返す (ヘルス表示用)。
	Provider() spec.KeyProvider
	// Healthy は ctx テナントの provider が到達可能かを返す。false のとき
	// fail-closed で新規署名を止める。
	Healthy(ctx context.Context) bool
}
