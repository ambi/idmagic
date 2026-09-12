package handlers_http_test

// 主要ユースケース追跡: REQ-AUTHENTICATION-016。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	"github.com/ambi/idmagic/backend/shared/policy/breaches_noop"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"

	"github.com/labstack/echo/v5"
)

//spec:covers EX-AUTHENTICATION-016-01: 通常経路、EX-AUTHENTICATION-016-04 確定済みのトークンは再利用できない。
func TestPasswordResetHTTPFlow(t *testing.T) {
	e, userRepo, sender, hasher := newPasswordResetHandler(t)
	csrf, cookie := passwordResetCSRF(t, e)

	forgot := serveJSON(t, e, "/api/auth/forgot_password", csrf, cookie, map[string]string{
		"email": "alice@example.com",
	})
	if forgot.Code != http.StatusNoContent {
		t.Fatalf("forgot status=%d body=%s", forgot.Code, forgot.Body.String())
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("sent emails=%d, want 1", len(sender.Sent))
	}
	token := resetTokenFromEmail(t, sender.Sent[0].Text)

	reset := serveJSON(t, e, "/api/auth/reset_password", csrf, cookie, map[string]string{
		"token": token, "new_password": "fresh-password-9182",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("reset status=%d body=%s", reset.Code, reset.Body.String())
	}
	user, err := userRepo.FindBySub(context.Background(), "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	matched, err := hasher.Verify("fresh-password-9182", user.PasswordHash)
	if err != nil || !matched {
		t.Fatalf("new password matched=%v err=%v", matched, err)
	}

	replay := serveJSON(t, e, "/api/auth/reset_password", csrf, cookie, map[string]string{
		"token": token, "new_password": "another-password-9182",
	})
	if replay.Code != http.StatusGone {
		t.Fatalf("replay status=%d body=%s", replay.Code, replay.Body.String())
	}
	// 拒否したのだから、二度目のパスワードは設定されていない。応答だけを読む検査は、
	// 拒否を書いた後で作用を行う実装にも同じように通ってしまう。
	after, err := userRepo.FindBySub(context.Background(), "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := hasher.Verify("another-password-9182", after.PasswordHash)
	if err != nil {
		t.Fatal(err)
	}
	if replayed {
		t.Fatal("the refused replay changed the password")
	}
}

// ブラウザーとメールスキャナーが行うのはリンクの GET と HEAD である。作用を起こすのは
// POST だけなので、先読みの後も同じトークンで更新できなければならない。
//
//spec:covers EX-AUTHENTICATION-016-03: リンクを開くだけではトークンを消費しない。
func TestPasswordResetPrefetchDoesNotConsumeTheToken(t *testing.T) {
	e, userRepo, sender, hasher := newPasswordResetHandler(t)
	csrf, cookie := passwordResetCSRF(t, e)

	forgot := serveJSON(t, e, "/api/auth/forgot_password", csrf, cookie, map[string]string{
		"email": "alice@example.com",
	})
	if forgot.Code != http.StatusNoContent {
		t.Fatalf("forgot status=%d body=%s", forgot.Code, forgot.Body.String())
	}
	token := resetTokenFromEmail(t, sender.Sent[0].Text)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		request := httptest.NewRequest(
			method, defaultRealmPath("/api/auth/reset_password")+"?token="+url.QueryEscape(token), http.NoBody,
		)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		if response.Code == http.StatusOK {
			t.Fatalf("%s on the reset endpoint answered 200; it must not be a consuming route", method)
		}
	}

	before, err := userRepo.FindBySub(context.Background(), "user-alice")
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := hasher.Verify("current-password-1", before.PasswordHash)
	if err != nil || !unchanged {
		t.Fatalf("prefetching changed the password: matched=%v err=%v", unchanged, err)
	}

	// 先読みの後でも同じトークンで更新できる。トークンが読み取りで消えていないことの証拠。
	reset := serveJSON(t, e, "/api/auth/reset_password", csrf, cookie, map[string]string{
		"token": token, "new_password": "fresh-password-9182",
	})
	if reset.Code != http.StatusOK {
		t.Fatalf("reset after prefetch status=%d body=%s", reset.Code, reset.Body.String())
	}
}

func TestForgotPasswordHTTPDoesNotRevealUnknownEmail(t *testing.T) {
	e, _, sender, _ := newPasswordResetHandler(t)
	csrf, cookie := passwordResetCSRF(t, e)
	response := serveJSON(t, e, "/api/auth/forgot_password", csrf, cookie, map[string]string{
		"email": "unknown@example.com",
	})
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if len(sender.Sent) != 0 {
		t.Fatalf("sent emails=%d, want 0", len(sender.Sent))
	}
}

func newPasswordResetHandler(
	t *testing.T,
) (*echo.Echo, *usermemory.UserRepository, *email_memory.NoopEmailSender, *passwords_argon2id.Argon2idPasswordHasher) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	historyRepo := passwordmemory.NewPasswordHistoryRepository()
	tokenStore := passwordmemory.NewPasswordResetTokenStore(userRepo, historyRepo)
	sender := &email_memory.NoopEmailSender{}
	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash("current-password-1")
	if err != nil {
		t.Fatal(err)
	}
	email := "alice@example.com"
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "user-alice", PreferredUsername: "alice", PasswordHash: hash,
		Email: &email, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	if err := historyRepo.Add(context.Background(), "user-alice", hash, now); err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", UserRepo: userRepo, PasswordHasher: hasher,
		PasswordHistoryRepo: historyRepo, Authentication: authentication.Module{PasswordResetTokenStore: tokenStore},
		EmailSender: sender, BreachedPasswordChecker: breaches_noop.NoopBreachedPasswordChecker{},
	})
	return e, userRepo, sender, hasher
}

func passwordResetCSRF(t *testing.T, e *echo.Echo) (string, *http.Cookie) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/password_reset_context", http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("context status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	result := response.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) != 1 || body.CSRFToken == "" {
		t.Fatalf("csrf=%q cookies=%v", body.CSRFToken, cookies)
	}
	return body.CSRFToken, cookies[0]
}

func serveJSON(
	t *testing.T,
	e *echo.Echo,
	path, csrf string,
	cookie *http.Cookie,
	body any,
) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, defaultRealmPath(path), bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func resetTokenFromEmail(t *testing.T, message string) string {
	t.Helper()
	start := strings.Index(message, "http://")
	if start < 0 {
		t.Fatalf("reset URL missing from email: %q", message)
	}
	end := strings.IndexByte(message[start:], '\n')
	rawURL := message[start:]
	if end >= 0 {
		rawURL = message[start : start+end]
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	token := parsed.Query().Get("token")
	if token == "" {
		t.Fatal("reset token missing")
	}
	return token
}

// defaultRealmPath は bare path を default テナントの正規ロケーション配下へ移す。
// bare path はどのテナントの正規ロケーションでもなくなったため、
// テストのリクエスト先も /realms/default 配下でなければ 404 になる。
func defaultRealmPath(path string) string {
	if strings.HasPrefix(path, "/realms/") {
		return path
	}
	return "/realms/default" + path
}
