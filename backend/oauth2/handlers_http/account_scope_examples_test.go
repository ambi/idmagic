package handlers_http_test

// docs/contexts/oauth2/scenarios.feature.md の REQ-OAUTH2-001 が宣言する通常経路を観測する。
//
// この具体例は「発行したトークンが account リソースサーバーで何を許すか」まで言っている
// ので、発行の観測だけでは足りない。トークンを組み立てて直接 API を叩くのでもない。
// 認可コード + PKCE を正式な入口で通し、そこで出たトークンをそのまま提示する。

import (
	"net/http"
	"net/http/httptest"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// EX-OAUTH2-001-01 の audience は、この配線では観測できない。製品は resource 指定の
// 無いアクセストークンの `aud` を client_id にしていて、レルムの IdMagic API を指す
// 経路が無い。account リソースサーバー側も audience を読んでいない。規範の判断が
// 要るので wi-570 が持つ。ここでは残りの Then を固定する。
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
