// Package exchange_http は、評価が発行した許可で下流宛のトークンを得て、ゲートウェイの鍵で証明を作る。
package exchange_http

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	maxResponseBytes = 1 << 20
	grantType        = "urn:ietf:params:oauth:grant-type:token-exchange"
	accessTokenType  = "urn:ietf:params:oauth:token-type:access_token"
)

// Credential は上流へ渡す資格情報である。ログへ出さない。
type Credential struct {
	Token string
	Proof string
}

// Client は機密クライアントの資格情報とゲートウェイの鍵で交換する。
type Client struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Resource     string
	Key          *ecdsa.PrivateKey
	HTTPClient   *http.Client
}

// Exchange は提示されたトークンを下流宛のトークンへ交換する。どの失敗でも資格情報を返さない。
func (c Client) Exchange(ctx context.Context, subjectToken, grant, correlationID string) (Credential, error) {
	proof, err := c.proof(c.TokenURL, "")
	if err != nil {
		return Credential{}, err
	}
	form := url.Values{
		"grant_type": {grantType}, "subject_token": {subjectToken}, "subject_token_type": {accessTokenType},
		"resource": {c.Resource}, "token_exchange_grant": {grant},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Credential{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Request-ID", correlationID)
	request.Header.Set("DPoP", proof)
	request.SetBasicAuth(c.ClientID, c.ClientSecret)
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return Credential{}, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Credential{}, fmt.Errorf("token exchange returned status %d", response.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&body); err != nil {
		return Credential{}, fmt.Errorf("token exchange response: %w", err)
	}
	if body.AccessToken == "" || !strings.EqualFold(body.TokenType, "DPoP") {
		return Credential{}, fmt.Errorf("token exchange returned token type %q", body.TokenType)
	}
	forward, err := c.proof(c.Resource, body.AccessToken)
	if err != nil {
		return Credential{}, err
	}
	return Credential{Token: body.AccessToken, Proof: forward}, nil
}

func (c Client) proof(htu, accessToken string) (string, error) {
	claims := jwt.MapClaims{"jti": rand.Text(), "htm": http.MethodPost, "htu": htu, "iat": time.Now().Unix()}
	if accessToken != "" {
		sum := sha256.Sum256([]byte(accessToken))
		claims["ath"] = base64.RawURLEncoding.EncodeToString(sum[:])
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	public, err := c.Key.PublicKey.ECDH()
	if err != nil {
		return "", err
	}
	// 非圧縮点は 0x04 の後に同じ長さの x と y が続く。
	point := public.Bytes()[1:]
	token.Header["jwk"] = map[string]string{
		"kty": "EC", "crv": "P-256",
		"x": base64.RawURLEncoding.EncodeToString(point[:len(point)/2]),
		"y": base64.RawURLEncoding.EncodeToString(point[len(point)/2:]),
	}
	return token.SignedString(c.Key)
}
