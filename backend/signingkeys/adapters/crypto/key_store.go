// Package crypto: 鍵ストアと JWT 署名 (PS256)。
//
// ローカル開発用 in-memory 鍵ストア。本番では KMS / HSM / Vault を使う想定。
// JWK サムプリント (RFC 7638) を kid として使用する。
package crypto

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"
)

func GenerateRSAJWKPair() (*rsa.PrivateKey, map[string]any, map[string]any, string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, "", err
	}
	publicJWK := rsaPublicJWK(&priv.PublicKey)
	kid, err := jwkThumbprint(publicJWK)
	if err != nil {
		return nil, nil, nil, "", err
	}
	publicJWK["kid"] = kid
	publicJWK["alg"] = string(signingdomain.SigAlgPS256)
	publicJWK["use"] = "sig"
	privateJWK := map[string]any{
		"kty": "RSA",
		"kid": kid,
		"alg": string(signingdomain.SigAlgPS256),
		"n":   base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigIntFromInt(priv.E)),
		"d":   base64.RawURLEncoding.EncodeToString(priv.D.Bytes()),
		"p":   base64.RawURLEncoding.EncodeToString(priv.Primes[0].Bytes()),
		"q":   base64.RawURLEncoding.EncodeToString(priv.Primes[1].Bytes()),
	}
	return priv, publicJWK, privateJWK, kid, nil
}

func ImportRSAJWK(publicJWK, privateJWK map[string]any) (crypto.PublicKey, crypto.PrivateKey, error) {
	pub, err := publicKeyFromJWK(publicJWK)
	if err != nil {
		return nil, nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("public JWK is not RSA")
	}
	decodeInt := func(name string) (*big.Int, error) {
		value, _ := privateJWK[name].(string)
		decoded, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil || len(decoded) == 0 {
			return nil, errors.New("private JWK missing or invalid " + name)
		}
		return new(big.Int).SetBytes(decoded), nil
	}
	d, err := decodeInt("d")
	if err != nil {
		return nil, nil, err
	}
	p, err := decodeInt("p")
	if err != nil {
		return nil, nil, err
	}
	q, err := decodeInt("q")
	if err != nil {
		return nil, nil, err
	}
	priv := &rsa.PrivateKey{PublicKey: *rsaPub, D: d, Primes: []*big.Int{p, q}}
	if err := priv.Validate(); err != nil {
		return nil, nil, err
	}
	priv.Precompute()
	return rsaPub, priv, nil
}

func publicKeyFromJWK(jwk map[string]any) (crypto.PublicKey, error) {
	if kty, _ := jwk["kty"].(string); kty != "RSA" {
		return nil, errors.New("public JWK is not RSA")
	}
	nValue, _ := jwk["n"].(string)
	eValue, _ := jwk["e"].(string)
	nBytes, err := base64.RawURLEncoding.DecodeString(nValue)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eValue)
	if err != nil {
		return nil, err
	}
	exponent := 0
	for _, b := range eBytes {
		exponent = exponent<<8 | int(b)
	}
	if len(nBytes) == 0 || exponent == 0 {
		return nil, errors.New("public JWK missing RSA modulus or exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}, nil
}

// tenantKeys は 1 テナント分の署名鍵集合と active kid を保持する。
type tenantKeys struct {
	keys   []*signingdomain.SigningKey
	active string
}

// InMemoryKeyStore は dev/test 用の tenant-aware な in-memory 鍵ストア。
// tenant scope は ctx (tenancy.TenantID) から解決する。
type InMemoryKeyStore struct {
	mu       sync.RWMutex
	byTenant map[string]*tenantKeys
}

func NewInMemoryKeyStore() (*InMemoryKeyStore, error) {
	ks := &InMemoryKeyStore{byTenant: map[string]*tenantKeys{}}
	// default テナントの鍵を先に作る (後方互換)。
	if _, err := ks.rotateInternal(tenancydomain.DefaultTenantID, time.Now().UTC(), 7*24*time.Hour); err != nil {
		return nil, err
	}
	return ks, nil
}

func (s *InMemoryKeyStore) GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error) {
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
	// まだ鍵の無いテナントは初回に遅延生成する。
	return s.rotateInternal(tenantID, time.Now().UTC(), 7*24*time.Hour)
}

func (s *InMemoryKeyStore) GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
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

func (s *InMemoryKeyStore) ListPublicKeys(ctx context.Context, now time.Time) ([]*signingdomain.SigningKey, error) {
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

func (s *InMemoryKeyStore) FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		return nil, nil //nolint:nilnil // repository contract: a missing key is not an adapter failure.
	}
	for _, k := range tk.keys {
		if k.Kid == kid {
			return k, nil
		}
	}
	return nil, nil //nolint:nilnil // repository contract: a missing key is not an adapter failure.
}

func (s *InMemoryKeyStore) Rotate(ctx context.Context, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	return s.rotateInternal(tenancy.TenantID(ctx), now, grace)
}

func (s *InMemoryKeyStore) Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	tenantID := tenancy.TenantID(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		return nil, nil //nolint:nilnil // repository contract: a missing key is not an adapter failure.
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

func (s *InMemoryKeyStore) ArchiveExpired(ctx context.Context, before time.Time) ([]*signingdomain.SigningKey, error) {
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

func (s *InMemoryKeyStore) Provider() signingdomain.KeyProvider {
	return signingdomain.KeyProviderLocal
}

func (s *InMemoryKeyStore) Healthy(_ context.Context) bool { return true }

func (s *InMemoryKeyStore) rotateInternal(tenantID string, now time.Time, grace time.Duration) (*signingdomain.SigningKey, error) {
	priv, jwk, _, kid, err := GenerateRSAJWKPair()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tk := s.byTenant[tenantID]
	if tk == nil {
		tk = &tenantKeys{}
		s.byTenant[tenantID] = tk
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if grace < 0 {
		return nil, errors.New("signing key grace period must not be negative")
	}
	for _, k := range tk.keys {
		k.Active = false
		k.RetiredAt = &now
		expires := now.Add(grace)
		k.ExpiresAt = &expires
	}
	key := &signingdomain.SigningKey{
		TenantID:   tenantID,
		Kid:        kid,
		Alg:        signingdomain.SigAlgPS256,
		Provider:   signingdomain.KeyProviderLocal,
		Usage:      signingdomain.KeyUsageSigning,
		PrivateKey: priv,
		PublicKey:  &priv.PublicKey,
		PublicJWK:  jwk,
		Active:     true,
		CreatedAt:  now,
	}
	tk.keys = append(tk.keys, key)
	tk.active = kid
	return key, nil
}

// rsaPublicJWK は RSA 公開鍵を JWK 形式の map に変換する。
func rsaPublicJWK(pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigIntFromInt(pub.E)),
	}
}

// bigIntFromInt は RSA 公開指数 (E) 等の非負整数を big-endian bytes に符号化する。
// big.Int 経由で先頭の 0x00 を取り除き、JWK の "e" メンバー形式に合わせる。
func bigIntFromInt(v int) []byte {
	return new(big.Int).SetInt64(int64(v)).Bytes()
}

// jwkThumbprint は RFC 7638 に従い JWK の SHA-256 サムプリントを base64url で返す。
// canonical JSON: required メンバーのみ、辞書順、空白なし。
func jwkThumbprint(jwk map[string]any) (string, error) {
	required := []string{"e", "kty", "n"}
	canon := map[string]any{}
	for _, k := range required {
		v, ok := jwk[k]
		if !ok {
			return "", errors.New("jwk missing required member: " + k)
		}
		canon[k] = v
	}
	b, err := json.Marshal(canon)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}
