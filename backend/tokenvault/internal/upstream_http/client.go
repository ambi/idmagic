// Package upstream_http は TokenVault の外向き HTTP を実装する。接続先の検査は
// shared/security/safehttp に委ね、本文の上限、リダイレクトの拒否、OAuth の応答の解釈を担う。
package upstream_http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/safehttp"
	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

const (
	requestTimeout       = 30 * time.Second
	maxTokenResponseSize = 1 << 16
)

// Client は ports.UpstreamClient の実装である。
type Client struct {
	HTTP *http.Client
}

// New は公開アドレスだけへ接続し、リダイレクトを追わないクライアントを返す。
func New() *Client {
	httpClient := safehttp.NewClient(safehttp.Config{
		DialTimeout: 5 * time.Second, TLSTimeout: 5 * time.Second, RequestTimeout: requestTimeout,
		ValidateURL: func(string) error { return nil },
	})
	return &Client{HTTP: withoutRedirects(httpClient)}
}

// NewWithHTTPClient は与えられた通信でクライアントを作る。リダイレクトは同じく追わない。
func NewWithHTTPClient(httpClient *http.Client) *Client {
	return &Client{HTTP: withoutRedirects(httpClient)}
}

// withoutRedirects はリダイレクトを結果として返す。追うと規則を通らない宛先へトークンが届く。
func withoutRedirects(client *http.Client) *http.Client {
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return client
}

// Send は代理実行の要求を送る。
func (c *Client) Send(ctx context.Context, request ports.OutboundRequest) (ports.OutboundResponse, error) {
	outbound, err := http.NewRequestWithContext(ctx, request.Method, request.URL, bytes.NewReader(request.Body))
	if err != nil {
		return ports.OutboundResponse{}, ports.ErrUpstreamUnreachable
	}
	for name, value := range request.Headers {
		outbound.Header.Set(name, value)
	}
	outbound.Header.Set("Authorization", "Bearer "+request.AccessToken)
	response, err := c.HTTP.Do(outbound)
	if err != nil {
		return ports.OutboundResponse{}, ports.ErrUpstreamUnreachable
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, domain.MaxResponseBodyBytes+1))
	if err != nil {
		return ports.OutboundResponse{}, ports.ErrUpstreamUnreachable
	}
	if len(body) > domain.MaxResponseBodyBytes {
		return ports.OutboundResponse{}, ports.ErrUpstreamResponseTooLarge
	}
	return ports.OutboundResponse{Status: response.StatusCode, ContentType: response.Header.Get("Content-Type"), Body: body}, nil
}

// ExchangeCode は認可コードをトークンへ交換する。
func (c *Client) ExchangeCode(ctx context.Context, client ports.OAuthClient, code, verifier, redirectURI string) (ports.TokenResponse, error) {
	return c.token(ctx, client, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {redirectURI},
	})
}

// Refresh は更新トークンでトークンを更新する。
func (c *Client) Refresh(ctx context.Context, client ports.OAuthClient, refreshToken string) (ports.TokenResponse, error) {
	return c.token(ctx, client, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}})
}

// Revoke は RFC 7009 の失効を試す。
func (c *Client) Revoke(ctx context.Context, client ports.OAuthClient, token, tokenTypeHint string) error {
	response, err := c.postForm(ctx, client, client.RevocationEndpoint, url.Values{"token": {token}, "token_type_hint": {tokenTypeHint}})
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("revocation returned status %d", response.StatusCode)
	}
	return nil
}

func (c *Client) postForm(ctx context.Context, client ports.OAuthClient, endpoint string, form url.Values) (*http.Response, error) {
	if client.AuthMethod == domain.AuthMethodClientSecretPost {
		form.Set("client_id", client.ClientID)
		form.Set("client_secret", client.ClientSecret)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, ports.ErrUpstreamUnreachable
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// JSON を既定にしない上流があるので明示する。
	request.Header.Set("Accept", "application/json")
	if client.AuthMethod != domain.AuthMethodClientSecretPost {
		// RFC 6749 §2.3.1: 資格情報は form-urlencoded してから Basic にする。
		request.SetBasicAuth(url.QueryEscape(client.ClientID), url.QueryEscape(client.ClientSecret))
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return nil, ports.ErrUpstreamUnreachable
	}
	return response, nil
}

func (c *Client) token(ctx context.Context, client ports.OAuthClient, form url.Values) (ports.TokenResponse, error) {
	response, err := c.postForm(ctx, client, client.TokenEndpoint, form)
	if err != nil {
		return ports.TokenResponse{}, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxTokenResponseSize+1))
	if err != nil || len(body) > maxTokenResponseSize {
		return ports.TokenResponse{}, &ports.TokenEndpointError{Status: response.StatusCode}
	}
	return parseTokenResponse(response, body)
}

func parseTokenResponse(response *http.Response, body []byte) (ports.TokenResponse, error) {
	var parsed struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    *int64 `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
	}
	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if mediaType != "application/json" || json.Unmarshal(body, &parsed) != nil {
		return ports.TokenResponse{}, &ports.TokenEndpointError{Status: response.StatusCode}
	}
	if response.StatusCode != http.StatusOK || parsed.Error != "" {
		return ports.TokenResponse{}, &ports.TokenEndpointError{Status: response.StatusCode, OAuthError: parsed.Error}
	}
	if parsed.AccessToken == "" || !strings.EqualFold(parsed.TokenType, "bearer") {
		return ports.TokenResponse{}, &ports.TokenEndpointError{Status: response.StatusCode}
	}
	result := ports.TokenResponse{AccessToken: parsed.AccessToken, RefreshToken: parsed.RefreshToken}
	if parsed.Scope != "" {
		result.Scopes = strings.Fields(parsed.Scope)
	}
	if parsed.ExpiresIn != nil {
		lifetime := time.Duration(*parsed.ExpiresIn) * time.Second
		result.ExpiresIn = &lifetime
	}
	return result, nil
}

var _ ports.UpstreamClient = (*Client)(nil)
