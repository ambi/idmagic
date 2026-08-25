package tokens_jose

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fuzzReplayStore はリプレイ判定を対象から外すためのスタブ。常に「初出」として受理する。
type fuzzReplayStore struct{}

func (fuzzReplayStore) RecordIfNew(context.Context, string, int, time.Time) (bool, error) {
	return true, nil
}

// fuzzJOSEKey は署名検証 target 用の固定鍵から JWK を 1 度だけ作る。
func fuzzJOSEKey(tb testing.TB) map[string]any {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatalf("generate key: %v", err)
	}
	jwk := rsaPublicJWK(&key.PublicKey)
	jwk["kid"] = "fuzz-key"
	return jwk
}

// unsignedJWT は署名部を差し替えた JWT を組み立てる。alg 混同の corpus を作るために使う。
func unsignedJWT(tb testing.TB, header, payload map[string]any, signature []byte) string {
	tb.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		tb.Fatalf("marshal header: %v", err)
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		tb.Fatalf("marshal payload: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(pb) + "." +
		base64.RawURLEncoding.EncodeToString(signature)
}

// FuzzVerifyClientAssertion は、こちらが署名していない client_assertion が受理されないことを表明する。
//
// この target は「必ず拒否する」しか言わないので、単体では「常に拒否する実装」も通ってしまう。
// 正当な assertion が受理されることは TestVerifyClientAssertion が別に押さえている。
func FuzzVerifyClientAssertion(f *testing.F) {
	const (
		clientID = "private-client"
		audience = "https://idp.example/token"
	)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	jwk := fuzzJOSEKey(f)

	claims := map[string]any{
		"iss": clientID, "sub": clientID, "aud": audience, "jti": "jti-1",
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	}
	// alg 混同: none、および RSA 公開鍵の modulus を HMAC 鍵に流用した HS256。
	f.Add(unsignedJWT(f, map[string]any{"alg": "none", "kid": "fuzz-key"}, claims, nil))
	modulus, _ := jwk["n"].(string)
	mac := hmac.New(sha256.New, []byte(modulus))
	mac.Write([]byte("signing input"))
	f.Add(unsignedJWT(f, map[string]any{"alg": "HS256", "kid": "fuzz-key"}, claims, mac.Sum(nil)))
	f.Add(unsignedJWT(f, map[string]any{"alg": "PS256", "kid": "fuzz-key"}, claims, []byte("not a signature")))
	f.Add("")
	f.Add("a.b.c")

	//nolint:unparam // keysFn は VerifyClientAssertion の引数の形に合わせる必要がある。
	keysFn := func(context.Context, string) ([]map[string]any, error) {
		return []map[string]any{jwk}, nil
	}

	f.Fuzz(func(t *testing.T, assertion string) {
		if len(assertion) > 64*1024 {
			return
		}
		result, err := VerifyClientAssertion(
			context.Background(), assertion, clientID, []string{audience},
			keysFn, fuzzReplayStore{}, now, nil,
		)
		if err == nil {
			t.Fatalf("accepted a client_assertion the test never signed: %q (result=%+v)", assertion, result)
		}
		if result != nil {
			t.Fatalf("VerifyClientAssertion returned %+v together with an error", result)
		}
	})
}

// FuzzVerifyDPoP は、こちらが署名していない DPoP 証明が受理されないことを表明する。
// 正当な証明の受理は TestVerifyDPoPAcceptsValidProof が押さえている。
func FuzzVerifyDPoP(f *testing.F) {
	const (
		method      = "POST"
		targetURI   = "https://idp.example/resource"
		accessToken = "access-token"
	)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	f.Add("", method, targetURI)
	f.Add("a.b.c", method, targetURI)
	f.Add(unsignedJWT(f,
		map[string]any{"alg": "none", "typ": "dpop+jwt"},
		map[string]any{"htm": method, "htu": targetURI, "jti": "1", "iat": now.Unix()},
		nil,
	), method, targetURI)

	f.Fuzz(func(t *testing.T, proof, htm, htu string) {
		if len(proof) > 64*1024 || len(htm) > 1024 || len(htu) > 4096 {
			return
		}
		result, err := VerifyDPoPForResource(
			context.Background(), proof, htm, htu, accessToken, fuzzReplayStore{}, now,
		)
		if err == nil {
			t.Fatalf("accepted a DPoP proof the test never signed: %q (result=%+v)", proof, result)
		}
		if result != nil {
			t.Fatalf("VerifyDPoPForResource returned %+v together with an error", result)
		}
	})
}

// FuzzVerifySecurityEventToken は、こちらが署名していない SET が受理されないことを表明する。
func FuzzVerifySecurityEventToken(f *testing.F) {
	const issuer = "https://transmitter.example"
	audience := []string{"https://idmagic.example"}
	jwk := fuzzJOSEKey(f)

	f.Add("")
	f.Add("a.b.c")
	f.Add(unsignedJWT(f,
		map[string]any{"alg": "none", "kid": "fuzz-key"},
		map[string]any{"iss": issuer, "aud": audience[0], "jti": "1"},
		nil,
	))

	f.Fuzz(func(t *testing.T, token string) {
		if len(token) > 64*1024 {
			return
		}
		claims, err := VerifySecurityEventToken(token, []map[string]any{jwk}, issuer, audience)
		if err == nil {
			t.Fatalf("accepted a security event token the test never signed: %q (claims=%+v)", token, claims)
		}
		if claims != nil {
			t.Fatalf("VerifySecurityEventToken returned %+v together with an error", claims)
		}
	})
}

// FuzzInlineJWKs は、インライン JWKS の受理が空を通さないことを表明する。
// 空の鍵集合を成功として返すと、呼び出し側は「鍵が 1 つも一致しない」を検証成功と取り違える。
func FuzzInlineJWKs(f *testing.F) {
	f.Add(`{"keys":[{"kty":"RSA","kid":"a","n":"AQ","e":"AQAB"}]}`)
	f.Add(`{"keys":[]}`)
	f.Add(`{"keys":"not-an-array"}`)
	f.Add(`{}`)
	f.Add(`{"keys":[null]}`)

	f.Fuzz(func(t *testing.T, document string) {
		if len(document) > 64*1024 {
			return
		}
		var jwks map[string]any
		if json.Unmarshal([]byte(document), &jwks) != nil {
			return
		}
		keys, err := InlineJWKs(jwks)
		if err != nil {
			if keys != nil {
				t.Fatalf("InlineJWKs returned %d keys together with an error", len(keys))
			}
			return
		}
		if len(keys) == 0 {
			t.Fatalf("InlineJWKs accepted an empty key set from %q", document)
		}
	})
}

// FuzzValidateJWKSURI は SSRF の拒否コントロールを表明する。
//
// 受理した文字列を改めて解析したとき、https であり、ホストを持ち、userinfo も fragment も
// 持たないことが成り立たなければならない。生文字列に対する接頭辞判定へ退行すると、
// "https://user@internal/" や "https:/\\/evil" のような入力で破れる。
func FuzzValidateJWKSURI(f *testing.F) {
	f.Add("https://idp.example/jwks")
	f.Add("http://idp.example/jwks")
	f.Add("https://user:pass@internal.example/jwks")
	f.Add("https://idp.example/jwks#fragment")
	f.Add("HTTPS://idp.example/jwks")
	f.Add("https://")
	f.Add("https:/\\/evil.example/jwks")
	f.Add("https://[::1]/jwks")

	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 8192 {
			return
		}
		if err := ValidateJWKSURI(raw); err != nil {
			return
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("accepted a jwks_uri that does not parse: %q", raw)
		}
		if parsed.Scheme != "https" {
			t.Fatalf("accepted a non-https jwks_uri: %q (scheme=%q)", raw, parsed.Scheme)
		}
		if parsed.Hostname() == "" {
			t.Fatalf("accepted a jwks_uri without a host: %q", raw)
		}
		if parsed.User != nil {
			t.Fatalf("accepted a jwks_uri carrying userinfo: %q", raw)
		}
		if strings.Contains(raw, "#") && parsed.Fragment != "" {
			t.Fatalf("accepted a jwks_uri carrying a fragment: %q", raw)
		}
	})
}
