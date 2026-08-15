package server_http_test

// 信頼済みデバイス (remember this device) のログイン経路の検証 (wi-91)。
// REQ-AUTHENTICATION-026 / 027 / 029 に対応する。

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	totpusecases "github.com/ambi/idmagic/backend/authentication/totp/usecases"
)

const (
	trustedDeviceCookieName = "idmagic_trusted_device"
	trustedDeviceMaxAge     = 30 * 24 * 60 * 60
)

type browserFlow struct {
	Next       string `json:"next"`
	RedirectTo string `json:"redirect_to"`
}

type authTransaction struct {
	Kind              string `json:"kind"`
	CSRFToken         string `json:"csrf_token"`
	CanRememberDevice bool   `json:"can_remember_device"`
}

// signInWithTOTP はパスワードと TOTP で 1 回サインインし、rememberDevice を送るかどうかを
// 切り替える。第二要素まで到達しなかった場合は失敗させる。
func signInWithTOTP(
	t *testing.T,
	client *http.Client,
	base string,
	rememberDevice bool,
) authTransaction {
	t.Helper()
	resp := startAuthorization(t, client, base, "verifier-for-trusted-device-test-1234567890", "state")
	resp.Body.Close()
	login := getJSON[authTransaction](t, client, base+"/api/auth/transaction")
	next := postJSON[browserFlow](t, client, base+"/api/auth/login", login.CSRFToken, map[string]any{
		"username": demoUsername, "password": demoPassword,
	})
	if next.Next != "/realms/default/totp" {
		t.Fatalf("login next=%q, want the second-factor step", next.Next)
	}
	second := getJSON[authTransaction](t, client, base+"/api/auth/transaction")
	code, err := totpusecases.GenerateTOTP(totpTestSecret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	done, setCookies := postForCookies(t, client, base+"/api/auth/totp", second.CSRFToken, map[string]any{
		"code": code, "remember_device": rememberDevice,
	})
	if done.Next != "/realms/default/consent" {
		t.Fatalf("totp next=%q, want the consent step", done.Next)
	}
	if issued := findCookie(setCookies, trustedDeviceCookieName); rememberDevice && issued != nil {
		// cookiejar は属性を落とすので、発行時の Set-Cookie でしか確認できない。
		if !issued.HttpOnly || issued.SameSite != http.SameSiteLaxMode || issued.Path == "" {
			t.Fatalf("trusted device cookie attributes = %+v, want HttpOnly / SameSite=Lax / realm path", issued)
		}
	}
	return second
}

// postForCookies は postJSON と同じ POST を行い、レスポンス本文に加えて Set-Cookie を返す。
// cookiejar は属性を保持しないため、cookie の属性はここでしか検証できない。
func postForCookies(
	t *testing.T,
	client *http.Client,
	target, csrf string,
	payload any,
) (browserFlow, []*http.Cookie) {
	t.Helper()
	body := mustJSONBytes(t, payload)
	req, _ := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", csrf)
	req.Header.Set("Origin", "http://test")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s status=%d body=%s", target, resp.StatusCode, raw)
	}
	var result browserFlow
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode %s: %v", target, err)
	}
	return result, resp.Cookies()
}

// signInWithPassword は「同じ端末での次回ログイン」を再現する。セッション cookie を
// 引き継がない新しいブラウザーセッションから、信頼済みデバイス cookie だけを持ち込んで
// パスワードを送り、そこで返る次の遷移先を返す。信頼済みデバイスが効いていれば第二要素の
// 画面を挟まずに同意へ進む。
func signInWithPassword(t *testing.T, client *http.Client, base string) string {
	t.Helper()
	next, _ := signInWithPasswordOn(t, nextLoginClient(t, client, base), base)
	return next
}

// signInWithPasswordOn は指定したブラウザーでパスワードだけのログインを行う。
func signInWithPasswordOn(t *testing.T, client *http.Client, base string) (string, *http.Client) {
	t.Helper()
	resp := startAuthorization(t, client, base, "verifier-for-trusted-device-test-0987654321", "state2")
	resp.Body.Close()
	login := getJSON[authTransaction](t, client, base+"/api/auth/transaction")
	if login.Kind != "login" {
		t.Fatalf("transaction kind=%q, want login", login.Kind)
	}
	next := postJSON[browserFlow](t, client, base+"/api/auth/login", login.CSRFToken, map[string]any{
		"username": demoUsername, "password": demoPassword,
	})
	return next.Next, client
}

// nextLoginClient は現在のブラウザーが持つ信頼済みデバイス cookie だけを引き継いだ、
// 新しいセッションのブラウザーを返す。セッション cookie は引き継がないので、次の
// /authorize は必ずログイン画面から始まる。
func nextLoginClient(t *testing.T, client *http.Client, base string) *http.Client {
	t.Helper()
	value := ""
	if cookie := trustedDeviceCookieOf(t, client, base); cookie != nil {
		value = cookie.Value
	}
	return browserClientWithCookie(t, base, value)
}

func trustedDeviceCookieOf(t *testing.T, client *http.Client, base string) *http.Cookie {
	t.Helper()
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse %s: %v", base, err)
	}
	return findCookie(client.Jar.Cookies(parsed), trustedDeviceCookieName)
}

// REQ-AUTHENTICATION-026: 第二要素の成立時に同意した端末は、次のログインで第二要素を
// 省略でき、そのたびに verifier が回転する。
func TestTrustedDeviceSkipsTheSecondFactorOnTheNextLogin(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	second := signInWithTOTP(t, client, base, true)
	if !second.CanRememberDevice {
		t.Fatal("the second-factor step must advertise that the tenant allows remembering the device")
	}
	issued := trustedDeviceCookieOf(t, client, base)
	if issued == nil || issued.Value == "" {
		t.Fatal("a trusted device cookie must be issued after a genuine second factor")
	}

	next, reused := signInWithPasswordOn(t, nextLoginClient(t, client, base), base)
	if next != "/realms/default/consent" {
		t.Fatalf("next=%q, want the second factor to be skipped", next)
	}
	rotated := trustedDeviceCookieOf(t, reused, base)
	if rotated == nil || rotated.Value == issued.Value {
		t.Fatal("the trusted device cookie must be rotated on every use")
	}
}

// REQ-AUTHENTICATION-026: 同意しなければ記憶しない。
func TestTrustedDeviceIsNotIssuedWithoutConsent(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	signInWithTOTP(t, client, base, false)

	if cookie := trustedDeviceCookieOf(t, client, base); cookie != nil && cookie.Value != "" {
		t.Fatal("a device must not be remembered without the user's explicit consent")
	}
	if next := signInWithPassword(t, client, base); next != "/realms/default/totp" {
		t.Fatalf("next=%q, want the second factor to still be required", next)
	}
}

// REQ-AUTHENTICATION-026: テナントが機能を無効にしていれば、同意しても記憶しない。
func TestTrustedDeviceIsNotIssuedWhenTheTenantDisablesIt(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{totpSecret: secret, requireMFA: true})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	second := signInWithTOTP(t, client, base, true)
	if second.CanRememberDevice {
		t.Fatal("a tenant with the feature disabled must not advertise remembering the device")
	}

	if cookie := trustedDeviceCookieOf(t, client, base); cookie != nil && cookie.Value != "" {
		t.Fatal("a tenant with the feature disabled must never issue a trusted device cookie")
	}
	if next := signInWithPassword(t, client, base); next != "/realms/default/totp" {
		t.Fatalf("next=%q, want the second factor to still be required", next)
	}
}

// REQ-AUTHENTICATION-027: allow_trusted_device=false のアプリは毎回 MFA を要求する。
func TestTrustedDeviceIsIgnoredWhenThePolicyRequiresMfaEveryTime(t *testing.T) {
	secret := totpTestSecret
	deny := false
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
		allowTrustedDevice: &deny,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	signInWithTOTP(t, client, base, true)
	if cookie := trustedDeviceCookieOf(t, client, base); cookie == nil || cookie.Value == "" {
		t.Fatal("the cookie is still issued; only its use as an MFA substitute is refused")
	}

	if next := signInWithPassword(t, client, base); next != "/realms/default/totp" {
		t.Fatalf("next=%q, want the second factor to be required every time", next)
	}
}

// REQ-AUTHENTICATION-027: 別のブラウザーが持ち出した cookie でも、回転後の値でなければ
// 第二要素を省略できない。
func TestTrustedDeviceRejectsTheCookieFromBeforeRotation(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	signInWithTOTP(t, client, base, true)
	stolen := trustedDeviceCookieOf(t, client, base)
	if stolen == nil {
		t.Fatal("no trusted device cookie was issued")
	}
	// 正規の利用者が 1 回使って verifier を回転させる。
	if next := signInWithPassword(t, client, base); next != "/realms/default/consent" {
		t.Fatalf("next=%q, want the legitimate use to succeed", next)
	}
	_ = stolen

	// 盗んだ側は回転前の cookie しか持たない。
	thief := browserClientWithCookie(t, base, stolen.Value)
	if next := signInWithPassword(t, thief, base); next != "/realms/default/totp" {
		t.Fatalf("next=%q, want the pre-rotation cookie to be refused", next)
	}
}

// REQ-AUTHENTICATION-027: 改竄した cookie は第二要素を省略できない。
func TestTrustedDeviceRejectsATamperedCookie(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	signInWithTOTP(t, client, base, true)
	issued := trustedDeviceCookieOf(t, client, base)
	if issued == nil {
		t.Fatal("no trusted device cookie was issued")
	}
	selector, _, _ := splitTrustedDeviceCookie(t, issued.Value)

	forged := browserClientWithCookie(t, base, selector+".forged-verifier")
	if next := signInWithPassword(t, forged, base); next != "/realms/default/totp" {
		t.Fatalf("next=%q, want a tampered verifier to be refused", next)
	}
}

// REQ-AUTHENTICATION-029: 信頼済みデバイスで成立したセッションはステップアップ済みでは
// なく、機微操作には再認証が要る。
func TestTrustedDeviceDoesNotSatisfyStepUp(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	signInWithTOTP(t, client, base, true)
	next, trustedSession := signInWithPasswordOn(t, nextLoginClient(t, client, base), base)
	if next != "/realms/default/consent" {
		t.Fatalf("next=%q, want the second factor to be skipped", next)
	}
	client = trustedSession

	devices := getJSON[struct {
		Devices []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
			Label   string `json:"label"`
		} `json:"devices"`
		MaxAgeSeconds int `json:"max_age_seconds"`
	}](t, client, base+"/api/account/v1/trusted_devices")
	if len(devices.Devices) != 1 || !devices.Devices[0].Current {
		t.Fatalf("trusted devices = %+v, want exactly one marked current", devices.Devices)
	}
	if devices.MaxAgeSeconds != trustedDeviceMaxAge {
		t.Fatalf("max_age_seconds = %d, want %d", devices.MaxAgeSeconds, trustedDeviceMaxAge)
	}
}

// REQ-AUTHENTICATION-028: パスワードを変えると記憶済みの端末はすべて失効し、次回の
// ログインで第二要素が再び要求される。
func TestTrustedDeviceIsRevokedWhenThePasswordChanges(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	second := signInWithTOTP(t, client, base, true)
	if trustedDeviceCookieOf(t, client, base) == nil {
		t.Fatal("no trusted device cookie was issued")
	}
	// ログイン直後なので step-up の recency 窓を満たしている。
	changePassword(t, client, base, second.CSRFToken, demoPassword, "another-strong-password-1")

	if next := signInWithPasswordAs(t, client, base, "another-strong-password-1"); next != "/realms/default/totp" {
		t.Fatalf("next=%q, want the second factor to be required after a password change", next)
	}
}

// REQ-AUTHENTICATION-028: 認証要素を解除すると記憶済みの端末はすべて失効する。
func TestTrustedDeviceIsRevokedWhenTheSecondFactorIsRemoved(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	second := signInWithTOTP(t, client, base, true)
	if trustedDeviceCookieOf(t, client, base) == nil {
		t.Fatal("no trusted device cookie was issued")
	}
	code, err := totpusecases.GenerateTOTP(secret, time.Now().UTC().Unix())
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}
	postNoContent(t, client, base+"/api/account/v1/mfa/totp/remove", second.CSRFToken, map[string]any{"code": code})

	devices := listTrustedDevices(t, client, base)
	if len(devices) != 0 {
		t.Fatalf("trusted devices = %d, want every device revoked after an authenticator change", len(devices))
	}
}

// REQ-AUTHENTICATION-029: 本人はステップアップ再認証のうえで記憶を個別に取り消せる。
func TestTrustedDeviceSelfRevocation(t *testing.T) {
	secret := totpTestSecret
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: secret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)

	second := signInWithTOTP(t, client, base, true)
	devices := listTrustedDevices(t, client, base)
	if len(devices) != 1 {
		t.Fatalf("trusted devices = %d, want 1", len(devices))
	}
	target := base + "/api/account/v1/trusted_devices/" + devices[0].ID + "/revoke"

	postNoContent(t, client, target, second.CSRFToken, map[string]any{})
	// 再送は idempotent に成功する。
	postNoContent(t, client, target, second.CSRFToken, map[string]any{})

	if remaining := listTrustedDevices(t, client, base); len(remaining) != 0 {
		t.Fatalf("trusted devices = %d, want 0 after self-revocation", len(remaining))
	}
	if next := signInWithPassword(t, client, base); next != "/realms/default/totp" {
		t.Fatalf("next=%q, want the second factor after revoking the device", next)
	}
}

type accountTrustedDevice struct {
	ID      string `json:"id"`
	Current bool   `json:"current"`
	Label   string `json:"label"`
}

func listTrustedDevices(t *testing.T, client *http.Client, base string) []accountTrustedDevice {
	t.Helper()
	return getJSON[struct {
		Devices []accountTrustedDevice `json:"devices"`
	}](t, client, base+"/api/account/v1/trusted_devices").Devices
}

// signInWithPasswordAs は指定したパスワードで「次回ログイン」を行う。
func signInWithPasswordAs(t *testing.T, client *http.Client, base, password string) string {
	t.Helper()
	fresh := nextLoginClient(t, client, base)
	resp := startAuthorization(t, fresh, base, "verifier-for-trusted-device-test-0987654321", "state3")
	resp.Body.Close()
	login := getJSON[authTransaction](t, fresh, base+"/api/auth/transaction")
	next := postJSON[browserFlow](t, fresh, base+"/api/auth/login", login.CSRFToken, map[string]any{
		"username": demoUsername, "password": password,
	})
	return next.Next
}

func changePassword(t *testing.T, client *http.Client, base, csrf, current, next string) {
	t.Helper()
	postNoContent(t, client, base+"/api/auth/change_password", csrf, map[string]any{
		"current_password": current, "new_password": next,
	})
}

// postNoContent は 204 を返す self-service 操作を叩く。
func postNoContent(t *testing.T, client *http.Client, target, csrf string, payload any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, target, bytes.NewReader(mustJSONBytes(t, payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", csrf)
	req.Header.Set("Origin", "http://test")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", target, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s status=%d body=%s", target, resp.StatusCode, raw)
	}
}

// browserClientWithCookie は指定した信頼済みデバイス cookie だけを持つ別ブラウザーを作る。
func browserClientWithCookie(t *testing.T, base, value string) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse %s: %v", base, err)
	}
	jar.SetCookies(parsed, []*http.Cookie{{Name: trustedDeviceCookieName, Value: value, Path: "/"}})
	return &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func splitTrustedDeviceCookie(t *testing.T, value string) (selector, verifier string, ok bool) {
	t.Helper()
	for i := range len(value) {
		if value[i] == '.' {
			return value[:i], value[i+1:], true
		}
	}
	t.Fatalf("cookie %q is not selector.verifier", value)
	return "", "", false
}
