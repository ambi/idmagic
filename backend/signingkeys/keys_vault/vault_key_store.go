// VaultKeyStore: HashiCorp Vault Transit secrets engine を使う本番用 KeyProvider。
//
// 秘密鍵マテリアルは Vault 外に出ない。署名は Vault の transit/sign へ委譲し、
// アプリは公開鍵 (JWKS 用) のミラーだけを保持する。tenant scope は ctx
// (tenancy.TenantID) から解決し、Vault の key name は prefix + tenantID とする。
package keys_vault

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"sync"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	keysJOSE "github.com/ambi/idmagic/backend/signingkeys/keys_jose"

	"github.com/ambi/idmagic/backend/tenancy"
)

type tenantKeys struct {
	keys   []*signingdomain.SigningKey
	active string
}

// TransitEngine は VaultKeyStore が必要とする Vault Transit の最小操作。
// 実装は HTTPTransitEngine、テストは fake で差し替える。
type TransitEngine interface {
	// EnsureKey は指定名の RSA-2048 署名鍵が無ければ作成する (冪等)。
	EnsureKey(ctx context.Context, name string) error
	// RotateKey は鍵バージョンを 1 つ進める。
	RotateKey(ctx context.Context, name string) error
	// LatestPublicKey は最新バージョンの公開鍵 (PKIX PEM) と version を返す。
	LatestPublicKey(ctx context.Context, name string) (pubPEM string, version int, err error)
	// Sign は prehashed digest に PSS(sha2-256) で署名し raw signature を返す。
	Sign(ctx context.Context, name string, version int, digest []byte) ([]byte, error)
	// Healthy は Vault が到達可能かを返す。
	Healthy(ctx context.Context) bool
}

// VaultKeyStore は TransitEngine を背にした tenant-aware な KeyStore。
type VaultKeyStore struct {
	engine   TransitEngine
	prefix   string
	mu       sync.RWMutex
	byTenant map[string]*tenantKeys
}

func NewVaultKeyStore(engine TransitEngine, keyNamePrefix string) *VaultKeyStore {
	if keyNamePrefix == "" {
		keyNamePrefix = "idmagic-signing-"
	}
	return &VaultKeyStore{engine: engine, prefix: keyNamePrefix, byTenant: map[string]*tenantKeys{}}
}

func (s *VaultKeyStore) keyName(tenantID string) string { return s.prefix + tenantID }

func (s *VaultKeyStore) GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	if tk := s.byTenant[tenantID]; tk != nil {
		for _, k := range tk.keys {
			if k.Kid == tk.active {
				s.mu.RUnlock()
				return k, nil
			}
		}
	}
	s.mu.RUnlock()
	// 未取得のテナントは Vault の最新鍵を取り込む (無ければ作成)。
	return s.loadOrRotate(ctx, tenantID, false, time.Now().UTC(), 7*24*time.Hour)
}

func (s *VaultKeyStore) GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		return []*signingdomain.SigningKey{}, nil
	}
	out := make([]*signingdomain.SigningKey, len(tk.keys))
	copy(out, tk.keys)
	return out, nil
}

func (s *VaultKeyStore) ListPublicKeys(ctx context.Context, now time.Time) ([]*signingdomain.SigningKey, error) {
	keys, err := s.GetAllKeys(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*signingdomain.SigningKey, 0, len(keys))
	for _, key := range keys {
		if key.ArchivedAt == nil && (key.ExpiresAt == nil || key.ExpiresAt.After(now)) {
			out = append(out, key)
		}
	}
	return out, nil
}

func (s *VaultKeyStore) FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		return nil, nil //nolint:nilnil // 契約: 見つからない場合は (nil, nil)。local / postgres と同一。
	}
	for _, k := range tk.keys {
		if k.Kid == kid {
			return k, nil
		}
	}
	return nil, nil //nolint:nilnil // 契約: 見つからない場合は (nil, nil)。
}

func (s *VaultKeyStore) Rotate(ctx context.Context, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	had := s.byTenant[tenantID] != nil && len(s.byTenant[tenantID].keys) > 0
	s.mu.RUnlock()
	return s.loadOrRotate(ctx, tenantID, had, now, grace)
}

func (s *VaultKeyStore) Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		return nil, nil //nolint:nilnil // 契約: 対象鍵が無ければ (nil, nil)。
	}
	remaining := tk.keys[:0]
	var disabled *signingdomain.SigningKey
	for _, k := range tk.keys {
		if k.Kid == kid {
			if tk.active == kid {
				return nil, signingdomain.ErrActiveSigningKeyCannotBeDisabled
			}
			disabled = k
			continue
		}
		remaining = append(remaining, k)
	}
	tk.keys = remaining
	return disabled, nil
}

func (s *VaultKeyStore) ArchiveExpired(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		return nil, nil
	}
	archived := []*signingdomain.SigningKey{}
	for _, key := range tk.keys {
		if key.ArchivedAt == nil && key.ExpiresAt != nil && !key.ExpiresAt.After(before) {
			at := before.UTC()
			key.ArchivedAt = &at
			archived = append(archived, key)
		}
	}
	return archived, nil
}

func (s *VaultKeyStore) Provider() signingdomain.KeyProvider {
	return signingdomain.KeyProviderVaultTransit
}

func (s *VaultKeyStore) Healthy(ctx context.Context) bool { return s.engine.Healthy(ctx) }

// loadOrRotate は Vault の鍵を用意し、最新公開鍵をミラーに取り込んで active にする。
// advance=true のときは新しいバージョンへ回転してから取り込む。
func (s *VaultKeyStore) loadOrRotate(ctx context.Context, tenantID string, advance bool, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if grace < 0 {
		return nil, errors.New("signing key grace period must not be negative")
	}
	name := s.keyName(tenantID)
	if err := s.engine.EnsureKey(ctx, name); err != nil {
		return nil, err
	}
	if advance {
		if err := s.engine.RotateKey(ctx, name); err != nil {
			return nil, err
		}
	}
	pubPEM, version, err := s.engine.LatestPublicKey(ctx, name)
	if err != nil {
		return nil, err
	}
	pub, err := parseRSAPublicKeyPEM(pubPEM)
	if err != nil {
		return nil, err
	}
	publicJWK := keysJOSE.PublicJWK(pub)
	kid, err := keysJOSE.Thumbprint(publicJWK)
	if err != nil {
		return nil, err
	}
	publicJWK["kid"] = kid
	publicJWK["alg"] = string(signingdomain.SigAlgPS256)
	publicJWK["use"] = "sig"
	key := &signingdomain.SigningKey{
		TenantID:   tenantID,
		Kid:        kid,
		Alg:        signingdomain.SigAlgPS256,
		Provider:   signingdomain.KeyProviderVaultTransit,
		Usage:      signingdomain.KeyUsageSigning,
		PrivateKey: vaultSigner{engine: s.engine, name: name, version: version, pub: pub},
		PublicKey:  pub,
		PublicJWK:  publicJWK,
		Active:     true,
		CreatedAt:  now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		tk = &tenantKeys{}
		s.byTenant[tenantID] = tk
	}
	// 同一 kid が既に取り込まれていれば active にするだけ。
	for _, existing := range tk.keys {
		if existing.Kid == kid {
			tk.active = kid
			return existing, nil
		}
	}
	for _, k := range tk.keys {
		k.Active = false
		k.RetiredAt = &now
		expires := now.Add(grace)
		k.ExpiresAt = &expires
	}
	tk.keys = append(tk.keys, key)
	tk.active = kid
	return key, nil
}

// vaultSigner は crypto.Signer を実装し、署名を Vault transit へ委譲する。
type vaultSigner struct {
	engine  TransitEngine
	name    string
	version int
	pub     *rsa.PublicKey
}

func (s vaultSigner) Public() crypto.PublicKey { return s.pub }

func (s vaultSigner) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	return s.engine.Sign(context.Background(), s.name, s.version, digest)
}

func parseRSAPublicKeyPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("vault public key is not PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("vault public key is not RSA")
	}
	return rsaPub, nil
}
