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
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

func get(t *testing.T, s *stack.Stack, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, stack.Issuer+path, http.NoBody)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
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
		if recorder := get(t, s, entry.path, ""); recorder.Code == http.StatusNotFound {
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
