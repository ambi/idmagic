package handlers_http_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// 小文字の方式名で、共通のクライアント認証を通る 4 つの入口へ送る。各入口の成功の
// 結果は認証を通った後にしか出ないので、401 の invalid_client との差が照合の観測点に
// なる。/token は未知の認可コードを送り、認証を通れば invalid_grant になる。
//
//spec:covers RFC9110-AUTH-SCHEME-CASE-INSENSITIVE: 小文字の basic を /token、/par、/introspect、/revoke が client_secret_basic として認証し、誤った秘密は 401 の invalid_client になる。
func TestClientSecretBasicAcceptsTheSchemeNameInAnyCase(t *testing.T) {
	s := stack.New(t, stack.WithBrowserFlow())
	post := func(path, scheme, secret string, form url.Values) (int, map[string]any) {
		t.Helper()
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
			stack.Issuer+"/realms/"+tenancydomain.DefaultRealm+path, strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		credentials := base64.StdEncoding.EncodeToString([]byte(stack.BrowserClientID + ":" + secret))
		request.Header.Set("Authorization", scheme+" "+credentials)
		recorder := httptest.NewRecorder()
		s.Echo.ServeHTTP(recorder, request)
		body := map[string]any{}
		_ = json.Unmarshal(recorder.Body.Bytes(), &body)
		return recorder.Code, body
	}

	cases := []struct {
		path       string
		form       url.Values
		wantStatus int
		wantError  string
	}{
		{"/token", url.Values{
			"grant_type": {"authorization_code"}, "code": {"unknown-code"},
			"code_verifier": {codeVerifier}, "redirect_uri": {stack.BrowserRedirectURI},
		}, http.StatusBadRequest, "invalid_grant"},
		{"/par", stack.AuthorizationQuery(codeVerifier, nil), http.StatusCreated, ""},
		{"/introspect", url.Values{"token": {"unknown-token"}}, http.StatusOK, ""},
		{"/revoke", url.Values{"token": {"unknown-token"}}, http.StatusOK, ""},
	}
	for _, scheme := range []string{"basic", "BASIC"} {
		for _, tc := range cases {
			status, body := post(tc.path, scheme, stack.BrowserClientSecret, tc.form)
			if status != tc.wantStatus || (tc.wantError != "" && body["error"] != tc.wantError) {
				t.Errorf("%s %s: status=%d body=%v, want status=%d error=%q",
					scheme, tc.path, status, body, tc.wantStatus, tc.wantError)
			}
		}
	}

	// 方式名を受け入れても、秘密は検査する。
	for _, tc := range cases {
		status, body := post(tc.path, "basic", "wrong-secret", tc.form)
		if status != http.StatusUnauthorized || body["error"] != "invalid_client" {
			t.Errorf("誤った秘密 %s: status=%d body=%v, want 401 invalid_client", tc.path, status, body)
		}
	}
}
