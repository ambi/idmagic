package tokens_jose

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"maps"
	"strings"
	"testing"
	"time"
)

func decodeJWTPart(t *testing.T, part string) map[string]any {
	t.Helper()
	decoded, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(decoded, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSignJWTES256ProducesAFixedLengthSignatureTheVerifierAccepts(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]string{"kty": "EC", "crv": "P-256", "x": "x", "y": "y"}
	token, err := SignJWT(key, "ES256",
		map[string]any{"alg": "none", "typ": "dpop+jwt", "jwk": jwk},
		map[string]any{"jti": "j1", "iat": 1700000000})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts", len(parts))
	}
	if err := verifyJWTSignature(parts, "ES256", &key.PublicKey); err != nil {
		t.Fatalf("signature: %v", err)
	}
	header := decodeJWTPart(t, parts[0])
	if header["alg"] != "ES256" || header["typ"] != "dpop+jwt" || len(header) != 3 {
		t.Fatalf("header = %v", header)
	}
	if got, _ := header["jwk"].(map[string]any); got["crv"] != "P-256" || got["x"] != "x" {
		t.Fatalf("jwk header = %v", header["jwk"])
	}
	if claims := decodeJWTPart(t, parts[1]); claims["jti"] != "j1" || claims["iat"] != float64(1700000000) || len(claims) != 2 {
		t.Fatalf("claims = %v", claims)
	}
}

func TestSignJWTRejectsUnsupportedKeysAndAlgorithms(t *testing.T) {
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signErr := errors.New("remote signing unavailable")
	for _, tc := range []struct {
		name string
		key  any
		alg  string
		want error
	}{
		{name: "missing signer", alg: "ES256"},
		{name: "P-384 key for ES256", key: p384, alg: "ES256"},
		{name: "unsupported algorithm", key: p256, alg: "ES384"},
		{name: "signing failure", key: unavailableSigner{Signer: p256, err: signErr}, alg: "ES256", want: signErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token, err := SignJWT(tc.key, tc.alg, nil, map[string]any{"sub": "s"})
			if token != "" || err == nil {
				t.Fatalf("token=%q error=%v, want no token and an error", token, err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
		})
	}
}

func TestVerifyJWT(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	base := map[string]any{"iss": "https://issuer.example", "aud": "client", "sub": "s", "exp": now.Unix() + 60}
	expect := JWTExpectation{Issuer: "https://issuer.example", Audience: "client", RequireExpiration: true, Now: now}
	fixedKey := func(map[string]any) (crypto.PublicKey, error) { return &key.PublicKey, nil } //nolint:unparam // VerifyJWT の keyFor の形に合わせる。鍵の選択の失敗は別の行が確かめる。
	sign := func(t *testing.T, alg string, header, claims map[string]any) string {
		t.Helper()
		token, err := SignJWT(key, alg, header, claims)
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	t.Run("accepts and passes the header to the key selection", func(t *testing.T) {
		var seen map[string]any
		claims, err := VerifyJWT(sign(t, "RS256", map[string]any{"kid": "k1"}, base), "RS256",
			func(header map[string]any) (crypto.PublicKey, error) { seen = header; return &key.PublicKey, nil }, expect)
		if err != nil || claims["sub"] != "s" {
			t.Fatalf("claims=%v err=%v", claims, err)
		}
		if seen["kid"] != "k1" || seen["alg"] != "RS256" {
			t.Fatalf("header = %v", seen)
		}
	})

	t.Run("skips unset expectations", func(t *testing.T) {
		claims := map[string]any{"sub": "s"}
		if _, err := VerifyJWT(sign(t, "PS256", nil, claims), "PS256", fixedKey, JWTExpectation{Now: now}); err != nil {
			t.Fatalf("err = %v", err)
		}
	})

	keyErr := errors.New("no key")
	valid := sign(t, "RS256", nil, base)
	parts := strings.Split(valid, ".")
	// 保護ヘッダーが別の alg を名乗り、署名は RS256 として正しいトークン。
	mislabeledInput := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS512"}`)) + "." + parts[1]
	mislabeledDigest := sha256.Sum256([]byte(mislabeledInput))
	mislabeledSig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, mislabeledDigest[:])
	if err != nil {
		t.Fatal(err)
	}
	mislabeled := mislabeledInput + "." + base64.RawURLEncoding.EncodeToString(mislabeledSig)
	for _, tc := range []struct {
		name  string
		token string
		alg   string
		key   func(map[string]any) (crypto.PublicKey, error)
	}{
		{name: "two segments", token: parts[0] + "." + parts[1]},
		{name: "four segments", token: valid + ".x"},
		{name: "header is not base64url", token: "!" + valid},
		{name: "header is not an object", token: base64.RawURLEncoding.EncodeToString([]byte("[]")) + "." + parts[1] + "." + parts[2]},
		{name: "claims are null", token: sign(t, "RS256", nil, nil)},
		{name: "other algorithm", token: valid, alg: "PS256"},
		{name: "header names another algorithm", token: mislabeled},
		{name: "key selection fails", token: valid, key: func(map[string]any) (crypto.PublicKey, error) { return nil, keyErr }},
		{name: "tampered claims", token: parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"t"}`)) + "." + parts[2]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			alg := tc.alg
			if alg == "" {
				alg = "RS256"
			}
			selectKey := tc.key
			if selectKey == nil {
				selectKey = fixedKey
			}
			if claims, err := VerifyJWT(tc.token, alg, selectKey, expect); err == nil || claims != nil {
				t.Fatalf("claims=%v err=%v, want rejection", claims, err)
			}
		})
	}

	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		accept bool
	}{
		{name: "missing required exp", mutate: func(c map[string]any) { delete(c, "exp") }},
		{name: "exp equals now", mutate: func(c map[string]any) { c["exp"] = now.Unix() }},
		{name: "exp fraction is truncated", mutate: func(c map[string]any) { c["exp"] = float64(now.Unix()) + 0.5 }},
		{name: "exp zero", mutate: func(c map[string]any) { c["exp"] = 0 }},
		{name: "nbf equals now", mutate: func(c map[string]any) { c["nbf"] = now.Unix() }, accept: true},
		{name: "nbf in the future", mutate: func(c map[string]any) { c["nbf"] = now.Unix() + 1 }},
		{name: "iat is not checked", mutate: func(c map[string]any) { c["iat"] = "later" }, accept: true},
		{name: "issuer mismatch", mutate: func(c map[string]any) { c["iss"] = "https://other.example" }},
		{name: "empty audience member", mutate: func(c map[string]any) { c["aud"] = []any{""} }},
		{name: "audience of another type", mutate: func(c map[string]any) { c["aud"] = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := maps.Clone(base)
			tc.mutate(claims)
			got, err := VerifyJWT(sign(t, "RS256", nil, claims), "RS256", fixedKey, expect)
			if tc.accept != (err == nil) || tc.accept != (got != nil) {
				t.Fatalf("claims=%v err=%v, accept=%v", got, err, tc.accept)
			}
		})
	}

	t.Run("exp is still checked when not required", func(t *testing.T) {
		claims := maps.Clone(base)
		claims["exp"] = now.Unix() - 1
		if _, err := VerifyJWT(sign(t, "RS256", nil, claims), "RS256", fixedKey, JWTExpectation{Now: now}); err == nil {
			t.Fatal("accepted an expired JWT")
		}
	})
}
