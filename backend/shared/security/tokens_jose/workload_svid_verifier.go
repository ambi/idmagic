// Package tokens_jose: 外部 workload attestation token (JWT-SVID / Kubernetes
// projected ServiceAccount token / クラウド instance identity token) の検証
// (ADR-023 の JWKS/JWT 検証基盤を再利用、[[wi-54-workload-identity-federation-spiffe]])。
package tokens_jose

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

// WorkloadSVIDClaims は検証を通過した外部 attestation token の claim。
type WorkloadSVIDClaims struct {
	Issuer    string
	Subject   string
	Audience  []string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// VerifyWorkloadSVID は外部 workload attestation token (JWT-SVID) を検証する
// (ADR-053 fail-closed)。alg は PS256 / ES256 / RS256 を受理する (外部 issuer は
// 本アプリの自己発行 JWT より alg の選択肢が広いため、[[wi-54-workload-identity-federation-spiffe]]
// では RS256 も受理する)。iss は登録済み WorkloadTrustBundle の issuer と完全一致、aud は
// 受理 audience のいずれかと一致、exp は clock skew を許容して未来、(exp - iat) は
// maxTTL 以下でなければならない。
func VerifyWorkloadSVID(
	token string,
	jwks []map[string]any,
	expectedIssuer string,
	acceptedAudiences []string,
	maxTTL time.Duration,
	now time.Time,
) (*WorkloadSVIDClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrWorkloadSVIDMalformed
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDMalformed, err)
	}
	var header map[string]any
	if err := json.Unmarshal(hb, &header); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDMalformed, err)
	}
	alg, _ := header["alg"].(string)
	if alg != "PS256" && alg != "ES256" && alg != "RS256" {
		return nil, fmt.Errorf("%w: alg %q not allowed", ErrWorkloadSVIDInvalidSignature, alg)
	}
	kid, _ := header["kid"].(string)

	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDMalformed, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(pb, &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDMalformed, err)
	}

	pub, err := pickJWK(jwks, kid)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDInvalidSignature, err)
	}
	if err := verifyJWTSignature(parts, alg, pub); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWorkloadSVIDInvalidSignature, err)
	}

	iss, _ := payload["iss"].(string)
	if iss == "" || iss != expectedIssuer {
		return nil, ErrWorkloadSVIDIssuerMismatch
	}
	if !verifyAudience(payload["aud"], acceptedAudiences) {
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
	if exp.Sub(iat) > maxTTL {
		return nil, ErrWorkloadSVIDTTLExceeded
	}

	return &WorkloadSVIDClaims{
		Issuer:    iss,
		Subject:   sub,
		Audience:  audienceStrings(payload["aud"]),
		IssuedAt:  iat,
		ExpiresAt: exp,
	}, nil
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
// WorkloadJWKSCache — last-known-good キャッシュ (ADR-053 bundle refresh)
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
