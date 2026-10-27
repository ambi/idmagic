// Package testing_clientauth は、private_key_jwt で認証するクライアントの鍵とアサーションを
// 作るテスト専用のフィクスチャである。Agent に束縛できるクライアントは鍵の所有を証明する
// 方式に限られる (AIMS00-AGENT-CLIENT-AUTH) ので、Agent のトークンを発行させるテストは
// モジュールを問わずこの資格情報を使う。
package testing_clientauth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/url"
	"testing"
	"time"
)

const (
	keyID         = "test-client-key"
	assertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
)

// PrivateKeyJWT は一つのクライアントの署名鍵である。
type PrivateKeyJWT struct {
	clientID string
	key      *rsa.PrivateKey
}

func NewPrivateKeyJWT(tb testing.TB, clientID string) *PrivateKeyJWT {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatal(err)
	}
	return &PrivateKeyJWT{clientID: clientID, key: key}
}

// JWKS は OAuth2Client.JWKS に登録する公開鍵の JWK Set である。
func (c *PrivateKeyJWT) JWKS() map[string]any {
	return map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": keyID,
		"n": base64.RawURLEncoding.EncodeToString(c.key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(c.key.PublicKey.E)).Bytes()),
	}}}
}

// Assertion は audience 宛ての、毎回異なる jti のクライアントアサーションである。
// 認可サーバーは jti の再利用を拒否するので、要求ごとに作り直す。
func (c *PrivateKeyJWT) Assertion(tb testing.TB, audience string) string {
	tb.Helper()
	now := time.Now().UTC()
	header, err := json.Marshal(map[string]any{"alg": "PS256", "kid": keyID})
	if err != nil {
		tb.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": c.clientID, "sub": c.clientID, "aud": audience, "jti": rand.Text(),
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	if err != nil {
		tb.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, c.key, crypto.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		tb.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// AuthenticateForm は form にクライアント認証の 3 項目を加えて返す。
func (c *PrivateKeyJWT) AuthenticateForm(tb testing.TB, form url.Values, audience string) url.Values {
	tb.Helper()
	form.Set("client_id", c.clientID)
	form.Set("client_assertion_type", assertionType)
	form.Set("client_assertion", c.Assertion(tb, audience))
	return form
}
