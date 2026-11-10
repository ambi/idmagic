package server_http_test

// JWT-SVID によるクライアント認証 (draft-ietf-oauth-spiffe-client-auth-03) を、製品と同じ
// `Register` の組み立てで確かめる。準備は公開の管理 API だけで行う。OAuth2 のクライアント
// 認証へ WorkloadIdentity の検証が結ばれていなければ、正しい JWT-SVID も拒否される。

import (
	"crypto/rand"
	"crypto/rsa"
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

const (
	spiffeAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-spiffe"
	spiffeWorkloadID    = "spiffe://example.org/ns/prod/sa/worker"
	spiffeRealmIssuer   = stack.Issuer + "/realms/default"
)

type spiffeClientAuthFixture struct {
	*publicSetupPath
	clientID    string
	otherClient string
	svidKey     *rsa.PrivateKey
}

// newSpiffeClientAuthFixture は、`spiffe_jwt` のクライアントを束縛した Agent と、その Agent へ
// ワークロードを対応付けた `spiffe` の信頼設定を公開 API で作る。もう一つの `spiffe_jwt` の
// クライアントは別の Agent に束縛し、対応付けの先と異なるクライアントとして使う。
func newSpiffeClientAuthFixture(t *testing.T) *spiffeClientAuthFixture {
	t.Helper()
	f := &spiffeClientAuthFixture{publicSetupPath: newPublicSetupPath(t)}
	owner := f.user(t, "spiffe-client-owner")
	f.clientID = f.spiffeClient(t)
	agentID := f.agent(t, owner, f.clientID, "autonomous")
	f.otherClient = f.spiffeClient(t)
	f.agent(t, owner, f.otherClient, "autonomous")

	key, jwk, _ := tpDPoPKey(t)
	jwk["kid"], jwk["use"] = "setup-key", "jwt-svid"
	f.svidKey = key
	bundle := f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles", map[string]any{
		"name": "SPIFFE client trust", "profile": "spiffe", "trust_domain": "example.org",
		"accepted_audiences": []string{"idmagic-workload"},
		"jwks":               map[string]any{"spiffe_sequence": 1, "keys": []any{jwk}},
	}, f.admin, http.StatusCreated)
	f.request(t, http.MethodPost, "/api/admin/v1/workload-identity/trust-bundles/"+setupString(t, bundle, "id")+"/bindings",
		map[string]any{"subject_pattern": "spiffe://example.org/ns/prod/sa/*", "agent_id": agentID}, f.admin, http.StatusCreated)
	return f
}

func (f *spiffeClientAuthFixture) spiffeClient(t *testing.T) string {
	t.Helper()
	registered := f.request(t, http.MethodPost, "/api/admin/v1/clients", map[string]any{
		"client_type": "confidential", "grant_types": []string{"client_credentials"}, "scope": "read",
		"token_endpoint_auth_method": "spiffe_jwt",
	}, f.admin, http.StatusCreated)
	client, ok := registered["client"].(map[string]any)
	if !ok {
		t.Fatalf("registration has no client: %v", registered)
	}
	return setupString(t, client, "client_id")
}

// svidClaims は SPIRE が発行する形の JWT-SVID のクレームを返す。SPIRE は `iss` と `jti` を
// 付けないことがあるので、どちらも持たせない。
func spiffeSVIDClaims() map[string]any {
	now := time.Now().UTC()
	return map[string]any{
		"sub": spiffeWorkloadID, "aud": []string{spiffeRealmIssuer},
		"iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
	}
}

func (f *spiffeClientAuthFixture) token(t *testing.T, form url.Values, claims map[string]any, want int) map[string]any {
	t.Helper()
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "read")
	form.Set("client_assertion_type", spiffeAssertionType)
	form.Set("client_assertion", setupJWT(t, f.svidKey, claims))
	return f.post(t, "/token", form, want)
}

func (f *spiffeClientAuthFixture) post(t *testing.T, path string, form url.Values, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, spiffeRealmIssuer+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, req)
	result := map[string]any{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("POST %s: decode response: %v", path, err)
		}
	}
	if rec.Code != want {
		t.Fatalf("POST %s: status=%d body=%v, want %d", path, rec.Code, result, want)
	}
	return result
}

func assertInvalidClient(t *testing.T, name string, body map[string]any) {
	t.Helper()
	if body["error"] != "invalid_client" || body["access_token"] != nil {
		t.Fatalf("%s: body=%v, want invalid_client without a token", name, body)
	}
}

//spec:covers SPIFFECA03-JWT-SVID (partial), REQ-OAUTH2-089, REQ-WORKLOADIDENTITY-017, AIMS00-AGENT-CLIENT-AUTH: 公開 API で登録した `spiffe_jwt` の Agent のクライアントが、`iss` と `jti` のない JWT-SVID だけで `client_credentials` のトークンを得る。`aud` が発行者だけでないもの、信頼設定の audience だけのもの、`iat` のないもの、未登録のトラストドメインのものは 401 invalid_client を受ける。
func TestRegisteredWorkloadAuthenticatesAgentClientWithJWTSVID(t *testing.T) {
	f := newSpiffeClientAuthFixture(t)
	issued := f.token(t, url.Values{"client_id": {f.clientID}}, spiffeSVIDClaims(), http.StatusOK)
	if claims := unverifiedClaims(t, setupString(t, issued, "access_token")); claims["sub"] != f.clientID {
		t.Fatalf("access token subject=%v, want %s", claims["sub"], f.clientID)
	}

	for name, mutate := range map[string]func(map[string]any){
		"aud が信頼設定の audience":    func(c map[string]any) { c["aud"] = []string{"idmagic-workload"} },
		"aud が発行者とほかの値":          func(c map[string]any) { c["aud"] = []string{spiffeRealmIssuer, "https://other.example"} },
		"aud がトークンエンドポイント":       func(c map[string]any) { c["aud"] = spiffeRealmIssuer + "/token" },
		"iat がない":                func(c map[string]any) { delete(c, "iat") },
		"exp が過去":                func(c map[string]any) { c["exp"] = time.Now().Add(-10 * time.Minute).Unix() },
		"sub が未登録のトラストドメイン":      func(c map[string]any) { c["sub"] = "spiffe://other.example/ns/prod/sa/worker" },
		"sub が関連付けに一致しない":        func(c map[string]any) { c["sub"] = "spiffe://example.org/ns/dev/sa/worker" },
		"sub が SPIFFE ID でない":    func(c map[string]any) { c["sub"] = "worker" },
		"iss だけが別の信頼設定を指しても選ばない": func(c map[string]any) { c["iss"] = "https://workloads.example" },
	} {
		t.Run(name+"は拒否する", func(t *testing.T) {
			claims := spiffeSVIDClaims()
			mutate(claims)
			want := http.StatusUnauthorized
			if name == "iss だけが別の信頼設定を指しても選ばない" {
				// SPIFFE の信頼設定に発行者を登録していないので `iss` は確かめず、トラストドメインで選ぶ。
				want = http.StatusOK
			}
			body := f.token(t, url.Values{"client_id": {f.clientID}}, claims, want)
			if want == http.StatusUnauthorized {
				assertInvalidClient(t, name, body)
			}
		})
	}
}

//spec:covers SPIFFECA03-CLIENT-ID (partial), REQ-WORKLOADIDENTITY-017: 関連付けの Agent に束縛していない `spiffe_jwt` のクライアント、`client_id` の省略（SPIFFE ID と等しい `client_id` がない）、`spiffe_jwt` でないクライアントは 401 invalid_client を受ける。
func TestJWTSVIDAuthenticatesOnlyClientsOfTheMappedAgent(t *testing.T) {
	f := newSpiffeClientAuthFixture(t)
	keyClient, _ := f.keyClient(t, "read", "client_credentials")
	for name, form := range map[string]url.Values{
		"別の Agent のクライアント":        {"client_id": {f.otherClient}},
		"client_id の省略":           {},
		"private_key_jwt のクライアント": {"client_id": {keyClient}},
		"未登録の client_id":          {"client_id": {"unknown-client"}},
	} {
		t.Run(name+"は拒否する", func(t *testing.T) {
			assertInvalidClient(t, name, f.token(t, form, spiffeSVIDClaims(), http.StatusUnauthorized))
		})
	}
}

//spec:covers SPIFFECA03-JTI (optional): `jti` を持つ JWT-SVID は一度だけ受け付け、`jti` のない同じ JWT-SVID は繰り返し受け付ける。
func TestJWTSVIDReplayIsRejectedOnlyWithJTI(t *testing.T) {
	f := newSpiffeClientAuthFixture(t)
	withoutJTI := spiffeSVIDClaims()
	assertion := setupJWT(t, f.svidKey, withoutJTI)
	for range 2 {
		f.post(t, "/token", url.Values{
			"grant_type": {"client_credentials"}, "scope": {"read"}, "client_id": {f.clientID},
			"client_assertion_type": {spiffeAssertionType}, "client_assertion": {assertion},
		}, http.StatusOK)
	}
	withJTI := spiffeSVIDClaims()
	withJTI["jti"] = rand.Text()
	f.token(t, url.Values{"client_id": {f.clientID}}, withJTI, http.StatusOK)
	assertInvalidClient(t, "jti の再利用", f.token(t, url.Values{"client_id": {f.clientID}}, withJTI, http.StatusUnauthorized))
}

//spec:covers SPIFFECA03-METADATA, SPIFFECA03-X509-SVID (excluded), SPIFFECA03-WIT-SVID (excluded): メタデータは `spiffe_jwt` を三つの認証方式の一覧に載せ、`spiffe_x509` と `spiffe_wit` を載せない。イントロスペクションも同じ JWT-SVID で認証できる。
func TestMetadataAdvertisesSpiffeJWTOnly(t *testing.T) {
	f := newSpiffeClientAuthFixture(t)
	metadata := f.request(t, http.MethodGet, "/.well-known/oauth-authorization-server", nil, nil, http.StatusOK)
	for _, field := range []string{
		"token_endpoint_auth_methods_supported", "introspection_endpoint_auth_methods_supported",
		"revocation_endpoint_auth_methods_supported",
	} {
		methods := stringsOf(t, metadata, field)
		if !slices.Contains(methods, "spiffe_jwt") || slices.Contains(methods, "spiffe_x509") || slices.Contains(methods, "spiffe_wit") {
			t.Fatalf("%s=%v, want spiffe_jwt without spiffe_x509 and spiffe_wit", field, methods)
		}
	}
	issued := f.token(t, url.Values{"client_id": {f.clientID}}, spiffeSVIDClaims(), http.StatusOK)
	active := f.post(t, "/introspect", url.Values{
		"token": {setupString(t, issued, "access_token")}, "client_id": {f.clientID},
		"client_assertion_type": {spiffeAssertionType}, "client_assertion": {setupJWT(t, f.svidKey, spiffeSVIDClaims())},
	}, http.StatusOK)
	if active["active"] != true {
		t.Fatalf("introspection with a JWT-SVID=%v, want active", active)
	}
}

//spec:covers SPIFFECA03-CIMD (excluded): URL 形式の `client_id` の `jwt-spiffe` の要求は、文書を取得せずに 401 invalid_client を受ける。
func TestJWTSVIDWithClientIDMetadataDocumentIsRejected(t *testing.T) {
	f := newSpiffeClientAuthFixture(t)
	body := f.token(t, url.Values{"client_id": {"https://client.example/metadata.json"}}, spiffeSVIDClaims(), http.StatusUnauthorized)
	assertInvalidClient(t, "CIMD の client_id", body)
}
