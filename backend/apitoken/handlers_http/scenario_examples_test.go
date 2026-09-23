package handlers_http_test

// ApiTokens が宣言する具体例のうち、認証、発行と失効、管理 API の粒度認可を、製品と同じ
// `server_http.Register` の組み立てで観測する。
//
// トークンは testing_stack の実署名器で作る。認証の拒否が「JWT のどの部分が不正だったか」に
// よるものかを区別するには、偽の検証器ではなく、製品が使う署名検証と記録の突き合わせを
// 通す必要がある。
//
// 管理 API の操作にはグループを使う。参照と変更で別の粒度スコープが宣言されていて、
// 変更の効果を保存先から読み直せるためである。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	realm          = tenancydomain.DefaultRealm
	realmAudience  = stack.Issuer + "/realms/" + tenancydomain.DefaultRealm
	apiTokensPath  = "/realms/" + tenancydomain.DefaultRealm + "/api/admin/v1/api-tokens"
	groupsPath     = "/realms/" + tenancydomain.DefaultRealm + "/api/admin/v1/groups"
	csrfTokenValue = "csrf-token-for-examples"
)

func newExampleStack(t *testing.T) *stack.Stack {
	t.Helper()
	return stack.New(t, stack.WithApiTokens(), stack.WithAuthorizationCodeFlow(), stack.WithTokenIssuance())
}

// ---- REQ-APITOKENS-002: AuthenticateApiToken ----

// authenticate は製品が組み立てるのと同じ overlay（記録の保存先と実署名器）で JWT を認証する。
func authenticate(t *testing.T, s *stack.Stack, token string) (apitokendomain.Principal, error) {
	t.Helper()
	service := apitokenusecases.New(s.ApiTokens, apitokenusecases.WithTokenIntrospector(s.Signer))
	return service.Authenticate(s.RealmContext(t, realm), token)
}

// managedToken は、JWT と管理記録をそれぞれ 1 か所だけ壊せるように別々に組み立てる。
// 既定値のまま作ったトークンは認証を通る。各拒否の事例はこの既定から 1 か所だけを変え、
// 拒否がその 1 か所によるものであることを、既定のままの成功と対にして示す。
type managedToken struct {
	signContext context.Context
	audience    string
	scopes      []string
	expiresAt   time.Time
	// record が nil のときは記録を保存しない。
	record func(*apitokendomain.ApiToken)
}

func validManagedToken(t *testing.T, s *stack.Stack) managedToken {
	t.Helper()
	return managedToken{
		signContext: s.RealmContext(t, realm),
		audience:    realmAudience,
		scopes:      []string{string(apitokendomain.ScopeGroupsRead)},
		expiresAt:   time.Now().Add(time.Hour),
		record:      func(*apitokendomain.ApiToken) {},
	}
}

func (m managedToken) sign(t *testing.T, s *stack.Stack) string {
	t.Helper()
	tenantID := s.TenantID(t, realm)
	literal, jti, err := s.Signer.SignAccessToken(m.signContext, oauthports.AccessTokenInput{
		Client:    &oauthdomain.OAuth2Client{TenantID: tenantID, ClientID: apitokendomain.BuiltinClientID},
		Sub:       stack.AdminUserID,
		Scopes:    m.scopes,
		Audiences: []string{m.audience},
		ExpiresAt: m.expiresAt.Unix(),
		Managed:   true,
	})
	if err != nil {
		t.Fatalf("sign managed token: %v", err)
	}
	if m.record == nil {
		return literal
	}
	recordExpiry := time.Now().Add(time.Hour)
	record := &apitokendomain.ApiToken{
		ID: "token-" + jti, TenantID: tenantID, UserID: stack.AdminUserID, JTI: jti,
		ClientID: apitokendomain.BuiltinClientID, Scopes: apitokendomain.Scopes{apitokendomain.ScopeGroupsRead},
		Audience: realmAudience, CreatedAt: time.Now().UTC(), ExpiresAt: &recordExpiry,
	}
	m.record(record)
	if err := s.ApiTokens.Save(s.RealmContext(t, realm), record); err != nil {
		t.Fatalf("save record: %v", err)
	}
	return literal
}

func assertDenied(t *testing.T, principal apitokendomain.Principal, err error) {
	t.Helper()
	if !errors.Is(err, apitokenusecases.ErrAccessDenied) {
		t.Fatalf("err = %v, want ErrAccessDenied", err)
	}
	if principal.UserID != "" || principal.TenantID != "" || principal.TokenID != "" || len(principal.Scopes) != 0 {
		t.Fatalf("denied authentication still returned a principal: %+v", principal)
	}
}

//spec:covers EX-APITOKENS-002-01: 管理コンソールと同じ発行経路で作った JWT を提示すると、発行したテナントの tenant_id、発行者の user_id、組み込みの client_id、発行時のスコープ集合を持つ主体が返ること。
func TestAuthenticateApiTokenReturnsTheIssuedPrincipal(t *testing.T) {
	s := newExampleStack(t)
	token, metadata := s.IssueApiToken(t, realm, apitokendomain.ScopeGroupsRead, apitokendomain.ScopeUsersWrite)

	principal, err := authenticate(t, s, token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if principal.TenantID != s.TenantID(t, realm) || principal.UserID != stack.AdminUserID ||
		principal.ClientID != apitokendomain.BuiltinClientID || principal.TokenID != metadata.ID {
		t.Fatalf("principal = %+v, want tenant %q, user %q, client %q, token %q",
			principal, s.TenantID(t, realm), stack.AdminUserID, apitokendomain.BuiltinClientID, metadata.ID)
	}
	want := apitokendomain.Scopes{apitokendomain.ScopeGroupsRead, apitokendomain.ScopeUsersWrite}.Strings()
	if got := principal.Scopes.Strings(); strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("scopes = %v, want %v", got, want)
	}
}

//spec:covers EX-APITOKENS-002-02: JWT の形式、署名、発行者、audience、exp のどれか 1 つだけが不正で記録は正しいトークンを、主体を返さずに AccessDeniedError で拒否すること。既定のままのトークンが通ることで、拒否がその 1 か所によるものだと示す。
func TestAuthenticateApiTokenRejectsAnInvalidJWT(t *testing.T) {
	s := newExampleStack(t)
	if _, err := authenticate(t, s, validManagedToken(t, s).sign(t, s)); err != nil {
		t.Fatalf("the unmodified token must authenticate, err = %v", err)
	}
	tenant, err := s.Tenants.FindByRealm(context.Background(), realm)
	if err != nil || tenant == nil {
		t.Fatalf("tenant: %v", err)
	}

	for _, tc := range []struct {
		name  string
		token func() string
	}{
		{"not a JWT", func() string { return "not-a-jwt" }},
		{"signature", func() string {
			return tamperSignature(t, validManagedToken(t, s).sign(t, s))
		}},
		{"issuer", func() string {
			token := validManagedToken(t, s)
			// 同じテナントの鍵で署名し、発行者だけを別の値にする。
			token.signContext = tenancy.WithTenant(context.Background(), tenant,
				"https://attacker.example/realms/"+realm, "/realms/"+realm)
			return token.sign(t, s)
		}},
		{"audience", func() string {
			token := validManagedToken(t, s)
			token.audience = stack.Issuer + "/realms/" + stack.OtherRealm
			return token.sign(t, s)
		}},
		{"exp", func() string {
			token := validManagedToken(t, s)
			token.expiresAt = time.Now().Add(-time.Minute)
			return token.sign(t, s)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			principal, err := authenticate(t, s, tc.token())
			assertDenied(t, principal, err)
		})
	}
}

//spec:covers EX-APITOKENS-002-03: JWT は正しく、管理記録が無い、失効済み、期限切れ、スコープ集合が空のどれかであるトークンを、主体を返さずに AccessDeniedError で拒否すること。既定のままのトークンが通ることで、拒否が記録の状態によるものだと示す。
func TestAuthenticateApiTokenRejectsAnUnusableRecord(t *testing.T) {
	s := newExampleStack(t)
	if _, err := authenticate(t, s, validManagedToken(t, s).sign(t, s)); err != nil {
		t.Fatalf("the unmodified token must authenticate, err = %v", err)
	}

	for _, tc := range []struct {
		name   string
		modify func(*managedToken)
	}{
		{"unknown", func(m *managedToken) { m.record = nil }},
		{"revoked", func(m *managedToken) {
			m.record = func(r *apitokendomain.ApiToken) {
				revokedAt := time.Now().Add(-time.Minute)
				r.RevokedAt = &revokedAt
			}
		}},
		{"expired", func(m *managedToken) {
			m.record = func(r *apitokendomain.ApiToken) {
				expiredAt := time.Now().Add(-time.Minute)
				r.ExpiresAt = &expiredAt
			}
		}},
		{"empty scope set", func(m *managedToken) {
			// JWT の scope も空にし、記録と JWT の不一致ではなく空であること自体で拒否させる。
			m.scopes = nil
			m.record = func(r *apitokendomain.ApiToken) { r.Scopes = nil }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := validManagedToken(t, s)
			tc.modify(&token)
			principal, err := authenticate(t, s, token.sign(t, s))
			assertDenied(t, principal, err)
		})
	}
}

// tamperSignature は署名部の中央の 1 文字を差し替える。末尾の文字は base64 の詰め物の
// ビットだけを変えることがあり、署名の値が変わらない場合がある。
func tamperSignature(t *testing.T, token string) string {
	t.Helper()
	signatureStart := strings.LastIndex(token, ".") + 1
	i := signatureStart + (len(token)-signatureStart)/2
	replacement := byte('A')
	if token[i] == 'A' {
		replacement = 'B'
	}
	return token[:i] + string(replacement) + token[i+1:]
}

// ---- REQ-APITOKENS-003: IssueApiToken / ListApiTokens / RevokeApiToken ----

// sessionClient はログインセッションを持つブラウザーとして管理 API を呼ぶ。
// Origin と CSRF の二重送信は成立させておき、拒否があれば認証と認可によるものだけにする。
type sessionClient struct {
	s         *stack.Stack
	sessionID string
	// csrfHeader は X-Csrf-Token に載せる値である。cookie 側は常に csrfTokenValue なので、
	// 空にすると二重送信が成立しない要求になる。
	csrfHeader string
}

func signedIn(t *testing.T, s *stack.Stack, sub string) sessionClient {
	t.Helper()
	authn, err := s.Sessions.Create(s.RealmContext(t, realm), sub, []string{"pwd"}, time.Now().UTC())
	if err != nil || authn == nil || authn.SessionID == "" {
		t.Fatalf("create session for %s: authn=%+v err=%v", sub, authn, err)
	}
	return sessionClient{s: s, sessionID: authn.SessionID, csrfHeader: csrfTokenValue}
}

func (c sessionClient) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	request := newJSONRequest(t, method, path, body)
	request.Header.Set("Origin", stack.Issuer)
	if c.csrfHeader != "" {
		request.Header.Set(support.CSRFHeader, c.csrfHeader)
	}
	request.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: c.sessionID})
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: csrfTokenValue})
	recorder := httptest.NewRecorder()
	c.s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func bearer(t *testing.T, s *stack.Stack, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	request := newJSONRequest(t, method, path, body)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	payload := []byte{}
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequestWithContext(t.Context(), method, stack.Issuer+path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func storedApiTokens(t *testing.T, s *stack.Stack) []*apitokendomain.ApiToken {
	t.Helper()
	tokens, err := s.ApiTokens.List(s.RealmContext(t, realm), s.TenantID(t, realm))
	if err != nil {
		t.Fatalf("list stored tokens: %v", err)
	}
	return tokens
}

func problemType(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, recorder.Body.String())
	}
	return problem.Type
}

// jwtPart は JWT の header または payload を復号する。
func jwtPart(t *testing.T, token string, index int) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want a JWS compact serialization", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[index])
	if err != nil {
		t.Fatalf("decode JWT part %d: %v", index, err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("unmarshal JWT part %d: %v", index, err)
	}
	return claims
}

//spec:covers EX-APITOKENS-003-01: 管理者のセッションで発行すると、sub が発行者で client_id が組み込みクライアントの RFC 9068 JWT (typ at+jwt) が応答にだけ載り、一覧は JWT 本文を含まずにライフサイクルを返し、失効後は同じトークンが管理 API と /introspect の双方で拒否されること。
func TestAdministratorIssuesListsAndRevokesAnApiToken(t *testing.T) {
	s := newExampleStack(t)
	admin := signedIn(t, s, stack.AdminUserID)

	issued := admin.do(t, http.MethodPost, apiTokensPath, map[string]any{
		"description": "Directory sync", "scopes": []string{"groups:read"}, "expiry_days": 7,
	})
	if issued.Code != http.StatusCreated || issued.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("issue status=%d cache=%q body=%s", issued.Code, issued.Header().Get("Cache-Control"), issued.Body.String())
	}
	var issueBody struct {
		Token string `json:"token"`
		Meta  struct {
			ID string `json:"id"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &issueBody); err != nil {
		t.Fatal(err)
	}
	if typ := jwtPart(t, issueBody.Token, 0)["typ"]; typ != "at+jwt" {
		t.Fatalf("JWT typ = %v, want at+jwt (RFC 9068)", typ)
	}
	claims := jwtPart(t, issueBody.Token, 1)
	if claims["sub"] != stack.AdminUserID || claims["client_id"] != apitokendomain.BuiltinClientID ||
		claims["iss"] != realmAudience || claims["aud"] != realmAudience || claims["scope"] != "groups:read" {
		t.Fatalf("JWT claims = %v", claims)
	}
	for _, claim := range []string{"exp", "iat", "jti"} {
		if _, ok := claims[claim]; !ok {
			t.Fatalf("JWT claims lack %s required by RFC 9068: %v", claim, claims)
		}
	}

	listed := admin.do(t, http.MethodGet, apiTokensPath, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}
	if strings.Contains(listed.Body.String(), issueBody.Token) || strings.Contains(listed.Body.String(), `"token"`) {
		t.Fatalf("listing returned the JWT again: %s", listed.Body.String())
	}
	var listBody struct {
		Tokens []struct {
			ID        string   `json:"id"`
			UserID    string   `json:"user_id"`
			Scopes    []string `json:"scopes"`
			ExpiresAt *string  `json:"expires_at"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Tokens) != 1 || listBody.Tokens[0].ID != issueBody.Meta.ID ||
		listBody.Tokens[0].UserID != stack.AdminUserID || listBody.Tokens[0].ExpiresAt == nil {
		t.Fatalf("listing = %+v, want the issued token's lifecycle", listBody.Tokens)
	}

	// 失効前に通ることを見ておく。これが無いと、常に拒否する配線と区別できない。
	if before := bearer(t, s, http.MethodGet, groupsPath, issueBody.Token, nil); before.Code != http.StatusOK {
		t.Fatalf("before revoke status=%d body=%s, want 200", before.Code, before.Body.String())
	}
	if revoked := admin.do(t, http.MethodDelete, apiTokensPath+"/"+issueBody.Meta.ID, nil); revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s", revoked.Code, revoked.Body.String())
	}
	after := bearer(t, s, http.MethodGet, groupsPath, issueBody.Token, nil)
	if after.Code != http.StatusUnauthorized || problemType(t, after) != "urn:idmagic:error:invalid_token" {
		t.Fatalf("after revoke status=%d body=%s, want 401 invalid_token", after.Code, after.Body.String())
	}
	if active := s.Introspect(t, realm, issueBody.Token)["active"]; active != false {
		t.Fatalf("introspection after revoke active=%v, want false", active)
	}
}

//spec:covers EX-APITOKENS-003-02: expiry_days が 0 と負の値の発行要求を invalid_request で拒否し、保存先に記録が 1 件も増えないこと。
func TestIssueApiTokenRejectsANonPositiveExpiry(t *testing.T) {
	s := newExampleStack(t)
	admin := signedIn(t, s, stack.AdminUserID)

	for _, expiryDays := range []int{0, -3} {
		rejected := admin.do(t, http.MethodPost, apiTokensPath, map[string]any{
			"description": "Directory sync", "scopes": []string{"groups:read"}, "expiry_days": expiryDays,
		})
		if rejected.Code != http.StatusBadRequest || problemType(t, rejected) != "urn:idmagic:error:invalid_request" {
			t.Fatalf("expiry_days=%d: status=%d body=%s, want 400 invalid_request", expiryDays, rejected.Code, rejected.Body.String())
		}
		if strings.Contains(rejected.Body.String(), `"token"`) {
			t.Fatalf("expiry_days=%d: refusal carried a token: %s", expiryDays, rejected.Body.String())
		}
	}
	if stored := storedApiTokens(t, s); len(stored) != 0 {
		t.Fatalf("stored tokens = %d, want none after the refusals", len(stored))
	}
}

//spec:covers EX-APITOKENS-003-03: admin も system_admin も持たない利用者のセッションで発行を要求すると access_denied で拒否し、保存先に記録が増えないこと。system_admin だけを持つ利用者の発行が通ることで、拒否の境界が 2 つのロールの双方にあることを示す。
func TestIssueApiTokenRequiresTheAdminOrSystemAdminRole(t *testing.T) {
	s := newExampleStack(t)
	systemAdmin, err := s.Users.FindBySub(context.Background(), stack.UserID)
	if err != nil || systemAdmin == nil {
		t.Fatalf("find user: %v", err)
	}
	withSystemAdmin := *systemAdmin
	withSystemAdmin.ID, withSystemAdmin.PreferredUsername = "system-admin-1", "system-admin"
	withSystemAdmin.Roles = []string{"system_admin"}
	s.Users.Seed(&withSystemAdmin)
	issueRequest := map[string]any{
		"description": "Directory sync", "scopes": []string{"groups:read"}, "expiry_days": 7,
	}

	rejected := signedIn(t, s, stack.UserID).do(t, http.MethodPost, apiTokensPath, issueRequest)
	if rejected.Code != http.StatusForbidden || problemType(t, rejected) != "urn:idmagic:error:access_denied" {
		t.Fatalf("status=%d body=%s, want 403 access_denied", rejected.Code, rejected.Body.String())
	}
	if stored := storedApiTokens(t, s); len(stored) != 0 {
		t.Fatalf("stored tokens = %d, want none after the refusal", len(stored))
	}

	systemAdminSession := signedIn(t, s, withSystemAdmin.ID)
	issued := systemAdminSession.do(t, http.MethodPost, apiTokensPath, issueRequest)
	if issued.Code != http.StatusCreated {
		t.Fatalf("system_admin issue status=%d body=%s, want 201", issued.Code, issued.Body.String())
	}
	stored := storedApiTokens(t, s)
	if len(stored) != 1 || stored[0].UserID != withSystemAdmin.ID {
		t.Fatalf("stored tokens = %+v, want one issued by the system_admin", stored)
	}
	if listed := systemAdminSession.do(t, http.MethodGet, apiTokensPath, nil); listed.Code != http.StatusOK ||
		!strings.Contains(listed.Body.String(), stored[0].ID) {
		t.Fatalf("system_admin list status=%d body=%s, want the issued token", listed.Code, listed.Body.String())
	}
	if revoked := systemAdminSession.do(t, http.MethodDelete, apiTokensPath+"/"+stored[0].ID, nil); revoked.Code != http.StatusNoContent {
		t.Fatalf("system_admin revoke status=%d body=%s, want 204", revoked.Code, revoked.Body.String())
	}
	if after := storedApiTokens(t, s); after[0].RevokedAt == nil {
		t.Fatalf("stored token after system_admin revoke = %+v, want it revoked", after[0])
	}
}

//spec:covers EX-APITOKENS-003-04: 存在しない id の失効を 204 で成功として扱い、既存トークンの記録は失効されず、そのトークンが管理 API へ引き続き到達できること。
func TestRevokeApiTokenForAnUnknownIDSucceedsWithoutSideEffects(t *testing.T) {
	s := newExampleStack(t)
	admin := signedIn(t, s, stack.AdminUserID)
	token, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeGroupsRead)

	revoked := admin.do(t, http.MethodDelete, apiTokensPath+"/00000000-0000-4000-8000-000000000000", nil)
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s, want 204", revoked.Code, revoked.Body.String())
	}
	stored := storedApiTokens(t, s)
	if len(stored) != 1 || stored[0].RevokedAt != nil {
		t.Fatalf("stored tokens = %+v, want the existing token unrevoked", stored)
	}
	if still := bearer(t, s, http.MethodGet, groupsPath, token, nil); still.Code != http.StatusOK {
		t.Fatalf("existing token status=%d body=%s, want 200", still.Code, still.Body.String())
	}
}

// ログインセッションは cookie で運ばれるので、別サイトから送られた要求でも付いてくる。
// 発行と失効は状態を変えるので、ほかの管理 API と同じく Origin と CSRF の二重送信が
// 揃わない要求を拒否し、トークンの記録を変えない。
func TestApiTokenStateChangesRequireTheCSRFToken(t *testing.T) {
	s := newExampleStack(t)
	admin := signedIn(t, s, stack.AdminUserID)
	_, existing := s.IssueApiToken(t, realm, apitokendomain.ScopeGroupsRead)
	forged := admin
	forged.csrfHeader = ""

	issued := forged.do(t, http.MethodPost, apiTokensPath, map[string]any{
		"description": "forged", "scopes": []string{"groups:write"}, "expiry_days": 7,
	})
	if issued.Code != http.StatusForbidden || problemType(t, issued) != "urn:idmagic:error:csrf_failed" {
		t.Fatalf("issue without CSRF status=%d body=%s, want 403 csrf_failed", issued.Code, issued.Body.String())
	}
	revoked := forged.do(t, http.MethodDelete, apiTokensPath+"/"+existing.ID, nil)
	if revoked.Code != http.StatusForbidden || problemType(t, revoked) != "urn:idmagic:error:csrf_failed" {
		t.Fatalf("revoke without CSRF status=%d body=%s, want 403 csrf_failed", revoked.Code, revoked.Body.String())
	}
	stored := storedApiTokens(t, s)
	if len(stored) != 1 || stored[0].ID != existing.ID || stored[0].RevokedAt != nil {
		t.Fatalf("stored tokens = %+v, want only the existing token, unrevoked", stored)
	}
}

// ---- REQ-APITOKENS-004: 管理 API の粒度スコープ ----

func storedGroupNames(t *testing.T, s *stack.Stack) []string {
	t.Helper()
	groups, err := s.Groups.ListAll(s.RealmContext(t, realm), s.TenantID(t, realm))
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	names := make([]string, len(groups))
	for i, group := range groups {
		names[i] = group.Name
	}
	return names
}

// portalAccessToken はブラウザーのポータルが提示する通常の OAuth アクセストークンを作る。
// 管理記録を持たず、粒度スコープではなくポータル境界のスコープ idmagic.admin を持つ。
func portalAccessToken(t *testing.T, s *stack.Stack) string {
	t.Helper()
	token, _, err := s.Signer.SignAccessToken(s.RealmContext(t, realm), oauthports.AccessTokenInput{
		Client:    &oauthdomain.OAuth2Client{TenantID: s.TenantID(t, realm), ClientID: "admin-portal"},
		Sub:       stack.AdminUserID,
		Scopes:    []string{"openid", "idmagic.admin"},
		Audiences: []string{realmAudience},
	})
	if err != nil {
		t.Fatalf("sign portal token: %v", err)
	}
	return token
}

//spec:covers EX-APITOKENS-004-01: 対応スコープと admin ロールを満たす API アクセストークンは参照と変更の操作を実行し、ポータルの OAuth アクセストークンとログインセッションは粒度スコープを持たずに同じ変更操作を実行できること。効果はグループの保存先で読み直す。
func TestAdminApiRunsAnOperationThatScopeAndRoleBothAllow(t *testing.T) {
	s := newExampleStack(t)

	reader, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeGroupsRead)
	if listed := bearer(t, s, http.MethodGet, groupsPath, reader, nil); listed.Code != http.StatusOK {
		t.Fatalf("groups:read list status=%d body=%s, want 200", listed.Code, listed.Body.String())
	}
	writer, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeGroupsWrite)
	for name, send := range map[string]func(body any) *httptest.ResponseRecorder{
		"api token": func(body any) *httptest.ResponseRecorder {
			return bearer(t, s, http.MethodPost, groupsPath, writer, body)
		},
		"portal token": func(body any) *httptest.ResponseRecorder {
			return bearer(t, s, http.MethodPost, groupsPath, portalAccessToken(t, s), body)
		},
		"login session": func(body any) *httptest.ResponseRecorder {
			return signedIn(t, s, stack.AdminUserID).do(t, http.MethodPost, groupsPath, body)
		},
	} {
		t.Run(name, func(t *testing.T) {
			created := send(map[string]any{"name": name})
			if created.Code != http.StatusCreated {
				t.Fatalf("create status=%d body=%s, want 201", created.Code, created.Body.String())
			}
			if names := storedGroupNames(t, s); !slices.Contains(names, name) {
				t.Fatalf("stored groups = %v, want %q to be created", names, name)
			}
		})
	}
}

//spec:covers EX-APITOKENS-004-02: 参照スコープだけの API アクセストークンで変更操作を要求すると、必要な groups:write を WWW-Authenticate に示す insufficient_scope で拒否し、グループの保存先が変わらないこと。
func TestAdminApiRejectsATokenWithoutTheOperationsScope(t *testing.T) {
	s := newExampleStack(t)
	reader, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeGroupsRead)

	rejected := bearer(t, s, http.MethodPost, groupsPath, reader, map[string]any{"name": "not-created"})
	assertInsufficientScope(t, rejected, "groups:write")
	if names := storedGroupNames(t, s); len(names) != 0 {
		t.Fatalf("stored groups = %v, want none after the refusal", names)
	}
}

//spec:covers EX-APITOKENS-004-04: 対話セッション限定の発行操作は、全スコープを持つ API アクセストークンでも interactive_session を示す insufficient_scope で拒否し、トークンの記録が増えないこと。
func TestAdminApiRejectsATokenForAnInteractiveSessionOnlyOperation(t *testing.T) {
	s := newExampleStack(t)
	all := make([]apitokendomain.Scope, 0)
	for _, scope := range apitokendomain.AllScopes() {
		all = append(all, apitokendomain.Scope(scope))
	}
	token, _ := s.IssueApiToken(t, realm, all...)

	rejected := bearer(t, s, http.MethodPost, apiTokensPath, token, map[string]any{
		"description": "escalation", "scopes": []string{"groups:write"}, "expiry_days": 7,
	})
	assertInsufficientScope(t, rejected, spec.InteractiveSessionScope)
	if stored := storedApiTokens(t, s); len(stored) != 1 {
		t.Fatalf("stored tokens = %d, want only the presenting token", len(stored))
	}
}

//spec:covers EX-APITOKENS-004-05: 対応スコープを持つ API アクセストークンでも、発行者が admin ロールを失った後は参照と変更の双方を insufficient_scope ではなく access_denied で拒否し、グループの保存先が変わらないこと。ロールを失う前に同じトークンが通ることで、拒否がロールによるものだと示す。
func TestAdminApiRejectsATokenWhoseIssuerLostTheRole(t *testing.T) {
	s := newExampleStack(t)
	token, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeGroupsRead, apitokendomain.ScopeGroupsWrite)
	if before := bearer(t, s, http.MethodGet, groupsPath, token, nil); before.Code != http.StatusOK {
		t.Fatalf("before losing the role status=%d body=%s, want 200", before.Code, before.Body.String())
	}

	issuer, err := s.Users.FindBySub(context.Background(), stack.AdminUserID)
	if err != nil || issuer == nil {
		t.Fatalf("find issuer: %v", err)
	}
	demoted := *issuer
	demoted.Roles = []string{"user"}
	s.Users.Seed(&demoted)

	for _, tc := range []struct {
		method string
		body   any
	}{
		{http.MethodGet, nil},
		{http.MethodPost, map[string]any{"name": "not-created"}},
	} {
		rejected := bearer(t, s, tc.method, groupsPath, token, tc.body)
		if rejected.Code != http.StatusForbidden || problemType(t, rejected) != "urn:idmagic:error:access_denied" {
			t.Fatalf("%s status=%d body=%s, want 403 access_denied", tc.method, rejected.Code, rejected.Body.String())
		}
	}
	if names := storedGroupNames(t, s); len(names) != 0 {
		t.Fatalf("stored groups = %v, want none after the refusal", names)
	}
}

func assertInsufficientScope(t *testing.T, recorder *httptest.ResponseRecorder, required string) {
	t.Helper()
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", recorder.Code, recorder.Body.String())
	}
	// 後ろに RFC 9728 の resource_metadata が続くので、error と scope の 2 つだけを読む。
	want := `Bearer error="insufficient_scope", scope="` + required + `"`
	if got := recorder.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, want) {
		t.Fatalf("WWW-Authenticate = %q, want it to start with %q", got, want)
	}
	if got := problemType(t, recorder); got != "urn:idmagic:error:insufficient_scope" {
		t.Fatalf("problem type = %q, want insufficient_scope", got)
	}
}
