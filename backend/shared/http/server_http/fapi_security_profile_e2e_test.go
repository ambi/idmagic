package server_http_test

// FAPI 2.0 Security Profile を選択したクライアントだけが追加の制約を受けることを、
// Register が組み立てたスタックの `/authorize`、`/register`、`/token` から観測する。
//
// **観測は必ず対で読む。** 制約を受けるクライアントと受けないクライアントは、
// `fapi_profile` 以外のすべてが同じである。クライアント認証方式も鍵も付与された
// グラントも scope もそろえてあるので、片方だけが拒否されたなら、その差はプロファイル
// の選択だけから来ている。制約が全クライアントへ漏れた実装は、対照側が落ちる。
//
// 単一のクライアントで拒否だけを読む形は採らない。それでは「FAPI に制約を課す実装」と
// 「全クライアントに同じ制約を課す実装」を区別できず、既存のクライアントを一斉に壊す
// 変更が緑のまま通ってしまう。

import (
	stdcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

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
	fpIssuer      = "http://test"
	fpRealmPath   = "/realms/default"
	fpRedirectURI = "https://app.example/cb"
	fpScope       = "openid profile read"
	fpUsername    = "alice"
	fpUserID      = "user_alice"
	fpPassword    = "fapi-profile-password-1234"
	fpVerifier    = "fapi-profile-pkce-verifier-01234567890123456"

	// 2 つのクライアントは `fapi_profile` 以外がすべて同じである。どちらも
	// private_key_jwt で、同じ鍵を使い、同じグラントと scope を持つ。
	fpFapiClientID  = "fapi-profile-selected-client"
	fpPlainClientID = "fapi-profile-unselected-client"
)

type fpFixture struct {
	base       string
	signingKey *rsa.PrivateKey
}

func newFapiProfileFixture(t *testing.T) *fpFixture {
	t.Helper()
	now := time.Now().UTC()

	signingKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "fapi-profile-key",
		"n": base64.RawURLEncoding.EncodeToString(signingKey.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(signingKey.PublicKey.E)).Bytes()),
	}}}

	clients := oauth2memory.NewClientRepository()
	seed := func(clientID string, profile domain.FapiProfile) {
		clients.Seed(&domain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID,
			ClientID: clientID, ClientType: spec.ClientConfidential,
			RedirectURIs: []string{fpRedirectURI},
			GrantTypes: []spec.GrantType{
				spec.GrantAuthorizationCode, spec.GrantClientCredentials,
			},
			ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
			TokenEndpointAuthMethod:  domain.AuthMethodPrivateKeyJwt,
			Scope:                    fpScope,
			JWKS:                     jwks,
			IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
			FapiProfile:              profile,
			CreatedAt:                now,
			UpdatedAt:                now,
		})
	}
	seed(fpFapiClientID, domain.FapiSecurityProfileV2)
	seed(fpPlainClientID, domain.FapiNone)

	hasher := testingpasswords.NewHasher()
	passwordHash, err := hasher.Hash(fpPassword)
	if err != nil {
		t.Fatal(err)
	}
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: fpUserID, PreferredUsername: fpUsername, PasswordHash: passwordHash,
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
	signer := tokensJOSE.NewJWTSigner(fpIssuer, keyStore)
	sessionManager := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())

	startupComplete := &atomic.Bool{}
	startupComplete.Store(true)
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          fpIssuer,
		TenantRepo:      tenants,
		StartupComplete: startupComplete,
		ShuttingDown:    &atomic.Bool{},
		Contract:        spec.CurrentRuntimeContract(),
		OAuth2: oauth2.Module{
			ClientRepo:   clients,
			ConsentRepo:  oauth2memory.NewConsentRepository(),
			RequestStore: oauth2memory.NewAuthorizationRequestStore(),
			CodeStore:    oauth2memory.NewAuthorizationCodeStore(),
			PARStore:     oauth2memory.NewPARStore(),
			RefreshStore: oauth2memory.NewRefreshTokenStore(),
			// DPoP の検証はこのストアが無いと丸ごと飛ばされる。送信者制約を
			// 観測する行が 1 つも読めなくなるので、必ず渡す。
			DpopReplayStore:            oauth2memory.NewDpopReplayStore(),
			ClientAssertionReplayStore: oauth2memory.NewClientAssertionReplayStore(),
			AccessTokenDenylist:        oauth2memory.NewAccessTokenDenylist(),
			McpResourceServerRepo:      oauth2memory.NewMcpResourceServerRepository(),
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
	return &fpFixture{base: server.URL + fpRealmPath, signingKey: signingKey}
}

// authorize は 1 つのクライアントで `/authorize` を 1 度叩き、状態と本文を返す。
// リダイレクトは追わない。追うと、拒否とログイン画面への遷移がどちらも 200 に
// 見えてしまう。
func (f *fpFixture) authorize(t *testing.T, clientID string) (int, string) {
	t.Helper()
	sum := sha256.Sum256([]byte(fpVerifier))
	query := url.Values{
		"client_id": {clientID}, "redirect_uri": {fpRedirectURI},
		"response_type": {"code"}, "scope": {fpScope}, "state": {"state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	client := browserClient(t)
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	response, err := client.Get(f.base + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

// fpProceeded は `/authorize` が拒否ではなく認証の続きへ進んだかを返す。製品は
// ログイン画面へ 303 で送るが、既にセッションがあれば 302 で戻し、画面を直接
// 返す経路では 200 になる。拒否は OAuth のエラー本文を持つ 4xx なので、この 3 つ
// のいずれでもない。
func fpProceeded(status int) bool {
	return status == http.StatusOK || status == http.StatusFound || status == http.StatusSeeOther
}

// clientAssertion は private_key_jwt のアサーションを 1 通署名する。
func (f *fpFixture) clientAssertion(t *testing.T, clientID string) string {
	t.Helper()
	now := time.Now()
	header, err := json.Marshal(map[string]any{"alg": "PS256", "kid": "fapi-profile-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": clientID, "sub": clientID,
		"aud": fpIssuer + fpRealmPath + "/token",
		"jti": "jti-" + clientID + "-" + t.Name() + "-" + fpNonce(now),
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, f.signingKey, stdcrypto.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// fpNonce は jti を毎回変えるための単調な接尾辞。アサーションのリプレイ検出に
// 引っかからないように、同じテスト内の 2 通目以降も別の jti を持たせる。
func fpNonce(now time.Time) string {
	return base64.RawURLEncoding.EncodeToString(big.NewInt(now.UnixNano()).Bytes())
}

// tokenWithoutProof は送信者制約の証拠を付けずに client_credentials を 1 度送る。
func (f *fpFixture) tokenWithoutProof(t *testing.T, clientID string) (int, string) {
	t.Helper()
	return f.postToken(t, clientID, "")
}

// tokenWithDPoP は DPoP 証明を付けて client_credentials を 1 度送る。
func (f *fpFixture) tokenWithDPoP(t *testing.T, clientID string) (int, string) {
	t.Helper()
	return f.postToken(t, clientID, f.dpopProof(t, clientID))
}

func (f *fpFixture) postToken(t *testing.T, clientID, proof string) (int, string) {
	t.Helper()
	form := url.Values{
		"grant_type": {"client_credentials"}, "scope": {"read"},
		"client_id":             {clientID},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {f.clientAssertion(t, clientID)},
	}
	request, err := http.NewRequest(http.MethodPost, f.base+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if proof != "" {
		request.Header.Set("DPoP", proof)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

// dpopProof は `/token` 宛ての DPoP 証明を 1 通署名する。
func (f *fpFixture) dpopProof(t *testing.T, clientID string) string {
	t.Helper()
	now := time.Now()
	jwk := map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(f.signingKey.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.signingKey.PublicKey.E)).Bytes()),
	}
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"htm": http.MethodPost, "htu": fpIssuer + fpRealmPath + "/token",
		"jti": "dpop-" + clientID + "-" + t.Name() + "-" + fpNonce(now),
		"iat": now.Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, f.signingKey, stdcrypto.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// register は `/register` へメタデータを 1 通送る。
func (f *fpFixture) register(t *testing.T, metadata map[string]any) (int, string) {
	t.Helper()
	raw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(f.base+"/register", "application/json", strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("POST /register: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

// fpAccessTokenClaims は `/token` の応答からアクセストークンのクレームを取り出す。
// 署名の検証はここでは行わない。この行が読むのは `cnf` の有無だけである。
func fpAccessTokenClaims(t *testing.T, body string) map[string]any {
	t.Helper()
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("/token 応答が JSON ではない: %v body=%s", err, body)
	}
	if parsed.AccessToken == "" {
		t.Fatalf("アクセストークンが返らなかった: %s", body)
	}
	segments := strings.Split(parsed.AccessToken, ".")
	if len(segments) != 3 {
		t.Fatalf("アクセストークンが JWT ではない: %q", parsed.AccessToken)
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

// 限られることを固定する。S256 PKCE の側は製品が全クライアントへ無条件に課して
// いるので、ここでは「選択したクライアントでも S256 以外は通らない」ことを対照と
// して併せて読む。
//
//spec:covers FAPI2-PAR-PKCE: プロファイルを選んだクライアントの認可リクエストが PAR 経由に
func TestFapi2ClientCannotStartAnAuthorizationRequestWithoutPAR(t *testing.T) {
	fixture := newFapiProfileFixture(t)

	status, body := fixture.authorize(t, fpFapiClientID)
	if fpProceeded(status) {
		t.Fatalf("プロファイルを選んだクライアントの直接の /authorize が通った: status=%d body=%s", status, body)
	}
	if !strings.Contains(body, "invalid_request") {
		t.Fatalf("拒否の理由が invalid_request ではない: status=%d body=%s", status, body)
	}

	// 対照: 同じリクエストを、プロファイル以外がすべて同じクライアントが送ると
	// ログインへ進む。ここが落ちるなら、制約は FAPI ではなく全クライアントに
	// 掛かっている。
	controlStatus, controlBody := fixture.authorize(t, fpPlainClientID)
	if !fpProceeded(controlStatus) {
		t.Fatalf("プロファイルを選んでいないクライアントまで拒否された: status=%d body=%s", controlStatus, controlBody)
	}
}

// できないことを、登録の入口で固定する。方式は保存されたクライアント自身の属性
// なので、保存できてしまえば以後どの経路からでも共有シークレットが通る。
//
//spec:covers FAPI2-CLIENT-AUTH: プロファイルを選んだクライアントが共有シークレットで認証
func TestRegisterRefusesAFapi2ClientThatAuthenticatesWithASharedSecret(t *testing.T) {
	fixture := newFapiProfileFixture(t)
	jwks := map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "registered-key",
		"n": base64.RawURLEncoding.EncodeToString(fixture.signingKey.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(fixture.signingKey.PublicKey.E)).Bytes()),
	}}}

	status, body := fixture.register(t, map[string]any{
		"client_name": "fapi with shared secret", "client_type": "confidential",
		"redirect_uris":              []string{fpRedirectURI},
		"grant_types":                []string{"authorization_code"},
		"token_endpoint_auth_method": "client_secret_basic",
		"fapi_profile":               "fapi_2_security_profile",
	})
	if status == http.StatusOK || status == http.StatusCreated {
		t.Fatalf("共有シークレットの FAPI クライアントが登録された: status=%d body=%s", status, body)
	}
	if !strings.Contains(body, "invalid_client_metadata") {
		t.Fatalf("拒否の理由が invalid_client_metadata ではない: status=%d body=%s", status, body)
	}

	// 対照 1: 同じメタデータの認証方式だけを非対称に変えると登録できる。
	asymmetric, asymmetricBody := fixture.register(t, map[string]any{
		"client_name": "fapi with private_key_jwt", "client_type": "confidential",
		"redirect_uris":              []string{fpRedirectURI},
		"grant_types":                []string{"authorization_code"},
		"token_endpoint_auth_method": "private_key_jwt",
		"jwks":                       jwks,
		"fapi_profile":               "fapi_2_security_profile",
	})
	if asymmetric != http.StatusOK && asymmetric != http.StatusCreated {
		t.Fatalf("非対称認証の FAPI クライアントが登録できなかった: status=%d body=%s", asymmetric, asymmetricBody)
	}

	// 対照 2: 非対称認証でも鍵が無ければ拒否される。対照 1 と対にすることで、
	// 登録できた理由が「鍵がそろっていること」であり、`private_key_jwt` という
	// 文字列そのものではないと分かる。この 2 つを対にしていなかった間、検証用の
	// 候補が構造的に不正だったせいで、鍵の有無によらず全件が同じ
	// invalid_client_metadata で落ちていた。
	keyless, keylessBody := fixture.register(t, map[string]any{
		"client_name": "fapi without keys", "client_type": "confidential",
		"redirect_uris":              []string{fpRedirectURI},
		"grant_types":                []string{"authorization_code"},
		"token_endpoint_auth_method": "private_key_jwt",
		"fapi_profile":               "fapi_2_security_profile",
	})
	if keyless == http.StatusOK || keyless == http.StatusCreated {
		t.Fatalf("鍵の無い private_key_jwt クライアントが登録された: status=%d body=%s", keyless, keylessBody)
	}

	// 対照 3: プロファイルを選ばなければ共有シークレットのままで登録できる。
	plain, plainBody := fixture.register(t, map[string]any{
		"client_name": "plain with shared secret", "client_type": "confidential",
		"redirect_uris":              []string{fpRedirectURI},
		"grant_types":                []string{"authorization_code"},
		"token_endpoint_auth_method": "client_secret_basic",
		"fapi_profile":               "none",
	})
	if plain != http.StatusOK && plain != http.StatusCreated {
		t.Fatalf("プロファイルを選ばない共有シークレットのクライアントまで拒否された: status=%d body=%s", plain, plainBody)
	}
}

// 持たないアクセストークンを受け取れないことを固定する。証拠を付けたときに `cnf` が
// 載ることを対にして読む。拒否だけでは、制約を課しながら束縛を付け忘れている実装と
// 区別できない。
//
//spec:covers FAPI2-SENDER-CONSTRAINT: プロファイルを選んだクライアントが、送信者制約の証拠を
func TestFapi2ClientCannotObtainAnUnconstrainedAccessToken(t *testing.T) {
	fixture := newFapiProfileFixture(t)

	status, body := fixture.tokenWithoutProof(t, fpFapiClientID)
	if status == http.StatusOK {
		t.Fatalf("証拠の無い FAPI クライアントにトークンが発行された: %s", body)
	}
	if strings.Contains(body, "access_token") {
		t.Fatalf("拒否の応答にアクセストークンが載っている: %s", body)
	}

	// 証拠を付ければ通り、そのトークンは提示した鍵へ束縛されている。
	boundStatus, boundBody := fixture.tokenWithDPoP(t, fpFapiClientID)
	if boundStatus != http.StatusOK {
		t.Fatalf("DPoP 証明を付けた FAPI クライアントが拒否された: status=%d body=%s", boundStatus, boundBody)
	}
	claims := fpAccessTokenClaims(t, boundBody)
	cnf, ok := claims["cnf"].(map[string]any)
	if !ok {
		t.Fatalf("送信者制約付きのはずのトークンに cnf が無い: %v", claims)
	}
	if jkt, _ := cnf["jkt"].(string); jkt == "" {
		t.Fatalf("cnf に jkt が無い: %v", cnf)
	}
}

// ことを、3 つの制約をまとめて対照側から読む。個々の行のテストも対照を 1 つずつ
// 持つが、ここでは「選んでいないクライアントには 1 つも掛からない」ことを 1 か所で
// 読む。制約が既定化した変更は、行ごとのテストより先にここが落ちる。
//
//spec:covers FAPI2-PROFILE-SELECTION: 追加制約がプロファイルを選んだクライアントだけに掛かる
func TestNonFapi2ClientKeepsWorkingWithoutTheProfileConstraints(t *testing.T) {
	fixture := newFapiProfileFixture(t)

	// PAR を経由しない認可リクエストが通る。
	status, body := fixture.authorize(t, fpPlainClientID)
	if !fpProceeded(status) {
		t.Fatalf("PAR 無しの /authorize が拒否された: status=%d body=%s", status, body)
	}

	// 送信者制約の証拠が無くてもトークンが出て、`cnf` は付かない。
	tokenStatus, tokenBody := fixture.tokenWithoutProof(t, fpPlainClientID)
	if tokenStatus != http.StatusOK {
		t.Fatalf("証拠の無い /token が拒否された: status=%d body=%s", tokenStatus, tokenBody)
	}
	if claims := fpAccessTokenClaims(t, tokenBody); claims["cnf"] != nil {
		t.Fatalf("証拠を出していないのに送信者制約が付いた: %v", claims["cnf"])
	}

	// 共有シークレットのクライアントが登録できる。
	registerStatus, registerBody := fixture.register(t, map[string]any{
		"client_name": "unconstrained", "client_type": "confidential",
		"redirect_uris":              []string{fpRedirectURI},
		"grant_types":                []string{"authorization_code"},
		"token_endpoint_auth_method": "client_secret_basic",
		"fapi_profile":               "none",
	})
	if registerStatus != http.StatusOK && registerStatus != http.StatusCreated {
		t.Fatalf("共有シークレットのクライアントが登録できなかった: status=%d body=%s", registerStatus, registerBody)
	}
}
