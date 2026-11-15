package domain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"maps"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-OAUTH2-017, CIMD02-PRIVATE-KEY-JWT: 鍵情報を取り込み、PKCE を保持した confidential クライアントを解決する。
func TestConfidentialCIMDRequiresKeysAndPKCE(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]any{
		"kty": "EC", "crv": "P-256",
		"x": base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))),
		"y": base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32))),
	}
	const clientID = "https://app.example/client.json"
	document := func(extra map[string]any) []byte {
		doc := map[string]any{
			"client_id": clientID, "client_name": "秘密鍵認証のクライアント",
			"redirect_uris":              []string{"https://app.example/callback"},
			"token_endpoint_auth_method": "private_key_jwt", "require_pkce": false,
		}
		maps.Copy(doc, extra)
		raw, marshalErr := json.Marshal(doc)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		return raw
	}
	for name, source := range map[string]map[string]any{
		"インライン": {"jwks": map[string]any{"keys": []any{jwk}}},
		"URL":   {"jwks_uri": "https://app.example/jwks.json"},
	} {
		t.Run(name, func(t *testing.T) {
			client, parseErr := ParseClientIDMetadataDocument(document(source), clientID, cimdTestNow)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if client.ClientType != spec.ClientConfidential || client.TokenEndpointAuthMethod != AuthMethodPrivateKeyJwt || !client.RequirePKCE {
				t.Fatalf("秘密鍵認証または PKCE が失われた: %+v", client)
			}
			if client.ApplicationID != "" || len(client.GrantTypes) != 1 || client.GrantTypes[0] != spec.GrantAuthorizationCode {
				t.Fatalf("文書から権限が追加された: %+v", client)
			}
			if client.JWKS == nil && client.JwksURI == nil {
				t.Fatal("鍵情報が失われた")
			}
		})
	}
	for name, source := range map[string]map[string]any{
		"鍵なし":         {},
		"空の鍵":         {"jwks": map[string]any{"keys": []any{}}},
		"鍵の二重指定":      {"jwks": map[string]any{"keys": []any{jwk}}, "jwks_uri": "https://app.example/jwks"},
		"null の URI":  {"jwks_uri": nil},
		"HTTP":        {"jwks_uri": "http://app.example/jwks"},
		"userinfo":    {"jwks_uri": "https://secret@app.example/jwks"},
		"fragment":    {"jwks_uri": "https://app.example/jwks#key"},
		"不正な URI の秘密": {"jwks_uri": "https://user:sensitive-password@app.example/%zz"},
		"秘密鍵":         {"jwks": map[string]any{"keys": []any{map[string]any{"kty": "EC", "d": "secret"}}}},
		"共有秘密":        {"jwks_uri": "https://app.example/jwks", "client_secret": nil},
		"自律発行":        {"jwks_uri": "https://app.example/jwks", "grant_types": []string{"client_credentials"}},
		"不正グラントの混在":   {"jwks_uri": "https://app.example/jwks", "grant_types": []string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:token-exchange"}},
	} {
		t.Run(name, func(t *testing.T) {
			client, parseErr := ParseClientIDMetadataDocument(document(source), clientID, cimdTestNow)
			if parseErr == nil || client != nil {
				t.Fatalf("拒否すべき文書を受理した: client=%+v err=%v", client, parseErr)
			}
			if strings.Contains(parseErr.Error(), "sensitive-password") {
				t.Fatal("拒否理由が URI の秘密を含む")
			}
		})
	}
	client, err := ParseClientIDMetadataDocument(document(map[string]any{
		"jwks_uri": "https://app.example/jwks", "grant_types": []string{"authorization_code", "refresh_token"},
	}), clientID, cimdTestNow)
	if err != nil || len(client.GrantTypes) != 2 {
		t.Fatalf("明示した refresh の許可を失った: client=%+v err=%v", client, err)
	}
}
