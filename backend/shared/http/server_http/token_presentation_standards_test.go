package server_http_test

// docs/contexts/oauth2/standards.md のうち、発行済みトークンの提示・内省・失効・
// 送信者制約に立つ 13 行を観測する。
//
// 入口は Register が組み立てたスタックの `/userinfo`、`/introspect`、`/revoke` である。
// この 13 行が言っているのは「受け取ったトークンをどう扱うか」なので、観測は必ず
// 発行済みのトークンを提示する形になる。検証関数の単体テストでは代わりにならない。
// 提示の検査はミドルウェアと同じ位置にあり、関数が正しくても配線されていなければ
// 素通りするからである。

import (
	stdcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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
	tpIssuer         = "http://test"
	tpClientID       = "token-presentation-client"
	tpClientSecret   = "token-presentation-client-secret"
	tpOtherClientID  = "token-presentation-other-client"
	tpOtherSecret    = "token-presentation-other-client-secret"
	tpRSClientID     = "token-presentation-resource-server"
	tpRSSecret       = "token-presentation-resource-server-secret"
	tpMTLSClientID   = "token-presentation-mtls-client"
	tpMTLSSubjectDN  = "CN=token-presentation-mtls-client"
	tpRedirectURI    = "https://app.example/cb"
	tpUsername       = "alice"
	tpUserID         = "user_alice"
	tpUserEmail      = "alice@example.test"
	tpPassword       = "token-presentation-password-1234"
	tpVerifier       = "token-presentation-standards-pkce-verifier-0123456789"
	tpUserInfoPath   = "/userinfo"
	tpIntrospectPath = "/introspect"
	tpRevokePath     = "/revoke"
	tpTokenPath      = "/token"
)

type tpFixture struct {
	base string
}

// newTokenPresentationFixture は本番と同じ Register で、トークンを発行する
// `/authorize` と `/token`、および発行済みトークンを受け取る `/userinfo`、
// `/introspect`、`/revoke` を 1 つのスタックへ載せる。
//
// DpopReplayStore と AccessTokenDenylist を持たせるのは、どちらも nil なら
// 該当の検査そのものが飛ぶためである。持たせずに観測すると、検査が働いた結果と
// 配線が無い結果を区別できない。
func newTokenPresentationFixture(t *testing.T) *tpFixture {
	t.Helper()
	now := time.Now().UTC()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(tpClientSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: tpClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{tpRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken, spec.GrantClientCredentials,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile email offline_access read",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	// 他クライアント所有のトークンを作るための 2 つ目のクライアント。
	otherHash := domain.HashClientSecret(tpOtherSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: tpOtherClientID, ClientSecretHash: &otherHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{tpRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken, spec.GrantClientCredentials,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "openid profile offline_access read",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	// 内省を要求するリソースサーバー。トークンを発行しないので、内省の応答が
	// 「自分が発行したものだから返せた」ではないことがこの分離で読める。
	rsHash := domain.HashClientSecret(tpRSSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: tpRSClientID, ClientSecretHash: &rsHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{tpRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantClientCredentials},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    "read",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	subjectDN := tpMTLSSubjectDN
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: tpMTLSClientID, ClientType: spec.ClientConfidential,
		RedirectURIs: []string{tpRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantClientCredentials,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodTlsClientAuth,
		TlsClientAuthSubjectDN:   &subjectDN,
		Scope:                    "openid profile read",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})

	hasher := testingpasswords.NewHasher()
	passwordHash, err := hasher.Hash(tpPassword)
	if err != nil {
		t.Fatal(err)
	}
	email := tpUserEmail
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: tpUserID, PreferredUsername: tpUsername, PasswordHash: passwordHash,
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
	signer := tokensJOSE.NewJWTSigner(tpIssuer, keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())

	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          tpIssuer,
		TenantRepo:      tenants,
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		OAuth2: oauth2.Module{
			ClientRepo: clients, ConsentRepo: oauth2memory.NewConsentRepository(),
			RequestStore:               oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:                  oauth2memory.NewAuthorizationCodeStore(),
			PARStore:                   oauth2memory.NewPARStore(),
			RefreshStore:               oauth2memory.NewRefreshTokenStore(),
			McpResourceServerRepo:      oauth2memory.NewMcpResourceServerRepository(),
			ClientAssertionReplayStore: oauth2memory.NewClientAssertionReplayStore(),
			DpopReplayStore:            oauth2memory.NewDpopReplayStore(),
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
	return &tpFixture{base: server.URL + "/realms/default"}
}

// htu は DPoP proof が名乗るべき絶対 URL を返す。製品が期待値を組み立てるのと
// 同じ形 (設定された issuer + リクエストのパス) をテスト側でも 1 か所に置く。
func tpHTU(path string) string {
	return tpIssuer + "/realms/default" + path
}

// postToken は `/token` へフォームを 1 通送る。decorate が無ければ既定の
// confidential クライアントの Basic 認証を付ける。
func (f *tpFixture) postToken(
	t *testing.T, form url.Values, decorate ...func(*http.Request),
) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, f.base+tpTokenPath, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if len(decorate) == 0 {
		request.SetBasicAuth(tpClientID, tpClientSecret)
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

type tpTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

func (f *tpFixture) mustToken(
	t *testing.T, form url.Values, decorate ...func(*http.Request),
) tpTokenResponse {
	t.Helper()
	status, body := f.postToken(t, form, decorate...)
	if status != http.StatusOK {
		t.Fatalf("/token status=%d body=%s", status, body)
	}
	var parsed tpTokenResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("/token 応答が JSON ではない: %v body=%s", err, body)
	}
	return parsed
}

// authorizationCode は正式な入口だけを通して認可コードを 1 本取る。
// 保存層へ直接置くと、交換の前提が製品の経路と違ってしまう。
func (f *tpFixture) authorizationCode(t *testing.T, clientID, scope string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(tpVerifier))
	query := url.Values{
		"client_id": {clientID}, "redirect_uri": {tpRedirectURI},
		"response_type": {"code"}, "scope": {scope}, "state": {"state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
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
		CSRFToken string `json:"csrf_token"`
	}](t, client, f.base+"/api/auth/transaction")
	next := postJSON[map[string]string](t, client, f.base+"/api/auth/login", transaction.CSRFToken,
		map[string]string{"username": tpUsername, "password": tpPassword})
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

func tpCodeExchangeForm(clientID, code string) url.Values {
	return url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"code_verifier": {tpVerifier}, "redirect_uri": {tpRedirectURI},
		"client_id": {clientID},
	}
}

// userToken は正式な入口だけを通して、利用者を主体とするアクセストークンを 1 本取る。
func (f *tpFixture) userToken(t *testing.T, scope string) tpTokenResponse {
	t.Helper()
	code := f.authorizationCode(t, tpClientID, scope)
	return f.mustToken(t, tpCodeExchangeForm(tpClientID, code))
}

// userInfo は `/userinfo` を 1 回叩く。Authorization の値をそのまま渡すので、
// スキームを変えた提示も、ヘッダーを付けない提示も同じ 1 つの関数で書ける。
func (f *tpFixture) userInfo(
	t *testing.T, authorization string, header http.Header, query url.Values,
) (int, map[string]any) {
	t.Helper()
	target := f.base + tpUserInfoPath
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	request, err := http.NewRequest(http.MethodGet, target, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	maps.Copy(request.Header, header)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET /userinfo: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("/userinfo 応答が JSON ではない: %v body=%s", err, raw)
	}
	return response.StatusCode, body
}

// reachedUserInfo は、利用者の主体が実際に返ったかを返す。状態コードだけでは、
// 拒否しながら claim を書く実装と区別できない。
func reachedUserInfo(status int, body map[string]any) bool {
	return status == http.StatusOK && body["sub"] == tpUserID
}

// introspect は `/introspect` を 1 回叩く。decorate が無ければリソースサーバーの
// Basic 認証を付ける。
func (f *tpFixture) introspect(
	t *testing.T, token, hint string, decorate ...func(*http.Request),
) (int, map[string]any) {
	t.Helper()
	form := url.Values{"token": {token}}
	if hint != "" {
		form.Set("token_type_hint", hint)
	}
	request, err := http.NewRequest(
		http.MethodPost, f.base+tpIntrospectPath, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if len(decorate) == 0 {
		request.SetBasicAuth(tpRSClientID, tpRSSecret)
	}
	for _, apply := range decorate {
		apply(request)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /introspect: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("/introspect 応答が JSON ではない: %v body=%s", err, raw)
	}
	return response.StatusCode, body
}

// revoke は `/revoke` を 1 回叩く。decorate が無ければトークンを発行した
// クライアントの Basic 認証を付ける。
func (f *tpFixture) revoke(
	t *testing.T, token string, decorate ...func(*http.Request),
) (int, string) {
	t.Helper()
	form := url.Values{"token": {token}}
	request, err := http.NewRequest(
		http.MethodPost, f.base+tpRevokePath, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if len(decorate) == 0 {
		request.SetBasicAuth(tpClientID, tpClientSecret)
	}
	for _, apply := range decorate {
		apply(request)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /revoke: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

// tpClientCertificate は自己署名のクライアント証明書を 1 枚作り、製品が受け取る
// ヘッダー値とサムプリントを返す。ヘッダーに生の改行は置けないので、
// 製品の復号と同じく URL エンコードした PEM を渡す。
func tpClientCertificate(t *testing.T, commonName string) (header, thumbprint string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(der)
	return url.QueryEscape(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))),
		base64.RawURLEncoding.EncodeToString(sum[:])
}

// tpDPoPProofInput は DPoP proof の 1 要素だけを崩すために、証明のすべての要素を
// 呼び出し側から与えられる形にする。
type tpDPoPProofInput struct {
	htm, htu, jti, ath string
	iat                time.Time
	typ, alg           string
}

// tpDPoPKey は DPoP の鍵と、その JWK およびサムプリント (jkt) を作る。
func tpDPoPKey(t *testing.T) (*rsa.PrivateKey, map[string]any, string) {
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

func tpDPoPProof(t *testing.T, key *rsa.PrivateKey, jwk map[string]any, in tpDPoPProofInput) string {
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
	signature, err := rsa.SignPSS(
		rand.Reader, key, stdcrypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// =====================================================================
// RFC 6750 — Bearer Token Usage
// =====================================================================

// RFC6750-AUTHORIZATION-HEADER (required) / RFC6750-QUERY-TOKEN (excluded):
// ベアラーアクセストークンを受け付ける提示の形は Authorization ヘッダーだけであり、
// URI のクエリパラメーターによる提示は提供していない。
//
// `RFC6750-QUERY-TOKEN` の Statement は製品の制約ではなく標準側の機能を書いているので、
// 観測は「その提示が通らないこと」と「通らなかった結果として主体が漏れていないこと」の
// 対になる。同じ 1 本のトークンをヘッダーとクエリで送り分けるのが要点である。別々の
// トークンで比べると、最初から無効だった実装と区別できない。
func TestBearerTokenIsAcceptedOnlyFromTheAuthorizationHeader(t *testing.T) {
	fixture := newTokenPresentationFixture(t)
	issued := fixture.userToken(t, "openid profile")

	// 対照: 同じ 1 本がヘッダーでは通る。
	status, body := fixture.userInfo(t, "Bearer "+issued.AccessToken, nil, nil)
	if !reachedUserInfo(status, body) {
		t.Fatalf("Authorization ヘッダーの Bearer が通らない: status=%d body=%v", status, body)
	}

	for _, tc := range []struct {
		name          string
		authorization string
		query         url.Values
	}{
		{
			name:  "query parameter instead of the header",
			query: url.Values{"access_token": {issued.AccessToken}},
		},
		{
			// ヘッダーを付けたうえでクエリにも載せる。ヘッダー側が有効なので、
			// クエリを読む実装であっても読まない実装であっても通ってしまう組み合わせは
			// 作れない。ここで読んでいるのは、クエリが提示の形として増えないことである。
			name:          "query parameter with no header",
			authorization: "",
			query:         url.Values{"access_token": {issued.AccessToken}, "token": {issued.AccessToken}},
		},
		{
			name:          "another authorization scheme",
			authorization: "Token " + issued.AccessToken,
		},
		{
			name:          "no authorization at all",
			authorization: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.userInfo(t, tc.authorization, nil, tc.query)
			if reachedUserInfo(status, body) {
				t.Fatalf("ヘッダー以外の提示で UserInfo へ到達した: status=%d", status)
			}
			for _, claim := range []string{"sub", "preferred_username", "email"} {
				if _, ok := body[claim]; ok {
					t.Fatalf("拒否された応答が %s を運んでいる: %v", claim, body)
				}
			}
		})
	}
}

// =====================================================================
// OpenID Connect Core 1.0 — UserInfo
// =====================================================================

// OIDC-CORE-USERINFO (required): `openid` スコープのアクセストークンに対して
// `sub` を含む UserInfo を返す。
//
// 200 が返ることだけでは足りない。`sub` が誰のものかを読まないと、別の利用者の
// 主体を返す実装を見分けられない。スコープが効いていることは、`openid` を持たない
// トークンと対で読む。`openid` の有無だけが違う 2 本を、同じ利用者について作る。
func TestUserInfoReturnsTheSubjectForAnOpenIDScopedToken(t *testing.T) {
	fixture := newTokenPresentationFixture(t)

	withOpenID := fixture.userToken(t, "openid profile email")
	status, body := fixture.userInfo(t, "Bearer "+withOpenID.AccessToken, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("/userinfo status=%d body=%v", status, body)
	}
	if body["sub"] != tpUserID {
		t.Fatalf("sub=%v, want %q", body["sub"], tpUserID)
	}
	// 同じ応答から、要求したスコープに対応する claim も読む。`sub` だけを読むテストは、
	// スコープを無視して常に最小の応答を返す実装を見分けられない。
	if body["preferred_username"] != tpUsername {
		t.Errorf("preferred_username=%v, want %q", body["preferred_username"], tpUsername)
	}
	if body["email"] != tpUserEmail {
		t.Errorf("email=%v, want %q", body["email"], tpUserEmail)
	}

	withoutOpenID := fixture.userToken(t, "profile email")
	status, body = fixture.userInfo(t, "Bearer "+withoutOpenID.AccessToken, nil, nil)
	if status != http.StatusForbidden {
		t.Fatalf("openid 無しの status=%d, want 403 (body=%v)", status, body)
	}
	if _, ok := body["sub"]; ok {
		t.Fatalf("openid 無しの応答が sub を運んでいる: %v", body)
	}
}

// =====================================================================
// RFC 7662 — Token Introspection
// =====================================================================

// RFC7662-INTROSPECT (required): 認証済みのリソースサーバーへ `active` と
// 許可されたメタデータを返す。
//
// 返した値が、提示したトークンそのもののものであることを 1 つずつ照合する。
// `active` だけを読むテストは、別のトークンの内容を返す実装も `scope` を落とす
// 実装も見分けられない。照合の相手はアクセストークンを復号した payload である。
//
// 「認証済みの」という限定は、認証を持たない同じリクエストが同じ答えを得ないこと
// でしか読めない。認証なしの内省を対に置く。
func TestIntrospectionAnswersAuthenticatedResourceServers(t *testing.T) {
	fixture := newTokenPresentationFixture(t)
	issued := fixture.userToken(t, "openid profile offline_access")
	_, claims := jwtParts(t, issued.AccessToken)

	status, body := fixture.introspect(t, issued.AccessToken, "access_token")
	if status != http.StatusOK {
		t.Fatalf("/introspect status=%d body=%v", status, body)
	}
	if body["active"] != true {
		t.Fatalf("active=%v, want true (body=%v)", body["active"], body)
	}
	for _, claim := range []string{"sub", "client_id", "exp", "iat", "jti"} {
		if body[claim] != claims[claim] {
			t.Errorf("%s=%v, want %v (発行したトークンの値)", claim, body[claim], claims[claim])
		}
	}
	if body["scope"] != claims["scope"] {
		t.Errorf("scope=%v, want %v", body["scope"], claims["scope"])
	}
	// `token_type` は、内省した値がアクセストークンとリフレッシュトークンの
	// どちらだったかを区別する。この行が要求しているのはメタデータが返ること
	// なので、ここで読んでいるのは 2 種類が同じ値にならないことである。返る値が
	// RFC 7662 §2.2 の言う RFC 6749 §5.1 の種別 (`Bearer`) ではないことは
	// [[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] が扱う。
	if body["token_type"] != "access_token" {
		t.Errorf("token_type=%v, want access_token", body["token_type"])
	}

	// リフレッシュトークンも同じエンドポイントで内省できる。JWT ではないので、
	// 照合の相手は発行時に返った値と、アクセストークンが名乗る主体になる。
	status, body = fixture.introspect(t, issued.RefreshToken, "refresh_token")
	if status != http.StatusOK || body["active"] != true {
		t.Fatalf("リフレッシュトークンの内省 status=%d body=%v", status, body)
	}
	if body["token_type"] != "refresh_token" {
		t.Errorf("token_type=%v, want refresh_token", body["token_type"])
	}
	if body["sub"] != claims["sub"] || body["client_id"] != claims["client_id"] {
		t.Errorf("リフレッシュトークンの主体 sub=%v client_id=%v, want %v / %v",
			body["sub"], body["client_id"], claims["sub"], claims["client_id"])
	}

	// 認証を持たないリソースサーバーには、同じトークンについて何も返らない。
	for _, tc := range []struct {
		name     string
		decorate func(*http.Request)
	}{
		{name: "no client authentication", decorate: func(*http.Request) {}},
		{name: "wrong client secret", decorate: func(request *http.Request) {
			request.SetBasicAuth(tpRSClientID, tpRSSecret+"-wrong")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.introspect(t, issued.AccessToken, "access_token", tc.decorate)
			if status == http.StatusOK {
				t.Fatalf("認証なしの内省が 200 を返した: %v", body)
			}
			if body["active"] == true {
				t.Fatalf("認証なしの内省が active=true を返した: %v", body)
			}
			for _, claim := range []string{"sub", "client_id", "scope", "jti"} {
				if _, ok := body[claim]; ok {
					t.Fatalf("認証なしの応答が %s を運んでいる: %v", claim, body)
				}
			}
		})
	}
}

// RFC7662-INACTIVE (required): 無効なトークンには `active=false` だけを返す。
//
// `active` が false であることに加えて、応答が他の鍵を 1 つも持たないことを読む。
// 4 通りの入力が同じ 1 つの本文になることが、存在を漏らさないということである。
// 未知のトークンと失効済みのトークンを見分けられる応答は、その差だけで
// 「そのトークンは実在した」と告げてしまう。
func TestIntrospectionRevealsNothingAboutInactiveTokens(t *testing.T) {
	fixture := newTokenPresentationFixture(t)

	revoked := fixture.userToken(t, "openid profile offline_access")
	if status, body := fixture.introspect(t, revoked.AccessToken, "access_token"); body["active"] != true {
		t.Fatalf("前提が壊れている: 失効前の内省が status=%d active=%v", status, body["active"])
	}
	if status, body := fixture.revoke(t, revoked.AccessToken); status != http.StatusOK {
		t.Fatalf("前提が壊れている: revoke status=%d body=%s", status, body)
	}

	rotated := fixture.userToken(t, "openid profile offline_access")
	// ローテーション済みのリフレッシュトークンを作る。使い終えた値は active ではない。
	fixture.mustToken(t, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {rotated.RefreshToken},
	})

	revokedRefresh := fixture.userToken(t, "openid profile offline_access")
	if status, body := fixture.revoke(t, revokedRefresh.RefreshToken); status != http.StatusOK {
		t.Fatalf("前提が壊れている: refresh の revoke status=%d body=%s", status, body)
	}

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "never issued", token: "not.a.token"},
		{name: "revoked access token", token: revoked.AccessToken},
		{name: "rotated refresh token", token: rotated.RefreshToken},
		{name: "revoked refresh token", token: revokedRefresh.RefreshToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.introspect(t, tc.token, "")
			if status != http.StatusOK {
				t.Fatalf("status=%d, want 200 (body=%v)", status, body)
			}
			if body["active"] != false {
				t.Fatalf("active=%v, want false (body=%v)", body["active"], body)
			}
			if len(body) != 1 {
				t.Fatalf("body=%v, want active だけ", body)
			}
		})
	}
}

// =====================================================================
// RFC 7009 — Token Revocation
// =====================================================================

// RFC7009-REVOCATION-ENDPOINT (required): 認証済みクライアントへトークン失効
// エンドポイントを提供する。
//
// 失効は失効前の成功と対で観測する。失効後の拒否だけでは、そのトークンが最初から
// 通らなかった実装と区別できない。アクセストークンは保護リソースへの到達可否で、
// リフレッシュトークンは新しいトークンが出るかどうかで読む。
//
// 「認証済みクライアントへ」の限定は、認証を持たない失効要求が同じ効果を持たない
// ことでしか読めない。資格情報を欠いた要求の後もトークンが生きていることを、
// 応答ではなく保護リソースへの到達で確かめる。
func TestRevocationIsOfferedToAuthenticatedClients(t *testing.T) {
	fixture := newTokenPresentationFixture(t)

	issued := fixture.userToken(t, "openid profile offline_access")
	if status, body := fixture.userInfo(t, "Bearer "+issued.AccessToken, nil, nil); !reachedUserInfo(status, body) {
		t.Fatalf("前提が壊れている: 失効前のトークンが通らない status=%d body=%v", status, body)
	}

	// 資格情報を欠いた失効要求は効果を持たない。
	if status, _ := fixture.revoke(t, issued.AccessToken, func(request *http.Request) {
		request.SetBasicAuth(tpClientID, tpClientSecret+"-wrong")
	}); status == http.StatusOK {
		t.Fatal("誤った資格情報の失効要求が受理された")
	}
	if status, body := fixture.userInfo(t, "Bearer "+issued.AccessToken, nil, nil); !reachedUserInfo(status, body) {
		t.Fatalf("認証されない失効要求がトークンを失効させた: status=%d body=%v", status, body)
	}

	// 認証済みクライアントの失効要求は、その場で効く。
	if status, body := fixture.revoke(t, issued.AccessToken); status != http.StatusOK {
		t.Fatalf("失効 status=%d body=%s", status, body)
	}
	if status, body := fixture.userInfo(t, "Bearer "+issued.AccessToken, nil, nil); reachedUserInfo(status, body) {
		t.Fatalf("失効させたアクセストークンが保護リソースへ到達した: status=%d", status)
	}
	if status, body := fixture.introspect(t, issued.AccessToken, "access_token"); body["active"] != false {
		t.Fatalf("失効させたアクセストークンの内省が active=%v (status=%d)", body["active"], status)
	}

	// リフレッシュトークンの失効も同じエンドポイントで受け付ける。
	if status, body := fixture.revoke(t, issued.RefreshToken); status != http.StatusOK {
		t.Fatalf("リフレッシュトークンの失効 status=%d body=%s", status, body)
	}
	status, body := fixture.postToken(t, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {issued.RefreshToken},
	})
	if status == http.StatusOK {
		t.Fatalf("失効させたリフレッシュトークンで新しいトークンが出た: %s", body)
	}
	assertNoTokenInBody(t, "失効させたリフレッシュトークン", body)
}

// RFC7009-UNKNOWN-TOKEN (required): 無効または他クライアント所有のトークンに対しても
// 成功応答を返し、情報を漏らさない。
//
// 3 通りの入力の応答が状態コードも本文も区別できないことを読む。片方だけを読む
// テストは、未知のトークンに 400 を返す実装を見分けられるが、本文で存在を漏らす
// 実装は見逃す。併せて、他クライアント所有のトークンが巻き添えで失効していない
// ことを、応答ではなく保護リソースへの到達で確かめる。成功応答を返しながら
// 失効させる実装は、応答だけを見るテストでは通ってしまう。
func TestRevokingAnUnknownOrForeignTokenIsAnIndistinguishableNoOp(t *testing.T) {
	fixture := newTokenPresentationFixture(t)

	// 他クライアントが所有するトークン。失効を要求するのは既定のクライアントである。
	foreignCode := fixture.authorizationCode(t, tpOtherClientID, "openid profile offline_access")
	foreign := fixture.mustToken(t, tpCodeExchangeForm(tpOtherClientID, foreignCode),
		func(request *http.Request) { request.SetBasicAuth(tpOtherClientID, tpOtherSecret) })
	if status, body := fixture.userInfo(t, "Bearer "+foreign.AccessToken, nil, nil); !reachedUserInfo(status, body) {
		t.Fatalf("前提が壊れている: 他クライアントのトークンが通らない status=%d body=%v", status, body)
	}

	// 対照: 自分が所有するトークンの失効要求。応答はこの 1 つと同じでなければならない。
	own := fixture.userToken(t, "openid profile offline_access")
	ownStatus, ownBody := fixture.revoke(t, own.AccessToken)
	if ownStatus != http.StatusOK {
		t.Fatalf("前提が壊れている: 自分のトークンの失効 status=%d body=%s", ownStatus, ownBody)
	}

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "never issued", token: "not.a.token"},
		{name: "access token owned by another client", token: foreign.AccessToken},
		{name: "refresh token owned by another client", token: foreign.RefreshToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.revoke(t, tc.token)
			if status != ownStatus || body != ownBody {
				t.Fatalf("応答が自分のトークンの失効と区別できる: status=%d body=%q, want status=%d body=%q",
					status, body, ownStatus, ownBody)
			}
		})
	}

	// 他クライアントのトークンは巻き添えで失効していない。
	if status, body := fixture.userInfo(t, "Bearer "+foreign.AccessToken, nil, nil); !reachedUserInfo(status, body) {
		t.Fatalf("他クライアントのアクセストークンが失効した: status=%d body=%v", status, body)
	}
	refreshed := fixture.mustToken(t, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {foreign.RefreshToken},
	}, func(request *http.Request) { request.SetBasicAuth(tpOtherClientID, tpOtherSecret) })
	if refreshed.AccessToken == "" {
		t.Fatal("他クライアントのリフレッシュトークンが失効した")
	}
}

// =====================================================================
// RFC 7800 / RFC 9700 — 確認鍵と送信者制約
// =====================================================================

// RFC7800-CONFIRMATION (optional) / RFC9700-SENDER-CONSTRAINT (optional):
// 送信者制約付きトークンの確認鍵情報を `cnf` クレームに格納し、送信者制約は
// DPoP と mTLS の 2 通りから選べる。
//
// 選べることの観測なので、制約なし・DPoP・mTLS の 3 本を同じ入口で作って比べる。
// 制約ありの 1 本だけを見ても、常に制約を付ける実装と区別できない。
//
// `cnf` は復号した payload から直接読む。到達できるかどうかの観測では、確認鍵を
// 内省の応答でだけ組み立てる実装 (トークン自体には載せない実装) を見分けられない。
// 併せて、載っている確認鍵が飾りでないことを、鍵を持たない提示が拒否されることで読む。
func TestSenderConstraintIsRecordedInCnfAndCheckedAtTheResource(t *testing.T) {
	fixture := newTokenPresentationFixture(t)
	key, jwk, jkt := tpDPoPKey(t)
	certificate, thumbprint := tpClientCertificate(t, "token-presentation-mtls-client")
	now := time.Now().UTC()

	// 1. 制約なし。cnf を持たず、ヘッダーだけで通る。
	plain := fixture.userToken(t, "openid profile")
	if _, claims := jwtParts(t, plain.AccessToken); claims["cnf"] != nil {
		t.Fatalf("制約なしのトークンが cnf を持っている: %v", claims["cnf"])
	}
	if plain.TokenType != "Bearer" {
		t.Errorf("制約なしの token_type=%q, want Bearer", plain.TokenType)
	}
	if status, body := fixture.userInfo(t, "Bearer "+plain.AccessToken, nil, nil); !reachedUserInfo(status, body) {
		t.Fatalf("制約なしのトークンが通らない: status=%d body=%v", status, body)
	}

	// 2. DPoP。cnf.jkt が提示鍵のサムプリントになる。
	dpopCode := fixture.authorizationCode(t, tpClientID, "openid profile")
	dpopIssued := fixture.mustToken(t, tpCodeExchangeForm(tpClientID, dpopCode),
		func(request *http.Request) {
			request.SetBasicAuth(tpClientID, tpClientSecret)
			request.Header.Set("DPoP", tpDPoPProof(t, key, jwk, tpDPoPProofInput{
				htm: http.MethodPost, htu: tpHTU(tpTokenPath), jti: "cnf-token", iat: now,
			}))
		})
	_, dpopClaims := jwtParts(t, dpopIssued.AccessToken)
	confirmation, ok := dpopClaims["cnf"].(map[string]any)
	if !ok || confirmation["jkt"] != jkt {
		t.Fatalf("DPoP トークンの cnf=%v, want jkt=%q", dpopClaims["cnf"], jkt)
	}
	// 確認鍵は飾りではない。証明を持たない提示は保護リソースへ届かない。
	if status, body := fixture.userInfo(t, "DPoP "+dpopIssued.AccessToken, nil, nil); reachedUserInfo(status, body) {
		t.Fatalf("DPoP 束縛トークンが証明なしで通った: status=%d", status)
	}
	proof := tpDPoPProof(t, key, jwk, tpDPoPProofInput{
		htm: http.MethodGet, htu: tpHTU(tpUserInfoPath), jti: "cnf-resource", iat: now,
		ath: tokensJOSE.AccessTokenHash(dpopIssued.AccessToken),
	})
	status, body := fixture.userInfo(t, "DPoP "+dpopIssued.AccessToken, http.Header{"Dpop": {proof}}, nil)
	if !reachedUserInfo(status, body) {
		t.Fatalf("DPoP 束縛トークンが正しい証明で通らない: status=%d body=%v", status, body)
	}

	// 3. mTLS。cnf["x5t#S256"] が提示証明書のサムプリントになる。
	mtlsCode := fixture.authorizationCode(t, tpMTLSClientID, "openid profile")
	mtlsIssued := fixture.mustToken(t, tpCodeExchangeForm(tpMTLSClientID, mtlsCode),
		func(request *http.Request) { request.Header.Set("X-Client-Certificate", certificate) })
	_, mtlsClaims := jwtParts(t, mtlsIssued.AccessToken)
	confirmation, ok = mtlsClaims["cnf"].(map[string]any)
	if !ok || confirmation["x5t#S256"] != thumbprint {
		t.Fatalf("mTLS トークンの cnf=%v, want x5t#S256=%q", mtlsClaims["cnf"], thumbprint)
	}
	if status, body := fixture.userInfo(t, "Bearer "+mtlsIssued.AccessToken, nil, nil); reachedUserInfo(status, body) {
		t.Fatalf("mTLS 束縛トークンが証明書なしで通った: status=%d", status)
	}
	status, body = fixture.userInfo(t, "Bearer "+mtlsIssued.AccessToken,
		http.Header{"X-Client-Certificate": {certificate}}, nil)
	if !reachedUserInfo(status, body) {
		t.Fatalf("mTLS 束縛トークンが正しい証明書で通らない: status=%d body=%v", status, body)
	}

	// 内省の応答も同じ確認鍵を運ぶ。リソースサーバーが束縛を読める経路はここしかない。
	_, introspected := fixture.introspect(t, dpopIssued.AccessToken, "access_token")
	if cnf, ok := introspected["cnf"].(map[string]any); !ok || cnf["jkt"] != jkt {
		t.Errorf("内省の cnf=%v, want jkt=%q", introspected["cnf"], jkt)
	}
	_, introspected = fixture.introspect(t, mtlsIssued.AccessToken, "access_token")
	if cnf, ok := introspected["cnf"].(map[string]any); !ok || cnf["x5t#S256"] != thumbprint {
		t.Errorf("内省の cnf=%v, want x5t#S256=%q", introspected["cnf"], thumbprint)
	}
}

// =====================================================================
// RFC 8705 — mTLS クライアント認証と証明書束縛トークン
// =====================================================================

// RFC8705-CLIENT-AUTH (optional) / RFC8705-CERT-BOUND (optional):
// 登録済みの Subject DN と検証済みクライアント証明書を照合してクライアントを認証し、
// 発行したアクセストークンを証明書のサムプリントへ束縛して、リソースへのアクセス時に
// 照合する。
//
// 照合が働いていることは、別の Subject DN を持つ正しい形式の証明書で読む。壊れた
// ヘッダーでは形式の検証で先に落ちるので、照合が無い実装でも同じ拒否になる。
// 束縛の側も同じで、別の正しい証明書を提示して、サムプリントの一致だけが差になる形で読む。
func TestMutualTLSAuthenticatesTheClientAndBindsTheAccessTokenToItsCertificate(t *testing.T) {
	fixture := newTokenPresentationFixture(t)
	registered, thumbprint := tpClientCertificate(t, "token-presentation-mtls-client")
	stranger, strangerThumbprint := tpClientCertificate(t, "someone-else")
	if thumbprint == strangerThumbprint {
		t.Fatal("前提が壊れている: 2 枚の証明書のサムプリントが同じである")
	}

	// 登録済みの Subject DN と一致する証明書はクライアントを認証する。
	issued := fixture.mustToken(t, url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"},
		"client_id": {tpMTLSClientID},
	}, func(request *http.Request) { request.Header.Set("X-Client-Certificate", registered) })
	if issued.AccessToken == "" {
		t.Fatal("mTLS クライアント認証でトークンが出ない")
	}

	// 別の Subject DN を名乗る証明書は、同じクライアント ID では認証されない。
	status, body := fixture.postToken(t, url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"},
		"client_id": {tpMTLSClientID},
	}, func(request *http.Request) { request.Header.Set("X-Client-Certificate", stranger) })
	if status == http.StatusOK {
		t.Fatalf("登録されていない Subject DN の証明書が受理された: %s", body)
	}
	assertNoTokenInBody(t, "別の Subject DN", body)

	// 発行したトークンは提示された証明書のサムプリントへ束縛される。
	code := fixture.authorizationCode(t, tpMTLSClientID, "openid profile")
	bound := fixture.mustToken(t, tpCodeExchangeForm(tpMTLSClientID, code),
		func(request *http.Request) { request.Header.Set("X-Client-Certificate", registered) })
	_, claims := jwtParts(t, bound.AccessToken)
	confirmation, ok := claims["cnf"].(map[string]any)
	if !ok || confirmation["x5t#S256"] != thumbprint {
		t.Fatalf("cnf=%v, want x5t#S256=%q", claims["cnf"], thumbprint)
	}

	// リソースへのアクセス時に照合される。
	status, responseBody := fixture.userInfo(t, "Bearer "+bound.AccessToken,
		http.Header{"X-Client-Certificate": {registered}}, nil)
	if !reachedUserInfo(status, responseBody) {
		t.Fatalf("束縛した証明書の提示で通らない: status=%d body=%v", status, responseBody)
	}
	for _, tc := range []struct {
		name   string
		header http.Header
	}{
		{name: "another valid certificate", header: http.Header{"X-Client-Certificate": {stranger}}},
		{name: "no certificate at all", header: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.userInfo(t, "Bearer "+bound.AccessToken, tc.header, nil)
			if reachedUserInfo(status, body) {
				t.Fatalf("束縛と一致しない提示でリソースへ到達した: status=%d", status)
			}
		})
	}
}

// =====================================================================
// RFC 9449 — DPoP
// =====================================================================

// RFC9449-PROOF (optional) / RFC9449-ATH (optional):
// DPoP proof の署名、`htm`、`htu`、`iat`、`jti` を検証し、保護リソースへ提示する
// proof には `ath` を要求して `base64url(SHA-256(access_token))` と照合する。
//
// 行が挙げる要素ごとに、1 要素だけを崩した証明を作る。崩していない要素が有効である
// ことは、無傷の証明が通ることで先に確かめる。まとめて壊した証明では、どの検証が
// 働いたのか分からない。
//
// トークンエンドポイント側は `client_credentials` で観測する。認可コードを使うと、
// 1 度成功した時点でコードが消えるので、`jti` のリプレイを「同じ入力の 2 回目」
// として送れない。`ath` はトークンエンドポイントでは要求されない (RFC 9449 §4.3:
// 束縛先のアクセストークンがまだ存在しない) ので、保護リソース側で観測する。
func TestDPoPProofElementsAreVerifiedAtTheTokenEndpointAndTheProtectedResource(t *testing.T) {
	fixture := newTokenPresentationFixture(t)
	key, jwk, jkt := tpDPoPKey(t)
	otherKey, _, _ := tpDPoPKey(t)
	now := time.Now().UTC()

	clientCredentials := url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"},
	}
	intact := func(jti string) tpDPoPProofInput {
		return tpDPoPProofInput{htm: http.MethodPost, htu: tpHTU(tpTokenPath), jti: jti, iat: now}
	}
	present := func(proof string) (int, string) {
		return fixture.postToken(t, clientCredentials, func(request *http.Request) {
			request.SetBasicAuth(tpClientID, tpClientSecret)
			request.Header.Set("DPoP", proof)
		})
	}

	status, body := present(tpDPoPProof(t, key, jwk, intact("intact-1")))
	if status != http.StatusOK {
		t.Fatalf("前提が壊れている: 無傷の DPoP 証明が通らない status=%d body=%s", status, body)
	}
	var first tpTokenResponse
	if err := json.Unmarshal([]byte(body), &first); err != nil {
		t.Fatal(err)
	}
	if _, claims := jwtParts(t, first.AccessToken); claims["cnf"] == nil {
		t.Fatal("前提が壊れている: 証明が通ったのにトークンが束縛されていない")
	}
	if first.TokenType != "DPoP" {
		t.Errorf("token_type=%q, want DPoP", first.TokenType)
	}

	// リプレイ: 直前に通った証明をそのまま送り直す。jti が同じでも通る実装をここで落とす。
	replayed := tpDPoPProof(t, key, jwk, intact("replayed-1"))
	if status, body := present(replayed); status != http.StatusOK {
		t.Fatalf("前提が壊れている: 1 回目の提示が通らない status=%d body=%s", status, body)
	}
	if status, body := present(replayed); status == http.StatusOK {
		t.Fatalf("同じ jti の証明が 2 回通った: %s", body)
	}

	for _, tc := range []struct {
		name  string
		proof func() string
	}{
		{name: "signature made by another key", proof: func() string {
			// jwk は無傷のまま、署名だけを別の鍵で作る。落ちるとすれば署名の検証しかない。
			return tpDPoPProof(t, otherKey, jwk, intact("bad-signature"))
		}},
		{name: "htm of another method", proof: func() string {
			in := intact("bad-htm")
			in.htm = http.MethodGet
			return tpDPoPProof(t, key, jwk, in)
		}},
		{name: "htu of another endpoint", proof: func() string {
			in := intact("bad-htu")
			in.htu = tpHTU(tpRevokePath)
			return tpDPoPProof(t, key, jwk, in)
		}},
		{name: "iat outside the clock skew", proof: func() string {
			in := intact("stale-iat")
			in.iat = now.Add(-2 * time.Hour)
			return tpDPoPProof(t, key, jwk, in)
		}},
		{name: "no jti", proof: func() string {
			return tpDPoPProof(t, key, jwk, intact(""))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := present(tc.proof())
			if status == http.StatusOK {
				t.Fatalf("崩した証明でトークンが出た: %s", body)
			}
			assertNoTokenInBody(t, tc.name, body)
		})
	}

	// 保護リソース側。ath を要求し、提示されたアクセストークンと照合する。
	code := fixture.authorizationCode(t, tpClientID, "openid profile")
	bound := fixture.mustToken(t, tpCodeExchangeForm(tpClientID, code), func(request *http.Request) {
		request.SetBasicAuth(tpClientID, tpClientSecret)
		request.Header.Set("DPoP", tpDPoPProof(t, key, jwk, intact("resource-token")))
	})
	if _, claims := jwtParts(t, bound.AccessToken); claims["cnf"] == nil {
		t.Fatalf("前提が壊れている: 束縛されていないトークンで ath を観測しようとしている jkt=%q", jkt)
	}
	resourceProof := func(jti, ath string) string {
		return tpDPoPProof(t, key, jwk, tpDPoPProofInput{
			htm: http.MethodGet, htu: tpHTU(tpUserInfoPath), jti: jti, ath: ath, iat: now,
		})
	}
	status, claims := fixture.userInfo(t, "DPoP "+bound.AccessToken,
		http.Header{"Dpop": {resourceProof("ath-intact", tokensJOSE.AccessTokenHash(bound.AccessToken))}}, nil)
	if !reachedUserInfo(status, claims) {
		t.Fatalf("前提が壊れている: 正しい ath の証明が通らない status=%d body=%v", status, claims)
	}

	other := fixture.userToken(t, "openid profile")
	for _, tc := range []struct {
		name string
		ath  string
	}{
		{name: "no ath", ath: ""},
		{name: "ath of another access token", ath: tokensJOSE.AccessTokenHash(other.AccessToken)},
		{name: "ath that is not a hash", ath: "not-a-hash"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := fixture.userInfo(t, "DPoP "+bound.AccessToken,
				http.Header{"Dpop": {resourceProof("ath-"+tc.name, tc.ath)}}, nil)
			if reachedUserInfo(status, body) {
				t.Fatalf("ath が一致しない証明で保護リソースへ到達した: status=%d", status)
			}
			if status != http.StatusUnauthorized {
				t.Fatalf("status=%d, want 401 (body=%v)", status, body)
			}
		})
	}
}
