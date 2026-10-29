// Package cimd_http fetches and resolves OAuth Client ID Metadata Documents
// (CIMD, draft-ietf-oauth-client-id-metadata-document-02) over an SSRF-safe
// HTTP client, and adapts resolution into the existing OAuth2ClientRepository
// lookup path via a decorator. Resolution results are never persisted.
package cimd_http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	clientdomain "github.com/ambi/idmagic/backend/oauth2/internal/client/domain"
	"github.com/ambi/idmagic/backend/shared/security/safehttp"
)

const maxDocumentBytes = 1 << 16 // 64 KiB; CIMD documents are small, fixed-shape JSON.

type cachedDocument struct {
	client    *clientdomain.OAuth2Client
	expiresAt time.Time
}

// Fetcher fetches and validates Client ID Metadata Documents, caching
// successful resolutions for 5 minutes.
type Fetcher struct {
	mu     sync.Mutex
	cache  map[string]cachedDocument
	client *http.Client
}

// NewFetcher builds a Fetcher whose HTTP client rejects non-https URLs and
// non-public IP targets (shared dialer from shared/security/safehttp, the
// same hardening tokens_jose.JWKResolver uses for jwks_uri).
func NewFetcher() *Fetcher {
	return newFetcherWithClient(safehttp.NewClient(safehttp.Config{
		DialTimeout:    2 * time.Second,
		TLSTimeout:     2 * time.Second,
		RequestTimeout: 3 * time.Second,
		ValidateURL:    validateClientIDURL,
	}))
}

// newFetcherWithClient lets tests exercise Fetch's status/size/cache logic
// against an httptest.Server without routing through the SSRF-safe dialer
// (which rejects loopback targets by design). The dialer itself is covered
// by shared/security/safehttp's own tests plus
// TestFetcher_RealFetcherRejectsLoopbackViaSSRFDialer.
func newFetcherWithClient(client *http.Client) *Fetcher {
	return &Fetcher{cache: map[string]cachedDocument{}, client: withoutRedirects(client)}
}

// withoutRedirects は client の複製を、リダイレクトを追従せず 3xx の応答をそのまま返す
// ように設定する (CIMD02-NO-REDIRECT)。3xx は 200 以外の応答として Fetch が拒否する。
// 共有の safehttp ではなくここで設定するのは、jwks_uri など別の用途のリダイレクト方針を
// 変えないためと、テストで差し替えた HTTP クライアントにも同じ方針を効かせるためである。
func withoutRedirects(client *http.Client) *http.Client {
	copied := *client
	copied.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &copied
}

func validateClientIDURL(raw string) error {
	if !clientdomain.IsClientIDMetadataDocumentURL(raw) {
		return errors.New("client id metadata document: client_id is not a valid https metadata document url")
	}
	return nil
}

// Fetch resolves clientIDURL into a synthesized, non-persisted OAuth2Client.
func (f *Fetcher) Fetch(ctx context.Context, clientIDURL string) (*clientdomain.OAuth2Client, error) {
	if err := validateClientIDURL(clientIDURL); err != nil {
		return nil, err
	}
	f.mu.Lock()
	cached, ok := f.cache[clientIDURL]
	f.mu.Unlock()
	if ok && time.Now().Before(cached.expiresAt) {
		return cached.client, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientIDURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch client id metadata document: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch client id metadata document: status %d", resp.StatusCode)
	}

	reader := io.LimitReader(resp.Body, maxDocumentBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(data) > maxDocumentBytes {
		return nil, errors.New("client id metadata document response exceeds 64 KiB")
	}

	client, err := clientdomain.ParseClientIDMetadataDocument(data, clientIDURL, time.Now())
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.cache[clientIDURL] = cachedDocument{client: client, expiresAt: time.Now().Add(5 * time.Minute)}
	f.mu.Unlock()
	return client, nil
}
