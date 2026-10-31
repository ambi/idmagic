package server_http_test

// ID-JAG の 2 ホップを製品と同じ `Register` の組み立てで通す。発行側を default、受け入れ側を
// 別のテナントにする。同じ配備でもテナントごとに発行者が分かれ、draft-05 の 9.4 節の
// トラストドメインをまたぐ。

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/testing_federation"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	xaaIssuerRealm   = stack.Issuer + "/realms/default"
	xaaAudienceRealm = stack.Issuer + "/realms/" + stack.OtherRealm
	xaaResource      = "https://api.chat.example/"
	xaaLocalUser     = "acme-local-user"
	xaaProvider      = "corporate-idp"
)

type xaaFixture struct {
	s        *stack.Stack
	idToken  string
	access   string
	issuer   *stack.Browser
	audience *stack.Browser
}

func newXAAFixture(t *testing.T) *xaaFixture {
	t.Helper()
	s := stack.New(t, stack.WithCrossAppAccess())
	ctx := context.Background()
	now := time.Now().UTC()
	otherTenant := s.TenantID(t, stack.OtherRealm)
	defaultTenant := s.TenantID(t, "default")

	client, err := s.Clients.FindByID(ctx, defaultTenant, stack.BrowserClientID)
	if err != nil || client == nil {
		t.Fatalf("client: %v %v", client, err)
	}
	client.GrantTypes = append(client.GrantTypes, spec.GrantTokenExchange)
	if err := s.Clients.Save(ctx, client); err != nil {
		t.Fatal(err)
	}
	target, err := s.Clients.FindByID(ctx, otherTenant, stack.BrowserClientID)
	if err != nil || target == nil {
		t.Fatalf("target client: %v %v", target, err)
	}
	target.GrantTypes = append(target.GrantTypes, spec.GrantJWTBearer)
	target.Scope = "chat.read chat.history"
	if err := s.Clients.Save(ctx, target); err != nil {
		t.Fatal(err)
	}
	if err := s.McpResourceServers.Save(ctx, &oauthdomain.McpResourceServer{
		TenantID: otherTenant, ID: "22222222-2222-4222-8222-222222222222", Resource: xaaResource, Name: "chat",
		Scopes: []string{"chat.read", "chat.history"}, State: oauthdomain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Users.Save(ctx, &userdomain.User{
		ID: xaaLocalUser, TenantID: otherTenant, PreferredUsername: "local", PasswordHash: "unused",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	testing_federation.SeedOIDCConnection(t, s.Federation, otherTenant, xaaProvider, xaaIssuerRealm, xaaIssuerRealm+"/jwks")
	testing_federation.LinkIdentity(t, s.Federation, otherTenant, xaaProvider, stack.UserID, xaaLocalUser)
	if err := s.CrossAppAccessPolicies.CreateIssuePolicy(ctx, &oauthdomain.CrossAppAccessIssuePolicy{
		TenantID: defaultTenant, ID: "11111111-1111-4111-8111-111111111111", ClientID: stack.BrowserClientID,
		Audience: xaaAudienceRealm, RelyingPartyClientID: stack.BrowserClientID, TargetClientID: stack.BrowserClientID,
		Resources: []string{xaaResource}, Scopes: []string{"chat.read"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.CrossAppAccessPolicies.CreateRedeemPolicy(ctx, &oauthdomain.CrossAppAccessRedeemPolicy{
		TenantID: otherTenant, ID: "33333333-3333-4333-8333-333333333333", ProviderID: xaaProvider,
		ClientID: stack.BrowserClientID, Scopes: []string{"chat.read"},
	}); err != nil {
		t.Fatal(err)
	}

	issuer := s.Browser(t, "default")
	verifier := strings.Repeat("v", 43)
	code, _ := issuer.AuthorizationCode(t, stack.AuthorizationQuery(verifier, nil))
	status, body := issuer.ExchangeCode(t, code, verifier)
	if status != http.StatusOK {
		t.Fatalf("code exchange %d %v", status, body)
	}
	idToken, _ := body["id_token"].(string)
	if idToken == "" {
		t.Fatalf("no id_token: %v", body)
	}
	accessToken, _ := body["access_token"].(string)
	return &xaaFixture{s: s, idToken: idToken, access: accessToken, issuer: issuer, audience: s.Browser(t, stack.OtherRealm)}
}

// 値は draft の 4.3.1 節の例の URN を写す。製品の定数から作らない。
func (f *xaaFixture) issue(t *testing.T) (int, map[string]any) {
	t.Helper()
	return f.issuer.PostToken(t, url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:id-jag"},
		"audience":             {xaaAudienceRealm},
		"resource":             {xaaResource},
		"scope":                {"chat.read"},
		"subject_token":        {f.idToken},
		"subject_token_type":   {"urn:ietf:params:oauth:token-type:id_token"},
	})
}

func xaaRedeemForm(assertion string) url.Values {
	return url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
}

func xaaDecodeSegment(t *testing.T, token string, index int) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %s", token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[index])
	if err != nil {
		t.Fatal(err)
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

//spec:covers REQ-OAUTH2-066, REQ-OAUTH2-069, REQ-OAUTH2-070, IDJAG-ISSUANCE, IDJAG-REDEMPTION, IDJAG-SIGNING-ALG: 発行、公開鍵、信頼、連携、再送の拒否を正式な入口から通す
func TestCrossAppAccessIssueAndRedeem_REQ_OAUTH2_066(t *testing.T) {
	f := newXAAFixture(t)
	status, issued := f.issue(t)
	if status != http.StatusOK {
		t.Fatalf("issue %d %v", status, issued)
	}
	if issued["issued_token_type"] != "urn:ietf:params:oauth:token-type:id-jag" || issued["token_type"] != "N_A" {
		t.Fatalf("issue response = %v", issued)
	}
	if _, leaked := issued["refresh_token"]; leaked {
		t.Fatal("the ID-JAG exchange returned a refresh token")
	}
	assertion, _ := issued["access_token"].(string)
	header := xaaDecodeSegment(t, assertion, 0)
	if header["typ"] != "oauth-id-jag+jwt" || header["alg"] != "RS256" {
		t.Fatalf("header = %v", header)
	}
	claims := xaaDecodeSegment(t, assertion, 1)
	if claims["iss"] != xaaIssuerRealm || claims["aud"] != xaaAudienceRealm || claims["sub"] != stack.UserID ||
		claims["client_id"] != stack.BrowserClientID || claims["resource"] != xaaResource {
		t.Fatalf("claims = %v", claims)
	}

	// ID-JAG は発行者のテナントでは償還できない (9.4 節)。
	status, body := f.issuer.PostToken(t, xaaRedeemForm(assertion))
	if status != http.StatusBadRequest && status != http.StatusUnauthorized {
		t.Fatalf("self redemption %d %v", status, body)
	}

	status, redeemed := f.audience.PostToken(t, xaaRedeemForm(assertion))
	if status != http.StatusOK {
		t.Fatalf("redeem %d %v", status, redeemed)
	}
	if redeemed["scope"] != "chat.read" {
		t.Fatalf("redeem response = %v", redeemed)
	}
	if _, leaked := redeemed["refresh_token"]; leaked {
		t.Fatal("redemption returned a refresh token")
	}
	token, _ := redeemed["access_token"].(string)
	access := xaaDecodeSegment(t, token, 1)
	if access["sub"] != xaaLocalUser || access["aud"] != xaaResource || access["iss"] != xaaAudienceRealm {
		t.Fatalf("access token = %v", access)
	}
	if active := f.s.Introspect(t, stack.OtherRealm, token); active["active"] != true || active["sub"] != xaaLocalUser {
		t.Fatalf("introspection = %v", active)
	}

	status, replay := f.audience.PostToken(t, xaaRedeemForm(assertion))
	if status != http.StatusBadRequest || replay["error"] != "invalid_grant" {
		t.Fatalf("replay %d %v", status, replay)
	}
	f.s.Events.AssertEmitted(t, "IdJagIssued", "IdJagRedeemed", "IdJagRejected")
}

//spec:covers REQ-OAUTH2-067: ID Token ではない自身のトークンを subject_token に受けないこと
func TestCrossAppAccessRefusesAccessTokenAsSubject_REQ_OAUTH2_067(t *testing.T) {
	f := newXAAFixture(t)
	f.idToken = f.access
	status, body := f.issue(t)
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Fatalf("issue %d %v", status, body)
	}
	f.s.Events.AssertNotEmitted(t, "IdJagIssued")
}

func (f *xaaFixture) admin(t *testing.T, realm, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	reader := strings.NewReader("")
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = strings.NewReader(string(raw))
	}
	request := httptest.NewRequest(method, stack.Issuer+"/realms/"+realm+path, reader)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", stack.Issuer)
	request.Header.Set(support.CSRFHeader, "xaa-csrf")
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "xaa-csrf"})
	request.AddCookie(f.s.AdminSessionCookie(t, realm))
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	return recorder
}

//verify:operation ListAdminCrossAppAccessIssuePolicies: 一覧の本文
//verify:operation CreateAdminCrossAppAccessIssuePolicy: 登録の本文、重複の 409、不正な入力の 400
//verify:operation DeleteAdminCrossAppAccessIssuePolicy: 削除の 204 と、別テナントと再度の削除の 404
//verify:operation ListAdminCrossAppAccessRedeemPolicies: 一覧の本文
//verify:operation CreateAdminCrossAppAccessRedeemPolicy: 登録の本文、重複の 409、不正な入力の 400
//verify:operation DeleteAdminCrossAppAccessRedeemPolicy: 削除の 204
//spec:covers REQ-OAUTH2-072: 登録、重複の拒否、不正な入力の拒否、テナント境界、削除後の発行と償還の拒否
func TestCrossAppAccessPolicyAdministration_REQ_OAUTH2_072(t *testing.T) {
	f := newXAAFixture(t)
	const issuePath = "/api/admin/v1/cross-app-access/issue-policies"
	const redeemPath = "/api/admin/v1/cross-app-access/redeem-policies"

	request := map[string]any{
		"client_id": stack.BrowserClientID, "audience": "https://second.example/",
		"relying_party_client_id": stack.BrowserClientID, "target_client_id": "t",
		"resources": []string{"https://r.example/"}, "scopes": []string{"s"},
	}
	created := f.admin(t, "default", http.MethodPost, issuePath, request)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body)
	}
	if again := f.admin(t, "default", http.MethodPost, issuePath, request); again.Code != http.StatusConflict {
		t.Fatalf("duplicate %d %s", again.Code, again.Body)
	}
	for name, mutate := range map[string]func(map[string]any){
		"no scopes":      func(r map[string]any) { r["scopes"] = []string{} },
		"no resources":   func(r map[string]any) { r["resources"] = []string{} },
		"http audience":  func(r map[string]any) { r["audience"] = "http://plain.example/" },
		"unknown client": func(r map[string]any) { r["client_id"] = "nobody" },
	} {
		invalid := maps.Clone(request)
		invalid["audience"] = "https://third.example/"
		mutate(invalid)
		if res := f.admin(t, "default", http.MethodPost, issuePath, invalid); res.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", name, res.Code, res.Body)
		}
	}
	if res := f.admin(t, "default", http.MethodPost, redeemPath, map[string]any{
		"provider_id": "missing", "client_id": stack.BrowserClientID, "scopes": []string{"s"},
	}); res.Code != http.StatusBadRequest {
		t.Errorf("unknown provider: %d %s", res.Code, res.Body)
	}

	redeemRequest := map[string]any{
		"provider_id": xaaProvider, "client_id": stack.ResourceServerClientID, "scopes": []string{"chat.read"},
	}
	createdRedeem := f.admin(t, stack.OtherRealm, http.MethodPost, redeemPath, redeemRequest)
	if createdRedeem.Code != http.StatusCreated || !strings.Contains(createdRedeem.Body.String(), `"provider_id":"`+xaaProvider+`"`) {
		t.Fatalf("create redeem %d %s", createdRedeem.Code, createdRedeem.Body)
	}
	if again := f.admin(t, stack.OtherRealm, http.MethodPost, redeemPath, redeemRequest); again.Code != http.StatusConflict {
		t.Fatalf("duplicate redeem %d %s", again.Code, again.Body)
	}
	if listed := f.admin(t, stack.OtherRealm, http.MethodGet, redeemPath, nil); !strings.Contains(listed.Body.String(), stack.ResourceServerClientID) {
		t.Fatalf("list redeem = %s", listed.Body)
	}

	var createdPolicy struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &createdPolicy); err != nil {
		t.Fatal(err)
	}
	if other := f.admin(t, stack.OtherRealm, http.MethodDelete, issuePath+"/"+createdPolicy.ID, nil); other.Code != http.StatusNotFound {
		t.Fatalf("another tenant deleted the policy: %d", other.Code)
	}
	listed := f.admin(t, "default", http.MethodGet, issuePath, nil)
	if !strings.Contains(listed.Body.String(), "https://second.example/") {
		t.Fatalf("list = %s", listed.Body)
	}
	if deleted := f.admin(t, "default", http.MethodDelete, issuePath+"/"+createdPolicy.ID, nil); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete %d %s", deleted.Code, deleted.Body)
	}
	if gone := f.admin(t, "default", http.MethodDelete, issuePath+"/"+createdPolicy.ID, nil); gone.Code != http.StatusNotFound {
		t.Fatalf("second delete %d", gone.Code)
	}
	f.s.Events.AssertEmitted(t, "CrossAppAccessPolicyCreated", "CrossAppAccessPolicyDeleted")

	status, issued := f.issue(t)
	if status != http.StatusOK {
		t.Fatalf("issue %d %v", status, issued)
	}
	assertion, _ := issued["access_token"].(string)
	// 受け入れの許可を消すと、発行済みの ID-JAG も償還できない。
	if deleted := f.admin(t, stack.OtherRealm, http.MethodDelete, redeemPath+"/33333333-3333-4333-8333-333333333333", nil); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete redeem %d %s", deleted.Code, deleted.Body)
	}
	status, refused := f.audience.PostToken(t, xaaRedeemForm(assertion))
	if status != http.StatusBadRequest || refused["error"] != "access_denied" {
		t.Fatalf("redeem after revocation %d %v", status, refused)
	}
	// 発行の許可を消すと、以後の発行を拒否する。
	if deleted := f.admin(t, "default", http.MethodDelete, issuePath+"/11111111-1111-4111-8111-111111111111", nil); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete issue %d %s", deleted.Code, deleted.Body)
	}
	status, refused = f.issue(t)
	if status != http.StatusBadRequest || refused["error"] != "invalid_target" {
		t.Fatalf("issue after revocation %d %v", status, refused)
	}
}

//spec:covers IDJAG-METADATA, REQ-SIGNINGKEYS-013: メタデータの広告と、ID-JAG の鍵が JWKS で RS256 として公開されること
func TestCrossAppAccessMetadataAndKeys_IDJAG_METADATA(t *testing.T) {
	f := newXAAFixture(t)
	if status, issued := f.issue(t); status != http.StatusOK {
		t.Fatalf("issue %d %v", status, issued)
	}
	get := func(path string) map[string]any {
		recorder := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, xaaIssuerRealm+path, http.NoBody))
		decoded := map[string]any{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("%s: %s", path, recorder.Body)
		}
		return decoded
	}
	metadata := get("/.well-known/oauth-authorization-server")
	if !strings.Contains(xaaJSON(t, metadata["identity_chaining_requested_token_types_supported"]), "urn:ietf:params:oauth:token-type:id-jag") ||
		!strings.Contains(xaaJSON(t, metadata["id_jag_signing_alg_values_supported"]), "RS256") ||
		!strings.Contains(xaaJSON(t, metadata["authorization_grant_profiles_supported"]), "urn:ietf:params:oauth:grant-profile:id-jag") ||
		!strings.Contains(xaaJSON(t, metadata["grant_types_supported"]), "urn:ietf:params:oauth:grant-type:jwt-bearer") {
		t.Fatalf("metadata = %v", metadata)
	}
	jwks := xaaJSON(t, get("/jwks")["keys"])
	if !strings.Contains(jwks, `"alg":"RS256"`) || !strings.Contains(jwks, `"alg":"PS256"`) {
		t.Fatalf("jwks = %s", jwks)
	}
}

func xaaJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

//spec:covers REQ-SIGNINGKEYS-013: 管理者が ID-JAG の鍵だけをローテーションでき、Signing の鍵が変わらず、用途の鍵にスコープを指定できないこと
func TestCrossAppAccessKeyRotation_REQ_SIGNINGKEYS_013(t *testing.T) {
	f := newXAAFixture(t)
	_, first := f.issue(t)
	firstToken, _ := first["access_token"].(string)
	firstKid := xaaDecodeSegment(t, firstToken, 0)["kid"]

	signingBefore := f.admin(t, "default", http.MethodGet, "/api/admin/v1/keys", nil).Body.String()
	rotated := f.admin(t, "default", http.MethodPost, "/api/admin/v1/keys/rotate", map[string]any{"usage": "IdJagSigning"})
	if rotated.Code != http.StatusOK {
		t.Fatalf("rotate %d %s", rotated.Code, rotated.Body)
	}
	if signingAfter := f.admin(t, "default", http.MethodGet, "/api/admin/v1/keys", nil).Body.String(); signingAfter != signingBefore {
		t.Fatalf("rotating the ID-JAG key changed the signing keys:\n%s\n%s", signingBefore, signingAfter)
	}
	_, second := f.issue(t)
	secondToken, _ := second["access_token"].(string)
	if xaaDecodeSegment(t, secondToken, 0)["kid"] == firstKid {
		t.Fatal("the ID-JAG key did not rotate")
	}
	// 回転の前後の ID-JAG は、JWKS に旧鍵が残るのでどちらも償還できる。
	for _, token := range []string{firstToken, secondToken} {
		if status, body := f.audience.PostToken(t, xaaRedeemForm(token)); status != http.StatusOK {
			t.Fatalf("redeem %d %v", status, body)
		}
	}
	if scoped := f.admin(t, "default", http.MethodPost, "/api/admin/v1/keys/rotate", map[string]any{"usage": "IdJagSigning", "scope_id": "other"}); scoped.Code != http.StatusBadRequest {
		t.Fatalf("scoped rotation %d %s", scoped.Code, scoped.Body)
	}
}

func xaaDPoPKey(t *testing.T) (*rsa.PrivateKey, map[string]any, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	// RFC 7638 の正規形。キーを辞書順に並べた JSON の SHA-256 である。
	sum := sha256.Sum256([]byte(`{"e":"` + e + `","kty":"RSA","n":"` + n + `"}`))
	return key, map[string]any{"kty": "RSA", "e": e, "n": n}, base64.RawURLEncoding.EncodeToString(sum[:])
}

func xaaDPoPProof(t *testing.T, key *rsa.PrivateKey, jwk map[string]any, endpoint, jti string) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := json.Marshal(map[string]any{"htm": http.MethodPost, "htu": endpoint, "jti": jti, "iat": time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func (f *xaaFixture) postDPoP(t *testing.T, realm string, form url.Values, key *rsa.PrivateKey, jwk map[string]any, jti string) (int, map[string]any) {
	t.Helper()
	endpoint := stack.Issuer + "/realms/" + realm + "/token"
	request := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(stack.BrowserClientID, stack.BrowserClientSecret)
	request.Header.Set("DPoP", xaaDPoPProof(t, key, jwk, endpoint, jti))
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	body := map[string]any{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", recorder.Body, err)
	}
	return recorder.Code, body
}

//spec:covers REQ-OAUTH2-068, REQ-OAUTH2-071, IDJAG-SENDER-CONSTRAINT: DPoP の証明が発行の cnf と償還の jwt-dpop を正式な入口で結び、jwt-bearer への格下げを拒否すること
func TestCrossAppAccessDPoPBinding_REQ_OAUTH2_071(t *testing.T) {
	f := newXAAFixture(t)
	ctx := context.Background()
	target, err := f.s.Clients.FindByID(ctx, f.s.TenantID(t, stack.OtherRealm), stack.BrowserClientID)
	if err != nil || target == nil {
		t.Fatalf("target client: %v %v", target, err)
	}
	target.GrantTypes = append(target.GrantTypes, spec.GrantJWTDPoP)
	if err := f.s.Clients.Save(ctx, target); err != nil {
		t.Fatal(err)
	}
	key, jwk, thumbprint := xaaDPoPKey(t)
	status, issued := f.postDPoP(t, "default", url.Values{
		"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"requested_token_type": {"urn:ietf:params:oauth:token-type:id-jag"},
		"audience":             {xaaAudienceRealm},
		"resource":             {xaaResource},
		"subject_token":        {f.idToken},
		"subject_token_type":   {"urn:ietf:params:oauth:token-type:id_token"},
	}, key, jwk, "proof-issue")
	if status != http.StatusOK {
		t.Fatalf("issue %d %v", status, issued)
	}
	assertion, _ := issued["access_token"].(string)
	cnf, _ := xaaDecodeSegment(t, assertion, 1)["cnf"].(map[string]any)
	if cnf["jkt"] != thumbprint {
		t.Fatalf("cnf = %v, want jkt %s", cnf, thumbprint)
	}

	status, body := f.audience.PostToken(t, xaaRedeemForm(assertion))
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" {
		t.Fatalf("downgrade %d %v", status, body)
	}
	form := url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-dpop"}, "assertion": {assertion}}
	status, body = f.audience.PostToken(t, form)
	if status != http.StatusBadRequest || body["error"] != "invalid_dpop_proof" {
		t.Fatalf("no proof %d %v", status, body)
	}
	status, body = f.postDPoP(t, stack.OtherRealm, form, key, jwk, "proof-redeem")
	if status != http.StatusOK || body["token_type"] != "DPoP" {
		t.Fatalf("redeem %d %v", status, body)
	}
	token, _ := body["access_token"].(string)
	accessCnf, _ := xaaDecodeSegment(t, token, 1)["cnf"].(map[string]any)
	if accessCnf["jkt"] != thumbprint {
		t.Fatalf("access token cnf = %v", accessCnf)
	}
}
