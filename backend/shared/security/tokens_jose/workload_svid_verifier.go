// Package tokens_jose: 外部 workload attestation token (JWT-SVID / Kubernetes
// projected ServiceAccount token / クラウド instance identity token) の検証
// (JWKS/JWT 検証基盤を再利用、[[wi-54-workload-identity-federation-spiffe]])。
package tokens_jose

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	_ "crypto/sha512" // SHA-384 と SHA-512 を crypto.Hash に登録する
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync"
	"time"
)

const workloadSVIDClockSkewSeconds = 60

var (
	ErrWorkloadSVIDMalformed        = errors.New("workload_svid: malformed JWT")
	ErrWorkloadSVIDInvalidSignature = errors.New("workload_svid: signature invalid")
	ErrWorkloadSVIDIssuerMismatch   = errors.New("workload_svid: iss does not match the registered trust bundle")
	ErrWorkloadSVIDAudienceMismatch = errors.New("workload_svid: aud does not match any accepted audience")
	ErrWorkloadSVIDExpired          = errors.New("workload_svid: token is expired")
	ErrWorkloadSVIDTTLExceeded      = errors.New("workload_svid: token lifetime exceeds the trust bundle max TTL")
	ErrWorkloadSVIDMissingClaim     = errors.New("workload_svid: required claim missing")
)

// WorkloadSVIDProfile は検証の種別。OIDC は OIDC 互換の JWT、SPIFFE は SPIFFE の
// JWT-SVID の JOSE の規則 (kid 必須、typ、alg の集合、iss 任意) を掛ける。SPIFFE ID の
// 構文とトラストドメインの一致は、SPIFFE ID を値型として持つ呼び出し側が判定する。
type WorkloadSVIDProfile int

const (
	WorkloadSVIDProfileOIDC WorkloadSVIDProfile = iota
	WorkloadSVIDProfileSPIFFE
)

// WorkloadSVIDExpectation は信頼設定から決まる検証の期待値。
type WorkloadSVIDExpectation struct {
	Profile WorkloadSVIDProfile
	// Issuer は iss と完全一致しなければならない発行者。SPIFFE で空なら iss を確かめない。
	Issuer            string
	AcceptedAudiences []string
	MaxTTL            time.Duration
}

// oidcWorkloadAlgs は外部 issuer のデフォルト (Kubernetes、クラウドは RS256 が多い) を含む
// OIDC の集合。本アプリの自己発行 JWT より広いが、wi-54 から変えていない。
var oidcWorkloadAlgs = []string{"PS256", "ES256", "RS256"}

// spiffeWorkloadAlgs は SPIFFE の JWT-SVID 仕様が列挙する集合。
var spiffeWorkloadAlgs = []string{
	"RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "PS256", "PS384", "PS512",
}

// WorkloadSVIDClaims は検証を通過した外部 attestation token の claim。
type WorkloadSVIDClaims struct {
	Issuer   string
	Subject  string
	Audience []string
	// JTI は文字列の `jti` クレームで、なければ空である。JWT-SVID は `jti` を必須としない。
	JTI       string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// VerifyWorkloadSVID は外部 workload attestation token を検証する (fail-closed)。
// alg は種別の集合のいずれか、iss は want.Issuer と完全一致 (SPIFFE で want.Issuer が空
// なら確かめない)、aud は受理 audience のいずれかと一致、exp は clock skew を許容して
// 未来、(exp - iat) は MaxTTL 以下でなければならない。SPIFFE では JOSE ヘッダーの kid
// を必須とし、typ は省略か JWT か JOSE に限る。
func VerifyWorkloadSVID(
	token string,
	jwks []map[string]any,
	want WorkloadSVIDExpectation,
	now time.Time,
) (*WorkloadSVIDClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrWorkloadSVIDMalformed
	}
	var header map[string]any
	if err := decodeJWTSegment(parts[0], &header); err != nil {
		return nil, err
	}
	spiffe := want.Profile == WorkloadSVIDProfileSPIFFE
	allowedAlgs := oidcWorkloadAlgs
	if spiffe {
		allowedAlgs = spiffeWorkloadAlgs
	}
	alg, _ := header["alg"].(string)
	if !slices.Contains(allowedAlgs, alg) {
		return nil, fmt.Errorf("%w: alg %q not allowed", ErrWorkloadSVIDInvalidSignature, alg)
	}
	kid, _ := header["kid"].(string)
	if spiffe {
		if kid == "" {
			return nil, fmt.Errorf("%w: kid is required for a JWT-SVID", ErrWorkloadSVIDInvalidSignature)
		}
		if typ, present := header["typ"]; present && typ != "JWT" && typ != "JOSE" {
			return nil, fmt.Errorf("%w: typ %v not allowed for a JWT-SVID", ErrWorkloadSVIDInvalidSignature, typ)
		}
	}

	var payload map[string]any
	if err := decodeJWTSegment(parts[1], &payload); err != nil {
		return nil, err
	}

	pub, err := pickWorkloadJWK(jwks, kid)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDInvalidSignature, err)
	}
	if err := verifyWorkloadSignature(parts, alg, pub); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDInvalidSignature, err)
	}

	iss, _ := payload["iss"].(string)
	if (want.Issuer != "" || !spiffe) && (iss == "" || iss != want.Issuer) {
		return nil, ErrWorkloadSVIDIssuerMismatch
	}
	if !verifyAudience(payload["aud"], want.AcceptedAudiences) {
		return nil, ErrWorkloadSVIDAudienceMismatch
	}
	sub, _ := payload["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("%w: sub", ErrWorkloadSVIDMissingClaim)
	}
	expF, ok := payload["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("%w: exp", ErrWorkloadSVIDMissingClaim)
	}
	iatF, ok := payload["iat"].(float64)
	if !ok {
		return nil, fmt.Errorf("%w: iat", ErrWorkloadSVIDMissingClaim)
	}
	exp := time.Unix(int64(expF), 0).UTC()
	iat := time.Unix(int64(iatF), 0).UTC()
	if exp.Add(workloadSVIDClockSkewSeconds * time.Second).Before(now) {
		return nil, ErrWorkloadSVIDExpired
	}
	if exp.Sub(iat) > want.MaxTTL {
		return nil, ErrWorkloadSVIDTTLExceeded
	}

	jti, _ := payload["jti"].(string)
	return &WorkloadSVIDClaims{
		Issuer:    iss,
		Subject:   sub,
		Audience:  audienceStrings(payload["aud"]),
		JTI:       jti,
		IssuedAt:  iat,
		ExpiresAt: exp,
	}, nil
}

func decodeJWTSegment(segment string, out any) error {
	b, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrWorkloadSVIDMalformed, err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("%w: %w", ErrWorkloadSVIDMalformed, err)
	}
	return nil
}

// pickWorkloadJWK は pickJWK と同じ規則で鍵を選ぶが、P-384 と P-521 の EC 鍵も読む。
// 鍵の読み方を広げるのはこの検証だけで、DPoP やクライアントの JWK の登録が受け付ける
// 鍵は変えない。
func pickWorkloadJWK(jwks []map[string]any, kid string) (crypto.PublicKey, error) {
	if kid == "" {
		if len(jwks) == 1 {
			return workloadPublicKeyFromJWK(jwks[0])
		}
		return nil, errors.New("kid required when JWKS has multiple keys")
	}
	for _, j := range jwks {
		if k, _ := j["kid"].(string); k == kid {
			return workloadPublicKeyFromJWK(j)
		}
	}
	return nil, fmt.Errorf("kid %q not found in JWKS", kid)
}

var workloadECCurves = map[string]elliptic.Curve{
	"P-256": elliptic.P256(),
	"P-384": elliptic.P384(),
	"P-521": elliptic.P521(),
}

func workloadPublicKeyFromJWK(jwk map[string]any) (crypto.PublicKey, error) {
	if kty, _ := jwk["kty"].(string); kty != "EC" {
		return publicKeyFromJWK(jwk)
	}
	crvName, _ := jwk["crv"].(string)
	curve, ok := workloadECCurves[crvName]
	if !ok {
		return nil, fmt.Errorf("unsupported EC curve %q", crvName)
	}
	size := (curve.Params().BitSize + 7) / 8
	xB64, _ := jwk["x"].(string)
	yB64, _ := jwk["y"].(string)
	x, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, err
	}
	y, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, err
	}
	if len(x) != size || len(y) != size {
		return nil, fmt.Errorf("invalid %s coordinate length", crvName)
	}
	point := make([]byte, 0, 1+2*size)
	point = append(point, 4)
	point = append(point, x...)
	point = append(point, y...)
	return ecdsa.ParseUncompressedPublicKey(curve, point)
}

// workloadAlg は JWS の alg ごとの鍵の種類、ハッシュ、EC の曲線 (RFC 7518 の 3.1 節)。
type workloadAlg struct {
	hash  crypto.Hash
	pss   bool
	curve string // 空なら RSA
}

var workloadAlgs = map[string]workloadAlg{
	"RS256": {hash: crypto.SHA256}, "RS384": {hash: crypto.SHA384}, "RS512": {hash: crypto.SHA512},
	"PS256": {hash: crypto.SHA256, pss: true}, "PS384": {hash: crypto.SHA384, pss: true}, "PS512": {hash: crypto.SHA512, pss: true},
	"ES256": {hash: crypto.SHA256, curve: "P-256"}, "ES384": {hash: crypto.SHA384, curve: "P-384"}, "ES512": {hash: crypto.SHA512, curve: "P-521"},
}

func verifyWorkloadSignature(parts []string, alg string, pub crypto.PublicKey) error {
	algSpec, ok := workloadAlgs[alg]
	if !ok {
		return fmt.Errorf("unsupported alg %q", alg)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	h := algSpec.hash.New()
	h.Write([]byte(parts[0] + "." + parts[1]))
	digest := h.Sum(nil)

	if algSpec.curve == "" {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("%s requires an RSA public key", alg)
		}
		if algSpec.pss {
			return rsa.VerifyPSS(rsaPub, algSpec.hash, digest, sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
		}
		return rsa.VerifyPKCS1v15(rsaPub, algSpec.hash, digest, sig)
	}
	ecPub, ok := pub.(*ecdsa.PublicKey)
	if !ok || ecPub.Curve != workloadECCurves[algSpec.curve] {
		return fmt.Errorf("%s requires an EC %s public key", alg, algSpec.curve)
	}
	size := (ecPub.Curve.Params().BitSize + 7) / 8
	if len(sig) != 2*size {
		return fmt.Errorf("%s signature must be %d bytes", alg, 2*size)
	}
	r := new(big.Int).SetBytes(sig[:size])
	s := new(big.Int).SetBytes(sig[size:])
	if !ecdsa.Verify(ecPub, digest, r, s) {
		return fmt.Errorf("%s signature invalid", alg)
	}
	return nil
}

func audienceStrings(aud any) []string {
	switch v := aud.(type) {
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

// =====================================================================
// WorkloadJWKSCache — last-known-good キャッシュ (bundle refresh)
// =====================================================================

const workloadJWKSMaxStaleness = 24 * time.Hour

type workloadJWKSCacheEntry struct {
	keys      []map[string]any
	fetchedAt time.Time
}

// WorkloadJWKSCache は WorkloadTrustBundle ごとの JWKS を last-known-good として
// キャッシュする。ライブ取得に失敗しても、直近の成功取得から workloadJWKSMaxStaleness
// 以内であれば古い鍵を使い続け (ネットワーク瞬断への耐性)、それを超えれば fail-closed で
// 拒否する ([[wi-54-workload-identity-federation-spiffe]])。
type WorkloadJWKSCache struct {
	mu      sync.Mutex
	entries map[string]workloadJWKSCacheEntry
}

func NewWorkloadJWKSCache() *WorkloadJWKSCache {
	return &WorkloadJWKSCache{entries: map[string]workloadJWKSCacheEntry{}}
}

// Get はライブ取得を試み、成功すればキャッシュを更新して返す (stale=false)。失敗した
// 場合、cacheKey の last-known-good が workloadJWKSMaxStaleness 以内なら stale=true で
// それを返す。live 取得が失敗し、使える cache も無ければ fail-closed でエラーを返す。
func (c *WorkloadJWKSCache) Get(
	ctx context.Context,
	cacheKey string,
	fetch func(ctx context.Context) ([]map[string]any, error),
	now time.Time,
) (keys []map[string]any, stale bool, err error) {
	keys, fetchErr := fetch(ctx)
	if fetchErr == nil {
		c.mu.Lock()
		c.entries[cacheKey] = workloadJWKSCacheEntry{keys: keys, fetchedAt: now}
		c.mu.Unlock()
		return keys, false, nil
	}

	c.mu.Lock()
	entry, ok := c.entries[cacheKey]
	c.mu.Unlock()
	if !ok {
		return nil, false, fmt.Errorf("workload_jwks_cache: fetch failed and no cached keys exist: %w", fetchErr)
	}
	if now.Sub(entry.fetchedAt) > workloadJWKSMaxStaleness {
		return nil, false, fmt.Errorf("workload_jwks_cache: fetch failed and last-known-good is stale beyond %s: %w", workloadJWKSMaxStaleness, fetchErr)
	}
	return entry.keys, true, nil
}
