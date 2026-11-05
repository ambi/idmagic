package protocol_oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

//spec:covers REQ-AUTHENTICATION-001, OIDC-CORE-ID-TOKEN-KID-OPTIONAL: 署名用 RSA 鍵の個数と kid の一致を判定し、省略を受け入れても署名とクレームの検証を維持する。
func TestVerifyUpstreamIDTokenSelectsOnlyAnUnambiguousSigningKey(t *testing.T) {
	connection, attempt, key, _, now := fuzzIDTokenFixture(t)
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicJWK := func(key *rsa.PrivateKey, kid, use string) jwk {
		return jwk{
			KTY: "RSA", Kid: kid, Alg: "RS256",
			Use: use,
			N:   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}
	}
	signing := publicJWK(key, "signing", "sig")
	encryption := publicJWK(other, "encryption", "enc")
	second := publicJWK(other, "second", "sig")
	unnamed := publicJWK(key, "", "sig")
	unspecifiedUse := publicJWK(key, "signing", "")
	encryptionWithSigningMaterial := publicJWK(key, "encryption", "enc")
	for _, tc := range []struct {
		name   string
		kid    any
		keys   []jwk
		mutate func(jwt.MapClaims)
		accept bool
	}{
		{name: "single signing key", keys: []jwk{signing}, accept: true},
		{name: "unspecified key use", keys: []jwk{unspecifiedUse}, accept: true},
		{name: "single unnamed key", keys: []jwk{unnamed}, accept: true},
		{name: "encryption key is excluded", keys: []jwk{signing, encryption}, accept: true},
		{name: "encryption key first", keys: []jwk{encryption, signing}, accept: true},
		{name: "no key"},
		{name: "only encryption", keys: []jwk{encryption}},
		{name: "encryption with matching signature", keys: []jwk{encryptionWithSigningMaterial}},
		{name: "multiple signing keys", keys: []jwk{signing, second}},
		{name: "unnamed key among multiple", keys: []jwk{unnamed, second}},
		{name: "duplicate key identifiers", keys: []jwk{signing, signing}},
		{name: "duplicate unnamed keys", keys: []jwk{unnamed, unnamed}},
		{name: "explicit key", kid: "signing", keys: []jwk{signing}, accept: true},
		{name: "explicit key among multiple", kid: "signing", keys: []jwk{second, signing}, accept: true},
		{name: "unknown identifier", kid: "unknown", keys: []jwk{signing}},
		{name: "encryption identifier", kid: "encryption", keys: []jwk{signing, encryption}},
		{name: "encryption identifier with matching signature", kid: "encryption", keys: []jwk{signing, encryptionWithSigningMaterial}},
		{name: "non-string identifier", kid: 123, keys: []jwk{signing}},
		{name: "empty identifier", kid: "", keys: []jwk{signing}},
		{name: "empty identifier among multiple", kid: "", keys: []jwk{unnamed, second}},
		{name: "wrong signature", keys: []jwk{second}},
		{name: "wrong issuer", keys: []jwk{signing}, mutate: func(c jwt.MapClaims) { c["iss"] = "https://attacker.example" }},
		{name: "wrong audience", keys: []jwk{signing}, mutate: func(c jwt.MapClaims) { c["aud"] = "other-client" }},
		{name: "wrong nonce", keys: []jwk{signing}, mutate: func(c jwt.MapClaims) { c["nonce"] = "other-nonce" }},
		{name: "expired", keys: []jwk{signing}, mutate: func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Minute).Unix() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := jwt.MapClaims{
				"iss": connection.Issuer, "aud": connection.ClientID, "sub": "upstream-user",
				"nonce": attempt.Nonce, "exp": now.Add(time.Minute).Unix(),
			}
			if tc.mutate != nil {
				tc.mutate(claims)
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			if tc.kid != nil {
				token.Header["kid"] = tc.kid
			}
			signed, err := token.SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			keySet, err := (jwksDocument{Keys: tc.keys}).rsaKeys()
			var got jwt.MapClaims
			if err == nil {
				got, err = verifyUpstreamIDToken(signed, keySet, connection, attempt, now)
			}
			if tc.accept {
				if err != nil || got["sub"] != "upstream-user" {
					t.Fatalf("claims=%v err=%v, want verified upstream-user", got, err)
				}
			} else if err == nil || got != nil {
				t.Fatalf("claims=%v err=%v, want rejection without claims", got, err)
			}
		})
	}
}
