package server_http_test

// docs/contexts/oauth2/standards.md のうち、クライアントの登録と、成立した認証方法の
// 記録を定める 2 行を観測する。
//
// 入口は Register が組み立てたスタックの `/register` と、`/authorize` からログインを
// 経て `/token` へ至る認可コードフローである。登録の使用例を組み立てる関数の単体
// テストでは代わりにならない。関数が正しくても、その結果が保存されず配線もされて
// いなければ、行が言っている振る舞いは 1 度も起きないからである。
//
// FAPI 2.0 の 4 行は同じ入口を共有するが、
// `fapi_security_profile_e2e_test.go` が実装と同時に消化した。
//
// 応答を読むだけでは足りない。`/register` は `client_id` を返すだけなら採番して
// 捨てる実装でも通ってしまうし、`amr` は成立していない方法を並べても「値がある」
// ことは満たしてしまう。2 行とも、返ったものが実際に効くことを対にして読む。

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/authentication"
	authndomain "github.com/ambi/idmagic/backend/authentication/domain"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	testingpasswords "github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	cpIssuer       = "http://test"
	cpRealmPath    = "/realms/default"
	cpClientID     = "client-profile-standards-client"
	cpClientSecret = "client-profile-standards-client-secret"
	cpRedirectURI  = "https://app.example/cb"
	cpScope        = "openid profile email"
	cpUsername     = "alice"
	cpUserID       = "user_alice"
	cpPassword     = "client-profile-standards-password-1234"
	cpVerifier     = "client-profile-standards-verifier-0123456789"
)

type cpFixture struct {
	base string
}

func newClientProfileFixture(t *testing.T) *cpFixture {
	t.Helper()
	now := time.Now().UTC()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(cpClientSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: cpClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{cpRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    cpScope,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
		UpdatedAt:                now,
	})

	hasher := testingpasswords.NewHasher()
	passwordHash, err := hasher.Hash(cpPassword)
	if err != nil {
		t.Fatal(err)
	}
	email := "alice@example.test"
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: cpUserID, PreferredUsername: cpUsername, PasswordHash: passwordHash,
		Email: &email, EmailVerified: true,
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
	})

	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(t.Context(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensJOSE.NewJWTSigner(cpIssuer, keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())

	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          cpIssuer,
		TenantRepo:      tenants,
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		Contract:        spec.CurrentRuntimeContract(),
		OAuth2: oauth2.Module{
			ClientRepo:                 clients,
			ConsentRepo:                oauth2memory.NewConsentRepository(),
			RequestStore:               oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:                  oauth2memory.NewAuthorizationCodeStore(),
			PARStore:                   oauth2memory.NewPARStore(),
			RefreshStore:               oauth2memory.NewRefreshTokenStore(),
			McpResourceServerRepo:      oauth2memory.NewMcpResourceServerRepository(),
			ClientAssertionReplayStore: oauth2memory.NewClientAssertionReplayStore(),
			DpopReplayStore:            oauth2memory.NewDpopReplayStore(),
			AccessTokenDenylist:        oauth2memory.NewAccessTokenDenylist(),
			TokenIssuer:                signer,
			TokenIntrospector:          signer,
		},
		IdManagement: idmanagement.Module{UserRepo: users},
		Authentication: authentication.Module{
			PasswordHasher: hasher,
			SessionManager: sessionManager,
			AuthnResolver:  sessionManager,
		},
		SigningKeys: signingkeys.Module{KeyStore: keyStore},
	})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	return &cpFixture{base: server.URL + cpRealmPath}
}

// signIn は正式な入口だけを通して 1 回分のトークンを取る。パスワードだけで通し、
// 第二要素は提示しない。
func (f *cpFixture) signIn(t *testing.T, clientID, clientSecret string) map[string]any {
	t.Helper()
	sum := sha256.Sum256([]byte(cpVerifier))
	query := url.Values{
		"client_id": {clientID}, "redirect_uri": {cpRedirectURI},
		"response_type": {"code"}, "scope": {cpScope}, "state": {"state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	client := browserClient(t)
	response, err := client.Get(f.base + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	_ = response.Body.Close()
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, f.base+"/api/auth/transaction")
	next := postJSON[map[string]string](t, client, f.base+"/api/auth/login", transaction.CSRFToken,
		map[string]string{"username": cpUsername, "password": cpPassword})
	redirect := next["redirect_to"]
	if redirect == "" {
		consent := getJSON[struct {
			CSRFToken string `json:"csrf_token"`
		}](t, client, f.base+"/api/auth/transaction")
		result := postJSON[map[string]string](t, client, f.base+"/api/auth/consent", consent.CSRFToken,
			map[string]string{"action": "allow"})
		redirect = result["redirect_to"]
	}
	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("認可コードが返らなかった: %s", redirect)
	}

	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"code_verifier": {cpVerifier}, "redirect_uri": {cpRedirectURI},
		"client_id": {clientID},
	}
	request, err := http.NewRequest(http.MethodPost, f.base+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientSecret))
	tokenResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer func() { _ = tokenResponse.Body.Close() }()
	raw, _ := io.ReadAll(tokenResponse.Body)
	if tokenResponse.StatusCode != http.StatusOK {
		t.Fatalf("/token status=%d body=%s", tokenResponse.StatusCode, raw)
	}
	body := map[string]any{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("/token 応答が JSON ではない: %v body=%s", err, raw)
	}
	return body
}

// cpClaims は JWS Compact Serialization のペイロードを返す。
func cpClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		t.Fatalf("JWT ではない: %q", token)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(segments[1])
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

func cpStrings(t *testing.T, claims map[string]any, field string) []string {
	t.Helper()
	raw, ok := claims[field].([]any)
	if !ok {
		t.Fatalf("%s が配列ではない: %v", field, claims[field])
	}
	values := make([]string, 0, len(raw))
	for _, entry := range raw {
		value, ok := entry.(string)
		if !ok {
			t.Fatalf("%s に文字列でない要素がある: %v", field, entry)
		}
		values = append(values, value)
	}
	return values
}

// RFC7591-REGISTER (optional): クライアントメタデータを受け取り、`client_id` と
// 登録結果を返すことを固定する。
//
// **採番した `client_id` を返すだけでは足りない。** それでは値を作って捨てる実装が
// 通ってしまう。返った `client_id` と `client_secret` で実際に認可コードフローを
// 通し、発行されたトークンがその `client_id` を名乗ることを対にして読む。これに
// より、応答が「登録結果」であって採番の記録でないことが分かる。
func TestDynamicRegistrationReturnsAClientIdentityThatActuallyWorks(t *testing.T) {
	fixture := newClientProfileFixture(t)

	metadata := map[string]any{
		"client_name":                "dynamically registered",
		"client_type":                "confidential",
		"redirect_uris":              []string{cpRedirectURI},
		"grant_types":                []string{"authorization_code"},
		"token_endpoint_auth_method": "client_secret_basic",
		"scope":                      cpScope,
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(fixture.base+"/register", "application/json", strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		t.Fatalf("/register status=%d body=%s", response.StatusCode, body)
	}

	registered := map[string]any{}
	if err := json.Unmarshal(body, &registered); err != nil {
		t.Fatalf("/register 応答が JSON ではない: %v body=%s", err, body)
	}
	clientID, _ := registered["client_id"].(string)
	if clientID == "" {
		t.Fatalf("client_id が返らなかった: %s", body)
	}
	clientSecret, _ := registered["client_secret"].(string)
	if clientSecret == "" {
		t.Fatalf("client_secret が返らなかった: %s", body)
	}
	// 登録結果は送ったメタデータを反映する。採番だけを返す実装はここで落ちる。
	if got, _ := registered["token_endpoint_auth_method"].(string); got != "client_secret_basic" {
		t.Fatalf("登録結果が送ったメタデータを反映していない: %s", body)
	}
	if got := registered["redirect_uris"]; got == nil {
		t.Fatalf("登録結果に redirect_uris が無い: %s", body)
	}

	// 返った識別情報で実際に認可コードフローが通り、トークンがその client_id を
	// 名乗る。ここが落ちるなら、応答は保存されていない値である。
	tokens := fixture.signIn(t, clientID, clientSecret)
	accessToken, _ := tokens["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("登録したクライアントでトークンが取れなかった: %v", tokens)
	}
	claims := cpClaims(t, accessToken)
	if got, _ := claims["client_id"].(string); got != clientID {
		t.Fatalf("発行されたトークンが別のクライアントを名乗っている: got=%q want=%q", got, clientID)
	}
}

// RFC8176-AMR (required): 実際に成立した認証方法を `amr` 値として記録することを
// 固定する。
//
// **成立した方法が載ることと、成立していない方法が載らないことを対で読む。**
// 片方だけでは、宣言された語彙をそのまま並べる実装と区別できない。ここではパスワード
// だけで通すので、`pwd` が載り、提示していない第二要素の値は 1 つも載らない。
func TestIDTokenRecordsOnlyTheAuthenticationMethodsThatActuallyHappened(t *testing.T) {
	fixture := newClientProfileFixture(t)
	tokens := fixture.signIn(t, cpClientID, cpClientSecret)

	idToken, _ := tokens["id_token"].(string)
	if idToken == "" {
		t.Fatalf("ID トークンが返らなかった: %v", tokens)
	}
	amr := cpStrings(t, cpClaims(t, idToken), "amr")

	if !slices.Contains(amr, authndomain.AMRPassword) {
		t.Fatalf("パスワードで通したのに amr に %q が無い: %v", authndomain.AMRPassword, amr)
	}

	// 提示していない方法は 1 つも載らない。語彙のうちパスワード以外を、成立して
	// いない方法として全件読む。1 つだけを選ぶと、残りを並べる実装が生き残る。
	for _, value := range authndomain.AMRVocabulary() {
		if value == authndomain.AMRPassword {
			continue
		}
		if slices.Contains(amr, value) {
			t.Fatalf("成立していない認証方法 %q が amr に載っている: %v", value, amr)
		}
	}

	// アクセストークンにも同じ記録が届く。ID トークンだけを見ると、リライング
	// パーティーが実際に読む経路の片方しか固定できない。
	accessToken, _ := tokens["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("アクセストークンが返らなかった: %v", tokens)
	}
	if got := cpStrings(t, cpClaims(t, accessToken), "amr"); !slices.Equal(got, amr) {
		t.Fatalf("アクセストークンの amr が ID トークンと違う: got=%v want=%v", got, amr)
	}
}
