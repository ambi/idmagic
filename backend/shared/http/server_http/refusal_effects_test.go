package server_http_test

// ブラウザーのログイン経路が宣言する拒否について、応答と「拒否が変えなかった状態」の
// 両方を確かめる。
//
// この経路の状態は LoginSession に載る。拒否の応答だけを読むテストは、その拒否が
// セッションを昇格させてしまう実装も、逆にセッションを壊してしまう実装も見分けられない。
// どちらも呼び出し元には同じ 401 に見える。

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
)

// postRefused は 2xx を期待しない POST を送る。postJSON は非 200 で失敗するので、
// 拒否そのものを読むテストはこちらを使う。
func postRefused(
	t *testing.T,
	client *http.Client,
	target, csrf string,
	payload any,
) (int, string) {
	t.Helper()
	request, _ := http.NewRequest(http.MethodPost, target, bytes.NewReader(mustJSONBytes(t, payload)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Csrf-Token", csrf)
	request.Header.Set("Origin", "http://test")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(body)
}

// sessionCookieValue は cookie jar が保持しているログインセッションの値を返す。
// 拒否がセッションそのものを差し替えていないことを読むために使う。
func sessionCookieValue(t *testing.T, client *http.Client, base string) string {
	t.Helper()
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	cookie := findCookie(client.Jar.Cookies(parsed), sessionusecases.SessionCookie)
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

// `authentication_pending` のまま残る。
//
// 「拒否を書いたうえで第二要素も成立させる」実装は、次画面の名前を読むテストを
// 素通りする。そこで拒否のあとに、同じセッションがまだ第二要素を待っていること、
// 通常のリソースへ到達できないこと、そして正しいコードで同じセッションが継続する
// ことの 3 つを読む。逆に「拒否のついでにセッションを捨てる」実装も 3 つ目で落ちる。
//
//spec:covers EX-AUTHENTICATION-017-02: 誤った TOTP コードの送信は拒否され、LoginSession は
func TestWrongTotpCodeLeavesTheLoginSessionPending(t *testing.T) {
	secret := totpTestSecret
	srv := newServerWithTOTPPolicy(t, secret, true)
	defer srv.Close()
	client := browserClient(t)

	response := startAuthorization(t, client, srv.URL+"/realms/default",
		"verifier-for-wrong-totp-test-123456789012345678", "state")
	response.Body.Close()
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction")
	loginResult := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login",
		transaction.CSRFToken, map[string]string{"username": demoUsername, "password": demoPassword})
	if loginResult["next"] != "/realms/default/totp" {
		t.Fatalf("前提が壊れている: login next=%q", loginResult["next"])
	}
	totpTransaction := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction")
	if totpTransaction.Kind != "totp" {
		t.Fatalf("前提が壊れている: kind=%q", totpTransaction.Kind)
	}
	pendingSession := sessionCookieValue(t, client, srv.URL+"/realms/default/")
	if pendingSession == "" {
		t.Fatal("前提が壊れている: 第二要素待ちのセッション Cookie が無い")
	}

	status, body := postRefused(t, client, srv.URL+"/realms/default/api/auth/totp",
		totpTransaction.CSRFToken, map[string]string{"code": "000000"})
	if status == http.StatusOK {
		t.Fatalf("誤ったコードが受理された: body=%s", body)
	}

	// 拒否が変えなかったもの その 1: セッションはまだ第二要素を待っている。
	stillPending := getJSON[struct {
		Kind string `json:"kind"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction")
	if stillPending.Kind != "totp" {
		t.Fatalf("拒否のあとに kind=%q になった、期待は totp のまま", stillPending.Kind)
	}
	if got := sessionCookieValue(t, client, srv.URL+"/realms/default/"); got != pendingSession {
		t.Fatalf("拒否がセッションを差し替えた: %q -> %q", pendingSession, got)
	}

	// その 2: 認証途中のままなので、通常のリソースへは到達できない。
	// これが無いと「拒否を返しつつ昇格させる」実装を kind だけでは見分けられない。
	sessionsResponse, err := client.Get(srv.URL + "/realms/default/api/account/v1/sessions")
	if err != nil {
		t.Fatal(err)
	}
	sessionsBody, _ := io.ReadAll(sessionsResponse.Body)
	sessionsResponse.Body.Close()
	if sessionsResponse.StatusCode == http.StatusOK {
		t.Fatalf("誤ったコードの拒否後に保護リソースが返った: body=%s", sessionsBody)
	}

	// その 3: 同じセッションで正しいコードを送れば認証は継続する。
	// 拒否がセッションを消費も失効もしていない証拠になる。
	code, err := totpusecases.GenerateTOTP(secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	accepted := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/totp",
		totpTransaction.CSRFToken, map[string]string{"code": code})
	if accepted["next"] != "/realms/default/consent" {
		t.Fatalf("前提が壊れている: 正しいコードの next=%q", accepted["next"])
	}
	if got := sessionCookieValue(t, client, srv.URL+"/realms/default/"); got != pendingSession {
		t.Fatalf("同じセッションで継続しなかった: %q -> %q", pendingSession, got)
	}
}
