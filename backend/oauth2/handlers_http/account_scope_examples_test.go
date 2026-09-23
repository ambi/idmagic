package handlers_http_test

// docs/domain/oauth2/scenarios.feature.md の REQ-OAUTH2-001 が宣言する通常経路を観測する。
//
// この具体例は「発行したトークンが account リソースサーバーで何を許すか」まで言っている
// ので、発行の観測だけでは足りない。トークンを組み立てて直接 API を叩くのでもない。
// 認可コード + PKCE を正式な入口で通し、そこで出たトークンをそのまま提示する。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// audience は resource を指定しない要求から推定する値なので、authorize にも交換にも
// resource を付けない。付けると推定を通らずに McpResourceServer の束縛を観測してしまう。
//
//spec:covers EX-OAUTH2-001-01, RFC9068-DEFAULT-AUDIENCE: account:read に同意した認可コードの交換は、sub が同意した User、aud がレルムの発行者識別子だけ、scope が account:read のトークンを返し、そのトークンで account API の参照は通り撤回は通らない。
func TestAccountScopedGrantIssuesAUserBoundTokenTheAccountApiAccepts(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithAccountApi())
	browser := s.Browser(t, tenancydomain.DefaultRealm)

	code, _ := browser.AuthorizationCode(t, stack.AuthorizationQuery(codeVerifier,
		map[string]string{"scope": "account:read"}))
	status, body := browser.ExchangeCode(t, code, codeVerifier)
	if status != http.StatusOK {
		t.Fatalf("認可コードの交換 status=%d body=%v", status, body)
	}
	accessToken, _ := body["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("応答が access_token を運んでいない: %v", body)
	}

	claims := jwtClaims(t, accessToken)
	if claims["sub"] != stack.UserID {
		t.Fatalf("sub=%#v, want %q (同意した利用者)", claims["sub"], stack.UserID)
	}
	if claims["scope"] != "account:read" {
		t.Fatalf("scope=%#v, want account:read", claims["scope"])
	}
	if realmAPI := stack.Issuer + "/realms/" + tenancydomain.DefaultRealm; claims["aud"] != realmAPI {
		t.Fatalf("aud=%#v, want %q (レルムの IdMagic API)", claims["aud"], realmAPI)
	}
	// 参照は通る。
	if code := accountRequest(t, s, http.MethodGet, "/api/account/v1/consents", accessToken); code != http.StatusOK {
		t.Fatalf("account:read のトークンで参照が通らない: status=%d", code)
	}

	// 同じトークンで書き換えは通らない。account:read が読み取りだけを許すことは、
	// 書き換えが拒否されることでしか読めない。通るものだけを見るテストは、
	// スコープを無視して一律に許す実装を素通りさせる。
	written := accountRequest(t, s, http.MethodPost,
		"/api/account/v1/consents/"+stack.BrowserClientID+"/revoke", accessToken)
	if written == http.StatusOK || written == http.StatusNoContent {
		t.Fatalf("account:read のトークンで撤回が通った: status=%d", written)
	}
}

// 製品の発行経路は account スコープのトークンに必ずレルムの IdMagic API を名乗らせるので、
// 別の audience を持つトークンは製品と同じ署名器とレルムの文脈で直接作る。対照として、
// 同じ組み立てでレルムの IdMagic API を名乗るトークンと、ポータル境界のスコープだけを
// 持つトークンが通ることも読む。これがなければ、拒否が audience によるものか署名や
// 文脈の組み立て違いによるものかを区別できない。
//
//spec:covers EX-OAUTH2-001-04: aud が client_id または別レルムの発行者識別子である account:read のトークンは account API の参照を 401 で拒否され、レルムの発行者識別子を名乗る同じトークンと idmagic.account だけのトークンは通る。
func TestAccountApiRefusesAccountScopedTokenForAnotherAudience(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithAccountApi())
	ctx := s.RealmContext(t, tenancydomain.DefaultRealm)
	client, err := s.Clients.FindByID(ctx, s.TenantID(t, tenancydomain.DefaultRealm), stack.BrowserClientID)
	if err != nil || client == nil {
		t.Fatalf("client=%v err=%v", client, err)
	}
	realmAPI := stack.Issuer + "/realms/" + tenancydomain.DefaultRealm
	sign := func(scope string, audiences ...string) string {
		t.Helper()
		token, _, err := s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
			Client: client, Sub: stack.UserID, Scopes: strings.Fields(scope), Audiences: audiences,
			AuthTime: time.Now().Unix(),
		})
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return token
	}

	for _, tc := range []struct {
		name  string
		token string
		want  int
	}{
		{"aud が client_id", sign("account:read", stack.BrowserClientID), http.StatusUnauthorized},
		{"aud が別レルムの発行者識別子", sign("account:read", stack.Issuer+"/realms/other"), http.StatusUnauthorized},
		{"対照: aud がレルムの IdMagic API", sign("account:read", realmAPI), http.StatusOK},
		{"対照: ポータル境界のスコープだけ", sign("openid idmagic.account", stack.BrowserClientID), http.StatusOK},
	} {
		if got := accountRequest(t, s, http.MethodGet, "/api/account/v1/consents", tc.token); got != tc.want {
			t.Errorf("%s: status=%d, want %d", tc.name, got, tc.want)
		}
	}
}

// accountRequest は account リソースサーバーへ Bearer で 1 回叩き、状態行を返す。
func accountRequest(t *testing.T, s *stack.Stack, method, path, token string) int {
	t.Helper()
	request := httptest.NewRequest(method,
		stack.Issuer+"/realms/"+tenancydomain.DefaultRealm+path, http.NoBody)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Origin", stack.Issuer)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder.Code
}
