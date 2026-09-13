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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/apitoken"
	apitokenmemory "github.com/ambi/idmagic/backend/apitoken/db_memory"
	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenports "github.com/ambi/idmagic/backend/apitoken/ports"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
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
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	testingpasswords "github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
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

	// BrowserClientID から UserPassword までは WithBrowserFlow が置く、ブラウザー経由の
	// 認可を 1 本通すために必要な最小の組である。
	BrowserClientID     = "web-app"
	BrowserClientSecret = "web-app-secret"
	BrowserRedirectURI  = "https://app.example.com/callback"
	UserPassword        = "testing-stack-password-1234"

	// WsFedRealm と WsFedReplyURL は WithWsFederation が登録する RP である。
	WsFedRealm    = "urn:idmagic:testing-stack-rp"
	WsFedReplyURL = "https://rp.example/wsfed"
)

// EventLog は `Deps.Emit` が受けたイベントを発行順に覚える。
//
// 宣言済みの具体例の `Then` は「どのイベントが発行されるか」で書かれていることが多く、
// 応答だけを読むテストは、状態は変えたが通知を出さない実装を通してしまう。記録は
// 常時行う。option にすると「イベントを読まないテストでは配線しない」が既定になり、
// 読みたくなったときに配線から書き直すことになる。
//
// 排他するのは、`httptest.Server` 越しに動かすテストが同じ log へ並行に書くためである。
type EventLog struct {
	mu     sync.Mutex
	events []spec.DomainEvent
}

func (l *EventLog) record(event spec.DomainEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
}

// Types は発行されたイベント型を発行順に返す。
func (l *EventLog) Types() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	types := make([]string, len(l.events))
	for i, event := range l.events {
		types[i] = event.EventType()
	}
	return types
}

// AssertEmitted は、名指したイベント型がすべて発行されたことを確かめる。
func (l *EventLog) AssertEmitted(t *testing.T, eventTypes ...string) {
	t.Helper()
	emitted := l.Types()
	for _, want := range eventTypes {
		if !slices.Contains(emitted, want) {
			t.Fatalf("%s が発行されていない: %v", want, emitted)
		}
	}
}

// AssertNotEmitted は、拒否がそのイベントを 1 件も残していないことを確かめる。
func (l *EventLog) AssertNotEmitted(t *testing.T, eventTypes ...string) {
	t.Helper()
	emitted := l.Types()
	for _, unwanted := range eventTypes {
		if slices.Contains(emitted, unwanted) {
			t.Fatalf("%s が発行されている: %v", unwanted, emitted)
		}
	}
}

// Stack は建てたスタックと、そこへ配線した保存先をまとめて持つ。
//
// 保存先を公開するのは、拒否が防いだ効果を読み直すためである。nil のままの field は
// その option を渡していないことを意味する。
type Stack struct {
	Echo     *echo.Echo
	Events   *EventLog
	Tenants  *tenancymemory.TenantRepository
	Users    *usermemory.UserRepository
	KeyStore *signingmemory.InMemoryKeyStore
	Signer   *tokensjose.JWTSigner

	Clients            *oauth2memory.OAuth2ClientRepository
	Consents           *consentmemory.ConsentRepository
	AuthzDetailTypes   *oauth2memory.AuthorizationDetailTypeRepository
	McpResourceServers *oauth2memory.McpResourceServerRepository
	Codes              *oauth2memory.AuthorizationCodeStore
	PAR                *oauth2memory.PARStore
	Refresh            *oauth2memory.RefreshTokenStore
	ApiTokens          apitokenports.Repository
	SamlSPs            *samlmemory.SamlServiceProviderRepository
	WsFedRPs           *wsfedmemory.WsFedRelyingPartyRepository
	Sessions           *sessionusecases.SessionManager
	// SessionStore は Sessions の保管先。サインアウトの具体例は「サーバー側の
	// セッションが失効したか」を言っていて、それは応答ではなくここにしか現れない。
	SessionStore *sessionmemory.SessionStore

	apiTokens *apitokenusecases.Service
	// owner は New を呼んだテストである。HTTP サーバーの後始末はここへ登録する。
	// Browser を呼んだ subtest へ登録すると、その subtest が終わった時点でサーバーが
	// 閉じ、同じスタックを使う次の subtest が接続できなくなる。
	owner  *testing.T
	server *httptest.Server
}

// Option は 1 つの入口を配線へ足す。
type Option func(*builder)

type builder struct {
	t     *testing.T
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
		b.stack.SessionStore = sessionmemory.NewSessionStore()
		b.stack.Sessions = sessionusecases.NewSessionManager(b.stack.SessionStore)
		b.deps.OAuth2.RequestStore = oauth2memory.NewAuthorizationRequestStore()
		b.deps.OAuth2.CodeStore = b.stack.Codes
		b.stack.PAR = oauth2memory.NewPARStore()
		b.deps.OAuth2.PARStore = b.stack.PAR
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

// WithBrowserFlow は `/authorize` からブラウザーのログインと同意を経て `/token` まで
// 通せる状態にする。配線だけでは通らないので、confidential クライアントと、パスワードを
// 持つ利用者も一緒に置く。
//
// wi-538 が `EX-OAUTH2-001-01` を子 work item へ回した理由がここだった。認可コードと
// `/token` を同時に配線した fixture は 94 個のどれかにあっても、そこへ「ログインできる
// 利用者」と「リダイレクト先を登録したクライアント」が揃っているものは無く、具体例を
// 1 件消化するたびに seed から書き直していた。
func WithBrowserFlow() Option {
	return func(b *builder) {
		WithAuthorizationCodeFlow()(b)
		if b.stack.Refresh == nil {
			WithTokenIssuance()(b)
		}
		secretHash := oauthdomain.HashClientSecret(BrowserClientSecret)
		for _, tenantID := range []string{tenancydomain.DefaultTenantID, OtherRealm} {
			b.stack.Clients.Seed(&oauthdomain.OAuth2Client{
				TenantID: tenantID, ClientID: BrowserClientID, ClientSecretHash: &secretHash,
				ClientType:   spec.ClientConfidential,
				RedirectURIs: []string{BrowserRedirectURI},
				GrantTypes: []spec.GrantType{
					spec.GrantAuthorizationCode, spec.GrantRefreshToken,
				},
				ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
				TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
				// account スコープまで許可しておく。REQ-OAUTH2-001 の具体例は、利用者に
				// 紐づくグラントが account リソースサーバーへ通ることを言っていて、
				// 許可スコープに無いクライアントではその経路そのものを作れない。
				Scope:                    "openid profile email offline_access account:read account:write",
				IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
				FapiProfile:              oauthdomain.FapiNone,
				CreatedAt:                time.Now().UTC(),
			})
		}
		hasher := testingpasswords.NewHasher()
		hash, err := hasher.Hash(UserPassword)
		if err != nil {
			b.t.Fatalf("seed password: %v", err)
		}
		user, err := b.stack.Users.FindBySub(context.Background(), UserID)
		if err != nil || user == nil {
			b.t.Fatalf("seed user %s: user=%v err=%v", UserID, user, err)
		}
		user.PasswordHash = hash
		b.stack.Users.Seed(user)
	}
}

// WithLoginThrottle はアカウント単位と IP 単位のログイン失敗の計数を配線する。
//
// 既定では配線しない。宣言済みの具体例が言う閾値 (900 秒で 10 回) をそのまま使うと、
// 1 件の観測に 10 往復かかる。閾値は呼び出し側が決め、時間枠と締め出しは具体例と同じ
// 900 秒に固定する。締め出しが「失敗を数えた結果」であることは、閾値までの試行が
// 通ることで示す。
func WithLoginThrottle(accountFailures, ipFailures int) Option {
	return func(b *builder) {
		b.deps.Authentication.LoginAttemptThrottle = sessionmemory.NewLoginAttemptThrottle(
			sessionports.LoginThrottleConfigs{
				Account: sessionports.LoginThrottleConfig{
					MaxFailures: accountFailures, WindowSeconds: 900, LockoutSeconds: 900,
				},
				IP: sessionports.LoginThrottleConfig{
					MaxFailures: ipFailures, WindowSeconds: 900, LockoutSeconds: 900,
				},
			})
		// IP 単位の計数は送信元 IP が解決できて初めて効く。信頼するホップ数が 0 の
		// ままだと `X-Forwarded-For` は読まれず、IP の閾値は一度も評価されない。
		b.deps.TrustedForwardedHops = 1
	}
}

// blockingRateLimiter は名指したポリシーだけを閾値超過として拒否する。
//
// `EndpointRateLimitPolicy` の時間枠そのものは Tenancy の設定であり、この具体例が
// 言っているのは「上限に達しているとき何が起きるか」である。実際に時間枠を埋めると、
// 観測したい拒否ではなく設定の再現に費用がかかる。
// 上限に達しているのは名指したポリシーと鍵の組だけである。鍵まで見るのは、
// 「同一 IP からの要求が」という具体例の主語を観測に残すためである。ポリシーだけで
// 拒否すると、送信元 IP が要求から読めていない実装でも同じ拒否が起きてしまう。
type blockingRateLimiter struct {
	policy string
	key    string
}

func (l *blockingRateLimiter) Allow(
	_ context.Context, policyID, key string, _ time.Time,
) (rlports.RateLimitResult, error) {
	if policyID == l.policy && key == l.key {
		return rlports.RateLimitResult{Allowed: false, RetryAfterSeconds: 30}, nil
	}
	return rlports.RateLimitResult{Allowed: true}, nil
}

// WithEndpointRateLimitReached は、名指したエンドポイントの流量制限が、名指した鍵
// (ログインなら送信元 IP) について上限に達している状態にする。ほかの鍵は通る。
func WithEndpointRateLimitReached(policy, key string) Option {
	return func(b *builder) {
		b.deps.RateLimiter = &blockingRateLimiter{policy: policy, key: key}
		b.deps.TrustedForwardedHops = 1
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
		b.deps.Saml = saml.Module{
			SPRepo: b.stack.SamlSPs, ProfileRepo: b.stack.SamlSPs,
			ReplayStore: samlmemory.NewAuthnRequestReplayStore(),
		}
		b.deps.FederationSigner = samltoken.KeyStoreSignerProvider{KeyStore: b.stack.KeyStore}
	}
}

// WithWsFederation は WS-Federation の保存先を配線し、返信先を登録した RP を 1 つ置く。
// サインアウトはその RP を名指しで呼ぶので、登録が無いと入口そのものが無い。
func WithWsFederation() Option {
	return func(b *builder) {
		b.stack.WsFedRPs = wsfedmemory.NewWsFedRelyingPartyRepository()
		b.stack.WsFedRPs.Seed(&feddomain.WsFedRelyingParty{
			Wtrealm:   WsFedRealm,
			ReplyURLs: []string{WsFedReplyURL},
			ClaimPolicy: claimdomain.ClaimMappingPolicy{
				NameID: claimdomain.NameIdConfiguration{
					Format:          "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
					SourceAttribute: "user_id",
				},
			},
		})
		b.deps.WsFederation = wsfederation.Module{RPRepo: b.stack.WsFedRPs}
		if b.deps.FederationSigner == nil {
			b.deps.FederationSigner = samltoken.KeyStoreSignerProvider{KeyStore: b.stack.KeyStore}
		}
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
		Echo: echo.New(), Events: &EventLog{}, owner: t,
		Tenants: tenants, Users: users, KeyStore: keyStore, Signer: signer,
	}
	b := &builder{
		t: t, stack: stack,
		// 渡すのは module だけである。`Deps` は移行期の互換入力として `UserRepo`、
		// `KeyStore`、`TokenIssuer` を平置きでも受けるが、bootstrap は module しか設定
		// しない。互換入力を使うと、この基盤を共有する全テストが製品と違う経路の
		// 組み立てを観測することになる。
		deps: httpadapter.Deps{
			Issuer: Issuer, Contract: spec.CurrentRuntimeContract(),
			Emit:         stack.Events.record,
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

// Browser はブラウザー経由の認可を 1 本通すための、cookie を持つクライアントである。
//
// `s.Echo.ServeHTTP` を直接呼ぶ形にしないのは、この経路が cookie に依存しているため
// である。認可トランザクション、CSRF、ログインセッションの 3 つが cookie で運ばれ、
// 手で付け替える fixture は「どの cookie を運ぶか」を毎回書き直すことになる。
type Browser struct {
	base   string
	client *http.Client
}

// Browser は realm の入口を指す browser を返す。スタックごとに 1 つの HTTP サーバーを
// 立て、テストの終了で閉じる。
func (s *Stack) Browser(t *testing.T, realm string) *Browser {
	t.Helper()
	if s.server == nil {
		s.server = httptest.NewServer(s.Echo)
		s.owner.Cleanup(s.server.Close)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return &Browser{
		base: s.server.URL + "/realms/" + realm,
		client: &http.Client{
			Jar: jar,
			// リダイレクトは追わない。Location そのものが観測対象だからである。
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// AuthorizationQuery は 1 本通る認可リクエストを返す。個々のテストはここから 1 か所だけを
// 崩す。崩していない事例が通ることを対照に置けるので、拒否の理由が崩した 1 か所であると
// 読める。
func AuthorizationQuery(verifier string, overrides map[string]string) url.Values {
	challenge := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id":             {BrowserClientID},
		"redirect_uri":          {BrowserRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile"},
		"state":                 {"opaque-state"},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
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

// Authorize は `/authorize` を 1 回叩く。応答は閉じずに返すので、呼び出し側が
// Location も本文も読める。
func (b *Browser) Authorize(t *testing.T, query url.Values) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet, b.base+"/authorize?"+query.Encode(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := b.client.Do(request)
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// Transaction は現在の認可トランザクションを返す。`kind` が login と consent のどちらで
// あるかが、同意画面を出し分ける規則の観測点になる。
func (b *Browser) Transaction(t *testing.T) map[string]any {
	t.Helper()
	request, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet, b.base+"/api/auth/transaction", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	response, err := b.client.Do(request)
	if err != nil {
		t.Fatalf("GET /api/auth/transaction: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/auth/transaction status=%d body=%s", response.StatusCode, raw)
	}
	transaction := map[string]any{}
	if err := json.Unmarshal(raw, &transaction); err != nil {
		t.Fatalf("transaction が JSON ではない body=%s: %v", raw, err)
	}
	return transaction
}

// SignIn はトランザクションの利用者を認証する。
func (b *Browser) SignIn(t *testing.T, username, password string) map[string]any {
	t.Helper()
	transaction := b.Transaction(t)
	csrf, _ := transaction["csrf_token"].(string)
	return b.postJSON(t, "/api/auth/login", csrf,
		map[string]string{"username": username, "password": password})
}

// TrustedProxyIP は、信頼するホップ 1 つぶんの代理としてテストが名乗るアドレスである。
// 製品は `X-Forwarded-For` の末尾から数えて信頼するホップ数だけ内側を送信元とみなすので、
// 送信元 IP を 1 つ名乗るには連なりが 2 つ必要になる。
const TrustedProxyIP = "10.0.0.1"

// SignInAttempt は SignIn と同じ要求を送り、拒否された応答もそのまま返す。
// 流量制限と失敗回数の具体例は、成立しない応答そのものが観測対象である。
// clientIP が空でなければ、信頼する代理を 1 つ挟んだ連なりとしてその値を名乗る。
func (b *Browser) SignInAttempt(
	t *testing.T, username, password, clientIP string,
) (int, map[string]any) {
	t.Helper()
	transaction := b.Transaction(t)
	csrf, _ := transaction["csrf_token"].(string)
	forwardedFor := ""
	if clientIP != "" {
		forwardedFor = clientIP + ", " + TrustedProxyIP
	}
	return b.postJSONAttempt(t, "/api/auth/login", csrf, forwardedFor,
		map[string]string{"username": username, "password": password})
}

// Consent は同意画面の判断を送り、リダイレクト先を返す。
func (b *Browser) Consent(t *testing.T, action string) string {
	t.Helper()
	transaction := b.Transaction(t)
	csrf, _ := transaction["csrf_token"].(string)
	result := b.postJSON(t, "/api/auth/consent", csrf, map[string]string{"action": action})
	redirect, _ := result["redirect_to"].(string)
	if redirect == "" {
		t.Fatalf("同意の応答がリダイレクト先を運んでいない: %v", result)
	}
	return redirect
}

// AuthorizationCode は `/authorize` からログインと同意までを通し、発行された認可コードを
// 返す。2 つ目の戻り値はリダイレクト先そのもので、`state` や `iss` の観測に使う。
func (b *Browser) AuthorizationCode(t *testing.T, query url.Values) (string, *url.URL) {
	t.Helper()
	_ = b.Authorize(t, query).Body.Close()
	b.SignIn(t, "user", UserPassword)
	redirect, err := url.Parse(b.Consent(t, "allow"))
	if err != nil {
		t.Fatalf("リダイレクト先を URL として読めない: %v", err)
	}
	code := redirect.Query().Get("code")
	if code == "" {
		t.Fatalf("リダイレクト先が認可コードを運んでいない: %s", redirect)
	}
	return code, redirect
}

// ExchangeCode は認可コードを `/token` で交換し、状態行と復号した本文を返す。
func (b *Browser) ExchangeCode(t *testing.T, code, verifier string) (int, map[string]any) {
	t.Helper()
	return b.PostToken(t, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {BrowserRedirectURI},
	})
}

// PostToken は `/token` へフォームを 1 通送る。クライアント認証は WithBrowserFlow が
// seed した confidential クライアントで行う。
func (b *Browser) PostToken(t *testing.T, form url.Values) (int, map[string]any) {
	t.Helper()
	request, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, b.base+"/token", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(BrowserClientID, BrowserClientSecret)
	response, err := b.client.Do(request)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	body := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &body)
	}
	return response.StatusCode, body
}

func (b *Browser) postJSON(t *testing.T, path, csrf string, payload any) map[string]any {
	t.Helper()
	status, result := b.postJSONAttempt(t, path, csrf, "", payload)
	if status != http.StatusOK {
		t.Fatalf("POST %s status=%d body=%v", path, status, result)
	}
	return result
}

// postJSONAttempt は成立しない応答も返す。csrf を空にすると二重送信が成立しない要求になる。
func (b *Browser) postJSONAttempt(
	t *testing.T, path, csrf, forwardedFor string, payload any,
) (int, map[string]any) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, b.base+path, bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	if csrf != "" {
		request.Header.Set("X-Csrf-Token", csrf)
	}
	request.Header.Set("Origin", Issuer)
	if forwardedFor != "" {
		request.Header.Set("X-Forwarded-For", forwardedFor)
	}
	response, err := b.client.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	result := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}
	return response.StatusCode, result
}

// SignInWithBadCSRF は CSRF の二重送信が成立しないログイン要求を送る。
func (b *Browser) SignInWithBadCSRF(t *testing.T, username, password string) (int, map[string]any) {
	t.Helper()
	_ = b.Transaction(t)
	return b.postJSONAttempt(t, "/api/auth/login", "tampered-csrf-token", "",
		map[string]string{"username": username, "password": password})
}

// SessionCookie は browser が保持しているセッション Cookie の値を返す。無ければ空文字。
func (b *Browser) SessionCookie(t *testing.T) string {
	t.Helper()
	base, err := url.Parse(b.base)
	if err != nil {
		t.Fatal(err)
	}
	for _, cookie := range b.client.Jar.Cookies(base) {
		if cookie.Name == sessionusecases.SessionCookie {
			return cookie.Value
		}
	}
	return ""
}

// PushAuthorizationRequest は `/par` へクライアント認証付きで 1 回送り、状態行と
// 復号した本文を返す。事前送信は認可の前段なので、browser の cookie は関係しない。
func (b *Browser) PushAuthorizationRequest(t *testing.T, query url.Values) (int, map[string]any) {
	t.Helper()
	request, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, b.base+"/par", strings.NewReader(query.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(BrowserClientID, BrowserClientSecret)
	response, err := b.client.Do(request)
	if err != nil {
		t.Fatalf("POST /par: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	body := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &body)
	}
	return response.StatusCode, body
}
