// Package issuer_metadata は信頼設定の入力支援に使う OIDC Discovery 文書を取得する。
package issuer_metadata

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/safehttp"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
)

type Client struct{ client *http.Client }

func NewClient() Client {
	return Client{client: safehttp.NewClient(safehttp.Config{
		DialTimeout: 2 * time.Second, TLSTimeout: 2 * time.Second, RequestTimeout: 3 * time.Second,
		MaxRedirects: 3, ValidateURL: tokens_jose.ValidateJWKSURI,
	})}
}

// Discover は候補だけを返す。JWKS 自体の取得と設定の保存は行わない。
func (c Client) Discover(ctx context.Context, issuer string) (string, error) {
	if err := tokens_jose.ValidateJWKSURI(issuer); err != nil {
		return "", err
	}
	parsed, err := url.Parse(issuer)
	if err != nil || parsed.RawQuery != "" || parsed.ForceQuery || strings.Contains(issuer, "#") {
		return "", errors.New("issuer query is not allowed")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(issuer, "/")+"/.well-known/openid-configuration", http.NoBody)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", errors.New("metadata endpoint did not return 200")
	}
	const maxBytes = 1 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxBytes {
		return "", errors.New("metadata is too large")
	}
	var document struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return "", err
	}
	if document.Issuer != issuer {
		return "", errors.New("metadata issuer mismatch")
	}
	if err := tokens_jose.ValidateJWKSURI(document.JWKSURI); err != nil {
		return "", err
	}
	return document.JWKSURI, nil
}
