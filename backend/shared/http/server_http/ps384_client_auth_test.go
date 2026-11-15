package server_http_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

//spec:covers REQ-OAUTH2-028, REQ-OAUTH2-014, REQ-OAUTH2-030, RFC7523-CLIENT-ASSERTION: 事前登録した通常のクライアントは CIMD も Poppy 設定も使わず PS384 で認証できる。
//verify:operation Token: 登録済みクライアントは PS384 で認証してトークンを取得する。
func TestRegisteredClientAuthenticatesWithPS384(t *testing.T) {
	f := newPublicSetupPath(t)
	clientID, key := f.keyClient(t, "read", "client_credentials")
	now := time.Now().UTC()
	header, err := json.Marshal(map[string]any{"alg": "PS384", "kid": "setup-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": clientID, "sub": clientID, "aud": stack.Issuer + "/realms/default",
		"jti": rand.Text(), "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha512.Sum384([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA384, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{
		"grant_type": {"client_credentials"}, "client_id": {clientID}, "scope": {"read"},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {input + "." + base64.RawURLEncoding.EncodeToString(signature)},
	}
	req := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, req)
	var tokens map[string]any
	if rec.Code != http.StatusOK {
		t.Fatalf("PS384 認証: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	setupString(t, tokens, "access_token")
	for _, path := range []string{"/.well-known/openid-configuration", "/.well-known/oauth-authorization-server"} {
		doc := f.request(t, http.MethodGet, path, nil, nil, http.StatusOK)
		for _, field := range []string{"token_endpoint_auth_signing_alg_values_supported", "id_token_signing_alg_values_supported", "dpop_signing_alg_values_supported"} {
			raw, ok := doc[field].([]any)
			if !ok {
				t.Fatalf("%s の %s がない", path, field)
			}
			if got, want := slices.Contains(raw, any("PS384")), field == "token_endpoint_auth_signing_alg_values_supported"; got != want {
				t.Fatalf("%s の %s=%v", path, field, raw)
			}
		}
	}
}
