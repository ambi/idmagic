package server_http

// docs/contexts/api-tokens/standards.md が宣言する行を、製品の正式な入口から観測する。
//
// API アクセストークンの検証はミドルウェアの位置にあるので、入口は保護されたエンドポイントで
// なければならない。トークン検証関数だけを呼ぶテストは、関数が正しくても配線されていない
// 実装を素通りさせる。ここでは Register が組み立てた経路をそのまま通す。

import (
	"context"
	cryptostd "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/apitoken"
	apitokenmemory "github.com/ambi/idmagic/backend/apitoken/db_memory"
	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	apiTokenIssuer     = "https://idp.example"
	apiTokenAdminSub   = "admin-1"
	apiTokenTargetSub  = "target-1"
	apiTokenRSClientID = "resource-server"
	apiTokenRSSecret   = "resource-server-secret"
	apiTokenOtherRealm = "acme"

	apiTokenListPath      = "/realms/default/api/admin/v1/users"
	apiTokenOtherListPath = "/realms/acme/api/admin/v1/users"
	apiTokenDisablePath   = "/realms/default/api/admin/v1/users/" + apiTokenTargetSub + "/disable"
	apiTokenRevokePath    = "/realms/default/revoke"
)

// apiTokenStack は Register が組み立てた 1 プロセス分のスタックと、テストが状態を読み直す
// ための保存先をまとめて持つ。
type apiTokenStack struct {
	e        *echo.Echo
	repo     apitokenports.Repository
	tokens   *apitokenusecases.Service
	users    *usermemory.UserRepository
	keyStore *signingmemory.InMemoryKeyStore
	signer   *tokensjose.JWTSigner
	tenants  *tenancymemory.TenantRepository
}

func newApiTokenStack(t *testing.T) *apiTokenStack {
	t.Helper()
	created := time.Now().UTC()
	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: created},
		{ID: apiTokenOtherRealm, Realm: apiTokenOtherRealm, DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: created},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}

	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: apiTokenAdminSub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: created, UpdatedAt: created,
	})
	users.Seed(&userdomain.User{
		ID: apiTokenTargetSub, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "target",
		PasswordHash: "unused", Roles: []string{"user"}, CreatedAt: created, UpdatedAt: created,
	})

	clients := oauth2memory.NewClientRepository()
	for _, tenantID := range []string{tenancydomain.DefaultTenantID, apiTokenOtherRealm} {
		secretHash := oauthdomain.HashClientSecret(apiTokenRSSecret)
		clients.Seed(&oauthdomain.OAuth2Client{
			TenantID: tenantID, ClientID: apiTokenRSClientID, ClientSecretHash: &secretHash,
			ClientType: spec.ClientConfidential, TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
			GrantTypes: []spec.GrantType{spec.GrantClientCredentials}, CreatedAt: created,
		})
	}

	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensjose.NewJWTSigner(apiTokenIssuer, keyStore)
	repo := apitokenmemory.NewRepository()

	e := echo.New()
	Register(e, Deps{
		Issuer: apiTokenIssuer, Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, UserRepo: users,
		SigningKeys: signingkeys.Module{KeyStore: keyStore},
		OAuth2: oauth2.Module{
			ClientRepo: clients, TokenIssuer: signer, TokenIntrospector: signer,
			RefreshStore:        oauth2memory.NewRefreshTokenStore(),
			AccessTokenDenylist: oauth2memory.NewAccessTokenDenylist(),
			DpopReplayStore:     oauth2memory.NewDpopReplayStore(),
		},
		ApiTokens: apitoken.Module{Repo: repo, TokenIssuer: signer, TokenIntrospector: signer},
	})
	return &apiTokenStack{
		e: e, repo: repo, users: users, keyStore: keyStore, signer: signer, tenants: tenants,
		tokens: apitokenusecases.New(repo,
			apitokenusecases.WithTokenIssuer(signer), apitokenusecases.WithTokenIntrospector(signer)),
	}
}

// realmContext は middleware が組み立てるのと同じテナント文脈を作る。署名鍵も audience も
// テナントごとなので、この文脈を通さずに発行したトークンは製品が発行するものと別物になる。
func (s *apiTokenStack) realmContext(t *testing.T, realm string) context.Context {
	t.Helper()
	tenant, err := s.tenants.FindByRealm(context.Background(), realm)
	if err != nil || tenant == nil {
		t.Fatalf("realm %q: tenant=%v err=%v", realm, tenant, err)
	}
	prefix := "/realms/" + realm
	return tenancy.WithTenant(context.Background(), tenant, apiTokenIssuer+prefix, prefix)
}

func (s *apiTokenStack) tenantID(t *testing.T, realm string) string {
	t.Helper()
	tenant, err := s.tenants.FindByRealm(context.Background(), realm)
	if err != nil || tenant == nil {
		t.Fatalf("realm %q: tenant=%v err=%v", realm, tenant, err)
	}
	return tenant.ID
}

// issue は管理コンソールと同じ Service を通して API アクセストークンを 1 本発行する。
// 発行のエンドポイントは対話セッション限定なので、テストはトークンを HTTP からは作れない。
func (s *apiTokenStack) issue(t *testing.T, realm, dpopJKT string, scopes ...apitokendomain.Scope) (string, apitokendomain.Metadata) {
	t.Helper()
	return s.issueWithClock(t, realm, dpopJKT, nil, scopes...)
}

// issueWithClock は発行時刻を差し替えて 1 本発行する。期限切れのトークンは、過去に
// 発行されたものでなければ作れない。
func (s *apiTokenStack) issueWithClock(t *testing.T, realm, dpopJKT string, now func() time.Time, scopes ...apitokendomain.Scope) (string, apitokendomain.Metadata) {
	t.Helper()
	service := s.tokens
	if now != nil {
		service = apitokenusecases.New(s.repo,
			apitokenusecases.WithTokenIssuer(s.signer), apitokenusecases.WithTokenIntrospector(s.signer),
			apitokenusecases.WithClock(now))
	}
	literal, metadata, err := service.Issue(
		s.realmContext(t, realm), s.tenantID(t, realm), apiTokenAdminSub, "standards test",
		apitokendomain.Scopes(scopes).Strings(), 1, dpopJKT,
	)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	return literal, metadata
}

func (s *apiTokenStack) do(request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	s.e.ServeHTTP(recorder, request)
	return recorder
}

// listUsers は保護された参照エンドポイントを 1 回叩く。Authorization の値をそのまま渡すので、
// スキームを変えた提示も、ヘッダーを付けない提示も同じ 1 つの関数で書ける。
func (s *apiTokenStack) listUsers(path, authorization string, header http.Header) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, apiTokenIssuer+path, http.NoBody)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	for name, values := range header {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	return s.do(request)
}

// reachedTheAdminAPI は、保護されたエンドポイントが実際に利用者の一覧を返したかを返す。
// 状態コードだけでは、拒否しながら本文を書く実装と区別できない。
func reachedTheAdminAPI(recorder *httptest.ResponseRecorder) bool {
	return recorder.Code == http.StatusOK && strings.Contains(recorder.Body.String(), apiTokenTargetSub)
}

// targetIsStillActive は、拒否が防いだ効果を保存先から読み直す。応答だけを見るテストは、
// 拒否を書いてから状態を変える実装を見分けられない。
func (s *apiTokenStack) targetIsStillActive(t *testing.T) bool {
	t.Helper()
	user, err := s.users.FindBySub(s.realmContext(t, tenancydomain.DefaultRealm), apiTokenTargetSub)
	if err != nil || user == nil {
		t.Fatalf("target user: %v %v", user, err)
	}
	return user.IsActive()
}

// decodeJWT は JWT の header と payload を復号する。署名は検証しない。
func decodeJWT(t *testing.T, token string) (map[string]any, map[string]any) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token is not a JWS compact serialization: %q", token)
	}
	decode := func(segment string) map[string]any {
		raw, err := base64.RawURLEncoding.DecodeString(segment)
		if err != nil {
			t.Fatalf("decode %q: %v", segment, err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("parse %s: %v", raw, err)
		}
		return out
	}
	return decode(parts[0]), decode(parts[1])
}

// resignManaged は、発行済みトークンの payload を土台に、指定した claim だけを差し替えた
// 管理発行トークンを default レルムの現行署名鍵で作る。値だけが違う正しい形式のトークンでないと、
// 形式の検証で落ちて、確かめたい照合まで届かない。
func (s *apiTokenStack) resignManaged(t *testing.T, literal string, overrides map[string]any) string {
	t.Helper()
	_, claims := decodeJWT(t, literal)
	for name, value := range overrides {
		if value == nil {
			delete(claims, name)
			continue
		}
		claims[name] = value
	}
	key, err := s.keyStore.GetActiveKey(s.realmContext(t, tenancydomain.DefaultRealm))
	if err != nil {
		t.Fatal(err)
	}
	token, err := tokensjose.SignPS256(key, map[string]string{"typ": "at+jwt"}, claims)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func (s *apiTokenStack) postForm(t *testing.T, path string, form url.Values, basicAuth bool) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, apiTokenIssuer+path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if basicAuth {
		request.SetBasicAuth(apiTokenRSClientID, apiTokenRSSecret)
	}
	return s.do(request)
}

func (s *apiTokenStack) introspect(t *testing.T, realm, token string) map[string]any {
	t.Helper()
	recorder := s.postForm(t, "/realms/"+realm+"/introspect", url.Values{
		"token": {token}, "token_type_hint": {"access_token"},
	}, true)
	if recorder.Code != http.StatusOK {
		t.Fatalf("introspect status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("introspect body %s: %v", recorder.Body.String(), err)
	}
	return body
}

// =====================================================================
// RFC 6750 — Bearer Token Usage
// =====================================================================

// RFC6750-API-TOKEN-HEADER: API アクセストークンを受け付ける提示の形が Authorization ヘッダーの
// Bearer と DPoP スキームだけであることを固定する。同じ 1 本のトークンを、スキームだけ変えて
// 提示する。トークンの側は毎回有効なので、到達できたかどうかの差は提示の形だけで決まる。
func TestApiTokenIsAcceptedOnlyFromTheAuthorizationHeaderScheme(t *testing.T) {
	stack := newApiTokenStack(t)
	literal, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)

	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "Bearer "+literal, nil)) {
		t.Fatal("前提が壊れている: Bearer スキームで管理 API へ到達できない")
	}

	for _, tc := range []struct {
		name          string
		authorization string
		accepted      bool
	}{
		// スキーム名の大文字小文字は同じスキームである。ここが落ちるなら、拒否の理由が
		// 「スキームが違う」ではなく「文字列が一致しない」になっている。
		{name: "lowercase bearer", authorization: "bearer " + literal, accepted: true},
		{name: "Basic scheme", authorization: "Basic " + literal},
		{name: "Token scheme", authorization: "Token " + literal},
		{name: "no scheme", authorization: literal},
		{name: "no Authorization header", authorization: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := stack.listUsers(apiTokenListPath, tc.authorization, nil)
			if tc.accepted {
				if !reachedTheAdminAPI(recorder) {
					t.Fatalf("status=%d body=%s, want the same scheme to be accepted", recorder.Code, recorder.Body.String())
				}
				return
			}
			if reachedTheAdminAPI(recorder) {
				t.Fatalf("status=%d body=%s, want the presentation to be refused", recorder.Code, recorder.Body.String())
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body=%s)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

// RFC6750-API-TOKEN-QUERY: URI クエリパラメーターによる提示を提供していないことを固定する。
// 提供していないことの観測なので、`excluded` の行は満たすことではなく満たさないことを読む。
//
// 受理されない一点だけでは、そのトークンが最初から無効だった実装と区別できない。そこで
// 同じ 1 本のトークンがヘッダーでは通ることを先に確かめ、クエリでは通らないことと対にする。
// 参照だけでなく状態を変える操作でも確かめ、拒否が防いだ効果 (利用者が無効化されていないこと)
// を保存先から読み直す。
func TestApiTokenIsNotAcceptedFromTheQueryString(t *testing.T) {
	stack := newApiTokenStack(t)
	literal, _ := stack.issue(t, tenancydomain.DefaultRealm, "",
		apitokendomain.ScopeUsersRead, apitokendomain.ScopeUsersWrite)

	// 対照 1: ヘッダーに載せた同じトークンは参照へ届く。
	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "Bearer "+literal, nil)) {
		t.Fatal("前提が壊れている: ヘッダーに載せたトークンで参照へ到達できない")
	}

	// クエリに載せた同じトークンでは参照へ届かない。
	queryRead := stack.listUsers(apiTokenListPath+"?access_token="+url.QueryEscape(literal), "", nil)
	if reachedTheAdminAPI(queryRead) {
		t.Fatalf("クエリのトークンで参照へ到達した: status=%d body=%s", queryRead.Code, queryRead.Body.String())
	}

	// クエリに載せた同じトークンでは状態も変わらない。CSRF と Origin は満たしておく。
	// 満たさないと、拒否の理由がクエリの扱いではなくブラウザー検証になってしまう。
	if !stack.targetIsStillActive(t) {
		t.Fatal("前提が壊れている: 対象の利用者が最初から無効")
	}
	queryWrite := stack.disableTarget(t, apiTokenDisablePath+"?access_token="+url.QueryEscape(literal), "")
	if queryWrite.Code == http.StatusNoContent {
		t.Fatalf("クエリのトークンで無効化が通った: body=%s", queryWrite.Body.String())
	}
	if queryWrite.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body=%s)", queryWrite.Code, queryWrite.Body.String())
	}
	if !stack.targetIsStillActive(t) {
		t.Fatal("クエリのトークンは拒否されたのに、利用者が無効化されている")
	}

	// 対照 2: 同じ操作をヘッダーで行えば通り、状態が変わる。ここが通らないと、上の 401 が
	// 「クエリを受け付けない」ではなく「その操作がそもそも通らない」ことの観測になる。
	headerWrite := stack.disableTarget(t, apiTokenDisablePath, "Bearer "+literal)
	if headerWrite.Code != http.StatusNoContent {
		t.Fatalf("前提が壊れている: ヘッダーでの無効化 status=%d body=%s", headerWrite.Code, headerWrite.Body.String())
	}
	if stack.targetIsStillActive(t) {
		t.Fatal("前提が壊れている: ヘッダーでの無効化が状態へ届いていない")
	}
}

// disableTarget は利用者の無効化を 1 回要求する。Authorization を持たない要求でも
// ブラウザー検証 (Origin + double-submit CSRF) だけは満たしておく。
func (s *apiTokenStack) disableTarget(t *testing.T, path, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, apiTokenIssuer+path, http.NoBody)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	request.Header.Set("Origin", apiTokenIssuer)
	request.Header.Set(support.CSRFHeader, "csrf-token-value")
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf-token-value"})
	return s.do(request)
}

// =====================================================================
// RFC 9068 — JWT Profile for OAuth 2.0 Access Tokens
// =====================================================================

// RFC9068-API-TOKEN-CLAIMS: 管理発行トークンが 8 つの claim を持つことを固定する。
// 復号した payload をそのまま読む。管理 API へ到達できることだけでは、認証が使わない
// claim (`iat`) を落とした実装を見分けられない。
func TestManagedApiTokenCarriesTheRFC9068Claims(t *testing.T) {
	stack := newApiTokenStack(t)
	literal, metadata := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)
	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "Bearer "+literal, nil)) {
		t.Fatal("前提が壊れている: 発行したトークンで管理 API へ到達できない")
	}
	_, claims := decodeJWT(t, literal)

	for _, tc := range []struct {
		claim string
		want  any
	}{
		{claim: "iss", want: apiTokenIssuer + "/realms/default"},
		{claim: "sub", want: apiTokenAdminSub},
		{claim: "aud", want: apiTokenIssuer + "/realms/default"},
		{claim: "client_id", want: apitokendomain.BuiltinClientID},
		{claim: "scope", want: string(apitokendomain.ScopeUsersRead)},
		{claim: "jti", want: metadata.JTI},
	} {
		if got := claims[tc.claim]; got != tc.want {
			t.Errorf("%s = %v, want %v", tc.claim, got, tc.want)
		}
	}

	issuedAt, ok := claims["iat"].(float64)
	if !ok || issuedAt <= 0 {
		t.Errorf("iat = %v, want the issuance time", claims["iat"])
	}
	expiresAt, ok := claims["exp"].(float64)
	if !ok || metadata.ExpiresAt == nil || int64(expiresAt) != metadata.ExpiresAt.Unix() {
		t.Errorf("exp = %v, want the lifecycle record's expiry %v", claims["exp"], metadata.ExpiresAt)
	}
	if expiresAt <= issuedAt {
		t.Errorf("exp %v is not after iat %v", expiresAt, issuedAt)
	}
}

// RFC9068-API-TOKEN-SIGNATURE: 管理発行トークンが通常の OAuth アクセストークンと同じ
// 非対称鍵で署名され、`typ` が `at+jwt` であることを固定する。
//
// header の値を読むだけでは、署名を検証していない実装を見分けられない。そこで、同じ kid を
// 名乗りながら別の鍵で署名したトークンが管理 API へ届かないことを併せて観測する。
func TestManagedApiTokenIsSignedWithTheTenantAccessTokenKey(t *testing.T) {
	stack := newApiTokenStack(t)
	ctx := stack.realmContext(t, tenancydomain.DefaultRealm)
	literal, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)
	header, _ := decodeJWT(t, literal)

	if got := header["typ"]; got != "at+jwt" {
		t.Errorf("typ = %v, want at+jwt", got)
	}
	if got := header["alg"]; got != string(signingdomain.SigAlgPS256) {
		t.Errorf("alg = %v, want PS256", got)
	}

	// 通常の OAuth アクセストークンを同じ signer から 1 本出し、鍵が同じであることを読む。
	ordinary, _, err := stack.signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{TenantID: tenancydomain.DefaultTenantID, ClientID: apiTokenRSClientID},
		Sub:    apiTokenAdminSub, Scopes: []string{"openid"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ordinaryHeader, _ := decodeJWT(t, ordinary)
	if header["kid"] == nil || header["kid"] != ordinaryHeader["kid"] {
		t.Fatalf("managed kid = %v, ordinary access token kid = %v; want the same key", header["kid"], ordinaryHeader["kid"])
	}

	active, err := stack.keyStore.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if header["kid"] != active.Kid {
		t.Fatalf("kid = %v, want the tenant's active signing key %q", header["kid"], active.Kid)
	}
	if err := verifyPS256WithKey(t, literal, active); err != nil {
		t.Fatalf("the tenant's active public key does not verify the managed token: %v", err)
	}

	// 同じ claim を、同じ kid を名乗る別の鍵で署名したトークンは届かない。
	foreign, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	_, claims := decodeJWT(t, literal)
	forged, err := tokensjose.SignPS256(
		&signingdomain.SigningKey{Kid: active.Kid, Alg: signingdomain.SigAlgPS256, PrivateKey: foreign},
		map[string]string{"typ": "at+jwt"}, claims,
	)
	if err != nil {
		t.Fatal(err)
	}
	if recorder := stack.listUsers(apiTokenListPath, "Bearer "+forged, nil); reachedTheAdminAPI(recorder) {
		t.Fatalf("別の鍵で署名したトークンが管理 API へ到達した: status=%d", recorder.Code)
	}
	// 対照: 差し替えたのが鍵だけであることを、正しい鍵で署名し直した同じ claim で確かめる。
	if recorder := stack.listUsers(apiTokenListPath, "Bearer "+stack.resignManaged(t, literal, nil), nil); !reachedTheAdminAPI(recorder) {
		t.Fatalf("前提が壊れている: 同じ claim をテナントの鍵で署名し直したトークンが届かない: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func verifyPS256WithKey(t *testing.T, token string, key *signingdomain.SigningKey) error {
	t.Helper()
	parts := strings.Split(token, ".")
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	public, ok := key.PublicKey.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("active key is not RSA: %T", key.PublicKey)
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	return rsa.VerifyPSS(public, cryptostd.SHA256, digest[:], signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
}

// =====================================================================
// RFC 9700 / BCP 240 — audience と送信者制約
// =====================================================================

// RFC9700-API-TOKEN-AUDIENCE: API アクセストークンが発行元レルムの API audience に固定され、
// 別のレルムまたはリソースでは拒否されることを固定する。
//
// 壊れた文字列では形式の検証で落ちるので、audience の照合が無い実装でも同じ拒否になる。
// そこで aud だけが違う、他はすべて有効なトークンをテナントの現行鍵で作って提示する。
func TestApiTokenIsBoundToTheIssuingRealmAudience(t *testing.T) {
	stack := newApiTokenStack(t)
	literal, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)

	// 対照: aud を差し替えない再署名は届く。差分が aud だけであることの担保になる。
	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "Bearer "+stack.resignManaged(t, literal, nil), nil)) {
		t.Fatal("前提が壊れている: aud を変えない再署名が届かない")
	}

	for _, tc := range []struct {
		name string
		aud  any
	}{
		{name: "another realm's API audience", aud: apiTokenIssuer + "/realms/acme"},
		{name: "another resource", aud: "https://api.example/resource"},
		{name: "the right audience among several", aud: []any{apiTokenIssuer + "/realms/default", "https://api.example/resource"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forged := stack.resignManaged(t, literal, map[string]any{"aud": tc.aud})
			if recorder := stack.listUsers(apiTokenListPath, "Bearer "+forged, nil); reachedTheAdminAPI(recorder) {
				t.Fatalf("aud=%v のトークンが管理 API へ到達した: status=%d", tc.aud, recorder.Code)
			}
		})
	}

	// 発行元でないレルムの同じ管理 API へ提示しても届かない。
	if recorder := stack.listUsers(apiTokenOtherListPath, "Bearer "+literal, nil); recorder.Code == http.StatusOK {
		t.Fatalf("別レルムの管理 API へ到達した: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

// RFC9700-API-TOKEN-SENDER-CONSTRAINT: 送信者制約を発行時に選べることを固定する。
// 選べることの観測なので、制約なしと制約ありの 2 本を同じ提示で比べる。制約ありの 1 本だけを
// 見ても、常に DPoP を要求する実装と区別できない。
func TestApiTokenSenderConstraintIsChosenAtIssuance(t *testing.T) {
	stack := newApiTokenStack(t)
	key, jwk, jkt := newApiTokenDPoPKey(t)

	unconstrained, unconstrainedMeta := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)
	constrained, constrainedMeta := stack.issue(t, tenancydomain.DefaultRealm, jkt, apitokendomain.ScopeUsersRead)

	if unconstrainedMeta.DPoPJKT != "" {
		t.Errorf("制約なしで発行した記録が dpop_jkt=%q を持っている", unconstrainedMeta.DPoPJKT)
	}
	if constrainedMeta.DPoPJKT != jkt {
		t.Errorf("dpop_jkt = %q, want %q", constrainedMeta.DPoPJKT, jkt)
	}

	// 制約なしのトークンは、証明を添えない Bearer の提示で通る。
	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "Bearer "+unconstrained, nil)) {
		t.Fatal("制約なしのトークンが証明なしで通らない")
	}
	// 制約ありのトークンは、同じ提示では通らない。
	if recorder := stack.listUsers(apiTokenListPath, "Bearer "+constrained, nil); reachedTheAdminAPI(recorder) {
		t.Fatalf("制約ありのトークンが証明なしで通った: status=%d", recorder.Code)
	}
	// 制約ありのトークンは、鍵の所持を証明すれば通る。
	proof := apiTokenDPoPProof(t, key, jwk, dpopProofInput{
		htm: http.MethodGet, htu: apiTokenListPath, jti: "sender-constraint-1",
		ath: tokensjose.AccessTokenHash(constrained), iat: time.Now().UTC(),
	})
	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "DPoP "+constrained, http.Header{"DPoP": {proof}})) {
		t.Fatal("制約ありのトークンが、鍵を証明しても通らない")
	}
	// cnf は発行したトークン自身に載る。記録だけに載る実装では、リソースサーバーが
	// 制約の存在をトークンから読めない。
	_, claims := decodeJWT(t, constrained)
	cnf, ok := claims["cnf"].(map[string]any)
	if !ok || cnf["jkt"] != jkt {
		t.Errorf("cnf = %v, want jkt %q", claims["cnf"], jkt)
	}
}

// =====================================================================
// RFC 9449 — DPoP
// =====================================================================

type dpopProofInput struct {
	htm, htu, jti, ath string
	iat                time.Time
	typ, alg           string
}

func newApiTokenDPoPKey(t *testing.T) (*rsa.PrivateKey, map[string]any, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(new(big.Int).SetInt64(int64(key.PublicKey.E)).Bytes()),
	}
	canonical, err := json.Marshal(map[string]any{"e": jwk["e"], "kty": jwk["kty"], "n": jwk["n"]})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return key, jwk, base64.RawURLEncoding.EncodeToString(sum[:])
}

func apiTokenDPoPProof(t *testing.T, key *rsa.PrivateKey, jwk map[string]any, in dpopProofInput) string {
	t.Helper()
	typ, alg := in.typ, in.alg
	if typ == "" {
		typ = "dpop+jwt"
	}
	if alg == "" {
		alg = "PS256"
	}
	header, err := json.Marshal(map[string]any{"typ": typ, "alg": alg, "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"htm": in.htm, "htu": in.htu, "jti": in.jti, "iat": in.iat.Unix()}
	if in.ath != "" {
		claims["ath"] = in.ath
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, key, cryptostd.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// RFC9449-API-TOKEN-DPOP: `dpop_jkt` に束縛したトークンについて、DPoP 証明の署名、`htm`、
// `htu`、`iat`、`jti` のリプレイ、およびサムプリントの一致が検証されることを固定する。
//
// 行が挙げる要素ごとに、1 要素だけを崩した証明を作る。崩していない要素が有効であることは、
// 無傷の証明が通ることで先に確かめる。まとめて壊した証明では、どの検証が働いたのか分からない。
func TestDPoPBoundApiTokenVerifiesEveryProofElement(t *testing.T) {
	stack := newApiTokenStack(t)
	key, jwk, jkt := newApiTokenDPoPKey(t)
	otherKey, otherJWK, _ := newApiTokenDPoPKey(t)
	literal, _ := stack.issue(t, tenancydomain.DefaultRealm, jkt, apitokendomain.ScopeUsersRead)
	ath := tokensjose.AccessTokenHash(literal)
	now := time.Now().UTC()

	present := func(proof string) *httptest.ResponseRecorder {
		return stack.listUsers(apiTokenListPath, "DPoP "+literal, http.Header{"DPoP": {proof}})
	}
	// 無傷の証明が持つ htu は、絶対 URL ではなくパスである。保護リソースの検証がそちらを
	// 期待しているからであり、その形が正しいという判断ではない。絶対 URL を送る適合クライアントが
	// 通らないことは [[wi-511-dpop-proof-htu-at-protected-resources-is-not-the-target-uri]] が扱う。ここが固定しているのは、htu が照合されること自体である。
	intact := func(jti string) dpopProofInput {
		return dpopProofInput{htm: http.MethodGet, htu: apiTokenListPath, jti: jti, ath: ath, iat: now}
	}

	if !reachedTheAdminAPI(present(apiTokenDPoPProof(t, key, jwk, intact("intact-1")))) {
		t.Fatal("前提が壊れている: 無傷の DPoP 証明が通らない")
	}

	// リプレイ: 直前に通った証明をそのまま送り直す。jti が同じでも通る実装をここで落とす。
	replayed := apiTokenDPoPProof(t, key, jwk, intact("replayed-1"))
	if !reachedTheAdminAPI(present(replayed)) {
		t.Fatal("前提が壊れている: 1 回目の提示が通らない")
	}
	if recorder := present(replayed); reachedTheAdminAPI(recorder) {
		t.Fatalf("同じ jti の証明が 2 回通った: status=%d", recorder.Code)
	}

	for _, tc := range []struct {
		name  string
		proof func() string
	}{
		{name: "signature made by another key", proof: func() string {
			// jwk は束縛された鍵のまま、署名だけを別の鍵で作る。サムプリントは一致するので、
			// 落ちるとすれば署名の検証しかない。
			return apiTokenDPoPProof(t, otherKey, jwk, intact("bad-signature"))
		}},
		{name: "thumbprint of another key", proof: func() string {
			// 署名と jwk は整合しているが、束縛されたサムプリントとは一致しない。
			return apiTokenDPoPProof(t, otherKey, otherJWK, intact("bad-thumbprint"))
		}},
		{name: "htm of another method", proof: func() string {
			in := intact("bad-htm")
			in.htm = http.MethodPost
			return apiTokenDPoPProof(t, key, jwk, in)
		}},
		{name: "htu of another resource", proof: func() string {
			in := intact("bad-htu")
			in.htu = apiTokenDisablePath
			return apiTokenDPoPProof(t, key, jwk, in)
		}},
		{name: "iat outside the clock skew", proof: func() string {
			in := intact("stale-iat")
			in.iat = now.Add(-2 * time.Hour)
			return apiTokenDPoPProof(t, key, jwk, in)
		}},
		{name: "no jti", proof: func() string {
			in := intact("")
			return apiTokenDPoPProof(t, key, jwk, in)
		}},
		{name: "ath of another access token", proof: func() string {
			in := intact("bad-ath")
			in.ath = tokensjose.AccessTokenHash("another-access-token")
			return apiTokenDPoPProof(t, key, jwk, in)
		}},
		{name: "no proof at all", proof: func() string { return "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := present(tc.proof())
			if reachedTheAdminAPI(recorder) {
				t.Fatalf("崩した証明で管理 API へ到達した: status=%d", recorder.Code)
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401 (body=%s)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

// =====================================================================
// RFC 7662 — Token Introspection
// =====================================================================

// RFC7662-API-TOKEN-INTROSPECT: 認証済みリソースサーバーへ返す内省の内容を固定する。
// 返した値が発行したトークンのものであることを 1 つずつ照合する。`active` だけを読むテストは、
// 別のトークンの内容を返す実装も、`scope` を落とす実装も見分けられない。
func TestApiTokenIntrospectionReturnsTheIssuedTokenClaims(t *testing.T) {
	stack := newApiTokenStack(t)
	_, _, jkt := newApiTokenDPoPKey(t)
	literal, metadata := stack.issue(t, tenancydomain.DefaultRealm, jkt,
		apitokendomain.ScopeUsersRead, apitokendomain.ScopeGroupsRead)
	_, claims := decodeJWT(t, literal)

	// リソースサーバーの認証が無い内省は成立しない。
	unauthenticated := stack.postForm(t, "/realms/default/introspect", url.Values{"token": {literal}}, false)
	if unauthenticated.Code == http.StatusOK {
		t.Fatalf("クライアント認証なしの内省が 200 を返した: body=%s", unauthenticated.Body.String())
	}

	body := stack.introspect(t, tenancydomain.DefaultRealm, literal)
	if body["active"] != true {
		t.Fatalf("active = %v, want true (body=%v)", body["active"], body)
	}
	if got := body["scope"]; got != claims["scope"] {
		t.Errorf("scope = %v, want %v", got, claims["scope"])
	}
	if got := body["sub"]; got != apiTokenAdminSub {
		t.Errorf("sub = %v, want %v", got, apiTokenAdminSub)
	}
	if got, ok := body["aud"].([]any); !ok || len(got) != 1 || got[0] != metadata.Audience {
		t.Errorf("aud = %v, want [%v]", body["aud"], metadata.Audience)
	}
	if got := body["jti"]; got != metadata.JTI {
		t.Errorf("jti = %v, want %v", got, metadata.JTI)
	}
	if got := body["iat"]; got != claims["iat"] {
		t.Errorf("iat = %v, want %v", got, claims["iat"])
	}
	if got := body["exp"]; got != claims["exp"] {
		t.Errorf("exp = %v, want %v", got, claims["exp"])
	}
	// cnf は任意だが、送信者制約を持つトークンでは返さなければ、リソースサーバーが
	// 制約の存在を知る手段が無くなる。
	cnf, ok := body["cnf"].(map[string]any)
	if !ok || cnf["jkt"] != jkt {
		t.Errorf("cnf = %v, want jkt %q", body["cnf"], jkt)
	}
}

// 内省が非活性を返す 4 通りの入力について、`active=false` だけを返すことを固定する。
// `active` が false であることに加えて、応答が他の鍵を 1 つも持たないことを読む。
// 4 通りの入力が同じ 1 つの本文になることが、存在を漏らさないということである。
//
// このテストは、api-tokens の standards.md が内省の非活性について宣言している行を名指さない。
// 行は失効済みのトークン全般について非活性を要求しているが、管理コンソールから失効させた
// トークンの内省はいま `active=true` と全 claim を返す。行を満たさないので、名指しは
// [[wi-510-introspection-ignores-the-managed-token-lifecycle-record]] の修正を待つ。ここで
// 観測しているのは、RFC 7009 の `/revoke` を通した失効を含む 4 通りだけである。
func TestApiTokenIntrospectionRevealsNothingAboutInactiveTokens(t *testing.T) {
	stack := newApiTokenStack(t)

	revoked, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)
	if body := stack.introspect(t, tenancydomain.DefaultRealm, revoked); body["active"] != true {
		t.Fatalf("前提が壊れている: 失効前の内省が active=%v", body["active"])
	}
	if recorder := stack.revoke(t, revoked); recorder.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: revoke status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	expired, _ := stack.issueWithClock(t, tenancydomain.DefaultRealm, "",
		func() time.Time { return time.Now().UTC().Add(-48 * time.Hour) }, apitokendomain.ScopeUsersRead)

	otherRealm, _ := stack.issue(t, apiTokenOtherRealm, "", apitokendomain.ScopeUsersRead)

	for _, tc := range []struct {
		name  string
		realm string
		token string
	}{
		{name: "unknown", realm: tenancydomain.DefaultRealm, token: "not.a.token"},
		{name: "revoked", realm: tenancydomain.DefaultRealm, token: revoked},
		{name: "expired", realm: tenancydomain.DefaultRealm, token: expired},
		{name: "issued by another realm", realm: tenancydomain.DefaultRealm, token: otherRealm},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := stack.introspect(t, tc.realm, tc.token)
			if body["active"] != false {
				t.Fatalf("active = %v, want false (body=%v)", body["active"], body)
			}
			if len(body) != 1 {
				t.Fatalf("body = %v, want only the active member", body)
			}
		})
	}
}

// =====================================================================
// RFC 7009 — Token Revocation
// =====================================================================

// revoke は組み込みの公開クライアント ID で /revoke を 1 回叩く。
func (s *apiTokenStack) revoke(t *testing.T, token string) *httptest.ResponseRecorder {
	t.Helper()
	return s.postForm(t, apiTokenRevokePath, url.Values{
		"token": {token}, "token_type_hint": {"access_token"}, "client_id": {apitokendomain.BuiltinClientID},
	}, false)
}

// RFC7009-API-TOKEN-REVOKE: `access_token` ヒントと組み込みの公開クライアント ID で提示した
// 管理発行 JWT が即時に失効することを固定する。
//
// 失効は失効前の成功と対で観測する。失効後の 401 だけでは、そのトークンが最初から
// 通らなかった実装と区別できない。応答に加えて、ライフサイクル記録の `revoked_at` と、
// 保護されたエンドポイントへの到達可否の 3 つを読む。
func TestRevokingAManagedApiTokenTakesEffectImmediately(t *testing.T) {
	stack := newApiTokenStack(t)
	literal, metadata := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)

	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "Bearer "+literal, nil)) {
		t.Fatal("前提が壊れている: 失効前のトークンで管理 API へ到達できない")
	}

	recorder := stack.revoke(t, literal)
	if recorder.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	if after := stack.listUsers(apiTokenListPath, "Bearer "+literal, nil); reachedTheAdminAPI(after) {
		t.Fatalf("失効させたトークンで管理 API へ到達した: status=%d", after.Code)
	}
	if body := stack.introspect(t, tenancydomain.DefaultRealm, literal); body["active"] != false {
		t.Fatalf("失効させたトークンの内省が active=%v", body["active"])
	}
	// 記録の側にも失効が届いている。届いていなければ、失効は署名鍵を回すまで解けない
	// denylist だけの話になり、記録を読む経路 (一覧、認証) には効かない。
	list, err := stack.tokens.List(stack.realmContext(t, tenancydomain.DefaultRealm), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range list {
		if item.JTI != metadata.JTI {
			continue
		}
		found = true
		if item.RevokedAt == nil {
			t.Fatalf("ライフサイクル記録に revoked_at が立っていない: %+v", item)
		}
	}
	if !found {
		t.Fatalf("失効させたトークンの記録が見つからない: %+v", list)
	}
}

// RFC7009-API-TOKEN-UNKNOWN: 未知または失効済みのトークンの失効要求も 200 の何もしない
// 処理であり、存在を漏らさないことを固定する。
//
// 3 通りの入力の応答が状態コードも本文も区別できないことを読む。片方だけを読むテストは、
// 未知のトークンに 400 を返す実装を見分けられるが、本文で存在を漏らす実装は見逃す。
// 併せて、未知のトークンの失効要求が他のトークンを巻き添えにしないことを観測する。
func TestRevokingAnUnknownApiTokenIsAnIndistinguishableNoOp(t *testing.T) {
	stack := newApiTokenStack(t)
	live, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)
	doomed, _ := stack.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)

	known := stack.revoke(t, doomed)
	if known.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 既知のトークンの失効 status=%d body=%s", known.Code, known.Body.String())
	}

	for _, tc := range []struct{ name, token string }{
		{name: "unparsable", token: "not.a.token"},
		{name: "well formed but never issued", token: stack.neverIssuedToken(t)},
		{name: "already revoked", token: doomed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := stack.revoke(t, tc.token)
			if recorder.Code != known.Code {
				t.Fatalf("status = %d, want %d (the status of revoking a token that exists)", recorder.Code, known.Code)
			}
			if recorder.Body.String() != known.Body.String() {
				t.Fatalf("body = %q, want %q (the body of revoking a token that exists)", recorder.Body.String(), known.Body.String())
			}
		})
	}

	// 何もしないというのは、他のトークンにも触らないということである。
	if !reachedTheAdminAPI(stack.listUsers(apiTokenListPath, "Bearer "+live, nil)) {
		t.Fatal("未知のトークンの失効要求が、別の有効なトークンを巻き添えにした")
	}
}

// neverIssuedToken は、発行された形をしているが記録を持たない管理発行 JWT を作る。
// 壊れた文字列だけでは、形式の検証で落ちる経路しか通らない。
func (s *apiTokenStack) neverIssuedToken(t *testing.T) string {
	t.Helper()
	literal, _ := s.issue(t, tenancydomain.DefaultRealm, "", apitokendomain.ScopeUsersRead)
	return s.resignManaged(t, literal, map[string]any{"jti": "never-issued-jti"})
}
