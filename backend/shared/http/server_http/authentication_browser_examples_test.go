package server_http_test

// docs/contexts/authentication/scenarios.feature.md の REQ-AUTHENTICATION-007 と 008 が
// 宣言する具体例を、ブラウザー経由のログインの入口から観測する。
//
// スタックは `testing_stack` が建てる。この具体例群が要求するのは「/authorize から
// ログインまで」「ログインできる利用者」「失敗回数と流量制限の配線」の 3 つで、
// `WithBrowserFlow` と `WithLoginThrottle` と `WithEndpointRateLimitReached` が配る。
//
// 通常経路の `Then` は Cookie の発行、認可コードの返送、イベントの発行の 3 つに分かれる。
// 応答だけを読むテストは、コードは返すが記録を残さない実装を通してしまうので、
// 発行されたイベントも併せて読む。

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const browserLoginVerifier = "authentication-browser-examples-pkce-verifier-01234"

//spec:covers EX-AUTHENTICATION-007-01: ブラウザーのログイン API に正しいパスワードを送るとセッション Cookie が発行され、認可コードが redirect_uri へ返り、UserAuthenticated が残ることを固定する。
func TestBrowserLoginIssuesASessionCookieACodeAndARecord(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)

	query := stack.AuthorizationQuery(browserLoginVerifier, nil)
	_ = browser.Authorize(t, query).Body.Close()
	if cookie := browser.SessionCookie(t); cookie != "" {
		t.Fatalf("前提が壊れている: 認証の前にセッション Cookie がある: %q", cookie)
	}

	browser.SignIn(t, "user", stack.UserPassword)
	if browser.SessionCookie(t) == "" {
		t.Fatal("ログインが成立したのにセッション Cookie が発行されていない")
	}

	redirect, err := url.Parse(browser.Consent(t, "allow"))
	if err != nil {
		t.Fatalf("リダイレクト先を URL として読めない: %v", err)
	}
	if got := redirect.String(); !strings.HasPrefix(got, stack.BrowserRedirectURI) {
		t.Fatalf("redirect=%q、登録された redirect_uri へ返っていない", got)
	}
	if redirect.Query().Get("code") == "" {
		t.Fatalf("redirect=%s が認可コードを運んでいない", redirect)
	}
	s.Events.AssertEmitted(t, "UserAuthenticated")
}

//spec:covers EX-AUTHENTICATION-007-02: SameSite の Cookie とリクエストのトークンが一致しないログイン要求は InvalidRequestError で拒否され、セッション Cookie も認証の記録も残らないことを固定する。
func TestBrowserLoginWithATamperedCSRFTokenLeavesNoSession(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	_ = browser.Authorize(t, stack.AuthorizationQuery(browserLoginVerifier, nil)).Body.Close()

	status, body := browser.SignInWithBadCSRF(t, "user", stack.UserPassword)
	if status != http.StatusForbidden && status != http.StatusBadRequest {
		t.Fatalf("status=%d body=%v、期待は CSRF の拒否", status, body)
	}
	if cookie := browser.SessionCookie(t); cookie != "" {
		t.Fatalf("CSRF の拒否後にセッション Cookie が発行された: %q", cookie)
	}
	s.Events.AssertNotEmitted(t, "UserAuthenticated")

	// 対照: 同じ往復で二重送信が成立すればログインは通る。拒否の理由が CSRF だと示す。
	browser.SignIn(t, "user", stack.UserPassword)
	if browser.SessionCookie(t) == "" {
		t.Fatal("前提が壊れている: CSRF が成立してもセッション Cookie が出ない")
	}
}

//spec:covers EX-AUTHENTICATION-007-03: アカウント単位の失敗回数が上限に達すると、正しいパスワードでも RateLimitedError で拒否され、LoginThrottled が残り、セッションが成立しないことを固定する。
func TestBrowserLoginRefusesTheCorrectPasswordOnceTheAccountIsThrottled(t *testing.T) {
	// 閾値は 2 にする。具体例が言う 10 回と時間枠 900 秒のうち、観測したいのは
	// 「上限に達したあとは正しいパスワードでも通らない」ことであって回数ではない。
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithLoginThrottle(2, 100))
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	_ = browser.Authorize(t, stack.AuthorizationQuery(browserLoginVerifier, nil)).Body.Close()

	for attempt := range 2 {
		status, body := browser.SignInAttempt(t, "user", "not-the-password", "198.51.100.9")
		if status != http.StatusUnauthorized {
			t.Fatalf("%d 回目の誤ったパスワード status=%d body=%v、期待は 401", attempt+1, status, body)
		}
	}

	status, body := browser.SignInAttempt(t, "user", stack.UserPassword, "198.51.100.9")
	if status != http.StatusTooManyRequests {
		t.Fatalf("上限到達後の正しいパスワード status=%d body=%v、期待は 429", status, body)
	}
	if retryAfter, _ := body["retry_after_seconds"].(float64); retryAfter <= 0 {
		if problemType, _ := body["type"].(string); problemType == "" {
			t.Fatalf("429 の応答が RateLimitedError を名乗っていない: %v", body)
		}
	}
	if cookie := browser.SessionCookie(t); cookie != "" {
		t.Fatalf("流量制限の拒否後にセッション Cookie が発行された: %q", cookie)
	}
	s.Events.AssertEmitted(t, "LoginThrottled")
	s.Events.AssertNotEmitted(t, "UserAuthenticated")
}

//spec:covers EX-AUTHENTICATION-007-04: 失敗回数によらず、同一 IP からのログイン要求が EndpointRateLimitPolicy の上限に達していると、正しいパスワードでも RateLimitedError で拒否され、セッションが成立しないことを固定する。
func TestBrowserLoginRefusesTheCorrectPasswordOnceTheEndpointLimitIsReached(t *testing.T) {
	// 失敗の計数は上限に達していない (10 回まで許す) 構成にする。それでも拒否される
	// ことが「失敗回数によらず」の観測点である。
	const clientIP = "203.0.113.5"
	s := stack.New(t, stack.WithBrowserFlow(), stack.WithLoginThrottle(10, 100),
		stack.WithEndpointRateLimitReached("login", clientIP))
	browser := s.Browser(t, tenancydomain.DefaultRealm)
	_ = browser.Authorize(t, stack.AuthorizationQuery(browserLoginVerifier, nil)).Body.Close()

	status, body := browser.SignInAttempt(t, "user", stack.UserPassword, clientIP)
	if status != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%v、期待は 429", status, body)
	}
	if cookie := browser.SessionCookie(t); cookie != "" {
		t.Fatalf("流量制限の拒否後にセッション Cookie が発行された: %q", cookie)
	}
	s.Events.AssertNotEmitted(t, "UserAuthenticated")

	// 対照: 上限に達しているのはこの送信元 IP だけである。同じスタックへ別の IP から
	// 送れば同じ資格情報で通る。拒否が「要求から読んだ送信元 IP」によることを示す。
	other := s.Browser(t, tenancydomain.DefaultRealm)
	_ = other.Authorize(t, stack.AuthorizationQuery(browserLoginVerifier, nil)).Body.Close()
	if status, body := other.SignInAttempt(t, "user", stack.UserPassword, "198.51.100.22"); status != http.StatusOK {
		t.Fatalf("前提が壊れている: 別の送信元 IP で status=%d body=%v", status, body)
	}
}

// 登録期限を過ぎてからの確定は、認証要素を保存せず、セッションを昇格させない。
//
// 期限そのものはセッションに書かれていて、確定の入口がそれを読む。応答だけを読むと、
// 拒否を書いたうえで保存も昇格も続ける実装を通してしまう。認証要素の有無と、
// セッションが通常のリソースへ届くかどうかの両方を読み直す。
//
//spec:covers EX-AUTHENTICATION-018-03: 登録期限を過ぎた確定が拒否され、認証要素が保存されず、LoginSession が認証完了へ昇格しないことを固定する。
func TestBrowserEnrollmentAfterTheDeadlineSavesNoFactorAndDoesNotPromote(t *testing.T) {
	const deadlineIn = 300 * time.Millisecond
	srv := newTOTPServer(t, totpServerOptions{
		requireMFA: true, enrollment: true, enrollmentDeadlineIn: deadlineIn,
	})
	defer srv.Close()
	client := browserClient(t)

	resp := startAuthorization(t, client, srv.URL+"/realms/default", "verifier-for-enrollment-deadline-12345678901", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction")
	loginResult := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login",
		transaction.CSRFToken, map[string]string{"username": demoUsername, "password": demoPassword})
	if loginResult["next"] != "/realms/default/mfa-enrollment" {
		t.Fatalf("前提が壊れている: login next=%q", loginResult["next"])
	}
	start := postJSON[struct {
		Secret string `json:"secret"`
	}](t, client, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/start",
		transaction.CSRFToken, map[string]string{})

	// ここで期限を跨ぐ。登録を始められたことと、始めた登録を確定できないことの差が
	// この具体例の中身である。
	time.Sleep(deadlineIn + 100*time.Millisecond)

	code, err := totpusecases.GenerateTOTP(start.Secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	refused := postAuthJSON(t, client, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/confirm",
		transaction.CSRFToken, map[string]string{"secret": start.Secret, "code": code})
	defer refused.Body.Close()
	if refused.StatusCode == http.StatusOK {
		t.Fatalf("登録期限を過ぎた確定が成立した: status=%d", refused.StatusCode)
	}

	// 昇格していないので、通常のリソースへは届かない。
	account, err := client.Get(srv.URL + "/realms/default/api/auth/account")
	if err != nil {
		t.Fatalf("GET /api/auth/account: %v", err)
	}
	defer account.Body.Close()
	if account.StatusCode != http.StatusUnauthorized {
		t.Fatalf("拒否されたのにセッションが認証完了へ昇格した: status=%d", account.StatusCode)
	}
	// 認証要素が保存されていれば、同じ利用者は次のログインで第二要素へ進める。
	// 保存されていないことは、登録バイパスの無い新しいログインが拒否されることで読む。
	other := browserClient(t)
	resp = startAuthorization(t, other, srv.URL+"/realms/default", "verifier-for-enrollment-deadline-check-98765", "state")
	resp.Body.Close()
	second := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, other, srv.URL+"/realms/default/api/auth/transaction")
	result := postJSON[map[string]string](t, other, srv.URL+"/realms/default/api/auth/login",
		second.CSRFToken, map[string]string{"username": demoUsername, "password": demoPassword})
	if !strings.Contains(result["redirect_to"], "error=access_denied") {
		t.Fatalf("認証要素が保存されている: redirect=%q", result["redirect_to"])
	}
}

// 登録中の誤った TOTP コードは、認証要素を作らず保留状態も解かない。
//
//spec:covers EX-AUTHENTICATION-018-04: 登録の確定に不正な TOTP コードを出すと InvalidRequestError で拒否され、認証要素が保存されず、LoginSession が Enrollment の保留のままであることを固定する。
func TestBrowserEnrollmentWithAWrongCodeKeepsThePendingSession(t *testing.T) {
	srv := newTOTPServer(t, totpServerOptions{requireMFA: true, enrollment: true})
	defer srv.Close()
	client := browserClient(t)

	resp := startAuthorization(t, client, srv.URL+"/realms/default", "verifier-for-enrollment-wrong-code-1234567", "state")
	resp.Body.Close()
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction")
	postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login",
		transaction.CSRFToken, map[string]string{"username": demoUsername, "password": demoPassword})
	start := postJSON[struct {
		Secret string `json:"secret"`
	}](t, client, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/start",
		transaction.CSRFToken, map[string]string{})

	refused := postAuthJSON(t, client, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/confirm",
		transaction.CSRFToken, map[string]string{"secret": start.Secret, "code": "000000"})
	body, _ := io.ReadAll(refused.Body)
	refused.Body.Close()
	if refused.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s、期待は 400", refused.StatusCode, body)
	}

	// 保留のままなので、通常のリソースへは届かない。
	account, err := client.Get(srv.URL + "/realms/default/api/auth/account")
	if err != nil {
		t.Fatalf("GET /api/auth/account: %v", err)
	}
	account.Body.Close()
	if account.StatusCode != http.StatusUnauthorized {
		t.Fatalf("誤ったコードの拒否後にセッションが昇格した: status=%d", account.StatusCode)
	}

	// 対照: 同じ保留のまま、正しいコードなら確定でき、認可が継続する。認証要素が
	// 作られていなかったことも、ここで作れることで示す。
	code, err := totpusecases.GenerateTOTP(start.Secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	completed := postJSON[map[string]string](t, client,
		srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/confirm",
		transaction.CSRFToken, map[string]string{"secret": start.Secret, "code": code})
	if completed["next"] != "/realms/default/consent" {
		t.Fatalf("前提が壊れている: 正しいコードの確定が next=%q", completed["next"])
	}
}

// postAuthJSON は postJSON と同じ要求を送り、拒否された応答もそのまま返す。
func postAuthJSON(t *testing.T, client *http.Client, endpoint, csrf string, payload any) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(mustJSONBytes(t, payload)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("Origin", "http://test")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", endpoint, err)
	}
	return response
}
