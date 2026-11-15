package cimd_http_test

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"io"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2"
	cimdhttp "github.com/ambi/idmagic/backend/oauth2/internal/client/cimd_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

//spec:covers REQ-OAUTH2-017, REQ-OAUTH2-028, REQ-OAUTH2-005, REQ-OAUTH2-009, REQ-OAUTH2-006, REQ-OAUTH2-019, CIMD02-PRIVATE-KEY-JWT:
//verify:operation Authorize: CIMD と PKCE、本人の同意を通す。
//verify:operation Token: 未登録のクライアントを PS384 で認証する。
//verify:operation Revoke: 同じ CIMD の認証で refresh token を失効する。
//verify:operation PushAuthorizationRequest: CIMD のクライアント認証と PKCE を含む PAR を実行する。
func TestConfidentialCIMDAuthorizationLifecycle(t *testing.T) {
	const redirect, verifier = "https://client.example/callback", "cimd-pkce-verifier-012345678901234567890123456789"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "cimd-key",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}}
	var clientID string
	var fetches atomic.Int64
	host := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		if r.URL.Path != "/client.json" {
			t.Errorf("取得先のパス=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"client_id": clientID, "client_name": "CIMD の秘密鍵クライアント",
			"redirect_uris": []string{redirect}, "grant_types": []string{"authorization_code", "refresh_token"},
			"scope": "openid profile offline_access", "token_endpoint_auth_method": "private_key_jwt", "jwks": jwks,
		})
	}))
	t.Cleanup(host.Close)
	clientID = host.URL + "/client.json"
	registered := oauth2.NewMemoryModule().ClientRepo
	repository := &cimdhttp.ClientRepositoryWithCIMD{OAuth2ClientRepository: registered, Fetcher: cimdhttp.NewTestFetcher(host.Client())}
	s := stack.New(t, stack.WithAuthorizationCodeFlow(), stack.WithTokenIssuance(), stack.WithAccountApi(),
		stack.WithOAuth2ClientRepository(repository))
	admin := s.AdminSessionCookie(t, "default")
	body, err := json.Marshal(map[string]any{"preferred_username": "cimd-user", "password": "cimd-password-1234567"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default/api/admin/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", stack.Issuer)
	req.Header.Set("X-Csrf-Token", "cimd-csrf")
	req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "cimd-csrf"})
	req.AddCookie(admin)
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("本人の公開 API での準備: status=%d body=%s", rec.Code, rec.Body.String())
	}
	browser := s.Browser(t, "default")
	query := stack.AuthorizationQuery(verifier, map[string]string{
		"client_id": clientID, "redirect_uri": redirect, "scope": "openid profile offline_access",
	})
	withoutPKCE := maps.Clone(query)
	withoutPKCE.Del("code_challenge")
	withoutPKCE.Del("code_challenge_method")
	refused := browser.Authorize(t, withoutPKCE)
	_ = refused.Body.Close()
	if refused.StatusCode != http.StatusBadRequest {
		t.Fatalf("PKCE なしの認可を受理した: %d", refused.StatusCode)
	}
	pushed := cimdSignedForm(t, s, "/par", query, clientID, key, http.StatusCreated)
	requestURI, _ := pushed["request_uri"].(string)
	if requestURI == "" {
		t.Fatalf("PAR の request_uri がない: %v", pushed)
	}
	response := browser.Authorize(t, url.Values{"client_id": {clientID}, "request_uri": {requestURI}})
	_ = response.Body.Close()
	if response.StatusCode != http.StatusFound && response.StatusCode != http.StatusSeeOther {
		t.Fatalf("認可が開始できない: %d", response.StatusCode)
	}
	browser.SignIn(t, "cimd-user", "cimd-password-1234567")
	callback, err := url.Parse(browser.Consent(t, "allow"))
	if err != nil {
		t.Fatal(err)
	}
	code := callback.Query().Get("code")
	if code == "" {
		t.Fatalf("本人の同意からコードを取得できない: %s", callback)
	}
	tokenForm := url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {redirect}, "code_verifier": {verifier},
	}
	// 誤署名を先に送り、認可コードを消費しないことを正しい鍵による成功で観測する。
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	bad := cimdSignedForm(t, s, "/token", tokenForm, clientID, wrongKey, http.StatusUnauthorized)
	if bad["error"] != "invalid_client" {
		t.Fatalf("誤署名の結果=%v", bad)
	}
	tokens := cimdSignedForm(t, s, "/token", tokenForm, clientID, key, http.StatusOK)
	refresh, ok := tokens["refresh_token"].(string)
	access, _ := tokens["access_token"].(string)
	if !ok || refresh == "" || access == "" {
		t.Fatalf("トークンを取得できない: %v", tokens)
	}
	refreshForm := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}}
	cimdSignedForm(t, s, "/token", refreshForm, clientID, wrongKey, http.StatusUnauthorized)
	rotated := cimdSignedForm(t, s, "/token", refreshForm, clientID, key, http.StatusOK)
	newRefresh, _ := rotated["refresh_token"].(string)
	if newRefresh == "" || newRefresh == refresh {
		t.Fatalf("refresh が更新されない: %v", rotated)
	}
	cimdSignedForm(t, s, "/revoke", url.Values{"token": {newRefresh}, "token_type_hint": {"refresh_token"}}, clientID, key, http.StatusOK)
	rejected := cimdSignedForm(t, s, "/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {newRefresh}}, clientID, key, http.StatusBadRequest)
	if rejected["error"] != "invalid_grant" {
		t.Fatalf("失効した refresh の結果=%v", rejected)
	}
	stored, err := registered.FindByID(t.Context(), "default", clientID)
	if err != nil || stored != nil || fetches.Load() != 1 {
		t.Fatalf("未登録とキャッシュの結果: stored=%v err=%v fetches=%d", stored, err, fetches.Load())
	}
}

func cimdSignedForm(t *testing.T, s *stack.Stack, path string, form url.Values, clientID string, key *rsa.PrivateKey, want int) map[string]any {
	t.Helper()
	now := time.Now().UTC()
	header, err := json.Marshal(map[string]any{"alg": "PS384", "kid": "cimd-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": clientID, "sub": clientID,
		"aud": stack.Issuer + "/realms/default", "jti": rand.Text(), "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha512.Sum384([]byte(input))
	sig, err := rsa.SignPSS(rand.Reader, key, crypto.SHA384, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	form.Set("client_id", clientID)
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", input+"."+base64.RawURLEncoding.EncodeToString(sig))
	req := httptest.NewRequest(http.MethodPost, stack.Issuer+"/realms/default"+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, req)
	result := map[string]any{}
	if rec.Body.Len() > 0 {
		raw, readErr := io.ReadAll(rec.Body)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
	}
	if rec.Code != want {
		t.Fatalf("POST %s: status=%d want=%d body=%v", path, rec.Code, want, result)
	}
	return result
}
