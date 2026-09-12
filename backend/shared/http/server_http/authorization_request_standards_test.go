package server_http_test

// docs/contexts/oauth2/standards.md のうち、認可リクエストの入口に立つ 15 行を観測する。
//
// 入口は Register が組み立てたスタックへの HTTP である。認可リクエストの検証は
// ハンドラー、use case、domain の 3 層に分かれて立っており、どれか 1 つの単体テストでは
// 「関数は正しいが配線が落ちている」実装を素通りさせる。
//
// 拒否の行では、拒否したことと、拒否が防いだ効果を対で読む。認可リクエストが保存
// されていないこと、認可コードが出ていないこと、トークンが出ていないことのいずれかで
// ある。保存してから拒否する実装は後続の経路へ材料を残すので、応答だけでは足りない。

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	authorizationdomain "github.com/ambi/idmagic/backend/oauth2/authorization/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	arIssuer       = "http://test"
	arClientID     = "authorization-request-client"
	arClientSecret = "authorization-request-client-secret"
	arRedirectURI  = "https://app.example/cb"
	arUsername     = "alice"
	arPassword     = "authorization-request-password-1234"
	arScope        = "openid profile payments"
	arResource     = "https://api.example/payments"
	arDetailType   = "payment_initiation"
)

// countingRequestStore は保存された認可リクエストを数える。「拒否したリクエストは
// 保存しない」を、保存そのものの回数で読む。保存層を後から覗く形にすると、保存して
// 削除した実装と区別できない。
type countingRequestStore struct {
	oauthports.AuthorizationRequestStore
	saves atomic.Int64
}

func (s *countingRequestStore) Save(ctx context.Context, request *domain.AuthorizationRequest) error {
	s.saves.Add(1)
	return s.AuthorizationRequestStore.Save(ctx, request)
}

type arFixture struct {
	server   *httptest.Server
	base     string
	requests *countingRequestStore
	clients  *oauth2memory.OAuth2ClientRepository
	consents *oauth2memory.ConsentRepository
	par      *oauth2memory.PARStore
}

// newAuthorizationRequestFixture は本番と同じ Register で認可・トークン・PAR・登録の
// 各エンドポイントを 1 つのスタックへ載せる。routes_e2e_test.go の newServer を使わない
// のは、あちらが authorization_details の type 登録簿と resource 登録簿を配線して
// おらず、RFC 9396 と token-exchange の行がその配線ごと観測できないためである。
func newAuthorizationRequestFixture(t *testing.T) *arFixture {
	t.Helper()
	now := time.Now().UTC()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(arClientSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: arClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{arRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken, spec.GrantTokenExchange,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    arScope,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})

	hasher := testing_passwords.NewHasher()
	passwordHash, err := hasher.Hash(arPassword)
	if err != nil {
		t.Fatalf("seed password: %v", err)
	}
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: arUsername, PasswordHash: passwordHash,
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
	})

	// payment_initiation は「金額の上限」と「操作の集合」を持つ型である。
	// 上限のある型でないと、RFC9396-MONOTONIC-NARROWING の「狭めることだけを許す」を
	// 区別する事例が作れない。
	detailTypes := oauth2memory.NewAuthorizationDetailTypeRepository()
	detailTypes.Seed(&authorizationdomain.AuthorizationDetailType{
		TenantID: tenancydomain.DefaultTenantID, Type: arDetailType,
		Description: "支払いの開始", DisplayTemplate: "{actions} を実行する",
		State: authorizationdomain.DetailTypeEnabled,
		Schema: authorizationdomain.AuthorizationDetailsSchema{
			Rules: []authorizationdomain.AuthorizationDetailFieldRule{
				{
					Name: "actions", Semantics: authorizationdomain.DetailFieldSet, Required: true,
					Allowed: []string{"initiate", "status"},
				},
				{Name: "instructedAmount", Semantics: authorizationdomain.DetailFieldAtMost, Required: true},
			},
		},
		CreatedAt: now, UpdatedAt: now,
	})

	resourceServers := oauth2memory.NewMcpResourceServerRepository()
	resourceServers.Seed(&domain.McpResourceServer{
		ID: "rs-payments", Resource: arResource, Name: "Payments API",
		Scopes: []string{"openid", "profile", "payments"},
		State:  domain.McpResourceServerActive,
	})

	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	signer := tokensJOSE.NewJWTSigner(arIssuer, keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	requests := &countingRequestStore{AuthorizationRequestStore: oauth2memory.NewAuthorizationRequestStore()}
	consents := oauth2memory.NewConsentRepository()
	parStore := oauth2memory.NewPARStore()

	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          arIssuer,
		TenantRepo:      tenants,
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		OAuth2: oauth2.Module{
			ClientRepo: clients, ConsentRepo: consents,
			RequestStore: requests, CodeStore: oauth2memory.NewAuthorizationCodeStore(),
			PARStore: parStore, RefreshStore: oauth2memory.NewRefreshTokenStore(),
			AuthzDetailTypeRepo: detailTypes, McpResourceServerRepo: resourceServers,
		},
		UserRepo:          users,
		KeyStore:          keyStore,
		TokenIssuer:       signer,
		TokenIntrospector: signer,
		PasswordHasher:    hasher,
		SessionManager:    sessionManager,
		AuthnResolver:     sessionManager,
	})
	server := httptest.NewServer(e)
	t.Cleanup(server.Close)
	return &arFixture{
		server: server, base: server.URL + "/realms/default",
		requests: requests, clients: clients, consents: consents, par: parStore,
	}
}

const arVerifier = "authorization-request-standards-pkce-verifier-0123456789"

func arChallenge() string {
	sum := sha256.Sum256([]byte(arVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// arQuery は 15 行すべてを満たす認可リクエストを返す。個々のテストはここから
// 1 か所だけを崩す。崩していない事例が通ることを対照に置くので、拒否の理由が
// 崩した 1 か所であることが読める。
func arQuery(overrides map[string]string) url.Values {
	query := url.Values{
		"client_id":             {arClientID},
		"redirect_uri":          {arRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile"},
		"state":                 {"opaque-state"},
		"code_challenge":        {arChallenge()},
		"code_challenge_method": {"S256"},
	}
	for key, value := range overrides {
		if value == "" {
			query.Del(key)
			continue
		}
		query.Set(key, value)
	}
	return query
}

// authorize は cookie を持たない client で /authorize を 1 回叩く。リダイレクトは
// 追わない。Location そのものが観測対象だからである。
func (f *arFixture) authorize(t *testing.T, query url.Values) *http.Response {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := client.Get(f.base + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// assertAuthorizationRefused は、拒否が応答に現れ、かつ認可リクエストが 1 件も
// 保存されていないことを読む。
func (f *arFixture) assertAuthorizationRefused(
	t *testing.T, query url.Values, what string,
) (*http.Response, string) {
	t.Helper()
	before := f.requests.saves.Load()
	response := f.authorize(t, query)
	body := readBody(t, response)
	if response.StatusCode == http.StatusSeeOther || response.StatusCode == http.StatusFound {
		t.Fatalf("%s: 拒否されるべき認可リクエストが %d で先へ進んだ Location=%q",
			what, response.StatusCode, response.Header.Get("Location"))
	}
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("%s: 応答に error が無い status=%d body=%s", what, response.StatusCode, body)
	}
	if got := f.requests.saves.Load() - before; got != 0 {
		t.Fatalf("%s: 拒否したのに認可リクエストが %d 件保存された", what, got)
	}
	return response, body
}

// accessTokenClaims は発行されたアクセストークンの payload を読む。
// authorization_details は署名済みトークンの中にしか現れないので、/token の応答本文を
// 眺めるだけでは、詳細を落として発行した実装と区別できない。
func accessTokenClaims(t *testing.T, accessToken string) map[string]any {
	t.Helper()
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		t.Fatalf("アクセストークンが JWT ではない: %q", accessToken)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("payload の復号: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("payload が JSON ではない: %v", err)
	}
	return claims
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

// completeFlow は login と consent を通し、クライアントへ返るリダイレクト先を返す。
// 認可コードは正式な入口を通ってしか手に入らない。
func (f *arFixture) completeFlow(t *testing.T, query url.Values) *url.URL {
	t.Helper()
	client := browserClient(t)
	response, err := client.Get(f.base + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("/authorize status=%d", response.StatusCode)
	}
	location := response.Header.Get("Location")
	if strings.HasPrefix(location, arRedirectURI) {
		// 同意が既にあると /authorize から直接クライアントへ戻る。
		parsed, parseErr := url.Parse(location)
		if parseErr != nil {
			t.Fatalf("parse redirect: %v", parseErr)
		}
		return parsed
	}

	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, f.base+"/api/auth/transaction")
	if transaction.Kind != "login" {
		t.Fatalf("最初の transaction が login ではない: %+v", transaction)
	}
	next := postJSON[map[string]string](t, client, f.base+"/api/auth/login", transaction.CSRFToken,
		map[string]string{"username": arUsername, "password": arPassword})
	if redirect := next["redirect_to"]; redirect != "" {
		parsed, parseErr := url.Parse(redirect)
		if parseErr != nil {
			t.Fatalf("parse redirect: %v", parseErr)
		}
		return parsed
	}

	consent := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, f.base+"/api/auth/transaction")
	if consent.Kind != "consent" {
		t.Fatalf("2 番目の transaction が consent ではない: %+v", consent)
	}
	result := postJSON[map[string]string](t, client, f.base+"/api/auth/consent", consent.CSRFToken,
		map[string]string{"action": "allow"})
	parsed, err := url.Parse(result["redirect_to"])
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	return parsed
}

// postToken は /token へフォームを 1 通送り、状態と本文を返す。
func (f *arFixture) postToken(t *testing.T, form url.Values) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, f.base+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(arClientID, arClientSecret)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

// assertNoCredentialIssued は、拒否された応答がトークンも認可コードも運んでいない
// ことを生の本文に対して確かめる。
func assertNoCredentialIssued(t *testing.T, what, body string) {
	t.Helper()
	for _, field := range []string{`"access_token"`, `"refresh_token"`, `"id_token"`, `"code"`} {
		if strings.Contains(body, field) {
			t.Fatalf("%s: 拒否された応答が %s を運んでいる body=%s", what, field, body)
		}
	}
}

// Authorization Code Grant を認可エンドポイントとトークンエンドポイントの両方で
// 提供し、単一値であるべきセキュリティパラメーターが認可リクエスト内で重複していれば
// invalid_request として拒否する。リダイレクトを使うフローは、そのコードグラントと
// PKCE で保護する。
//
// 提供していることは「/authorize が 303 を返す」では足りない。コードがトークンへ
// 交換できて初めて、2 つの入口の両方で提供されていると言える。
//
//spec:covers RFC6749-AUTHORIZATION-CODE / RFC9700-AUTHORIZATION-CODE:
func TestAuthorizationCodeGrantSpansBothEndpointsAndRejectsDuplicatedParameters(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	// 対照: 正式な入口だけを通って、認可コードがアクセストークンになる。
	redirect := fixture.completeFlow(t, arQuery(nil))
	code := redirect.Query().Get("code")
	if code == "" {
		t.Fatalf("認可コードが返らなかった: %s", redirect)
	}
	status, body := fixture.postToken(t, url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"code_verifier": {arVerifier}, "redirect_uri": {arRedirectURI},
	})
	if status != http.StatusOK || !strings.Contains(body, `"access_token"`) {
		t.Fatalf("/token status=%d body=%s", status, body)
	}

	// 重複した単一値パラメーターは、値が両方とも妥当でも拒否する。攻撃者は 2 つ目を
	// 足すだけで、検証を通す値と実際に使われる値を食い違わせられる。
	for _, key := range []string{"client_id", "redirect_uri", "response_type", "scope", "state"} {
		t.Run("重複した "+key, func(t *testing.T) {
			query := arQuery(nil)
			query.Add(key, query.Get(key))
			if _, body := fixture.assertAuthorizationRefused(t, query, key); !strings.Contains(body, `"invalid_request"`) {
				t.Fatalf("error=invalid_request を期待した: %s", body)
			}
		})
	}

	// PKCE の無いリダイレクトフローは受け付けない。
	t.Run("code_challenge が無い", func(t *testing.T) {
		fixture.assertAuthorizationRefused(t, arQuery(map[string]string{"code_challenge": ""}), "code_challenge なし")
	})
}

// Implicit Grant も、OpenID Connect の Implicit / Hybrid Flow も提供しない。
//
// この 2 行の Statement は製品の制約ではなく標準側の機能を書いているので、観測は
// 「その機能を要求するリクエストが通らないこと」と「通らなかった結果として資格情報が
// 1 つも出ていないこと」の対になる。フラグメントに直接トークンを載せる流儀なので、
// 応答が資格情報を運んでいないことを本文とリダイレクト先の両方で読む。
//
//spec:covers RFC6749-IMPLICIT / OIDC-CORE-HYBRID-IMPLICIT (excluded):
func TestImplicitAndHybridResponseTypesAreRefused(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	// 対照: code だけは通る。以下の拒否が response_type の扱いによるものだと分かる。
	if redirect := fixture.completeFlow(t, arQuery(nil)); redirect.Query().Get("code") == "" {
		t.Fatalf("前提が壊れている: response_type=code で認可コードが出ない")
	}

	for _, responseType := range []string{
		"token",               // RFC 6749 Implicit
		"id_token",            // OIDC Implicit
		"id_token token",      // OIDC Implicit
		"code id_token",       // OIDC Hybrid
		"code token",          // OIDC Hybrid
		"code id_token token", // OIDC Hybrid
	} {
		t.Run("response_type="+responseType, func(t *testing.T) {
			response, body := fixture.assertAuthorizationRefused(t,
				arQuery(map[string]string{"response_type": responseType}), responseType)
			assertNoCredentialIssued(t, responseType, body)
			if location := response.Header.Get("Location"); location != "" {
				t.Fatalf("%s: 拒否が Location=%q を返した", responseType, location)
			}
		})
	}
}

// トークンリクエストの code_verifier を認可時の code_challenge と照合し、認可リクエスト
// 内で PKCE パラメーターが重複していれば拒否する。Statement が 2 つのことを言っている
// ので、観測も 2 つ置く。
//
//spec:covers RFC7636-VERIFY:
func TestPKCEVerifierIsCheckedAtTheTokenEndpointAndDuplicatesAreRefused(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	// 照合: 同じコードに対し、違う verifier では交換できず、正しい verifier では
	// できる。1 本のコードで対にしないと、コードが最初から無効だった場合と
	// 区別できない。
	redirect := fixture.completeFlow(t, arQuery(nil))
	code := redirect.Query().Get("code")
	if code == "" {
		t.Fatal("認可コードが返らなかった")
	}
	exchange := func(verifier string) (int, string) {
		return fixture.postToken(t, url.Values{
			"grant_type": {"authorization_code"}, "code": {code},
			"code_verifier": {verifier}, "redirect_uri": {arRedirectURI},
		})
	}
	status, body := exchange(arVerifier + "-tampered")
	if status == http.StatusOK {
		t.Fatalf("誤った code_verifier で交換できた: %s", body)
	}
	assertNoCredentialIssued(t, "誤った code_verifier", body)
	if status, body = exchange(arVerifier); status != http.StatusOK {
		t.Fatalf("正しい code_verifier で交換できない: status=%d body=%s", status, body)
	}

	// 重複: PKCE パラメーターが 2 つ現れる認可リクエストは受け付けない。
	for _, key := range []string{"code_challenge", "code_challenge_method"} {
		t.Run("重複した "+key, func(t *testing.T) {
			query := arQuery(nil)
			query.Add(key, query.Get(key))
			fixture.assertAuthorizationRefused(t, query, key)
		})
	}
}

// code_challenge_method は S256 だけを許可し、RFC 7636 が定める plain 方式は提供しない。
//
// plain の事例は、challenge と verifier が RFC 上は正しく対応する組（plain では
// challenge == verifier）で送る。壊れた値で送ると、method を見ていない実装でも同じ
// 拒否になり、method の扱いを区別できない。
//
//spec:covers RFC7636-S256 / RFC7636-PLAIN (excluded):
func TestOnlyS256CodeChallengeMethodIsAccepted(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	if redirect := fixture.completeFlow(t, arQuery(nil)); redirect.Query().Get("code") == "" {
		t.Fatal("前提が壊れている: S256 で認可コードが出ない")
	}

	for name, overrides := range map[string]map[string]string{
		"plain (RFC 7636 の対応する組)": {"code_challenge_method": "plain", "code_challenge": arVerifier},
		"PLAIN":                   {"code_challenge_method": "PLAIN", "code_challenge": arVerifier},
		"s256 (小文字)":              {"code_challenge_method": "s256"},
		"method 無し":               {"code_challenge_method": ""},
	} {
		t.Run(name, func(t *testing.T) {
			_, body := fixture.assertAuthorizationRefused(t, arQuery(overrides), name)
			assertNoCredentialIssued(t, name, body)
		})
	}
}

// クライアント認証済みの PAR を保存して短命な request_uri を返し、その request_uri は
// 一度だけ使える。
//
// optional の行なので、まず提供していることを確かめる。提供していなければ行の
// Adoption が誤っていることになるので、その場合は規範の変更として切り出す。
//
//spec:covers RFC9126-PAR (optional) / RFC9126-SINGLE-USE:
func TestPushedAuthorizationRequestIsAuthenticatedShortLivedAndSingleUse(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	form := arQuery(nil)
	post := func(authenticate bool) *http.Response {
		request, err := http.NewRequest(http.MethodPost, fixture.base+"/par", strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if authenticate {
			request.SetBasicAuth(arClientID, arClientSecret)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("POST /par: %v", err)
		}
		t.Cleanup(func() { _ = response.Body.Close() })
		return response
	}

	// クライアント認証の無い push は request_uri を返さない。
	unauthenticated := post(false)
	if unauthenticated.StatusCode == http.StatusCreated {
		t.Fatal("クライアント認証の無い PAR が受理された")
	}
	if body := readBody(t, unauthenticated); strings.Contains(body, "request_uri") {
		t.Fatalf("拒否した PAR が request_uri を返した: %s", body)
	}

	// 認証済みの push は短命な request_uri を返す。
	authenticated := post(true)
	if authenticated.StatusCode != http.StatusCreated {
		t.Fatalf("PAR status=%d body=%s", authenticated.StatusCode, readBody(t, authenticated))
	}
	var pushed struct {
		RequestURI string `json:"request_uri"`
		ExpiresIn  int    `json:"expires_in"`
	}
	if err := json.Unmarshal([]byte(readBody(t, authenticated)), &pushed); err != nil {
		t.Fatalf("PAR 応答が JSON ではない: %v", err)
	}
	if !strings.HasPrefix(pushed.RequestURI, "urn:ietf:params:oauth:request_uri:") {
		t.Fatalf("request_uri=%q", pushed.RequestURI)
	}
	// 「短命」は RFC 9126 §4 が 60 秒程度を推奨する。上限を読まないと、期限を
	// 事実上無効にした実装が通る。
	if pushed.ExpiresIn <= 0 || pushed.ExpiresIn > 90 {
		t.Fatalf("expires_in=%d, want 0 < expires_in <= 90", pushed.ExpiresIn)
	}
	// 広告した expires_in と、保存した記録の実際の寿命は別々に決まっている。
	// 応答の数字だけを読むと、90 と広告しながら 1 日生かす実装が通ってしまう。
	stored, err := fixture.par.Find(context.Background(), pushed.RequestURI)
	if err != nil || stored == nil {
		t.Fatalf("push した記録が保存されていない: %#v err=%v", stored, err)
	}
	if lifetime := stored.ExpiresAt.Sub(stored.IssuedAt); lifetime != time.Duration(pushed.ExpiresIn)*time.Second {
		t.Fatalf("保存した寿命は %v、広告した expires_in は %d 秒", lifetime, pushed.ExpiresIn)
	}

	consume := url.Values{"request_uri": {pushed.RequestURI}, "client_id": {arClientID}}

	// 1 回目は受理される。
	first := fixture.authorize(t, consume)
	if first.StatusCode != http.StatusSeeOther {
		t.Fatalf("1 回目の /authorize status=%d body=%s", first.StatusCode, readBody(t, first))
	}

	// 2 回目は拒否され、認可リクエストも増えない。
	_, body := fixture.assertAuthorizationRefused(t, consume, "2 回目の request_uri")
	if !strings.Contains(body, `"invalid_request_uri"`) {
		t.Fatalf("error=invalid_request_uri を期待した: %s", body)
	}
}

// 認可レスポンス、および安全に確定した redirect_uri へ返す認可エラーに発行者の
// 識別子を含める。
//
// 「安全に確定した」を読むために、確定していない場合を対に置く。登録されていない
// redirect_uri へのリクエストは、iss を付けてリダイレクトするのではなく、そもそも
// リダイレクトしない。ここを読まないと、未検証の宛先へ iss 付きで飛ばす実装が通る。
//
//spec:covers RFC9207-ISS:
func TestIssuerIdentifierAccompaniesAuthorizationResponsesAndRedirectedErrors(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)
	const wantIssuer = arIssuer + "/realms/default"

	// 成功した認可レスポンス。
	redirect := fixture.completeFlow(t, arQuery(nil))
	if got := redirect.Query().Get("iss"); got != wantIssuer {
		t.Errorf("認可レスポンスの iss=%q, want %q", got, wantIssuer)
	}

	// 安全に確定した redirect_uri へ返すエラー。session の無い prompt=none は
	// 登録済みの宛先へ login_required を返す。
	errorResponse := fixture.authorize(t, arQuery(map[string]string{"prompt": "none"}))
	location := errorResponse.Header.Get("Location")
	if !strings.HasPrefix(location, arRedirectURI) {
		t.Fatalf("prompt=none が登録済みの宛先へ返らない: status=%d Location=%q",
			errorResponse.StatusCode, location)
	}
	errorURL, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	if errorURL.Query().Get("error") == "" {
		t.Fatalf("エラーが載っていない: %s", location)
	}
	if got := errorURL.Query().Get("iss"); got != wantIssuer {
		t.Errorf("認可エラーの iss=%q, want %q", got, wantIssuer)
	}

	// 確定していない redirect_uri へは、iss を付けてさえリダイレクトしない。
	unverified := fixture.authorize(t,
		arQuery(map[string]string{"redirect_uri": "https://attacker.example/cb"}))
	if location := unverified.Header.Get("Location"); location != "" {
		t.Fatalf("未検証の redirect_uri へリダイレクトした: %q", location)
	}
}

// authorization_details は、テナントが事前登録した type とそのスキーマに対して
// 検証する。未登録の型やスキーマの不一致は部分的に受理せず拒否する。
//
// 「部分的に受理せず」は、妥当な detail と不当な detail を 1 通に混ぜた事例でしか
// 読めない。妥当な方だけを取り込んで進む実装は、不当な方だけの事例では捕まらない。
//
//spec:covers RFC9396-REGISTERED-TYPES:
func TestAuthorizationDetailsAreValidatedAgainstRegisteredTypes(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	valid := `{"type":"` + arDetailType + `","actions":["initiate"],"fields":{"instructedAmount":100}}`

	// 対照: 登録済みの型でスキーマに適合する detail は通る。
	if redirect := fixture.completeFlow(t,
		arQuery(map[string]string{"scope": "openid profile payments", "authorization_details": "[" + valid + "]"})); redirect.Query().Get("code") == "" {
		t.Fatal("登録済みの型の authorization_details が拒否された")
	}

	refused := map[string]string{
		"未登録の型":         `[{"type":"unregistered_type","actions":["initiate"]}]`,
		"許可されない action": `[{"type":"` + arDetailType + `","actions":["cancel"],"fields":{"instructedAmount":100}}]`,
		"必須フィールドが無い":    `[{"type":"` + arDetailType + `","actions":["initiate"]}]`,
		"型が空":           `[{"type":"","actions":["initiate"]}]`,
		"JSON が配列ではない":  `{"type":"` + arDetailType + `"}`,
		// 妥当なものと不当なものを混ぜる。部分的に受理する実装だけがここで割れる。
		"妥当なものと未登録の型を混ぜる": `[` + valid + `,{"type":"unregistered_type"}]`,
	}
	for name, details := range refused {
		t.Run(name, func(t *testing.T) {
			_, body := fixture.assertAuthorizationRefused(t,
				arQuery(map[string]string{"authorization_details": details}), name)
			assertNoCredentialIssued(t, name, body)
		})
	}

	// 同じ検証は push の時点でも立つ。/par が素通りすれば、確定検証まで不正な
	// detail が保存されたまま運ばれる。
	t.Run("PAR も同じ検証を行う", func(t *testing.T) {
		form := arQuery(map[string]string{"authorization_details": `[{"type":"unregistered_type"}]`})
		request, err := http.NewRequest(http.MethodPost, fixture.base+"/par", strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.SetBasicAuth(arClientID, arClientSecret)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode == http.StatusCreated {
			t.Fatal("未登録の型を持つ PAR が受理された")
		}
	})
}

// 発行または交換するトークンが持てるのは、同意した権限の部分集合に限る。後続の交換は
// 権限を狭めることだけを許し、広げる要求は拒否する。
//
// Statement が 2 つのことを言っているので、観測も 2 つ置く。前半は同意した detail が
// そのまま発行されたトークンに載ること、後半は そのトークンを subject_token とする
// 交換が、狭める要求だけを通すことである。後半は正式な入口 (/token の
// token-exchange) から観測する。domain の DetailsSubsetOf を直接呼ぶ形では、
// 交換の経路に配線されていない実装を素通りさせる。
//
//spec:covers RFC9396-MONOTONIC-NARROWING:
func TestAuthorizationDetailsCanOnlyNarrowAcrossExchange(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	granted := `[{"type":"` + arDetailType + `","actions":["initiate","status"],"fields":{"instructedAmount":100}}]`
	redirect := fixture.completeFlow(t, arQuery(map[string]string{
		"scope": "openid profile payments", "authorization_details": granted,
	}))
	code := redirect.Query().Get("code")
	if code == "" {
		t.Fatal("認可コードが返らなかった")
	}
	status, body := fixture.postToken(t, url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"code_verifier": {arVerifier}, "redirect_uri": {arRedirectURI},
	})
	if status != http.StatusOK {
		t.Fatalf("/token status=%d body=%s", status, body)
	}
	var issued struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(body), &issued); err != nil {
		t.Fatalf("/token 応答が JSON ではない: %v", err)
	}
	// 同意した detail が、そのまま発行されたトークンに載る。応答本文ではなく
	// 署名済み payload を読むのは、リソースサーバーが信頼するのがそちらだからである。
	claims := accessTokenClaims(t, issued.AccessToken)
	carried, ok := claims["authorization_details"].([]any)
	if !ok || len(carried) != 1 {
		t.Fatalf("アクセストークンが同意した authorization_details を運んでいない: %v",
			claims["authorization_details"])
	}
	first, _ := carried[0].(map[string]any)
	if first["type"] != arDetailType {
		t.Fatalf("運ばれた detail の type=%v, want %q", first["type"], arDetailType)
	}

	exchange := func(details string) (int, string) {
		return fixture.postToken(t, url.Values{
			"grant_type":            {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":         {issued.AccessToken},
			"subject_token_type":    {"urn:ietf:params:oauth:token-type:access_token"},
			"resource":              {arResource},
			"scope":                 {"payments"},
			"authorization_details": {details},
		})
	}

	// 狭める交換は通る。これが無いと、以下の拒否が交換そのものの失敗と区別できない。
	narrowed := `[{"type":"` + arDetailType + `","actions":["status"],"fields":{"instructedAmount":10}}]`
	if status, body := exchange(narrowed); status != http.StatusOK {
		t.Fatalf("狭める交換が拒否された: status=%d body=%s", status, body)
	}

	for name, details := range map[string]string{
		"金額の上限を上げる":   `[{"type":"` + arDetailType + `","actions":["status"],"fields":{"instructedAmount":1000}}]`,
		"許可されない操作を足す": `[{"type":"` + arDetailType + `","actions":["cancel"],"fields":{"instructedAmount":10}}]`,
	} {
		t.Run(name, func(t *testing.T) {
			status, body := exchange(details)
			if status == http.StatusOK {
				t.Fatalf("権限を広げる交換が受理された: %s", body)
			}
			assertNoCredentialIssued(t, name, body)
		})
	}
}

// 同じ領域で type と粗い scope が重なる場合は構造化された詳細の上限を優先し、
// authorization_details で制限した領域を scope が再び広げる要求は拒否する。
//
// 製品でこれが現れるのは同意の判定である。過去に scope 全体へ与えた同意は、構造化
// された detail の同意を代替しない。scope の同意だけで detail 付きのリクエストが
// 自動で通ってしまう実装は、粗い scope が構造化された上限を上書きしたことになる。
//
//spec:covers RFC9396-SCOPE-PRECEDENCE:
func TestCoarseScopeConsentDoesNotCoverStructuredAuthorizationDetails(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	// 1 回目のフローで scope への同意を残す。
	if redirect := fixture.completeFlow(t, arQuery(nil)); redirect.Query().Get("code") == "" {
		t.Fatal("最初のフローで認可コードが出ない")
	}

	// 対照: 同じ scope だけのリクエストは、残った同意で consent 画面を挟まずに通る。
	client := browserClient(t)
	loginThen := func(query url.Values) *http.Response {
		response, err := client.Get(fixture.base + "/authorize?" + query.Encode())
		if err != nil {
			t.Fatalf("GET /authorize: %v", err)
		}
		t.Cleanup(func() { _ = response.Body.Close() })
		return response
	}
	first := loginThen(arQuery(nil))
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, fixture.base+"/api/auth/transaction")
	if transaction.Kind != "login" {
		t.Fatalf("最初の transaction が login ではない: %+v", transaction)
	}
	_ = first
	next := postJSON[map[string]string](t, client, fixture.base+"/api/auth/login", transaction.CSRFToken,
		map[string]string{"username": arUsername, "password": arPassword})
	if next["redirect_to"] == "" {
		t.Fatalf("scope だけのリクエストが同意済みなのに consent を要求した: %+v", next)
	}

	// 構造化された detail を足すと、同じ利用者・同じクライアント・同じ scope でも
	// 同意を求め直す。scope は 1 回目とまったく同じにする。scope を広げてしまうと、
	// 同意を求め直す理由が scope の不足になり、detail の扱いを観測できない。
	detailed := arQuery(map[string]string{
		"authorization_details": `[{"type":"` + arDetailType + `","actions":["initiate"],"fields":{"instructedAmount":100}}]`,
	})
	detailedClient := browserClient(t)
	response, err := detailedClient.Get(fixture.base + "/authorize?" + detailed.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	_ = response.Body.Close()
	loginTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, detailedClient, fixture.base+"/api/auth/transaction")
	afterLogin := postJSON[map[string]string](t, detailedClient, fixture.base+"/api/auth/login",
		loginTransaction.CSRFToken, map[string]string{"username": arUsername, "password": arPassword})
	if afterLogin["redirect_to"] != "" {
		t.Fatalf("構造化された authorization_details が過去の scope 同意で自動承認された: %+v", afterLogin)
	}
	if afterLogin["next"] == "" || !strings.HasSuffix(afterLogin["next"], "/consent") {
		t.Fatalf("consent を要求していない: %+v", afterLogin)
	}
}

// redirect_uri は登録値と完全に一致させ、未検証の URI へリダイレクトしない。
//
// 「完全に一致」を読むため、別ホストだけでなく末尾スラッシュ 1 文字違いとクエリの
// 追加も試す。正規化や前方一致で受け入れる実装へ緩めば、攻撃者は登録値を接頭辞に
// 持つ別の宛先へ認可コードを配送できる。
//
//spec:covers RFC9700-REDIRECT-MATCH:
func TestRedirectURIMustMatchARegisteredValueExactly(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	if redirect := fixture.completeFlow(t, arQuery(nil)); redirect.Query().Get("code") == "" {
		t.Fatal("前提が壊れている: 登録値と完全一致する redirect_uri で認可コードが出ない")
	}

	for name, redirectURI := range map[string]string{
		"別ホスト":     "https://attacker.example/cb",
		"末尾スラッシュ":  arRedirectURI + "/",
		"クエリを足した":  arRedirectURI + "?next=/",
		"パスを足した":   arRedirectURI + "/extra",
		"スキームだけ違う": strings.Replace(arRedirectURI, "https://", "http://", 1),
	} {
		t.Run(name, func(t *testing.T) {
			response, body := fixture.assertAuthorizationRefused(t,
				arQuery(map[string]string{"redirect_uri": redirectURI}), name)
			// 拒否が防いだもの: 未検証の宛先へは、エラーとしてもリダイレクトしない。
			if location := response.Header.Get("Location"); location != "" {
				t.Fatalf("%s: 未検証の URI へリダイレクトした Location=%q", name, location)
			}
			assertNoCredentialIssued(t, name, body)
		})
	}
}

// Authorization Code Grant を利用するクライアントには redirect_uri の登録を要求する。
//
// 登録の入口で拒否することと、拒否が防いだ効果 (redirect_uri を持たないクライアントが
// 登録簿に現れないこと) を対で読む。
//
//spec:covers RFC7591-REDIRECT-URI:
func TestDynamicRegistrationRequiresARedirectURIForTheCodeGrant(t *testing.T) {
	fixture := newAuthorizationRequestFixture(t)

	register := func(payload string) (int, string) {
		request, err := http.NewRequest(http.MethodPost, fixture.base+"/register", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("POST /register: %v", err)
		}
		defer func() { _ = response.Body.Close() }()
		body, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(body)
	}

	// 対照: redirect_uris を持つコードグラントのクライアントは登録できる。
	status, body := register(`{
		"client_name": "With Redirect",
		"client_type": "confidential",
		"redirect_uris": ["https://registered.example/cb"],
		"token_endpoint_auth_method": "client_secret_post",
		"grant_types": ["authorization_code"],
		"response_types": ["code"],
		"scope": "openid"
	}`)
	if status != http.StatusCreated {
		t.Fatalf("redirect_uris 付きの登録が失敗した: status=%d body=%s", status, body)
	}

	for name, payload := range map[string]string{
		"redirect_uris が無い": `{
			"client_name": "No Redirect",
			"client_type": "confidential",
			"token_endpoint_auth_method": "client_secret_post",
			"grant_types": ["authorization_code"],
			"response_types": ["code"],
			"scope": "openid"
		}`,
		"redirect_uris が空": `{
			"client_name": "Empty Redirect",
			"client_type": "confidential",
			"redirect_uris": [],
			"token_endpoint_auth_method": "client_secret_post",
			"grant_types": ["authorization_code"],
			"response_types": ["code"],
			"scope": "openid"
		}`,
	} {
		t.Run(name, func(t *testing.T) {
			status, body := register(payload)
			if status == http.StatusCreated {
				t.Fatalf("%s: redirect_uri を持たないコードグラントのクライアントが登録された: %s", name, body)
			}
			// 拒否が防いだもの: そのクライアントは登録簿に現れない。
			all, err := fixture.clients.FindAll(context.Background(), tenancydomain.DefaultTenantID)
			if err != nil {
				t.Fatal(err)
			}
			for _, client := range all {
				if client.ClientName != nil && strings.Contains(*client.ClientName, "Redirect") &&
					len(client.RedirectURIs) == 0 {
					t.Fatalf("%s: redirect_uri を持たないクライアントが登録簿にある: %#v", name, client)
				}
			}
		})
	}
}
