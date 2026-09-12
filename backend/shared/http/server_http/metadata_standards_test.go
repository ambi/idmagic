package server_http_test

// docs/contexts/oauth2/standards.md のうち、クライアントとリソースサーバーが製品の
// 設定を読む経路に立つ 7 行を観測する。
//
// 入口は Register が組み立てたスタックの `/.well-known/openid-configuration`、
// `/.well-known/oauth-authorization-server`、`/.well-known/oauth-protected-resource`、
// `/jwks`、および保護リソースが返す `WWW-Authenticate` である。文書を組み立てる関数の
// 単体テストでは代わりにならない。関数が正しくても、その文書が well-known のパスから
// 配られていなければ、適合クライアントは製品の設定を 1 度も読めないからである。
//
// **配信された宣言を読むだけでは足りない。** メタデータは宣言と実装がずれても取得
// できてしまうので、7 行とも「宣言した内容」と「宣言どおりに動くこと」を対で読む。
// 広告されたエンドポイントは実在するか、広告された鍵は発行済みトークンの署名を
// 検証できるか、広告された bearer 提示方法は実際に通るか、広告された
// `resource_metadata` URL は実際に文書を返すか、である。
//
// 発行済みトークンは正式な入口 (`/authorize` → ログイン → `/token`) だけを通して取る。
// 署名鍵や ID トークンをテスト側で組み立てると、「製品が配った鍵で製品が発行した
// トークンを検証できる」という当の対応関係が観測から消える。

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/authentication"
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
	mdIssuer       = "http://test"
	mdRealmPath    = "/realms/default"
	mdClientID     = "metadata-standards-client"
	mdClientSecret = "metadata-standards-client-secret"
	mdRedirectURI  = "https://app.example/cb"
	mdUsername     = "alice"
	mdUserID       = "user_alice"
	mdPassword     = "metadata-standards-password-1234"
	mdVerifier     = "metadata-standards-pkce-verifier-01234567890123"
	mdScope        = "openid profile email"

	// 登録済み McpResourceServer を 2 件置く。1 件だけでは、resource ごとに導出する
	// 実装と、resource を無視して固定の 1 文書を返す実装を区別できない。
	mdResourceA        = "https://tools.example/mcp"
	mdResourceB        = "https://reports.example/mcp"
	mdDisabledResource = "https://retired.example/mcp"
)

var (
	mdScopesA = []string{"tools:read", "tools:write"}
	mdScopesB = []string{"reports:read"}
)

type mdFixture struct {
	base string
}

// newMetadataFixture は cmd/internal/bootstrap と同じモジュール構成で、メタデータを
// 配る経路 (`/.well-known/*`、`/jwks`) と、その宣言どおりに動くことを読むための経路
// (`/authorize`、`/token`、`/userinfo`、保護された管理 API) を 1 つのスタックへ載せる。
//
// McpResourceServerRepo を空のまま渡さないのは、RFC9728-METADATA が「登録済みの
// McpResourceServer ごとに」と言っているためである。登録が 0 件のスタックでは、
// その「ごとに」が観測対象として存在しない。
func newMetadataFixture(t *testing.T) *mdFixture {
	t.Helper()
	now := time.Now().UTC()

	clients := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(mdClientSecret)
	clients.Seed(&domain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: mdClientID, ClientSecretHash: &secretHash,
		ClientType: spec.ClientConfidential, RedirectURIs: []string{mdRedirectURI},
		GrantTypes: []spec.GrantType{
			spec.GrantAuthorizationCode, spec.GrantRefreshToken,
		},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		Scope:                    mdScope,
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})

	resourceServers := oauth2memory.NewMcpResourceServerRepository()
	resourceServers.Seed(&domain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ID: "mcp-tools",
		Resource: mdResourceA, Name: "Tools", Scopes: mdScopesA,
		State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now,
	})
	resourceServers.Seed(&domain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ID: "mcp-reports",
		Resource: mdResourceB, Name: "Reports", Scopes: mdScopesB,
		State: domain.McpResourceServerActive, CreatedAt: now, UpdatedAt: now,
	})
	resourceServers.Seed(&domain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ID: "mcp-retired",
		Resource: mdDisabledResource, Name: "Retired", Scopes: []string{"retired:read"},
		State: domain.McpResourceServerDisabled, CreatedAt: now, UpdatedAt: now,
	})

	hasher := testingpasswords.NewHasher()
	passwordHash, err := hasher.Hash(mdPassword)
	if err != nil {
		t.Fatal(err)
	}
	email := "alice@example.test"
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: mdUserID, PreferredUsername: mdUsername, PasswordHash: passwordHash,
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
	signer := tokensJOSE.NewJWTSigner(mdIssuer, keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())

	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          mdIssuer,
		TenantRepo:      tenants,
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		// メタデータ文書のエンドポイントは TypeSpec 由来の実行時契約から導出される。
		// 契約を渡さないスタックは文書そのものを組み立てられないので、本項目の 7 行が
		// 1 つも観測できない。
		Contract: spec.CurrentRuntimeContract(),
		OAuth2: oauth2.Module{
			ClientRepo:                 clients,
			ConsentRepo:                oauth2memory.NewConsentRepository(),
			RequestStore:               oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:                  oauth2memory.NewAuthorizationCodeStore(),
			PARStore:                   oauth2memory.NewPARStore(),
			RefreshStore:               oauth2memory.NewRefreshTokenStore(),
			McpResourceServerRepo:      resourceServers,
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
	return &mdFixture{base: server.URL + mdRealmPath}
}

// document は well-known の 1 文書を取得する。
func (f *mdFixture) document(t *testing.T, path string, query url.Values) map[string]any {
	t.Helper()
	target := f.base + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	return getJSON[map[string]any](t, http.DefaultClient, target)
}

// reach は広告された絶対 URL をそのまま 1 回叩き、状態コードを返す。文書が名乗った
// エンドポイントが実在するかは、名前の形ではなくこの応答でしか読めない。
//
// 広告される URL は本番の issuer (`http://test`) を前もつ。httptest のポートはそれと
// 違うので、issuer の前置きだけをテストサーバーの実アドレスへ読み替える。読み替えの
// 対象はホストだけで、製品が組み立てたパスとクエリには触れない。
//
// リダイレクトは追わない。追うと、経路が実在して 3xx を返したことと、その転送先が
// 存在しないことが 1 つの状態コードに潰れる。`/end_session` は引数なしのリクエストを
// 転送するので、実際にここで潰れた。
func (f *mdFixture) reach(t *testing.T, method, advertised string) int {
	t.Helper()
	local := strings.Replace(advertised, mdIssuer+mdRealmPath, f.base, 1)
	if local == advertised {
		t.Fatalf("広告された URL %q が issuer %q で始まっていない", advertised, mdIssuer+mdRealmPath)
	}
	request, err := http.NewRequest(method, local, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, local, err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode
}

// assertRouted は、広告された URL に経路が実在することを確かめる。
//
// 探りに使うメソッドは実行時契約から取る。echo はパスが登録済みでもメソッドが違えば
// 404 を返すので、GET で一律に探ると POST のエンドポイントを「存在しない」と読んで
// しまい、逆に本当に配線が落ちている場合と区別できない。
//
// 状態コードは 404 でないことだけを見る。ここで確かめたいのは経路の存在であって、
// 引数を持たないリクエストに対する各エンドポイントの正しい拒否ではない。
func (f *mdFixture) assertRouted(t *testing.T, field, advertised, operationName string) {
	t.Helper()
	operation, ok := spec.CurrentRuntimeContract().Operation(operationName)
	if !ok {
		t.Fatalf("実行時契約に %s が無い", operationName)
	}
	if want := mdRealmIssuer + operation.Path; advertised != want {
		t.Errorf("%s = %q, want %q", field, advertised, want)
		return
	}
	if status := f.reach(t, operation.Method, advertised); status == http.StatusNotFound {
		t.Errorf("%s が広告する %s %q は 404 を返し、経路として存在しない", field, operation.Method, advertised)
	}
}

// bearerJSON は広告された絶対 URL へ Authorization ヘッダーでトークンを提示し、
// 応答の JSON を返す。広告された提示方法が実際に通ることは、この形でしか読めない。
func (f *mdFixture) bearerJSON(t *testing.T, method, advertised, accessToken string) map[string]any {
	t.Helper()
	local := strings.Replace(advertised, mdRealmIssuer, f.base, 1)
	request, err := http.NewRequest(method, local, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, local, err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%s %s status=%d body=%s", method, local, response.StatusCode, raw)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("%s %s の応答が JSON ではない: %v body=%s", method, local, err, raw)
	}
	return body
}

// challenge は保護リソースを 1 回叩き、状態コードと `WWW-Authenticate` を返す。
func (f *mdFixture) challenge(t *testing.T, path, accessToken string) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, f.base+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, response.Body)
	return response.StatusCode, response.Header.Get("WWW-Authenticate")
}

// challengeParameter は `WWW-Authenticate` から 1 つの認証パラメーターの値を取り出す。
func challengeParameter(t *testing.T, header, name string) string {
	t.Helper()
	for part := range strings.SplitSeq(header, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		// 先頭の要素は `Bearer error="..."` の形で scheme を伴う。
		key = strings.TrimSpace(strings.TrimPrefix(key, "Bearer "))
		if key != name {
			continue
		}
		unquoted, err := strconv.Unquote(strings.TrimSpace(value))
		if err != nil {
			t.Fatalf("認証パラメーター %s の値が引用符付き文字列ではない: %q", name, value)
		}
		return unquoted
	}
	return ""
}

type mdTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
}

// tokens は正式な入口だけを通して、この利用者を主体とするトークンを 1 組取る。
func (f *mdFixture) tokens(t *testing.T) mdTokenResponse {
	t.Helper()
	sum := sha256.Sum256([]byte(mdVerifier))
	query := url.Values{
		"client_id": {mdClientID}, "redirect_uri": {mdRedirectURI},
		"response_type": {"code"}, "scope": {mdScope}, "state": {"state"},
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
		map[string]string{"username": mdUsername, "password": mdPassword})
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
		"code_verifier": {mdVerifier}, "redirect_uri": {mdRedirectURI},
		"client_id": {mdClientID},
	}
	request, err := http.NewRequest(http.MethodPost, f.base+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(url.QueryEscape(mdClientID), url.QueryEscape(mdClientSecret))
	tokenResponse, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer func() { _ = tokenResponse.Body.Close() }()
	raw, _ := io.ReadAll(tokenResponse.Body)
	if tokenResponse.StatusCode != http.StatusOK {
		t.Fatalf("/token status=%d body=%s", tokenResponse.StatusCode, raw)
	}
	var parsedTokens mdTokenResponse
	if err := json.Unmarshal(raw, &parsedTokens); err != nil {
		t.Fatalf("/token 応答が JSON ではない: %v body=%s", err, raw)
	}
	return parsedTokens
}

// jwtHeader は JWS Compact Serialization の保護ヘッダーを返す。`alg` と `kid` は
// 署名検証の前に読む必要があるので、検証とは別に取り出す。
func jwtHeader(t *testing.T, token string) map[string]any {
	t.Helper()
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		t.Fatalf("JWS Compact Serialization ではない: %q", token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(segments[0])
	if err != nil {
		t.Fatalf("保護ヘッダーが base64url ではない: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("保護ヘッダーが JSON ではない: %v", err)
	}
	return parsed
}

// verifyWithPublishedKeys は、配信された JWK Set だけを鍵の出所として token の署名を
// 検証し、その claim を返す。鍵は `kid` で選ぶので、集合に含まれていない鍵で署名された
// トークンはここで落ちる。
//
// 鍵素材をテスト側で保持せず配信された文書から組み立てるのが、この関数の要点である。
// 署名器を直接呼んで検証すると、「配った鍵で検証できる」ではなく「同じ処理を 2 度
// 走らせた」ことしか読めない。
func verifyWithPublishedKeys(t *testing.T, jwks map[string]any, token string) jwt.MapClaims {
	t.Helper()
	keys, ok := jwks["keys"].([]any)
	if !ok {
		t.Fatalf("JWK Set に keys 配列が無い: %#v", jwks)
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		for _, entry := range keys {
			jwk, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if kid != "" && jwk["kid"] != kid {
				continue
			}
			return rsaPublicKeyFromJWK(t, jwk), nil
		}
		return nil, jwt.ErrTokenUnverifiable
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		t.Fatalf("配信された鍵でトークンの署名を検証できなかった: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("配信された鍵で検証したトークンが有効にならなかった")
	}
	return claims
}

// rsaPublicKeyFromJWK は RFC 7517 の RSA JWK から公開鍵を組み立てる。`kty` が RSA で
// ないもの、`n` と `e` を持たないものは検証鍵として使えないので落とす。
func rsaPublicKeyFromJWK(t *testing.T, jwk map[string]any) *rsa.PublicKey {
	t.Helper()
	if jwk["kty"] != "RSA" {
		t.Fatalf("RSA 以外の JWK が配信されている: %#v", jwk["kty"])
	}
	modulus := decodeBase64URLField(t, jwk, "n")
	exponent := decodeBase64URLField(t, jwk, "e")
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulus),
		E: int(new(big.Int).SetBytes(exponent).Int64()),
	}
}

func decodeBase64URLField(t *testing.T, jwk map[string]any, field string) []byte {
	t.Helper()
	value, ok := jwk[field].(string)
	if !ok || value == "" {
		t.Fatalf("JWK に %s が無い: %#v", field, jwk)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("JWK の %s が base64url ではない: %v", field, err)
	}
	return decoded
}

// stringsOf は文書の配列値を文字列のスライスにする。要素が文字列でなければ落とす。
func stringsOf(t *testing.T, doc map[string]any, field string) []string {
	t.Helper()
	raw, ok := doc[field]
	if !ok {
		t.Fatalf("%s が文書に無い", field)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("%s が配列ではない: %#v", field, raw)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		value, ok := item.(string)
		if !ok {
			t.Fatalf("%s の要素が文字列ではない: %#v", field, item)
		}
		values = append(values, value)
	}
	return values
}

func stringOf(t *testing.T, doc map[string]any, field string) string {
	t.Helper()
	value, ok := doc[field].(string)
	if !ok || value == "" {
		t.Fatalf("%s が文字列として入っていない: %#v", field, doc[field])
	}
	return value
}

// mdMissingScopes は want のうち got に無いものを返す。
func mdMissingScopes(got, want []string) []string {
	missing := make([]string, 0, len(want))
	for _, scope := range want {
		if !slices.Contains(got, scope) {
			missing = append(missing, scope)
		}
	}
	return missing
}

// apiTokenScopeSample は `account`、`management`、SCIM のそれぞれから 1 つずつ選ぶ。
// AllScopes() をそのまま期待値にすると、製品と同じ式を書き写すだけになり、3 系統の
// どれかが欠けても気づかない。
var apiTokenScopeSample = []string{
	string(apitokendomain.ScopeAccountRead),
	string(apitokendomain.ScopeAccountWrite),
	string(apitokendomain.ScopeUsersRead),
	string(apitokendomain.ScopeTenantsRead),
	string(apitokendomain.ScopeScimUsersRead),
	string(apitokendomain.ScopeScimGroupsWrite),
}

// 行は 1 つの動詞に見えるが、実際は 3 つのことを言っている。JWK Set の形で配ること、
// 配るのが「検証鍵」であること、配ってよいのが「公開可能な」ものだけであることである。
// 3 つとも別々に崩せるので、3 つとも読む。
//
// 検証鍵であることは、配信された鍵だけで製品が発行した ID トークンの署名を検証して
// 読む。鍵の形を眺めるだけでは、正しい形をした無関係な鍵を配る実装と区別できない。
// 公開可能であることは、RSA 秘密鍵の成分がどの JWK にも入っていないことで読む。
//
// 観測の順が「発行してから取得する」なのは、テナントの署名鍵が最初に必要になった
// 時点で作られるためである。1 度も発行していない realm の JWK Set は空で、これは
// メモリと PostgreSQL のどちらの鍵ストアでも同じである。行が言っているのは製品が
// 実際に使う検証鍵を配ることなので、検証鍵が存在する状態で読む。
//
//spec:covers RFC7517-JWKS (required): 公開可能な検証鍵を JWK Set として配布する。
func TestJWKSPublishesOnlyPublicVerificationKeys(t *testing.T) {
	fixture := newMetadataFixture(t)

	tokens := fixture.tokens(t)
	if tokens.IDToken == "" {
		t.Fatal("ID トークンが発行されなかった")
	}

	jwks := fixture.document(t, "/jwks", nil)
	keys, ok := jwks["keys"].([]any)
	if !ok || len(keys) == 0 {
		t.Fatalf("発行済みトークンがあるのに JWK Set が空である: %#v", jwks)
	}

	// 配ってよいのは公開部分だけである。RSA 秘密鍵の成分が 1 つでも載れば、その鍵で
	// 誰でもトークンを発行できるようになる。
	private := []string{"d", "p", "q", "dp", "dq", "qi", "oth", "k"}
	for index, entry := range keys {
		jwk, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("keys[%d] が JWK ではない: %#v", index, entry)
		}
		for _, field := range private {
			if _, present := jwk[field]; present {
				t.Errorf("keys[%d] に秘密鍵の成分 %q が載っている", index, field)
			}
		}
		if jwk["kty"] == nil {
			t.Errorf("keys[%d] に kty が無く、JWK として読めない", index)
		}
	}

	// 配られた鍵が「検証鍵」であることは、それだけで製品の発行物を検証できることで読む。
	// 鍵は保護ヘッダーの kid で選ぶので、発行に使った鍵を配っていなければここで落ちる。
	if kid, _ := jwtHeader(t, tokens.IDToken)["kid"].(string); kid == "" {
		t.Fatal("ID トークンが kid を名乗っていないので、配信された鍵と対応付けられない")
	}
	claims := verifyWithPublishedKeys(t, jwks, tokens.IDToken)
	subject, err := claims.GetSubject()
	if err != nil || subject != mdUserID {
		t.Fatalf("検証できたのが当の利用者のトークンではない: sub=%q err=%v", subject, err)
	}
}

// mdRealmIssuer は realm の正式な発行者識別子である。製品が広告する URL はこれを前に
// 置くので、期待値の側でも 1 か所に置く。
const mdRealmIssuer = mdIssuer + mdRealmPath

// Authorization Server Metadata として公開する。
//
// 行は 3 つのことを言っているので 3 つとも読む。発行者、利用可能なエンドポイント、
// 機能である。そして 3 つとも、宣言と実装がずれても取得はできてしまう。
//
//   - 発行者は、実際に発行されたトークンが名乗る `iss` と一致することで読む。文字列が
//     入っているだけなら、別の発行者を名乗る文書を配る実装と区別できない。
//   - エンドポイントは、広告された URL を実際に叩いて 404 が返らないことで読む。
//     RFC 8414 の要点は、クライアントがこの文書だけを頼りに経路を組み立てられることに
//     ある。存在しない経路を広告する文書は、形が正しくてもその役に立たない。
//   - 機能は、広告した `token_endpoint_auth_methods_supported` と
//     `code_challenge_methods_supported` を実際に使ったリクエストが通ることで読む。
//
//spec:covers RFC8414-METADATA (required): 発行者と利用可能なエンドポイントおよび機能を
func TestAuthorizationServerMetadataPublishesTheIssuerEndpointsAndCapabilities(t *testing.T) {
	fixture := newMetadataFixture(t)

	metadata := fixture.document(t, "/.well-known/oauth-authorization-server", nil)

	if got := stringOf(t, metadata, "issuer"); got != mdRealmIssuer {
		t.Errorf("issuer = %q, want %q", got, mdRealmIssuer)
	}

	// RFC 8414 が定める中核のエンドポイント群。広告した以上、すべて実在しなければ
	// クライアントはこの文書だけでは動けない。
	endpoints := map[string]string{
		"authorization_endpoint":                "Authorize",
		"token_endpoint":                        "Token",
		"jwks_uri":                              "GetJwks",
		"introspection_endpoint":                "Introspect",
		"revocation_endpoint":                   "Revoke",
		"registration_endpoint":                 "RegisterClient",
		"pushed_authorization_request_endpoint": "PushAuthorizationRequest",
		"device_authorization_endpoint":         "DeviceAuthorization",
	}
	for field, operation := range endpoints {
		fixture.assertRouted(t, field, stringOf(t, metadata, field), operation)
	}

	// 広告した機能が実際に効くこと。ここで使う Basic 認証と S256 は、下の
	// リクエストがそのまま使う組み合わせである。
	if methods := stringsOf(t, metadata, "token_endpoint_auth_methods_supported"); !slices.Contains(methods, "client_secret_basic") {
		t.Fatalf("token_endpoint_auth_methods_supported に client_secret_basic が無い: %v", methods)
	}
	if methods := stringsOf(t, metadata, "code_challenge_methods_supported"); !slices.Contains(methods, "S256") {
		t.Fatalf("code_challenge_methods_supported に S256 が無い: %v", methods)
	}
	if types := stringsOf(t, metadata, "grant_types_supported"); !slices.Contains(types, "authorization_code") {
		t.Fatalf("grant_types_supported に authorization_code が無い: %v", types)
	}

	// 広告どおりの組み合わせ (authorization_code + S256 + client_secret_basic) で
	// 発行できること、そのトークンが広告された発行者を名乗ることを対で読む。
	tokens := fixture.tokens(t)
	claims := verifyWithPublishedKeys(t, fixture.document(t, "/jwks", nil), tokens.IDToken)
	issuer, err := claims.GetIssuer()
	if err != nil {
		t.Fatalf("発行されたトークンから iss を読めない: %v", err)
	}
	if issuer != mdRealmIssuer {
		t.Errorf("発行されたトークンの iss = %q だが、文書は issuer = %q と広告している", issuer, mdRealmIssuer)
	}
}

// 対応機能を Discovery Metadata として公開する。
//
// RFC8414-METADATA と入口が違う。OpenID Provider の設定は
// `/.well-known/openid-configuration` から取るものであり、OAuth の
// `/.well-known/oauth-authorization-server` から取れることは代わりにならない。
// 観測する中身も違う。OpenID Connect Discovery が固有に要求するのは
// `userinfo_endpoint`、`subject_types_supported`、
// `id_token_signing_alg_values_supported` であり、いずれも RFC 8414 には無い。
//
// 「対応機能」は、広告した ID トークンの署名アルゴリズムが実際に発行された ID トークンの
// `alg` と合うこと、広告した `userinfo_endpoint` が実際にその主体を返すことで読む。
//
//spec:covers OIDC-DISCOVERY-CONFIGURATION (required): well-known 設定から発行者、エンドポイント、
func TestOpenIDProviderConfigurationPublishesTheIssuerEndpointsAndCapabilities(t *testing.T) {
	fixture := newMetadataFixture(t)

	configuration := fixture.document(t, "/.well-known/openid-configuration", nil)

	if got := stringOf(t, configuration, "issuer"); got != mdRealmIssuer {
		t.Errorf("issuer = %q, want %q", got, mdRealmIssuer)
	}
	for field, operation := range map[string]string{
		"authorization_endpoint": "Authorize",
		"token_endpoint":         "Token",
		"userinfo_endpoint":      "UserInfo",
		"jwks_uri":               "GetJwks",
		"end_session_endpoint":   "EndSession",
	} {
		fixture.assertRouted(t, field, stringOf(t, configuration, field), operation)
	}
	if types := stringsOf(t, configuration, "response_types_supported"); !slices.Contains(types, "code") {
		t.Errorf("response_types_supported に code が無い: %v", types)
	}
	if types := stringsOf(t, configuration, "subject_types_supported"); len(types) == 0 {
		t.Error("subject_types_supported が空である")
	}
	if _, ok := configuration["claims_supported"]; !ok {
		t.Error("claims_supported が無い")
	}

	tokens := fixture.tokens(t)
	if tokens.IDToken == "" {
		t.Fatal("openid スコープを求めたのに ID トークンが発行されなかった")
	}

	// 広告した署名アルゴリズムと、実際に発行された ID トークンの alg が合うこと。
	algorithms := stringsOf(t, configuration, "id_token_signing_alg_values_supported")
	algorithm, _ := jwtHeader(t, tokens.IDToken)["alg"].(string)
	if !slices.Contains(algorithms, algorithm) {
		t.Errorf("発行された ID トークンの alg=%q は広告 %v に含まれていない", algorithm, algorithms)
	}

	// 広告した jwks_uri が、その ID トークンを検証できる鍵を配っていること。
	jwks := getJSON[map[string]any](t, http.DefaultClient,
		strings.Replace(stringOf(t, configuration, "jwks_uri"), mdRealmIssuer, fixture.base, 1))
	verifyWithPublishedKeys(t, jwks, tokens.IDToken)

	// 広告した userinfo_endpoint が、そのアクセストークンの主体を実際に返すこと。
	userinfo := fixture.bearerJSON(t, http.MethodGet,
		stringOf(t, configuration, "userinfo_endpoint"), tokens.AccessToken)
	if userinfo["sub"] != mdUserID {
		t.Errorf("userinfo_endpoint が返した sub = %#v, want %q", userinfo["sub"], mdUserID)
	}
}

// 指定して Protected Resource Metadata を取得できるようにする。
//
// この行が固定しているのは配信の場所と指定の方法である。中身が正しくても、
// 場所とパラメーター名が違えば適合クライアントは辿り着けない。
//
// パラメーター名を実際に読んでいることは、別名で同じ値を送った要求が当のリソースの
// 文書を返さないことで読む。名前を無視して「URL らしい値」を拾う実装は、指定した
// つもりのないリソースの文書を返してしまう。
//
//spec:covers RFC9728-WELL-KNOWN (required): `/.well-known/oauth-protected-resource` で `resource` を
func TestProtectedResourceMetadataIsFetchedFromTheWellKnownPathByResource(t *testing.T) {
	fixture := newMetadataFixture(t)

	metadata := fixture.document(t, "/.well-known/oauth-protected-resource",
		url.Values{"resource": {mdResourceA}})
	if got := stringOf(t, metadata, "resource"); got != mdResourceA {
		t.Errorf("resource = %q, want %q", got, mdResourceA)
	}

	// 指定は `resource` という名前でしか効かない。別名で送った同じ値は、指定なしと
	// 同じ realm の文書へ落ちる。
	byOtherName := fixture.document(t, "/.well-known/oauth-protected-resource",
		url.Values{"aud": {mdResourceA}})
	if got := stringOf(t, byOtherName, "resource"); got == mdResourceA {
		t.Errorf("`aud` で送った値が resource として効いている: %q", got)
	}
}

// 対応する `authorization_servers` と対応スコープを含む Protected Resource Metadata を
// 配信する。
//
// 「ごとに」は 2 件以上の登録でしか読めない。1 件だけを見ると、登録から導出する実装と、
// 登録を無視して固定の 1 文書を返す実装が同じ観測になる。
//
// 「登録済みの」は、登録されていないリソースと Disabled なリソースが拒否されることで
// 読む。要求された値をそのまま書き戻す実装は、この 2 つを通してしまう。
//
// `authorization_servers` は、広告された URL が実際に Authorization Server Metadata を
// 返すことまで読む。到達できない発行者を指す文書は、クライアントの経路を組み立てない。
//
//spec:covers RFC9728-METADATA (required): 登録済みの `McpResourceServer` ごとに、対象リソースに
func TestProtectedResourceMetadataIsDerivedFromEachRegisteredResourceServer(t *testing.T) {
	fixture := newMetadataFixture(t)

	for _, testCase := range []struct {
		resource string
		scopes   []string
	}{
		{mdResourceA, mdScopesA},
		{mdResourceB, mdScopesB},
	} {
		metadata := fixture.document(t, "/.well-known/oauth-protected-resource",
			url.Values{"resource": {testCase.resource}})

		if got := stringOf(t, metadata, "resource"); got != testCase.resource {
			t.Errorf("resource = %q, want %q", got, testCase.resource)
		}
		if got := stringsOf(t, metadata, "scopes_supported"); !slices.Equal(got, testCase.scopes) {
			t.Errorf("%s の scopes_supported = %v, want %v", testCase.resource, got, testCase.scopes)
		}
		servers := stringsOf(t, metadata, "authorization_servers")
		if !slices.Equal(servers, []string{mdRealmIssuer}) {
			t.Errorf("%s の authorization_servers = %v, want [%q]", testCase.resource, servers, mdRealmIssuer)
		}
		// 広告された認可サーバーが実際にメタデータを配っていること。
		advertised := servers[0] + "/.well-known/oauth-authorization-server"
		if status := fixture.reach(t, http.MethodGet, advertised); status != http.StatusOK {
			t.Errorf("広告された authorization_servers の %q が status=%d を返した", advertised, status)
		}
	}

	// 登録されていないリソースと Disabled なリソースは配信の対象ではない。
	for _, resource := range []string{"https://unregistered.example/mcp", mdDisabledResource} {
		status := fixture.reach(t, http.MethodGet,
			mdRealmIssuer+"/.well-known/oauth-protected-resource?resource="+url.QueryEscape(resource))
		if status != http.StatusBadRequest {
			t.Errorf("登録されていない、または Disabled な %q に status=%d を返した; want 400", resource, status)
		}
	}
}

// 対する Protected Resource Metadata と `account`、`management`、SCIM の各スコープ、
// 対応する `bearer_methods_supported` を公開する。
//
// 行は 3 つのことを言っている。未指定のときの対象が realm の IdMagic API であること、
// 3 系統のスコープを載せること、対応する提示方法を載せることである。
//
// スコープは 3 系統からそれぞれ標本を取って読む。`AllScopes()` をそのまま期待値に
// 写すと、製品と同じ式を 2 度書くだけになり、どれか 1 系統が落ちても気づかない。
//
// `bearer_methods_supported` は、載っている名前だけでなく、その方法が実際に通ることを
// 対で読む。`header` と書いてあるのにヘッダー提示が通らない実装は、宣言だけを見ている
// 限り区別できない。
//
//spec:covers RFC9728-IDMAGIC-API (required): `resource` が未指定であれば、realm の IdMagic API に
func TestProtectedResourceMetadataWithoutAResourceDescribesTheRealmApi(t *testing.T) {
	fixture := newMetadataFixture(t)

	metadata := fixture.document(t, "/.well-known/oauth-protected-resource", nil)

	if got := stringOf(t, metadata, "resource"); got != mdRealmIssuer {
		t.Errorf("resource = %q, want realm の IdMagic API %q", got, mdRealmIssuer)
	}
	if servers := stringsOf(t, metadata, "authorization_servers"); !slices.Equal(servers, []string{mdRealmIssuer}) {
		t.Errorf("authorization_servers = %v, want [%q]", servers, mdRealmIssuer)
	}

	scopes := stringsOf(t, metadata, "scopes_supported")
	if missing := mdMissingScopes(scopes, apiTokenScopeSample); len(missing) > 0 {
		t.Errorf("scopes_supported に %v が無い: %v", missing, scopes)
	}

	methods := stringsOf(t, metadata, "bearer_methods_supported")
	if !slices.Contains(methods, "header") {
		t.Fatalf("bearer_methods_supported に header が無い: %v", methods)
	}

	// 広告した提示方法が実際に通ること。
	tokens := fixture.tokens(t)
	userinfo := fixture.bearerJSON(t, http.MethodGet, mdRealmIssuer+"/userinfo", tokens.AccessToken)
	if userinfo["sub"] != mdUserID {
		t.Errorf("header 提示でリソースへ届かなかった: %#v", userinfo)
	}
}

// `403 insufficient_scope` レスポンスでは、当該 realm の Protected Resource Metadata URL を
// `resource_metadata` 認証パラメーターで提示する。
//
// 行は 2 つの応答を名指ししているので、2 つとも読む。片方だけを観測すると、拒否の分岐が
// 2 本あるうち 1 本にしか付けていない実装を見逃す。
//
// 提示された URL は、それ自体が Protected Resource Metadata を返すところまで読む。
// この認証パラメーターの目的は、トークンを持たないクライアントに次の一手を教えることに
// あるので、辿れない URL では行の言うことが果たされない。
//
//spec:covers RFC9728-CHALLENGE (required): ベアラー保護リソースの `401 invalid_token` と
func TestBearerChallengesPointAtTheRealmProtectedResourceMetadata(t *testing.T) {
	fixture := newMetadataFixture(t)
	tokens := fixture.tokens(t)

	for _, testCase := range []struct {
		name       string
		path       string
		token      string
		wantStatus int
		wantError  string
	}{
		{
			// 検証できないトークンの提示。
			name: "invalid_token", path: "/api/auth/account", token: "not.a.valid.token",
			wantStatus: http.StatusUnauthorized, wantError: "invalid_token",
		},
		{
			// 有効だが、この API が要求するスコープを持たないトークンの提示。
			name: "insufficient_scope", path: "/api/admin/v1/clients", token: tokens.AccessToken,
			wantStatus: http.StatusForbidden, wantError: "insufficient_scope",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			status, header := fixture.challenge(t, testCase.path, testCase.token)
			if status != testCase.wantStatus {
				t.Fatalf("status = %d, want %d (challenge=%q)", status, testCase.wantStatus, header)
			}
			if got := challengeParameter(t, header, "error"); got != testCase.wantError {
				t.Fatalf("error = %q, want %q (challenge=%q)", got, testCase.wantError, header)
			}

			advertised := challengeParameter(t, header, "resource_metadata")
			want := mdRealmIssuer + "/.well-known/oauth-protected-resource"
			if advertised != want {
				t.Fatalf("resource_metadata = %q, want %q", advertised, want)
			}

			// 提示された URL が実際に当の realm の文書を返すこと。
			metadata := getJSON[map[string]any](t, http.DefaultClient,
				strings.Replace(advertised, mdRealmIssuer, fixture.base, 1))
			if got := stringOf(t, metadata, "resource"); got != mdRealmIssuer {
				t.Errorf("提示された URL が返した resource = %q, want %q", got, mdRealmIssuer)
			}
		})
	}
}
