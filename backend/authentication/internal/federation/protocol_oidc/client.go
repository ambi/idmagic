// Package protocol_oidc implements the Authentication broker's upstream OIDC RP port.
package protocol_oidc

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/internal/federation/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
)

const maxResponseBytes = 1 << 20

type Client struct {
	HTTPClient     *http.Client
	SecretResolver federationports.SecretResolver
}

type discoveryDocument struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// Discover は設定の入力候補を返す。取得処理に保存済みの接続を渡さない。
func (c Client) Discover(ctx context.Context, issuer string) (*domain.MetadataPreview, error) {
	var document discoveryDocument
	discoveryURL := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	if err := c.getJSON(ctx, discoveryURL, &document); err != nil {
		return nil, err
	}
	if document.Issuer != issuer {
		return nil, errors.New("OIDC discovery issuer mismatch")
	}
	for _, endpoint := range []string{document.AuthorizationEndpoint, document.TokenEndpoint, document.JWKSURI} {
		if err := validateRemoteURL(endpoint); err != nil {
			return nil, err
		}
	}
	var keys jwksDocument
	if err := c.getJSON(ctx, document.JWKSURI, &keys); err != nil {
		return nil, err
	}
	if _, err := keys.rsaKeys(); err != nil {
		return nil, err
	}
	return &domain.MetadataPreview{
		Issuer: issuer, AuthorizationEndpoint: document.AuthorizationEndpoint,
		TokenEndpoint: document.TokenEndpoint, JWKSURI: document.JWKSURI,
	}, nil
}

// TestConnection reports, without performing any authorization_code flow, whether the
// connection's fixed trust-source endpoints are reachable and its secret_reference resolves.
// It never returns the secret value, tokens, or certificate bodies; an empty slice means success.
func (c Client) TestConnection(ctx context.Context, connection domain.IdentityProviderConnection) []string {
	if connection.Protocol != domain.ProtocolOIDC {
		return []string{"connection is not OIDC"}
	}
	client := c.HTTPClient
	if client == nil {
		client = safeHTTPClient()
	}
	var failures []string
	for _, endpoint := range []struct{ label, url string }{
		{"authorization endpoint", connection.AuthorizationEndpoint},
		{"token endpoint", connection.TokenEndpoint},
		{"JWKS URI", connection.JWKSURI},
	} {
		if err := reachable(ctx, client, endpoint.url); err != nil {
			failures = append(failures, fmt.Sprintf("%s is unreachable", endpoint.label))
		}
	}
	if _, err := c.resolveSecret(ctx, connection.SecretReference); err != nil {
		failures = append(failures, "secret reference cannot be resolved")
	}
	return failures
}

// resolveSecret returns the client secret ready to use. A value using the legacy "env:" scheme
// is resolved via SecretResolver; anything else (including "") is already the real secret value
// — the repository dual-reads legacy env: references and decrypted ciphertext into the same
// SecretReference field, so only the legacy case still needs resolution here.
func (c Client) resolveSecret(ctx context.Context, reference string) (string, error) {
	if reference == "" || !strings.HasPrefix(reference, "env:") {
		return reference, nil
	}
	if c.SecretResolver == nil {
		return "", errors.New("OIDC secret resolver unavailable")
	}
	return c.SecretResolver.Resolve(ctx, reference)
}

func reachable(ctx context.Context, client *http.Client, rawURL string) error {
	if err := validateRemoteURL(rawURL); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, http.NoBody)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	return nil
}

func AuthorizationURL(connection domain.IdentityProviderConnection, attempt domain.FederatedLoginAttempt, redirectURI string) (string, error) {
	if !connection.Active() || connection.Protocol != domain.ProtocolOIDC {
		return "", errors.New("OIDC connection is not active")
	}
	if attempt.State == "" || attempt.Nonce == "" || len(attempt.PKCEVerifier) < 43 {
		return "", errors.New("state, nonce, and PKCE verifier are required")
	}
	if err := validateRemoteURL(connection.AuthorizationEndpoint); err != nil {
		return "", err
	}
	challenge := sha256.Sum256([]byte(attempt.PKCEVerifier))
	values := url.Values{
		"client_id":             {connection.ClientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {attempt.State},
		"nonce":                 {attempt.Nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	return connection.AuthorizationEndpoint + "?" + values.Encode(), nil
}

func (c Client) ExchangeAndValidate(
	ctx context.Context,
	connection domain.IdentityProviderConnection,
	attempt domain.FederatedLoginAttempt,
	code, redirectURI string,
	now time.Time,
) (domain.NormalizedClaims, error) {
	var empty domain.NormalizedClaims
	if !connection.Active() || connection.Protocol != domain.ProtocolOIDC || code == "" {
		return empty, errors.New("invalid OIDC callback")
	}
	secret, err := c.resolveSecret(ctx, connection.SecretReference)
	if err != nil {
		return empty, err
	}
	values := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {connection.ClientID},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {attempt.PKCEVerifier},
	}
	if secret != "" {
		values.Set("client_secret", secret)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, connection.TokenEndpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return empty, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var tokenResponse struct {
		IDToken string `json:"id_token"`
	}
	if err := c.doJSON(request, &tokenResponse); err != nil {
		return empty, err
	}
	if tokenResponse.IDToken == "" {
		return empty, errors.New("OIDC token response has no ID token")
	}
	keys := jwksDocument{}
	if err := c.getJSON(ctx, connection.JWKSURI, &keys); err != nil {
		return empty, err
	}
	keySet, err := keys.rsaKeys()
	if err != nil {
		return empty, err
	}
	claims, err := verifyUpstreamIDToken(tokenResponse.IDToken, keySet, connection, attempt, now)
	if err != nil {
		return empty, err
	}
	return normalizeClaims(connection.ClaimMapping, claims)
}

// NormalizeClaims applies a saved provider mapping to already validated claims.
func NormalizeClaims(mapping domain.ClaimMapping, claims map[string]any) (domain.NormalizedClaims, error) {
	out := domain.NormalizedClaims{
		Subject:  stringClaim(claims, mapping.Subject),
		Username: stringClaim(claims, mapping.Username),
		Email:    stringClaim(claims, mapping.Email),
		Name:     stringClaim(claims, mapping.Name),
	}
	if mapping.EmailVerified != "" {
		out.EmailVerified, _ = claims[mapping.EmailVerified].(bool)
	}
	if out.Subject == "" || out.Username == "" {
		return domain.NormalizedClaims{}, errors.New("required mapped claims are missing")
	}
	return out, nil
}

func normalizeClaims(mapping domain.ClaimMapping, claims map[string]any) (domain.NormalizedClaims, error) {
	return NormalizeClaims(mapping, claims)
}

func stringClaim(claims map[string]any, name string) string {
	if name == "" {
		return ""
	}
	value, _ := claims[name].(string)
	return strings.TrimSpace(value)
}

func (c Client) getJSON(ctx context.Context, rawURL string, output any) error {
	if err := validateRemoteURL(rawURL); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return err
	}
	return c.doJSON(request, output)
}

func (c Client) doJSON(request *http.Request, output any) error {
	client := c.HTTPClient
	if client == nil {
		client = safeHTTPClient()
	}
	response, err := client.Do(request) //nolint:gosec // G704: rawURL is HTTPS-only and the production transport rejects private/local DNS results.
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("upstream returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes+1))
	if err := decoder.Decode(output); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("upstream response exceeds allowed JSON value")
	}
	return nil
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KTY string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type rsaSigningKey struct {
	kid       string
	publicKey *rsa.PublicKey
}

func (d jwksDocument) rsaKeys() ([]rsaSigningKey, error) {
	var out []rsaSigningKey
	for _, key := range d.Keys {
		if key.KTY != "RSA" || key.Alg != "RS256" || (key.Use != "" && key.Use != "sig") {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			return nil, err
		}
		e, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			return nil, err
		}
		exponent := new(big.Int).SetBytes(e)
		if exponent.Sign() <= 0 || !exponent.IsInt64() {
			return nil, errors.New("invalid RSA exponent")
		}
		out = append(out, rsaSigningKey{
			kid: key.Kid, publicKey: &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent.Int64())},
		})
	}
	if len(out) == 0 {
		return nil, errors.New("JWKS has no supported signing key")
	}
	return out, nil
}

func validateRemoteURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return errors.New("upstream URL must be absolute HTTPS without userinfo or fragment")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && unsafeIP(ip) {
		return errors.New("upstream URL resolves to a private or local address")
	}
	return nil
}

func safeHTTPClient() *http.Client {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		baseTransport = &http.Transport{}
	}
	transport := baseTransport.Clone()
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, errors.New("upstream host did not resolve")
		}
		if slices.ContainsFunc(ips, unsafeIP) {
			return nil, errors.New("upstream resolved to a private or local address")
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
	return &http.Client{
		Timeout: 10 * time.Second, Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errors.New("upstream redirects are not allowed")
		},
	}
}

func unsafeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

// verifyUpstreamIDToken は上流 IdP の ID Token を検証して claims を返す。
//
// 署名アルゴリズムは RS256 に限り、kid が JWKS の鍵と一致するか、kid の省略時には署名用の
// 鍵が一つだけであることを要求する。issuer、audience、有効期限、nonce も接続と試行に
// 一致することを要求する。ここを緩めると、上流を名乗る
// 任意の相手が任意のローカルユーザーとして認証を通せる (REQ-AUTHENTICATION-001)。
//
// HTTP からは切り離してあり、鍵集合と時刻を引数で受け取る。
func verifyUpstreamIDToken(
	idToken string,
	keySet []rsaSigningKey,
	connection domain.IdentityProviderConnection,
	attempt domain.FederatedLoginAttempt,
	now time.Time,
) (map[string]any, error) {
	claims, err := tokens_jose.VerifyJWT(idToken, "RS256", func(header map[string]any) (crypto.PublicKey, error) {
		if _, present := header["kid"]; !present {
			// Core §10.1 の省略を認めるが、複数鍵を順に試して曖昧な選択を隠さない。
			if len(keySet) == 1 {
				return keySet[0].publicKey, nil
			}
			return nil, errors.New("ID token signing key is ambiguous or missing")
		}
		kid, ok := header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("invalid ID token key identifier")
		}
		for _, key := range keySet {
			if key.kid == kid {
				return key.publicKey, nil
			}
		}
		return nil, errors.New("ID token key not found")
	}, tokens_jose.JWTExpectation{
		Issuer: connection.Issuer, Audience: connection.ClientID,
		RequireExpiration: true, Now: normalizedNow(now),
	})
	if err != nil {
		return nil, fmt.Errorf("validate ID token: %w", err)
	}
	if nonce, _ := claims["nonce"].(string); nonce == "" || nonce != attempt.Nonce {
		return nil, errors.New("ID token nonce mismatch")
	}
	return claims, nil
}

func normalizedNow(now time.Time) time.Time {
	if now.IsZero() {
		return time.Now().UTC()
	}
	return now.UTC()
}
