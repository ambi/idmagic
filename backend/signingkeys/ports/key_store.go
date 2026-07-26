package ports

import (
	"context"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
)

type (
	keyUsageContextKey struct{}
	keyScopeContextKey struct{}
)

const DefaultKeyScope = "default"

// WithKeyUsage selects the isolated key lifecycle used by a caller.
// Callers that do not opt in retain the OAuth/OIDC Signing lifecycle.
func WithKeyUsage(ctx context.Context, usage signingdomain.KeyUsage) context.Context {
	return context.WithValue(ctx, keyUsageContextKey{}, usage)
}

func KeyUsage(ctx context.Context) signingdomain.KeyUsage {
	if usage, ok := ctx.Value(keyUsageContextKey{}).(signingdomain.KeyUsage); ok && usage.Valid() {
		return usage
	}
	return signingdomain.KeyUsageSigning
}

// WithKeyScope selects an independent lifecycle within a tenant and usage.
func WithKeyScope(ctx context.Context, scopeID string) context.Context {
	if scopeID == "" {
		scopeID = DefaultKeyScope
	}
	return context.WithValue(ctx, keyScopeContextKey{}, scopeID)
}

func KeyScope(ctx context.Context) string {
	if scopeID, ok := ctx.Value(keyScopeContextKey{}).(string); ok && scopeID != "" {
		return scopeID
	}
	return DefaultKeyScope
}

// SigningKey は本実装では RSA を想定。alg=PS256 のみ。
// 公開鍵 JWK は JWKS 配布用。鍵はテナントに帰属する (TenantID)。
// VaultTransit provider では PrivateKey は nil で、署名は provider が担う。
// KeyStore はテナント帰属の署名鍵を扱う。tenant scope は ctx (tenancy.TenantID)
// から解決し、列挙・検索・回転・署名鍵選択はすべて ctx のテナントに閉じる。
type KeyStore interface {
	GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error)
	GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error)
	ListPublicKeys(ctx context.Context, now time.Time) ([]*signingdomain.SigningKey, error)
	FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error)
	Rotate(ctx context.Context, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error)
	ArchiveExpired(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error)
	// Disable は ctx テナントの鍵 1 件を無効化 (JWKS から除去) する。
	Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error)
	// Provider はこの KeyStore の KeyProvider を返す (ヘルス表示用)。
	Provider() signingdomain.KeyProvider
	// Healthy は ctx テナントの provider が到達可能かを返す。false のとき
	// fail-closed で新規署名を止める。
	Healthy(ctx context.Context) bool
}

// DueKeyRotator is an optional atomic compare-and-rotate capability used by
// externally scheduled batches. Implementations re-evaluate cadence inside
// their tenant-scoped serialization boundary.
type DueKeyRotator interface {
	RotateIfDue(ctx context.Context, now time.Time, cadence, grace time.Duration) (*signingdomain.SigningKey, error)
}
