package tokens_jose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/safehttp"
)

const maxJWKSBytes = 1 << 20

type cachedJWKS struct {
	keys      []map[string]any
	expiresAt time.Time
}

type JWKResolver struct {
	mu       sync.Mutex
	cache    map[string]cachedJWKS
	resolver *net.Resolver
	client   *http.Client
	// now は再利用の期限の時計。nil なら time.Now を使う。
	now func() time.Time
}

func NewJWKResolver() *JWKResolver {
	r := &JWKResolver{cache: map[string]cachedJWKS{}, resolver: net.DefaultResolver}
	r.client = safehttp.NewClient(safehttp.Config{
		DialTimeout:    2 * time.Second,
		TLSTimeout:     2 * time.Second,
		RequestTimeout: 3 * time.Second,
		MaxRedirects:   3,
		ValidateURL:    ValidateJWKSURI,
	})
	return r
}

func ValidateJWKSURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("jwks_uri: %w", err)
	}
	if parsed.Scheme != "https" {
		return errors.New("jwks_uri: https is required")
	}
	if parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("jwks_uri: invalid authority, userinfo, or fragment")
	}
	return nil
}

type JWKSource struct {
	JWKS    map[string]any
	JwksURI *string
}

func (r *JWKResolver) Resolve(ctx context.Context, client JWKSource) ([]map[string]any, error) {
	if keys, err := InlineJWKs(client.JWKS); err == nil {
		return keys, nil
	}
	if client.JwksURI == nil || *client.JwksURI == "" {
		return nil, errors.New("private_key_jwt client has no jwks or jwks_uri")
	}
	return r.fetch(ctx, *client.JwksURI)
}

// ResolveJWKSSource fetches keys from an inline JWKS document if present,
// otherwise from jwksURI (with the same cache/SSRF-safety as Resolve).
// Shared by any registrant of an external JWKS source; used by workload
// identity federation ([[wi-54-workload-identity-federation-spiffe]])
// to resolve a WorkloadTrustBundle's signing keys without depending on the
// OAuth2Client shape.
func (r *JWKResolver) ResolveJWKSSource(ctx context.Context, jwksURI *string, inlineJWKS map[string]any) ([]map[string]any, error) {
	if inlineJWKS != nil {
		if keys, err := InlineJWKs(inlineJWKS); err == nil {
			return keys, nil
		}
	}
	if jwksURI == nil || *jwksURI == "" {
		return nil, errors.New("no jwks or jwks_uri configured")
	}
	return r.fetch(ctx, *jwksURI)
}

func InlineJWKs(jwks map[string]any) ([]map[string]any, error) {
	var raw []any
	switch keys := jwks["keys"].(type) {
	case []any:
		raw = keys
	case []map[string]any:
		raw = make([]any, len(keys))
		for i := range keys {
			raw[i] = keys[i]
		}
	default:
		return nil, errors.New("inline jwks is missing keys")
	}
	if len(raw) == 0 {
		return nil, errors.New("inline jwks is empty")
	}
	keys := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		key, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("inline jwks contains a non-object key")
		}
		keys = append(keys, key)
	}
	return keys, nil
}

const jwksReuse = 5 * time.Minute

func (r *JWKResolver) fetch(ctx context.Context, raw string) ([]map[string]any, error) {
	return r.fetchDocument(ctx, raw, "", func(document map[string]any) ([]map[string]any, time.Duration, error) {
		keys, err := InlineJWKs(document)
		return keys, jwksReuse, err
	})
}

// fetchDocument は raw の文書を取得し、read が選んだ鍵を、read が返す期間だけ
// cacheKeyPrefix+raw の名前で再利用する。read のエラーは取得の失敗として返す。
func (r *JWKResolver) fetchDocument(
	ctx context.Context,
	raw, cacheKeyPrefix string,
	read func(document map[string]any) ([]map[string]any, time.Duration, error),
) ([]map[string]any, error) {
	if err := ValidateJWKSURI(raw); err != nil {
		return nil, err
	}
	parsed, _ := url.Parse(raw)
	if _, err := r.safeIPs(ctx, parsed.Hostname()); err != nil {
		return nil, err
	}
	cacheKey := cacheKeyPrefix + raw
	r.mu.Lock()
	cached, ok := r.cache[cacheKey]
	r.mu.Unlock()
	if ok && r.clock().Before(cached.expiresAt) {
		return cached.keys, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks_uri: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks_uri: status %d", resp.StatusCode)
	}
	var document map[string]any
	reader := io.LimitReader(resp.Body, maxJWKSBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(data) > maxJWKSBytes {
		return nil, errors.New("jwks_uri response exceeds 1 MiB")
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("decode jwks_uri: %w", err)
	}
	keys, reuseFor, err := read(document)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.cache[cacheKey] = cachedJWKS{keys: keys, expiresAt: r.clock().Add(reuseFor)}
	r.mu.Unlock()
	return keys, nil
}

func (r *JWKResolver) clock() time.Time {
	if r.now != nil {
		return r.now()
	}
	return time.Now()
}

// safeIPs delegates to safehttp.SafeIPs; kept as a method (rather than
// calling safehttp directly at call sites) so existing tests and the
// pre-fetch check in fetch() don't need to know about the shared resolver.
func (r *JWKResolver) safeIPs(ctx context.Context, host string) ([]net.IP, error) {
	return safehttp.SafeIPs(ctx, r.resolver, host)
}
