package testing_stack_test

// この基盤が要求を満たしているかは、[[wi-538]] が「94 個のどの fixture にも無い」として
// 子 work item へ回した組み立て — 認可コードの交換から account リソースサーバーまで — を、
// option の合成だけで建てられるかで決まる。件数ではなくその 1 件で判定する。
//
// 当の具体例の id はここに書かない。被覆の検査は id を名指したテストの存在で判定するので、
// 到達可能性を示しただけの本ファイルが名指すと、消化済みと数えられてしまう。

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

func get(t *testing.T, s *stack.Stack, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, stack.Issuer+path, http.NoBody)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

// 認可コード + PKCE の交換、トークンの発行、そして account リソースサーバーが、
// 3 つの option の合成で 1 つのスタックに建つ。
//
// 建ったことは、3 つの入口がそれぞれ経路として存在することで読む。ルートが登録されて
// いなければ echo は 404 を返すので、404 でないことがその入口に届いた証拠になる。
// 具体例そのものの消化は [[wi-559]] が持つので、ここでは到達可能性だけを固定する。
func TestAuthorizationCodeAndAccountApiComposeIntoOneStack(t *testing.T) {
	s := stack.New(t,
		stack.WithAuthorizationCodeFlow(),
		stack.WithTokenIssuance(),
		stack.WithAccountApi())

	if s.Clients == nil || s.Consents == nil || s.Codes == nil || s.Refresh == nil {
		t.Fatalf("保存先が配線されていない: clients=%v consents=%v codes=%v refresh=%v",
			s.Clients != nil, s.Consents != nil, s.Codes != nil, s.Refresh != nil)
	}

	for _, entry := range []struct{ name, path string }{
		{"認可", "/realms/default/authorize"},
		{"Discovery", "/realms/default/.well-known/openid-configuration"},
		{"account 同意", "/realms/default/api/account/v1/consents"},
	} {
		if recorder := get(t, s, entry.path); recorder.Code == http.StatusNotFound {
			t.Fatalf("%s の入口が登録されていない: %s", entry.name, entry.path)
		}
	}
}

// 既定が製品の組み立てと一致しているかを、いちばん間違えやすい 1 点で読む。
//
// `WithApiTokens` が `OAuth2.TokenIntrospector` へ渡すのは、管理発行トークンのライフ
// サイクル記録を重ねた overlay である。生の署名検証器を渡すと、管理コンソールからの失効は
// 記録にしか載らないので、失効したトークンが `/introspect` で `active: true` に見える。
// それは製品の欠陥ではなく組み立ての違いであり、この基盤を共有する全テストへ一度に広がる。
//
// 観測は `/introspect` に置く。管理 API の入口は `ApiTokens` module 側の照合を通るので、
// overlay を外しても落ちない。組み立ての違いは、それが効く入口でしか読めない。
func TestApiTokenIntrospectionSeesTheRevocationRecord(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithOAuth2Clients(), stack.WithTokenIssuance())
	token, metadata := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeUsersRead)

	// 失効させる前は active に見える。これが無いと、常に false を返す配線と区別できない。
	if active := s.Introspect(t, tenancydomain.DefaultRealm, token)["active"]; active != true {
		t.Fatalf("発行直後のトークンが active=%v である", active)
	}

	if err := s.ApiTokens.Revoke(
		s.RealmContext(t, tenancydomain.DefaultRealm),
		s.TenantID(t, tenancydomain.DefaultRealm),
		metadata.ID, time.Now().UTC(),
	); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	if active := s.Introspect(t, tenancydomain.DefaultRealm, token)["active"]; active != false {
		t.Fatalf("失効したトークンが active=%v である。overlay ではなく生の署名検証器が"+
			"渡っていると、記録にしかない失効が見えない", active)
	}
}

// 同意の保存先は 2 つの option が要求する。どちらを単独で渡しても配線されなければ
// ならない。
//
// これを別のテストにするのは、両方を渡す組み合わせだけでは片方の配線漏れを検出できない
// ためである。変異テストで実際にそうなった。`WithAuthorizationCodeFlow` 側の冪等ガードを
// 反転させても `WithAccountApi` が作るので合成のテストは通り、逆も同じで、生存した変異が
// 2 件残った。単独で渡す観測を置くと両方とも死ぬ。
func TestEachOptionThatNeedsConsentsWiresItOnItsOwn(t *testing.T) {
	for _, entry := range []struct {
		name   string
		option stack.Option
	}{
		{"WithAuthorizationCodeFlow", stack.WithAuthorizationCodeFlow()},
		{"WithAccountApi", stack.WithAccountApi()},
	} {
		t.Run(entry.name, func(t *testing.T) {
			if s := stack.New(t, entry.option); s.Consents == nil {
				t.Fatalf("%s だけでは同意の保存先が配線されない", entry.name)
			}
		})
	}
}

// ブラウザー経由の認可を 1 本通す駆動部そのものを、この package のテストでも踏む。
//
// 変異テストが示したのは、`Browser` の全行がこの package からは 1 度も実行されて
// いないことだった。呼び出し側 (backend/oauth2/handlers_http) のテストが落ちれば
// 気づけるとはいえ、そのときに壊れているのが製品なのか駆動部なのかが分からない。
// 基盤は自分が動くことを自分で言えなければならない。
//
// 具体例の id はここに書かない。被覆の検査は id を名指したテストの存在で判定するので、
// 駆動部の健全性を示しただけの本ファイルが名指すと、消化済みと数えられてしまう。
func TestBrowserFlowDrivesAuthorizationThroughToAToken(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	const verifier = "testing-stack-browser-flow-pkce-verifier-0123456789"

	// WithBrowserFlow 単独でトークンの保管先まで配線される。ここを読まないと、
	// 発行の保管先が無いスタックでも「認可コードは出た」で通ってしまう。
	if s.Refresh == nil {
		t.Fatal("WithBrowserFlow だけではリフレッシュトークンの保管先が配線されない")
	}

	code, redirect := browser.AuthorizationCode(t, stack.AuthorizationQuery(verifier, nil))
	if code == "" {
		t.Fatalf("認可コードが返っていない: %s", redirect)
	}
	if got := redirect.Query().Get("state"); got != "opaque-state" {
		t.Fatalf("state=%q が伝播していない: %s", got, redirect)
	}

	status, body := browser.ExchangeCode(t, code, verifier)
	if status != http.StatusOK {
		t.Fatalf("交換 status=%d body=%v", status, body)
	}
	if token, _ := body["access_token"].(string); token == "" {
		t.Fatalf("交換がアクセストークンを返さない: %v", body)
	}

	// 事前送信の駆動部も同じ 1 本で踏む。
	pushStatus, pushed := browser.PushAuthorizationRequest(t,
		stack.AuthorizationQuery(verifier, nil))
	if pushStatus != http.StatusCreated {
		t.Fatalf("/par status=%d body=%v", pushStatus, pushed)
	}
	if requestURI, _ := pushed["request_uri"].(string); requestURI == "" {
		t.Fatalf("/par が request_uri を返さない: %v", pushed)
	}

	// 記録が配線されていることを 1 点で読む。ここが空なら、イベントを読む
	// 呼び出し側のテストは「発行されていない」と「配線していない」を区別できない。
	if len(s.Events.Types()) == 0 {
		t.Fatal("認可からトークン発行まで通したのにイベントが 1 件も記録されていない")
	}
}

// サインインの駆動部を、成立する側と成立しない側の両方で踏む。
//
// 拒否された応答を返す入口 (SignInAttempt、SignInWithBadCSRF) と、発行された
// セッション Cookie を読み出す入口 (SessionCookie) は、呼び出し側のテストからしか
// 実行されていなかった。そこで落ちたとき、壊れているのが製品なのか駆動部なのかを
// 見分ける手掛かりが無い。
//
// 具体例の id はここに書かない。被覆の検査は id を名指したテストの存在で判定するので、
// 駆動部の健全性を示しただけの本ファイルが名指すと、消化済みと数えられてしまう。
func TestBrowserDrivesBothSidesOfSignIn(t *testing.T) {
	const clientIP = "198.51.100.3"
	// 上限に達しているのはこの送信元 IP だけである。名乗った IP が要求として届いて
	// いなければ、最後の 429 は起きない。駆動部が連なりを組み立てていることは、
	// ここでしか読めない。
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithLoginThrottle(5, 100),
		stack.WithEndpointRateLimitReached("login", clientIP))
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	const verifier = "testing-stack-sign-in-driver-pkce-verifier-01234567"
	_ = browser.Authorize(t, stack.AuthorizationQuery(verifier, nil)).Body.Close()

	if cookie := browser.SessionCookie(t); cookie != "" {
		t.Fatalf("認証の前から SessionCookie が値を返す: %q", cookie)
	}
	if status, body := browser.SignInWithBadCSRF(t, "user", stack.UserPassword); status == http.StatusOK {
		t.Fatalf("二重送信の成立しない要求が status=%d body=%v で通った", status, body)
	}
	if status, body := browser.SignInAttempt(t, "user", "not-the-password", "198.51.100.9"); status != http.StatusUnauthorized {
		t.Fatalf("上限に達していない送信元からの誤ったパスワード status=%d body=%v、期待は 401", status, body)
	}
	if status, body := browser.SignInAttempt(t, "user", stack.UserPassword, clientIP); status != http.StatusTooManyRequests {
		t.Fatalf("上限に達した送信元 status=%d body=%v、期待は 429", status, body)
	}
	if cookie := browser.SessionCookie(t); cookie != "" {
		t.Fatalf("成立しなかった要求のあとに SessionCookie が値を返す: %q", cookie)
	}

	// 成立する側。失敗の計数はアカウント単位に残るので、別のスタックで踏む。
	allowed := stack.New(t, stack.WithBrowserFlow())
	fresh := allowed.Browser(t, tenancydomain.DefaultRealm)
	_ = fresh.Authorize(t, stack.AuthorizationQuery(verifier, nil)).Body.Close()
	if status, body := fresh.SignInAttempt(t, "user", stack.UserPassword, "198.51.100.4"); status != http.StatusOK {
		t.Fatalf("正しいパスワード status=%d body=%v", status, body)
	}
	if fresh.SessionCookie(t) == "" {
		t.Fatal("認証が成立したのに SessionCookie が空を返す")
	}
}

// WS-Federation の配線が、SAML と同じスタックへ option 1 つで乗る。
func TestWsFederationComposesWithSaml(t *testing.T) {
	s := stack.New(t, stack.WithAuthorizationCodeFlow(), stack.WithSaml(), stack.WithWsFederation())
	if s.WsFedRPs == nil {
		t.Fatal("WithWsFederation が RP の保存先を配らない")
	}

	// 入口があることは、ルート未登録の 404 でないことで読む。
	signOut := get(t, s, "/realms/"+tenancydomain.DefaultRealm+
		"/wsfed?wa=wsignout1.0&wtrealm="+url.QueryEscape(stack.WsFedRealm))
	if signOut.Code == http.StatusNotFound {
		t.Fatalf("WS-Federation の入口が登録されていない: body=%s", signOut.Body.String())
	}
	metadata := get(t, s, "/realms/"+tenancydomain.DefaultRealm+
		"/federationmetadata/2007-06/federationmetadata.xml")
	if metadata.Code != http.StatusOK {
		t.Fatalf("フェデレーションメタデータ status=%d", metadata.Code)
	}

	// WS-Federation 単独でも署名者が配線される。SAML と合成したときだけ配線される
	// 実装では、この 1 本が署名できない。
	alone := stack.New(t, stack.WithWsFederation())
	if aloneMetadata := get(t, alone, "/realms/"+tenancydomain.DefaultRealm+
		"/federationmetadata/2007-06/federationmetadata.xml"); aloneMetadata.Code != http.StatusOK {
		t.Fatalf("WS-Federation 単独のメタデータ status=%d body=%s",
			aloneMetadata.Code, aloneMetadata.Body.String())
	}
}
