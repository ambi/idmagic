package server_http_test

// 認可の要求が acr_values で MFA の水準を求めたとき、その水準を満たすまで認可を進めない
// (REQ-OAUTH2-005、REQ-AUTHENTICATION-027)。RFC 9470 の再認可はこの規則に頼るので、
// 満たさないまま発行すると、リソースサーバーの拒否と再認可が往復する。

import (
	"crypto/sha256"
	"encoding/base64"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const requestedMFA = "urn:idmagic:acr:mfa"

// startAuthorizationWith は startAuthorization と同じ認可の要求に extra のパラメーターを足す。
func startAuthorizationWith(
	t *testing.T, client *http.Client, baseURL, verifier, state string, extra url.Values,
) *http.Response {
	t.Helper()
	sum := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"client_id":             {demoClientID},
		"redirect_uri":          {demoRedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid profile email offline_access"},
		"state":                 {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(sum[:])},
		"code_challenge_method": {"S256"},
	}
	maps.Copy(query, extra)
	resp, err := client.Get(baseURL + "/authorize?" + query.Encode())
	if err != nil {
		t.Fatalf("GET /authorize: %v", err)
	}
	return resp
}

// loginWithPasswordAfter は extra を付けた認可の要求から、パスワードでログインした結果を返す。
func loginWithPasswordAfter(t *testing.T, client *http.Client, base string, extra url.Values) browserFlow {
	t.Helper()
	resp := startAuthorizationWith(t, client, base, "verifier-for-requested-mfa-test-1234567890", "state", extra)
	resp.Body.Close()
	login := getJSON[authTransaction](t, client, base+"/api/auth/transaction")
	return postJSON[browserFlow](t, client, base+"/api/auth/login", login.CSRFToken, map[string]any{
		"username": demoUsername, "password": demoPassword,
	})
}

//spec:covers REQ-AUTHENTICATION-027: MFA の水準と再認証を同時に求めた認可では、信頼済みデバイスで第二要素を省略せず、通常のログインでは省略を続けることを固定する。
func TestRequestedMfaReauthenticationDoesNotAcceptTheTrustedDevice(t *testing.T) {
	srv := newTOTPServer(t, totpServerOptions{
		totpSecret: totpTestSecret, requireMFA: true, trustedDeviceMaxAgeSeconds: trustedDeviceMaxAge,
	})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)
	signInWithTOTP(t, client, base, true)

	for _, reauthentication := range []url.Values{
		{"max_age": {"0"}},
		{"prompt": {"login"}},
	} {
		// 信頼済みデバイスの Cookie は使うたびに回転するので、直前に通常のログインで省略が
		// 効くことを確かめ、その回転後の Cookie で再認証を求める。
		next, rotated := signInWithPasswordOn(t, nextLoginClient(t, client, base), base)
		if next != "/realms/default/consent" {
			t.Fatalf("通常のログイン next=%q, want 信頼済みデバイスで第二要素を省略して同意へ進む", next)
		}
		client = rotated
		extra := url.Values{"acr_values": {requestedMFA}}
		maps.Copy(extra, reauthentication)
		reauthenticating := nextLoginClient(t, client, base)
		requested := loginWithPasswordAfter(t, reauthenticating, base, extra)
		if requested.Next != "/realms/default/totp" {
			t.Fatalf("%v で next=%q redirect=%q, want 第二要素の画面", reauthentication, requested.Next, requested.RedirectTo)
		}
		// 再認証の要求は記憶済みの端末を照合しないので、Cookie は回転せず次の通常のログインでも使える。
		client = reauthenticating
	}
}

//spec:covers REQ-OAUTH2-005: サインインポリシーが MFA を求めないアプリでも、acr_values が MFA を求めた認可はログインの後に第二要素へ進めることを固定する。
func TestRequestedMfaIsEnforcedAfterLoginWithoutAPolicy(t *testing.T) {
	srv := newTOTPServer(t, totpServerOptions{totpSecret: totpTestSecret})
	defer srv.Close()
	base := srv.URL + "/realms/default"

	next := loginWithPasswordAfter(t, browserClient(t), base, url.Values{"acr_values": {requestedMFA}})
	if next.Next != "/realms/default/totp" {
		t.Fatalf("next=%q redirect=%q, want 第二要素の画面", next.Next, next.RedirectTo)
	}
}

//spec:covers REQ-OAUTH2-005: 既存のセッションに MFA の水準を求めたとき、TOTP 以外の第二要素だけを持つ利用者も第二要素の画面へ進めることを固定する。
func TestRequestedMfaOnAnExistingSessionOffersAnyEnrolledFactor(t *testing.T) {
	srv := newTOTPServer(t, totpServerOptions{recoveryCode: true})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	client := browserClient(t)
	if next := loginWithPasswordAfter(t, client, base, nil); next.Next != "/realms/default/consent" {
		t.Fatalf("最初のログイン next=%q, want 同意", next.Next)
	}

	resp := startAuthorizationWith(t, client, base, "verifier-for-requested-mfa-test-0987654321", "state2",
		url.Values{"acr_values": {requestedMFA}})
	resp.Body.Close()
	if location := resp.Header.Get("Location"); resp.StatusCode != http.StatusSeeOther || location != "/realms/default/totp" {
		t.Fatalf("status=%d location=%q, want 第二要素の画面", resp.StatusCode, location)
	}
}

//spec:covers REQ-OAUTH2-005, OIDC-UNMET-AUTHENTICATION-REQUIREMENTS: 第二要素を一つも持たない利用者に MFA の水準を求めたとき、既存のセッションでもログインの後でも unmet_authentication_requirements を返し、認可コードを発行しないことを固定する。
func TestRequestedMfaWithoutAnyFactorIsUnmet(t *testing.T) {
	srv := newTOTPServer(t, totpServerOptions{})
	defer srv.Close()
	base := srv.URL + "/realms/default"
	extra := url.Values{"acr_values": {requestedMFA}}

	t.Run("ログインの後", func(t *testing.T) {
		next := loginWithPasswordAfter(t, browserClient(t), base, extra)
		assertUnmet(t, next.RedirectTo)
	})
	t.Run("既存のセッション", func(t *testing.T) {
		client := browserClient(t)
		if next := loginWithPasswordAfter(t, client, base, nil); next.Next != "/realms/default/consent" {
			t.Fatalf("最初のログイン next=%q, want 同意", next.Next)
		}
		resp := startAuthorizationWith(t, client, base, "verifier-for-requested-mfa-test-0987654321", "state2", extra)
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("status=%d, want 303", resp.StatusCode)
		}
		assertUnmet(t, resp.Header.Get("Location"))
	})
}

func assertUnmet(t *testing.T, location string) {
	t.Helper()
	parsed, err := url.Parse(location)
	if err != nil || !strings.HasPrefix(location, demoRedirectURI) {
		t.Fatalf("location=%q, want クライアントの redirect_uri", location)
	}
	query := parsed.Query()
	if query.Get("error") != "unmet_authentication_requirements" || query.Get("code") != "" {
		t.Fatalf("location=%q, want error=unmet_authentication_requirements でコードなし", location)
	}
}
