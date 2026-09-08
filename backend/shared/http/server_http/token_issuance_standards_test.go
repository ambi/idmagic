package server_http_test

// docs/contexts/oauth2/standards.md のうち、トークンの発行と交換の入口に立つ 15 行を
// 観測する。
//
// 入口は Register が組み立てたスタックの `/token` である。この 15 行が言っているのは
// 「何を出すか」なので、出たトークンの payload を復号して直接読む。応答が 200 である
// ことや、そのトークンで保護リソースへ到達できることでは足りない。
// [[wi-500-back-api-tokens-standards-rows-with-tests]] が測ったとおり、行が列挙して
// いる claim のうち認証が読まないものは、到達できるという観測では固定できない。

import (
	stdcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
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

	"github.com/labstack/echo/v5"

	testingpasswords "github.com/ambi/idmagic/backend/shared/security/testing_passwords"

	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	tiIssuer          = "http://test"
	tiClientID        = "token-issuance-client"
	tiClientSecret    = "token-issuance-client-secret"
	tiPublicClientID  = "token-issuance-public-client"
	tiAssertionClient = "token-issuance-assertion-client"
	tiRedirectURI     = "https://app.example/cb"
	tiUsername        = "alice"
	tiUserID          = "user_alice"
	tiPassword        = "token-issuance-password-1234"
	tiResource        = "https://api.example/orders"
	tiOtherResource   = "https://api.example/invoices"
	tiVerifier        = "token-issuance-standards-pkce-verifier-01234567890"
)

type tiFixture struct {
	base          string
	assertionKey  *rsa.PrivateKey
	tenants       *tenancymemory.TenantRepository
	refreshTokens *oauth2memory.RefreshTokenStore
}

// newTokenIssuanceFixture は本番と同じ Register で `/authorize` と `/token` を 1 つの
// スタックへ載せる。テナント記録を持たせるのは、委譲深さの上限がテナントから
// 解決されるためである。解決器を差し替えると、その配線ごと観測できなくなる。
func newTokenIssuanceFixture(t *testing.T) *tiFixture {
	t.Helper()
	now := time.Now().UTC()

	assertionKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(tiClientSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: tiClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{tiRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
			spec.GrantClientCredentials, spec.GrantTokenExchange,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile offline_access read write",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	// public クライアントは client_credentials を宣言していても使えない。
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: tiPublicClientID, ClientType: spec.ClientPublic,
		RedirectURIs:             []string{tiRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantClientCredentials},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodNone,
		Scope:                    "openid read",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: tiAssertionClient, ClientType: spec.ClientConfidential,
		RedirectURIs:            []string{tiRedirectURI},
		GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt,
		Scope:                   "read",
		JWKS: map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "assertion-key",
			"n": base64.RawURLEncoding.EncodeToString(assertionKey.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(assertionKey.PublicKey.E)).Bytes()),
		}}},
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})

	hasher := testingpasswords.NewHasher()
	passwordHash, err := hasher.Hash(tiPassword)
	if err != nil {
		t.Fatal(err)
	}
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: tiUserID, PreferredUsername: tiUsername, PasswordHash: passwordHash,
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
	})

	resourceServers := oauth2memory.NewMcpResourceServerRepository()
	resourceServers.Seed(&domain.McpResourceServer{
		ID: "rs-orders", Resource: tiResource, Name: "Orders API",
		Scopes: []string{"openid", "profile", "offline_access", "read", "write"},
		State:  domain.McpResourceServerActive,
	})
	// 無効な資源サーバー。登録済みでも `Active` でなければ audience にできない。
	resourceServers.Seed(&domain.McpResourceServer{
		ID: "rs-invoices", Resource: tiOtherResource, Name: "Invoices API",
		Scopes: []string{"read"}, State: domain.McpResourceServerDisabled,
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
	signer := tokensJOSE.NewJWTSigner(tiIssuer, keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	refreshTokens := oauth2memory.NewRefreshTokenStore()

	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          tiIssuer,
		TenantRepo:      tenants,
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		OAuth2: oauth2.Module{
			ClientRepo: clients, ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore: oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:    oauth2memory.NewAuthorizationCodeStore(),
			PARStore:     oauth2memory.NewPARStore(), RefreshStore: refreshTokens,
			McpResourceServerRepo:      resourceServers,
			ClientAssertionReplayStore: oauth2memory.NewClientAssertionReplayStore(),
			AccessTokenDenylist:        oauth2memory.NewAccessTokenDenylist(),
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
	return &tiFixture{
		base: server.URL + "/realms/default", assertionKey: assertionKey,
		tenants: tenants, refreshTokens: refreshTokens,
	}
}

// postToken は `/token` へフォームを 1 通送る。decorate が無ければ confidential
// クライアントの Basic 認証を付ける。
func (f *tiFixture) postToken(
	t *testing.T, form url.Values, decorate ...func(*http.Request),
) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, f.base+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if len(decorate) == 0 {
		request.SetBasicAuth(tiClientID, tiClientSecret)
	}
	for _, apply := range decorate {
		apply(request)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

// tokenResponse は成功した `/token` 応答を読む。
type tiTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

func (f *tiFixture) mustToken(t *testing.T, form url.Values) tiTokenResponse {
	t.Helper()
	status, body := f.postToken(t, form)
	if status != http.StatusOK {
		t.Fatalf("/token status=%d body=%s", status, body)
	}
	var parsed tiTokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("/token 応答が JSON ではない: %v body=%s", err, body)
	}
	return parsed
}

// jwtParts は JWT の header と payload を復号して返す。
func jwtParts(t *testing.T, token string) (header, payload map[string]any) {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT ではない: %q", token)
	}
	decode := func(segment string) map[string]any {
		raw, err := base64.RawURLEncoding.DecodeString(segment)
		if err != nil {
			t.Fatalf("復号: %v", err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("JSON ではない: %v", err)
		}
		return out
	}
	return decode(parts[0]), decode(parts[1])
}

// authorizationCode は正式な入口だけを通して認可コードを 1 本取る。
func (f *tiFixture) authorizationCode(t *testing.T, extra url.Values) string {
	t.Helper()
	sum := sha256.Sum256([]byte(tiVerifier))
	query := url.Values{
		"client_id": {tiClientID}, "redirect_uri": {tiRedirectURI},
		"response_type": {"code"}, "scope": {"openid profile offline_access"},
		"state":                 {"state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	maps.Copy(query, extra)
	client := browserClient(t)
	response, err := client.Get(f.base + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("/authorize status=%d", response.StatusCode)
	}
	transaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, f.base+"/api/auth/transaction")
	next := postJSON[map[string]string](t, client, f.base+"/api/auth/login", transaction.CSRFToken,
		map[string]string{"username": tiUsername, "password": tiPassword})
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
	return code
}

func (f *tiFixture) codeExchangeForm(code string, extra url.Values) url.Values {
	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"code_verifier": {tiVerifier}, "redirect_uri": {tiRedirectURI},
	}
	maps.Copy(form, extra)
	return form
}

func assertNoTokenInBody(t *testing.T, what, body string) {
	t.Helper()
	for _, field := range []string{`"access_token"`, `"refresh_token"`, `"id_token"`} {
		if strings.Contains(body, field) {
			t.Fatalf("%s: 拒否された応答が %s を運んでいる body=%s", what, field, body)
		}
	}
}

// RFC6749-CLIENT-CREDENTIALS (optional) / RFC6749-PASSWORD-GRANT (excluded):
// Client Credentials Grant は confidential クライアントに限って許可し、
// Resource Owner Password Credentials Grant は提供しない。
//
// `RFC6749-PASSWORD-GRANT` の Statement は製品の制約ではなく標準側の機能を書いて
// いるので、観測は「そのグラントを要求するリクエストが通らないこと」と「通らなかった
// 結果としてトークンが 1 つも出ていないこと」の対になる。利用者の正しい資格情報を
// 載せて送るのが要点である。誤った資格情報で送ると、グラントを提供している実装でも
// 同じ拒否になり、提供の有無を区別できない。
func TestClientCredentialsIsConfidentialOnlyAndPasswordGrantIsNotOffered(t *testing.T) {
	fixture := newTokenIssuanceFixture(t)

	// 対照: confidential クライアントの client_credentials は通る。
	issued := fixture.mustToken(t, url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"},
	})
	if issued.AccessToken == "" {
		t.Fatal("confidential クライアントの client_credentials でトークンが出ない")
	}

	// public クライアントは、同じグラントを宣言していても使えない。
	status, body := fixture.postToken(t, url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"},
		"client_id": {tiPublicClientID},
	}, func(request *http.Request) {})
	if status == http.StatusOK {
		t.Fatalf("public クライアントの client_credentials が受理された: %s", body)
	}
	assertNoTokenInBody(t, "public クライアント", body)

	// password グラントは、利用者の正しい資格情報を載せても提供されない。
	status, body = fixture.postToken(t, url.Values{
		"grant_type": {"password"},
		"username":   {tiUsername}, "password": {tiPassword},
		"scope": {"openid"},
	})
	if status == http.StatusOK {
		t.Fatalf("password グラントが受理された: %s", body)
	}
	assertNoTokenInBody(t, "password グラント", body)
}

// RFC9068-CLAIMS / RFC9068-ASYMMETRIC-SIGNATURE / RFC7518-SIGNATURE-ALGORITHMS /
// RFC7519-REGISTERED-CLAIMS / OIDC-CORE-ID-TOKEN / RFC8707-AUDIENCE:
// 発行するトークンの中身と署名を、復号した payload から直接読む。
//
// 到達できることだけでは足りない。認証が読まない claim は、入口の観測では固定
// できないからである。署名は、テナントの公開鍵で実際に検証が通ることと、
// アルゴリズムが非対称であることの両方を読む。
func TestIssuedTokensCarryTheRegisteredClaimsAndAnAsymmetricSignature(t *testing.T) {
	fixture := newTokenIssuanceFixture(t)
	code := fixture.authorizationCode(t, nil)
	issued := fixture.mustToken(t, fixture.codeExchangeForm(code, nil))

	accessHeader, accessClaims := jwtParts(t, issued.AccessToken)

	// RFC9068-CLAIMS: JWT アクセストークンが列挙された 7 つの claim を持つ。
	for _, claim := range []string{"iss", "sub", "aud", "exp", "iat", "jti", "client_id"} {
		if _, ok := accessClaims[claim]; !ok {
			t.Errorf("アクセストークンに %s が無い: %v", claim, accessClaims)
		}
	}
	if accessHeader["typ"] != "at+jwt" {
		t.Errorf("アクセストークンの typ=%v, want at+jwt", accessHeader["typ"])
	}

	// RFC7519-REGISTERED-CLAIMS: 発行した値が用途に合っている。
	wantIssuer := tiIssuer + "/realms/default"
	if accessClaims["iss"] != wantIssuer {
		t.Errorf("iss=%v, want %q", accessClaims["iss"], wantIssuer)
	}
	if accessClaims["sub"] != tiUserID {
		t.Errorf("sub=%v, want %q", accessClaims["sub"], tiUserID)
	}
	if accessClaims["client_id"] != tiClientID {
		t.Errorf("client_id=%v, want %q", accessClaims["client_id"], tiClientID)
	}
	exp, expOK := accessClaims["exp"].(float64)
	iat, iatOK := accessClaims["iat"].(float64)
	if !expOK || !iatOK || exp <= iat {
		t.Errorf("exp=%v iat=%v: exp は iat より後でなければならない", accessClaims["exp"], accessClaims["iat"])
	}
	if jti, _ := accessClaims["jti"].(string); jti == "" {
		t.Error("jti が空である")
	}

	// RFC8707-AUDIENCE: 空でない audience を持つ。resource 未指定なら client_id。
	switch audience := accessClaims["aud"].(type) {
	case string:
		if audience == "" {
			t.Error("aud が空である")
		}
		if audience != tiClientID {
			t.Errorf("resource 未指定の aud=%q, want %q", audience, tiClientID)
		}
	case []any:
		if len(audience) == 0 {
			t.Error("aud が空の配列である")
		}
	default:
		t.Errorf("aud=%v が空である", accessClaims["aud"])
	}

	// RFC9068-ASYMMETRIC-SIGNATURE / RFC7518-SIGNATURE-ALGORITHMS:
	// 署名は PS256 または ES256 であり、対称鍵ではない。
	for name, header := range map[string]map[string]any{
		"access_token": accessHeader,
		"id_token":     mustHeader(t, issued.IDToken),
	} {
		alg, _ := header["alg"].(string)
		if alg != "PS256" && alg != "ES256" {
			t.Errorf("%s の alg=%q, want PS256 or ES256", name, alg)
		}
		if strings.HasPrefix(alg, "HS") || strings.HasPrefix(alg, "none") {
			t.Errorf("%s が対称鍵または無署名である: alg=%q", name, alg)
		}
		if kid, _ := header["kid"].(string); kid == "" {
			t.Errorf("%s に kid が無い。公開鍵での検証ができない", name)
		}
	}
	// 公開鍵で実際に検証できることを JWKS 経由で確かめる。alg の宣言だけでは、
	// 署名が対応する鍵で作られたことの証拠にならない。
	assertVerifiableWithPublishedKey(t, fixture.base, issued.AccessToken)

	// OIDC-CORE-ID-TOKEN: ID トークンが 5 つの claim と認証コンテキストを持つ。
	_, idClaims := jwtParts(t, issued.IDToken)
	for _, claim := range []string{"iss", "sub", "aud", "exp", "iat"} {
		if _, ok := idClaims[claim]; !ok {
			t.Errorf("ID トークンに %s が無い: %v", claim, idClaims)
		}
	}
	if idClaims["aud"] != tiClientID {
		t.Errorf("ID トークンの aud=%v, want %q", idClaims["aud"], tiClientID)
	}
	for _, claim := range []string{"auth_time", "amr"} {
		if _, ok := idClaims[claim]; !ok {
			t.Errorf("ID トークンに認証コンテキスト %s が無い: %v", claim, idClaims)
		}
	}
}

func mustHeader(t *testing.T, token string) map[string]any {
	t.Helper()
	if token == "" {
		t.Fatal("トークンが空である")
	}
	header, _ := jwtParts(t, token)
	return header
}

// assertVerifiableWithPublishedKey は、公開している JWKS の中に、当のトークンの
// kid を持つ非対称鍵があることを確かめる。
func assertVerifiableWithPublishedKey(t *testing.T, base, token string) {
	t.Helper()
	header, _ := jwtParts(t, token)
	kid, _ := header["kid"].(string)
	response, err := http.Get(base + "/jwks")
	if err != nil {
		t.Fatalf("GET jwks: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(body, &jwks); err != nil {
		t.Fatalf("jwks が JSON ではない: %v body=%s", err, body)
	}
	for _, key := range jwks.Keys {
		if key["kid"] != kid {
			continue
		}
		kty, _ := key["kty"].(string)
		if kty != "RSA" && kty != "EC" {
			t.Fatalf("公開鍵の kty=%q が非対称ではない", kty)
		}
		if _, hasSecret := key["k"]; hasSecret {
			t.Fatal("公開している鍵が対称鍵の素材を含んでいる")
		}
		return
	}
	t.Fatalf("kid=%q の公開鍵が JWKS に無い。公開鍵での検証ができない: %s", kid, body)
}

// RFC8707-MCP-RESOURCE-BINDING:
// `resource` で指定された McpResourceServer に audience を厳格に限定し、未登録・無効・
// 複数指定は fail-closed で拒否する。`resource` が未指定なら client_id を audience とする。
//
// 行は「全経路へ一様に適用する」と言っているので、認可コードの交換と
// client_credentials の 2 経路で同じ観測を繰り返す。1 経路だけを見るテストは、
// 別の経路で検査が抜けている実装を通してしまう。
func TestAccessTokenAudienceIsBoundToTheRequestedResource(t *testing.T) {
	fixture := newTokenIssuanceFixture(t)

	audienceOf := func(t *testing.T, token string) []string {
		t.Helper()
		_, claims := jwtParts(t, token)
		switch value := claims["aud"].(type) {
		case string:
			return []string{value}
		case []any:
			out := make([]string, 0, len(value))
			for _, item := range value {
				text, _ := item.(string)
				out = append(out, text)
			}
			return out
		default:
			t.Fatalf("aud=%v", claims["aud"])
			return nil
		}
	}

	t.Run("認可コードの交換", func(t *testing.T) {
		// resource は認可リクエストと交換の双方で一致していなければならない。
		code := fixture.authorizationCode(t, url.Values{"resource": {tiResource}})
		issued := fixture.mustToken(t, fixture.codeExchangeForm(code, url.Values{"resource": {tiResource}}))
		if got := audienceOf(t, issued.AccessToken); len(got) != 1 || got[0] != tiResource {
			t.Fatalf("aud=%v, want [%q]", got, tiResource)
		}
	})

	t.Run("client_credentials", func(t *testing.T) {
		issued := fixture.mustToken(t, url.Values{
			"grant_type": {"client_credentials"}, "scope": {"read"}, "resource": {tiResource},
		})
		if got := audienceOf(t, issued.AccessToken); len(got) != 1 || got[0] != tiResource {
			t.Fatalf("aud=%v, want [%q]", got, tiResource)
		}
	})

	t.Run("resource 未指定なら client_id", func(t *testing.T) {
		issued := fixture.mustToken(t, url.Values{
			"grant_type": {"client_credentials"}, "scope": {"read"},
		})
		if got := audienceOf(t, issued.AccessToken); len(got) != 1 || got[0] != tiClientID {
			t.Fatalf("aud=%v, want [%q]", got, tiClientID)
		}
	})

	// fail-closed の 3 事例。どれも「拒否したうえでトークンが出ていない」ことを読む。
	for name, resource := range map[string][]string{
		"未登録の resource": {"https://api.example/unregistered"},
		"無効な resource":  {tiOtherResource},
		"複数指定":          {tiResource, tiOtherResource},
	} {
		t.Run(name, func(t *testing.T) {
			form := url.Values{"grant_type": {"client_credentials"}, "scope": {"read"}}
			form["resource"] = resource
			status, body := fixture.postToken(t, form)
			if status == http.StatusOK {
				t.Fatalf("%s が受理された: %s", name, body)
			}
			assertNoTokenInBody(t, name, body)
		})
	}
}

// RFC9700-REFRESH-REPLAY:
// リフレッシュトークンをローテーションし、再利用を検知したら関連トークンを失効させる。
//
// Statement が 2 つのことを言っているので、観測も 2 つ置く。1 つ目はローテーション
// （交換のたびに別の値が返り、古い値は使えない）、2 つ目は再利用の検知が「その 1 本」
// ではなく family を落とすことである。後者を読まないと、古い値を拒否するだけで
// 攻撃者が先に奪った新しい値を生かし続ける実装が通る。
func TestRefreshTokenRotatesAndReuseRevokesTheWholeFamily(t *testing.T) {
	fixture := newTokenIssuanceFixture(t)
	code := fixture.authorizationCode(t, nil)
	first := fixture.mustToken(t, fixture.codeExchangeForm(code, nil))
	if first.RefreshToken == "" {
		t.Fatal("offline_access を要求したのにリフレッシュトークンが返らない")
	}

	refresh := func(token string) (int, string) {
		return fixture.postToken(t, url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {token},
		})
	}

	// ローテーション: 交換すると別の値が返る。
	status, body := refresh(first.RefreshToken)
	if status != http.StatusOK {
		t.Fatalf("1 回目のリフレッシュ status=%d body=%s", status, body)
	}
	var rotated tiTokenResponse
	if err := json.Unmarshal([]byte(body), &rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == "" || rotated.RefreshToken == first.RefreshToken {
		t.Fatalf("ローテーションしていない: %q", rotated.RefreshToken)
	}

	// 再利用の検知: 使い終えた値をもう一度出すと拒否される。
	status, body = refresh(first.RefreshToken)
	if status == http.StatusOK {
		t.Fatalf("使用済みのリフレッシュトークンが受理された: %s", body)
	}
	assertNoTokenInBody(t, "使用済みの再利用", body)

	// 再利用が防いだもの: 検知の時点で family ごと落ちるので、攻撃者が先に
	// 奪っていた新しい値も、その後は使えない。
	status, body = refresh(rotated.RefreshToken)
	if status == http.StatusOK {
		t.Fatalf("再利用を検知した後もローテーション後の値が使えた: %s", body)
	}
	assertNoTokenInBody(t, "family の失効", body)
}

// RFC8693-DELEGATION-DEFAULT / RFC8693-IMPERSONATION (optional) /
// RFC8693-SUBJECT-TOKEN / RFC8693-DELEGATION-DEPTH:
// 交換の既定は委譲であり、`sub` は元の利用者のまま、現在の行為者が `act` に入り、
// 以前の行為者は §4.1 に従って内側へ入れ子になる。受け付ける `subject_token` は
// 自身が発行しイントロスペクションを通過したものに限る。`act` チェーンの長さは
// テナントの実効委譲深さで制限し、テナントは既定を下げられるが上げられない。
//
// なりすまし（`act` を落として `sub` を置き換える形）は、明示的に許可した場合だけ
// 受け付ける。製品はその許可を持たないので、観測は「どの交換でも `sub` が入れ替わらず
// `act` が必ず載る」ことになる。
func TestTokenExchangeDelegatesByDefaultAndBoundsTheActorChain(t *testing.T) {
	fixture := newTokenIssuanceFixture(t)
	code := fixture.authorizationCode(t, nil)
	subject := fixture.mustToken(t, fixture.codeExchangeForm(code, nil))

	exchange := func(subjectToken string) (int, string) {
		return fixture.postToken(t, url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {subjectToken},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
			"resource":           {tiResource}, "scope": {"profile"},
		})
	}

	// RFC8693-DELEGATION-DEFAULT / RFC8693-IMPERSONATION:
	// 既定の交換は sub を保ち、act に現在の行為者を入れる。
	status, body := exchange(subject.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("交換 status=%d body=%s", status, body)
	}
	var exchanged tiTokenResponse
	if err := json.Unmarshal([]byte(body), &exchanged); err != nil {
		t.Fatal(err)
	}
	_, claims := jwtParts(t, exchanged.AccessToken)
	if claims["sub"] != tiUserID {
		t.Fatalf("交換後の sub=%v, want %q (委譲は元の利用者を保つ)", claims["sub"], tiUserID)
	}
	act, ok := claims["act"].(map[string]any)
	if !ok {
		t.Fatalf("交換後のトークンに act が無い。なりすましの形になっている: %v", claims)
	}
	if act["sub"] != tiClientID {
		t.Fatalf("act.sub=%v, want %q", act["sub"], tiClientID)
	}
	if _, nested := act["act"]; nested {
		t.Fatalf("1 段目の交換で act が入れ子になっている: %v", act)
	}

	// 2 段目は以前の行為者を内側へ入れ子にする (§4.1)。
	status, body = exchange(exchanged.AccessToken)
	if status != http.StatusOK {
		t.Fatalf("2 段目の交換 status=%d body=%s", status, body)
	}
	var second tiTokenResponse
	if err := json.Unmarshal([]byte(body), &second); err != nil {
		t.Fatal(err)
	}
	_, secondClaims := jwtParts(t, second.AccessToken)
	if secondClaims["sub"] != tiUserID {
		t.Fatalf("2 段目の sub=%v, want %q", secondClaims["sub"], tiUserID)
	}
	secondAct, _ := secondClaims["act"].(map[string]any)
	inner, nested := secondAct["act"].(map[string]any)
	if !nested || inner["sub"] != tiClientID {
		t.Fatalf("以前の行為者が内側へ入れ子になっていない: %v", secondAct)
	}

	// RFC8693-SUBJECT-TOKEN: 自身が発行していないトークンは受け付けない。
	for name, token := range map[string]string{
		"JWT ではない不透明な値":  "not-a-token",
		"claim を差し替えた偽物": tamperedSubject(t, subject.AccessToken),
	} {
		t.Run(name, func(t *testing.T) {
			status, body := exchange(token)
			if status == http.StatusOK {
				t.Fatalf("%s が subject_token として受理された: %s", name, body)
			}
			assertNoTokenInBody(t, name, body)
		})
	}

	// RFC8693-DELEGATION-DEPTH: テナントが上限を下げると、その深さを超える
	// チェーンは拒否される。ここで観測しているのは、解決器が `/token` へ
	// 配線されていることである。上限そのものの規則は
	// backend/oauth2/token/usecases/exchange_token_delegation_policy_test.go が持つ。
	t.Run("テナントが下げた上限を超える交換は拒否される", func(t *testing.T) {
		lowered := 1
		if err := fixture.tenants.Save(t.Context(), &tenancydomain.Tenant{
			ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
			Status: tenancydomain.TenantStatusActive, MaxDelegationDepth: &lowered,
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = fixture.tenants.Save(t.Context(), &tenancydomain.Tenant{
				ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
				Status: tenancydomain.TenantStatusActive,
			})
		})
		// 深さ 1 のチェーンを持つトークン (1 段目の交換結果) をもう一度交換すると
		// 深さ 2 になり、下げた上限を超える。
		status, body := exchange(exchanged.AccessToken)
		if status == http.StatusOK {
			t.Fatalf("テナントが下げた上限を超える交換が受理された: %s", body)
		}
		assertNoTokenInBody(t, "深さ超過", body)
	})
}

// tamperedSubject は claim だけを差し替えた、署名の合わないトークンを作る。
// 壊れた文字列では、そもそも JWT として読めずに落ちるので、「自身が発行した
// ものに限る」という判断を区別できない。
func tamperedSubject(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT ではない: %q", token)
	}
	_, payload := jwtParts(t, token)
	payload["sub"] = "attacker"
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return parts[0] + "." + base64.RawURLEncoding.EncodeToString(raw) + "." + parts[2]
}

// RFC7523-CLIENT-ASSERTION (optional):
// クライアントアサーションの署名、発行者、subject、audience、有効期限、`jti` を
// 検証する。6 つの要素それぞれについて、1 つだけを崩したアサーションが拒否される
// ことを読む。崩していないアサーションが通ることを対照に置くので、拒否の理由が
// 崩した 1 か所であることが分かる。
func TestClientAssertionVerifiesEveryDeclaredElement(t *testing.T) {
	fixture := newTokenIssuanceFixture(t)
	// audience は httptest の URL ではなく、製品が発行者として名乗る値である。
	audience := tiIssuer + "/realms/default/token"

	assert := func(t *testing.T, mutate func(map[string]any)) (int, string) {
		t.Helper()
		now := time.Now()
		claims := map[string]any{
			"iss": tiAssertionClient, "sub": tiAssertionClient,
			"aud": audience, "jti": "jti-" + t.Name(),
			"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
		}
		mutate(claims)
		signWith := fixture.assertionKey
		if replacement, ok := claims["__wrong_key"].(*rsa.PrivateKey); ok {
			signWith = replacement
			delete(claims, "__wrong_key")
		}
		header, err := json.Marshal(map[string]any{"alg": "PS256", "kid": "assertion-key"})
		if err != nil {
			t.Fatal(err)
		}
		payload, err := json.Marshal(claims)
		if err != nil {
			t.Fatal(err)
		}
		signingInput := base64.RawURLEncoding.EncodeToString(header) + "." +
			base64.RawURLEncoding.EncodeToString(payload)
		digest := sha256.Sum256([]byte(signingInput))
		signature, err := rsa.SignPSS(rand.Reader, signWith, stdcrypto.SHA256, digest[:],
			&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
		if err != nil {
			t.Fatal(err)
		}
		assertion := signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
		return fixture.postToken(t, url.Values{
			"grant_type": {"client_credentials"}, "scope": {"read"},
			"client_id":             {tiAssertionClient},
			"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
			"client_assertion":      {assertion},
		}, func(*http.Request) {})
	}

	// 対照: 6 要素がそろったアサーションは通る。
	if status, body := assert(t, func(map[string]any) {}); status != http.StatusOK {
		t.Fatalf("正しいアサーションが拒否された: status=%d body=%s", status, body)
	}

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"署名が別の鍵":   func(c map[string]any) { c["__wrong_key"] = otherKey },
		"iss が別の値": func(c map[string]any) { c["iss"] = "someone-else" },
		// iss と sub をそろえたまま client_id から外す。iss == sub の検査と
		// iss == client_id の検査は別々に立っているので、片方だけを崩す事例が
		// 無いと、もう片方が肩代わりして変異が生き残る。
		"iss と sub がそろって別のクライアント": func(c map[string]any) {
			c["iss"] = "someone-else"
			c["sub"] = "someone-else"
		},
		"sub が別の値": func(c map[string]any) { c["sub"] = "someone-else" },
		"aud が別の値": func(c map[string]any) { c["aud"] = "https://other.example/token" },
		// 許容する時計のずれ (60 秒) より十分に過去へ置く。境界ぎりぎりでは、
		// 期限を見ていない実装と時計のずれの許容を区別できない。
		"exp が過去": func(c map[string]any) { c["exp"] = time.Now().Add(-10 * time.Minute).Unix() },
		"jti が無い": func(c map[string]any) { delete(c, "jti") },
	} {
		t.Run(name, func(t *testing.T) {
			status, body := assert(t, mutate)
			if status == http.StatusOK {
				t.Fatalf("%s のアサーションが受理された: %s", name, body)
			}
			assertNoTokenInBody(t, name, body)
		})
	}
}
