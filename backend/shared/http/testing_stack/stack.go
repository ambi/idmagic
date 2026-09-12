// Package testing_stack は、`server_http.Register` が組み立てるのと同じスタックを
// テストから 1 行で建てるための組み立て器である。
//
// これがある理由は測定にもとづく。94 個のテストファイルが 44 フィールドの `Deps` を
// 手で埋めていて、宣言済みの具体例へテストを対応付ける作業のたびに「必要な配線を持った
// fixture はどれか」を探すところから始まっていた。[[wi-538]] は
// `EX-OAUTH2-001-01` を子 work item へ回したが、その理由は主張の難しさではなく、
// 認可コードと `/token` と account リソースサーバーを同時に配線したスタックが
// 94 個のどれにも無かったことである。観測したい振る舞いではなく fixture の有無が
// 作業の範囲を決めていた。詳細は wi-565。
//
// **既定は製品の組み立てと一致させる。** 共有化は間違いを 1 箇所へ集める代わり、
// 間違えたときの影響を全呼び出し元へ広げる。とくに `WithApiTokens` が
// `TokenIntrospector` へ渡すのは生の署名検証器ではなく `apitokenusecases` の overlay で
// あり、ここを取り違えると失効したトークンが `/introspect` で有効に見える。それは製品の
// 欠陥ではなく組み立ての違いになる。組み立ての正は backend/cmd/idmagic/server.go である。
//
// option は `Deps` のフィールド単位ではなく、**具体例が観測したい入口**の単位で切る。
// 44 フィールドを 44 個の option に置き換えても、呼び出し側が払う費用は変わらない。
//
// 保存先は型付きの field として公開する。応答だけを読むテストは「拒否を書いてから保存
// する」実装を見分けられないので、効果を読み直せることがこの基盤の要件である。
package testing_stack

import (
	"context"
	"encoding/json"
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
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	consentmemory "github.com/ambi/idmagic/backend/oauth2/consent/db_memory"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	testingpasswords "github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"
)

// 基盤が用意する主体とレルム。テストが自分で seed しなくても、テナント境界を
// またぐ観測が 1 行で書けるようにしてある。
const (
	Issuer      = "https://idp.example"
	AdminUserID = "admin-1"
	UserID      = "user-1"
	// OtherRealm はレルム越えの提示を観測するための 2 つ目のテナント。id と realm は
	// 同じ文字列にしてある。
	OtherRealm = "acme"

	// ResourceServerClientID と ResourceServerSecret は、WithOAuth2Clients が両テナントへ
	// seed する confidential クライアント。`/introspect` と `/revoke` はクライアント認証を
	// 要求するので、これが無いとトークンの状態を製品の入口から読めない。
	ResourceServerClientID = "resource-server"
	ResourceServerSecret   = "resource-server-secret"
)

// Stack は建てたスタックと、そこへ配線した保存先をまとめて持つ。
//
// 保存先を公開するのは、拒否が防いだ効果を読み直すためである。nil のままの field は
// その option を渡していないことを意味する。
type Stack struct {
	Echo     *echo.Echo
	Tenants  *tenancymemory.TenantRepository
	Users    *usermemory.UserRepository
	KeyStore *signingmemory.InMemoryKeyStore
	Signer   *tokensjose.JWTSigner

	Clients            *oauth2memory.OAuth2ClientRepository
	Consents           *consentmemory.ConsentRepository
	AuthzDetailTypes   *oauth2memory.AuthorizationDetailTypeRepository
	McpResourceServers *oauth2memory.McpResourceServerRepository
	Codes              *oauth2memory.AuthorizationCodeStore
	Refresh            *oauth2memory.RefreshTokenStore
	ApiTokens          apitokenports.Repository
	SamlSPs            *samlmemory.SamlServiceProviderRepository
	Sessions           *sessionusecases.SessionManager

	apiTokens *apitokenusecases.Service
}

// Option は 1 つの入口を配線へ足す。
type Option func(*builder)

type builder struct {
	deps  httpadapter.Deps
	stack *Stack
}

// WithOAuth2Clients は OAuth2 クライアントと、その管理 API が読み書きする保存先を配線する。
// 認可詳細タイプと MCP リソースサーバーも同じ管理 API 群に属するので一緒に配る。
// REQ-OAUTH2-003 の粒度スコープは resource ごとに別のスコープを割り当てるため、
// 拒否が防いだ効果は resource ごとに読み直す必要がある。
func WithOAuth2Clients() Option {
	return func(b *builder) {
		b.stack.Clients = oauth2memory.NewClientRepository()
		b.stack.AuthzDetailTypes = oauth2memory.NewAuthorizationDetailTypeRepository()
		b.stack.McpResourceServers = oauth2memory.NewMcpResourceServerRepository()
		secretHash := oauthdomain.HashClientSecret(ResourceServerSecret)
		for _, tenantID := range []string{tenancydomain.DefaultTenantID, OtherRealm} {
			b.stack.Clients.Seed(&oauthdomain.OAuth2Client{
				TenantID: tenantID, ClientID: ResourceServerClientID, ClientSecretHash: &secretHash,
				ClientType:              spec.ClientConfidential,
				TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
				GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
				CreatedAt:               time.Now().UTC(),
			})
		}
		b.deps.OAuth2.ClientRepo = b.stack.Clients
		b.deps.OAuth2.AuthzDetailTypeRepo = b.stack.AuthzDetailTypes
		b.deps.OAuth2.McpResourceServerRepo = b.stack.McpResourceServers
	}
}

// WithAuthorizationCodeFlow は `/authorize` から `/token` までを通す配線を足す。
// 認可リクエスト、認可コード、PAR、同意、そしてセッションと認証文脈の解決である。
// クライアントの保存先も要るので WithOAuth2Clients を含む。
func WithAuthorizationCodeFlow() Option {
	return func(b *builder) {
		if b.stack.Clients == nil {
			WithOAuth2Clients()(b)
		}
		if b.stack.Consents == nil {
			b.stack.Consents = consentmemory.NewConsentRepository()
			b.deps.OAuth2.ConsentRepo = b.stack.Consents
		}
		b.stack.Codes = oauth2memory.NewAuthorizationCodeStore()
		b.stack.Sessions = sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
		b.deps.OAuth2.RequestStore = oauth2memory.NewAuthorizationRequestStore()
		b.deps.OAuth2.CodeStore = b.stack.Codes
		b.deps.OAuth2.PARStore = oauth2memory.NewPARStore()
		b.deps.SessionManager = b.stack.Sessions
		b.deps.AuthnResolver = b.stack.Sessions
		b.deps.PasswordHasher = testingpasswords.NewHasher()
	}
}

// WithTokenIssuance はトークンを発行して提示できるようにする。リフレッシュの保管、
// 失効リスト、DPoP のリプレイ記録である。署名鍵と署名器は基盤が常に持つ。
func WithTokenIssuance() Option {
	return func(b *builder) {
		b.stack.Refresh = oauth2memory.NewRefreshTokenStore()
		b.deps.OAuth2.RefreshStore = b.stack.Refresh
		b.deps.OAuth2.AccessTokenDenylist = oauth2memory.NewAccessTokenDenylist()
		b.deps.OAuth2.DpopReplayStore = oauth2memory.NewDpopReplayStore()
		b.deps.OAuth2.ClientAssertionReplayStore = oauth2memory.NewClientAssertionReplayStore()
	}
}

// WithApiTokens は管理発行の API アクセストークンを配線する。
//
// `OAuth2.TokenIntrospector` へ渡すのは生の署名検証器ではなく、管理発行トークンの
// ライフサイクル記録を重ねた `apitokenusecases` の overlay である。組み立ての正は
// backend/cmd/idmagic/server.go であり、そこがこの形で渡している。生の検証器にすると、
// 管理コンソールからの失効が記録にしか載らないため、失効したトークンが `/introspect` で
// 有効に見える。入口を通すことに意味があるのは、入口の組み立てが製品と同じときだけである。
func WithApiTokens() Option {
	return func(b *builder) {
		repo := apitokenmemory.NewRepository()
		b.stack.ApiTokens = repo
		b.stack.apiTokens = apitokenusecases.New(repo,
			apitokenusecases.WithTokenIssuer(b.stack.Signer),
			apitokenusecases.WithTokenIntrospector(b.stack.Signer))
		b.deps.ApiTokens = apitoken.Module{
			Repo: repo, TokenIssuer: b.stack.Signer, TokenIntrospector: b.stack.Signer,
		}
		b.deps.OAuth2.TokenIntrospector = apitokenusecases.New(repo,
			apitokenusecases.WithTokenIntrospector(b.stack.Signer))
	}
}

// WithAccountApi は利用者が自分の同意を参照し撤回する `/api/account/v1/` を配線する。
func WithAccountApi() Option {
	return func(b *builder) {
		if b.stack.Consents == nil {
			b.stack.Consents = consentmemory.NewConsentRepository()
			b.deps.OAuth2.ConsentRepo = b.stack.Consents
		}
	}
}

// WithSaml は SAML の保存先と管理 API を配線する。
func WithSaml() Option {
	return func(b *builder) {
		b.stack.SamlSPs = samlmemory.NewSamlServiceProviderRepository()
		b.deps.Saml = saml.Module{SPRepo: b.stack.SamlSPs, ProfileRepo: b.stack.SamlSPs}
		b.deps.FederationSigner = samltoken.KeyStoreSignerProvider{KeyStore: b.stack.KeyStore}
	}
}

// New は既定のテナントと利用者を持つスタックを建て、渡された option の配線を足す。
//
// 基盤が常に持つのはテナント、利用者、署名鍵、署名器、そして `Register` である。
// この 5 つはどの入口も前提にしていて、外しても呼び出し側が同じものを建て直すだけになる。
func New(t *testing.T, options ...Option) *Stack {
	t.Helper()
	created := time.Now().UTC()

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{
			ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
			DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: created,
		},
		{
			ID: OtherRealm, Realm: OtherRealm,
			DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: created,
		},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatalf("seed tenant %s: %v", tenant.Realm, err)
		}
	}

	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: AdminUserID, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: created, UpdatedAt: created,
	})
	users.Seed(&userdomain.User{
		ID: UserID, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "user",
		PasswordHash: "unused", Roles: []string{"user"},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: created, UpdatedAt: created,
	})

	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	signer := tokensjose.NewJWTSigner(Issuer, keyStore)

	stack := &Stack{
		Echo: echo.New(), Tenants: tenants, Users: users, KeyStore: keyStore, Signer: signer,
	}
	b := &builder{
		stack: stack,
		// 渡すのは module だけである。`Deps` は移行期の互換入力として `UserRepo`、
		// `KeyStore`、`TokenIssuer` を平置きでも受けるが、bootstrap は module しか設定
		// しない。互換入力を使うと、この基盤を共有する全テストが製品と違う経路の
		// 組み立てを観測することになる。
		deps: httpadapter.Deps{
			Issuer: Issuer, Contract: spec.CurrentRuntimeContract(),
			TenantRepo:   tenants,
			IdManagement: idmanagement.Module{UserRepo: users},
			SigningKeys:  signingkeys.Module{KeyStore: keyStore},
			OAuth2:       oauth2.Module{TokenIssuer: signer, TokenIntrospector: signer},
		},
	}
	for _, option := range options {
		option(b)
	}
	httpadapter.Register(stack.Echo, b.deps)
	return stack
}

// RealmContext は middleware が組み立てるのと同じテナント文脈を返す。署名鍵も audience も
// テナントごとなので、この文脈を通さずに発行したトークンは製品が発行するものと別物になる。
func (s *Stack) RealmContext(t *testing.T, realm string) context.Context {
	t.Helper()
	tenant, err := s.Tenants.FindByRealm(context.Background(), realm)
	if err != nil || tenant == nil {
		t.Fatalf("realm %q: tenant=%v err=%v", realm, tenant, err)
	}
	prefix := "/realms/" + realm
	return tenancy.WithTenant(context.Background(), tenant, Issuer+prefix, prefix)
}

// TenantID はレルムに対応するテナント id を返す。
func (s *Stack) TenantID(t *testing.T, realm string) string {
	t.Helper()
	tenant, err := s.Tenants.FindByRealm(context.Background(), realm)
	if err != nil || tenant == nil {
		t.Fatalf("realm %q: tenant=%v err=%v", realm, tenant, err)
	}
	return tenant.ID
}

// IssueApiToken は管理コンソールと同じ Service を通して API アクセストークンを 1 本発行する。
// 発行のエンドポイントは対話セッション限定なので、テストは HTTP からトークンを作れない。
func (s *Stack) IssueApiToken(
	t *testing.T, realm string, scopes ...apitokendomain.Scope,
) (string, apitokendomain.Metadata) {
	t.Helper()
	if s.apiTokens == nil {
		t.Fatal("IssueApiToken には WithApiTokens が要る")
	}
	literal, metadata, err := s.apiTokens.Issue(
		s.RealmContext(t, realm), s.TenantID(t, realm), AdminUserID, "testing_stack",
		apitokendomain.Scopes(scopes).Strings(), 1, "",
	)
	if err != nil {
		t.Fatalf("issue api token: %v", err)
	}
	return literal, metadata
}

// Introspect は `/introspect` へ 1 回問い合わせ、応答を復号して返す。
//
// クライアント認証は WithOAuth2Clients が seed する confidential クライアントで行う。
// トークンの状態を製品の入口から読むための経路であり、保存先を直接読むのとは別物である。
// 失効が記録にしか載らない管理発行トークンでは、この差が組み立ての正しさそのものになる。
func (s *Stack) Introspect(t *testing.T, realm, token string) map[string]any {
	t.Helper()
	form := url.Values{"token": {token}, "token_type_hint": {"access_token"}}
	request := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, Issuer+"/realms/"+realm+"/introspect",
		strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(ResourceServerClientID, ResourceServerSecret)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("introspect status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("introspect body %s: %v", recorder.Body.String(), err)
	}
	return body
}
