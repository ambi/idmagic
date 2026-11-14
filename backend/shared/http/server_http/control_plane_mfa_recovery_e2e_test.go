package server_http_test

// MFA の端末と復旧コードを失った制御面の運用者が、デフォルトのテナントの管理者による認証器の
// リセットから再登録して、MFA で成立したセッションへ戻れることを確かめる
// (REQ-AUTHENTICATION-022)。システムコンソールが MFA を求める前に、締め出された運用者の
// 復旧の経路が成立することを固定する。

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
)

//spec:covers REQ-AUTHENTICATION-022: system_admin だけを持つ運用者の認証器を別の管理者が全リセットすると、運用者は次のログインで再登録し、system_admin のまま MFA で成立したセッションを得ることを固定する。
func TestControlPlaneOperatorRecoversMfaThroughAnotherAdmin(t *testing.T) {
	srv, _ := newServerForAuthenticatorResetWithEvents(t, resetServerOptions{aliceRoles: []string{"system_admin"}})
	defer srv.Close()

	adminClient := loginDirectAdmin(t, srv, resetAdminUsername, resetAdminPassword, resetAdminTOTP)
	resetAllAuthenticators(t, srv, adminClient, "user_alice")

	operator := reenrollAfterAuthenticatorReset(t, srv)
	assertRecoveredControlPlaneOperator(t, srv, operator)
}

//spec:covers REQ-AUTHENTICATION-022, REQ-OAUTH2-005: admin と system_admin を持つ運用者は、デフォルトのポリシーが MFA を求めない間、パスワードで管理画面へ入って自分の認証器を全リセットし、アカウントの画面で新しい TOTP を登録し、MFA の水準を求める再認可でその TOTP を使って MFA で成立したセッションを得ることを固定する。
func TestControlPlaneOperatorRecoversMfaBySelfResetWithoutDefaultMfa(t *testing.T) {
	srv, _ := newServerForAuthenticatorResetWithEvents(t, resetServerOptions{
		aliceRoles: []string{"admin", "system_admin"}, noDefaultMFA: true,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"

	// 端末と復旧コードを失っていても、MFA を求めないポリシーではパスワードで管理画面へ入れる。
	self := loginDirectAdmin(t, srv, demoUsername, demoPassword, "")
	resetAllAuthenticators(t, srv, self, "user_alice")

	// MFA を求めないポリシーでは、リセットの後もログインは再登録を求めない。運用者はログインした
	// ばかりのセッションで、アカウントの画面から新しい TOTP を登録する。
	csrf := adminCSRFToken(t, self, srv)
	enrollment := postJSON[struct {
		Secret string `json:"secret"`
	}](t, self, base+"/api/account/v1/mfa/totp/enroll/start", csrf, map[string]string{})
	code, err := authentication.GenerateTOTP(enrollment.Secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	postNoContent(t, self, base+"/api/account/v1/mfa/totp/enroll/confirm", csrf, map[string]any{
		"secret": enrollment.Secret, "code": code,
	})

	// システムコンソールの再認可と同じく、MFA の水準を求める認可で第二要素を求められる。
	resp := startAuthorizationWith(t, self, base, "verifier-for-control-plane-recovery-123456", "state",
		map[string][]string{"acr_values": {"urn:idmagic:acr:mfa"}})
	resp.Body.Close()
	if location := resp.Header.Get("Location"); location != "/realms/default/totp" {
		t.Fatalf("status=%d location=%q, want 第二要素の画面", resp.StatusCode, location)
	}
	second := getJSON[authTransaction](t, self, base+"/api/auth/transaction")
	code, err = authentication.GenerateTOTP(enrollment.Secret, time.Now().UTC().Add(30*time.Second).Unix())
	if err != nil {
		t.Fatal(err)
	}
	postJSON[browserFlow](t, self, base+"/api/auth/totp", second.CSRFToken, map[string]any{"code": code})
	assertRecoveredControlPlaneOperator(t, srv, self)
}

func resetAllAuthenticators(t *testing.T, srv *httptest.Server, actor *http.Client, sub string) {
	t.Helper()
	result := postJSON[map[string]any](
		t, actor, srv.URL+"/realms/default/api/admin/v1/users/"+sub+"/authenticator-reset", adminCSRFToken(t, actor, srv),
		map[string]any{"targets": []string{"totp", "webauthn", "recovery_code"}},
	)
	if required, _ := result["reenrollment_required"].(bool); !required {
		t.Fatalf("reset result=%+v, want reenrollment_required=true", result)
	}
}

// reenrollAfterAuthenticatorReset は、リセットされた alice がパスワードでログインし、再登録の
// 保留から新しい TOTP を登録して管理画面へ戻るまでを行い、そのブラウザーを返す。
func reenrollAfterAuthenticatorReset(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()
	client := browserClient(t)
	returnTo := "/realms/default/admin"
	transaction := getJSON[struct {
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	login := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/login", transaction.CSRFToken, map[string]string{
		"username": demoUsername, "password": demoPassword, "return_to": returnTo,
	})
	if login["next"] == "" || login["redirect_to"] != "" {
		t.Fatalf("login result=%+v, want the pending re-enrollment", login)
	}
	enrollment := getJSON[struct {
		Kind      string `json:"kind"`
		CSRFToken string `json:"csrf_token"`
	}](t, client, srv.URL+"/realms/default/api/auth/transaction?return_to="+returnTo)
	if enrollment.Kind != "mfa_enrollment" {
		t.Fatalf("kind=%q, want mfa_enrollment", enrollment.Kind)
	}
	start := postJSON[struct {
		Secret string `json:"secret"`
	}](t, client, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/start", enrollment.CSRFToken, map[string]string{})
	code, err := authentication.GenerateTOTP(start.Secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatal(err)
	}
	confirmed := postJSON[map[string]string](t, client, srv.URL+"/realms/default/api/auth/mfa/enrollment/totp/confirm", enrollment.CSRFToken, map[string]string{
		"secret": start.Secret, "code": code, "return_to": returnTo,
	})
	if confirmed["redirect_to"] != returnTo {
		t.Fatalf("post-enrollment redirect_to=%q, want %q", confirmed["redirect_to"], returnTo)
	}
	return client
}

// assertRecoveredControlPlaneOperator は、再登録した運用者のセッションが実際の第二要素で MFA を
// 満たし、system_admin のロールを保っていることを確かめる。システムコンソールの判定は、この
// amr と所属とロールを読む。
func assertRecoveredControlPlaneOperator(t *testing.T, srv *httptest.Server, operator *http.Client) {
	t.Helper()
	base := srv.URL + "/realms/default"
	session := currentSessionOf(t, operator, base)
	if !slices.Contains(session.AMR, "otp") || session.ACR != "urn:idmagic:acr:mfa" {
		t.Fatalf("amr=%v acr=%q, want otp と mfa", session.AMR, session.ACR)
	}
	account := getJSON[struct {
		Roles []string `json:"roles"`
	}](t, operator, base+"/api/auth/account")
	if !slices.Contains(account.Roles, "system_admin") {
		t.Fatalf("roles=%v, want system_admin を保つ", account.Roles)
	}
}
