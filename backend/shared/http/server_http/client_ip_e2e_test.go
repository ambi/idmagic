package server_http_test

// REQ-SYSTEM-032 のクライアント IP の解決を、IP 単位のログイン抑制の入口から観測する。
//
// 解決した値そのものは応答に出ないので、同じ送信元とみなされたかどうかを IP 単位の
// 閾値で読む。アカウント単位の閾値は高くして、締め出しが IP の軸だけで起きるようにする。
// テストの HTTP クライアントは loopback から接続するので、直接の接続の相手は常に
// 127.0.0.1 である。

import (
	"net/http"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const clientIPVerifier = "client-ip-examples-pkce-verifier-0123456789abcdefgh"

//spec:covers REQ-SYSTEM-032: 信頼する代理を設定しない構成では、X-Forwarded-For を変えても接続相手が同じなら同じ送信元として数える。
func TestLoginIPThrottleKeysOnDirectPeer_REQ_SYSTEM_032(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithLoginThrottle(100, 2), stack.WithTrustedProxies())
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	_ = browser.Authorize(t, stack.AuthorizationQuery(clientIPVerifier, nil)).Body.Close()

	for i, forwardedFor := range []string{"198.51.100.1", "198.51.100.2"} {
		status, body := browser.SignInAttemptForwardedFor(t, "user", "not-the-password", forwardedFor)
		if status != http.StatusUnauthorized {
			t.Fatalf("%d 回目の誤ったパスワード status=%d body=%v、期待は 401", i+1, status, body)
		}
	}
	status, body := browser.SignInAttemptForwardedFor(t, "user", stack.UserPassword, "198.51.100.3")
	if status != http.StatusTooManyRequests {
		t.Fatalf("同じ接続相手からの 3 回目 status=%d body=%v、期待は IP 単位の閾値による 429", status, body)
	}
}

//spec:covers REQ-SYSTEM-032: 信頼する代理の範囲を設定した構成では、代理が付けた送信元を採り、クライアントが左に送った値は採らない。
func TestLoginIPThrottleIgnoresForgedEntriesBehindProxy_REQ_SYSTEM_032(t *testing.T) {
	// 代理は接続相手の 127.0.0.1 の 1 段だけである。代理はクライアントが送った
	// X-Forwarded-For の右に、自分が見た送信元 203.0.113.7 を付ける。
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithLoginThrottle(100, 2), stack.WithTrustedProxies("127.0.0.1/32"))
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	_ = browser.Authorize(t, stack.AuthorizationQuery(clientIPVerifier, nil)).Body.Close()

	for i, forged := range []string{"198.51.100.1", "198.51.100.2"} {
		status, body := browser.SignInAttemptForwardedFor(t, "user", "not-the-password", forged+", 203.0.113.7")
		if status != http.StatusUnauthorized {
			t.Fatalf("%d 回目の誤ったパスワード status=%d body=%v、期待は 401", i+1, status, body)
		}
	}
	status, body := browser.SignInAttemptForwardedFor(t, "user", stack.UserPassword, "198.51.100.3, 203.0.113.7")
	if status != http.StatusTooManyRequests {
		t.Fatalf("同じ送信元からの 3 回目 status=%d body=%v、期待は IP 単位の閾値による 429", status, body)
	}

	// 対照：代理が付けた送信元が違えば別の送信元として数え、締め出さない。
	other := stack.New(t, stack.WithBrowserFlow(), stack.WithLoginThrottle(100, 2), stack.WithTrustedProxies("127.0.0.1/32"))
	fresh := other.Browser(t, tenancydomain.DefaultRealm)
	_ = fresh.Authorize(t, stack.AuthorizationQuery(clientIPVerifier, nil)).Body.Close()
	for i, origin := range []string{"203.0.113.1", "203.0.113.2"} {
		if status, body := fresh.SignInAttemptForwardedFor(t, "user", "not-the-password", "198.51.100.1, "+origin); status != http.StatusUnauthorized {
			t.Fatalf("対照の %d 回目 status=%d body=%v、期待は 401", i+1, status, body)
		}
	}
	if status, body := fresh.SignInAttemptForwardedFor(t, "user", stack.UserPassword, "198.51.100.1, 203.0.113.3"); status != http.StatusOK {
		t.Fatalf("別の送信元からの正しいパスワード status=%d body=%v、期待は 200", status, body)
	}
}
