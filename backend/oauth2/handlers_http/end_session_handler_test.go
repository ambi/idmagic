package handlers_http_test

// SCL シナリオ "RP-Initiated Logout は登録済み post_logout_redirect_uri にだけリダイレクトする"
// と "未登録 post_logout_redirect_uri は拒否される" を /end_session 経由で検証する。

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

const (
	logoutClientID    = "logout-client"
	logoutRedirectURI = "https://app.example.com/post-logout"
)

func newEndSessionServer(t *testing.T) *echo.Echo {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	clientRepo.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID,
		ClientID: logoutClientID, ClientType: spec.ClientPublic,
		RedirectURIs:             []string{logoutRedirectURI},
		GrantTypes:               []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:            []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:  oauthdomain.AuthMethodNone,
		Scope:                    "openid",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              oauthdomain.FapiNone,
		CreatedAt:                time.Now().UTC(),
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", OAuth2: oauth2.Module{ClientRepo: clientRepo},
	})
	return e
}

//verify:operation EndSession: GET・POST の 302 Location に state を返すことと不正な返送先の 400
//verify:operation PostEndSession: GET・POST の 302 Location に state を返すことと不正な返送先の 400
//spec:covers REQ-OAUTH2-023: GET と POST は要求した state を登録済みの戻り先へ返し、未登録の URI を拒否する。
func TestEndSessionHTTPContracts(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			e := newEndSessionServer(t)
			for _, redirect := range []string{logoutRedirectURI, "https://unregistered.example/cb"} {
				form := url.Values{"client_id": {logoutClientID}, "post_logout_redirect_uri": {redirect}, "state": {"contract-state"}}
				path := "/realms/default/end_session"
				var body io.Reader = http.NoBody
				if method == http.MethodGet {
					path += "?" + form.Encode()
				} else {
					body = strings.NewReader(form.Encode())
				}
				req := httptest.NewRequest(method, path, body)
				if method == http.MethodPost {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
				rec := httptest.NewRecorder()
				e.ServeHTTP(rec, req)
				if redirect != logoutRedirectURI {
					oauthContractError(t, rec, http.StatusBadRequest, "invalid_request")
					if rec.Header().Get("Location") != "" {
						t.Fatal("refused logout redirected")
					}
					continue
				}
				if rec.Code != http.StatusFound || rec.Body.Len() != 0 {
					t.Fatalf("logout status=%d body=%s", rec.Code, rec.Body.String())
				}
				location, err := url.Parse(rec.Header().Get("Location"))
				if err != nil || location.Scheme+"://"+location.Host+location.Path != logoutRedirectURI || location.Query().Get("state") != "contract-state" {
					t.Fatalf("Location=%v err=%v", location, err)
				}
			}
		})
	}
}

//verify:operation PostEndSession: form state の返送、query の優先および空 state の省略
//spec:covers REQ-OAUTH2-023: POST 本文の state を URL 符号化して返し、既存の query 優先を維持する。
func TestEndSessionPOSTStateContracts(t *testing.T) {
	for _, tc := range []struct {
		name, queryState, formState, want string
	}{
		{"form", "", "opaque&state=日本語+#", "opaque&state=日本語+#"},
		{"query takes precedence", "query-state", "form-state", "query-state"},
		{"absent", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEndSessionServer(t)
			form := url.Values{"client_id": {logoutClientID}, "post_logout_redirect_uri": {logoutRedirectURI}, "state": {tc.formState}}
			query := url.Values{"state": {tc.queryState}}
			request := httptest.NewRequest(http.MethodPost, "/realms/default/end_session?"+query.Encode(), strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, request)
			location, err := url.Parse(rec.Header().Get("Location"))
			if err != nil || rec.Code != http.StatusFound || rec.Body.Len() != 0 || location.Scheme+"://"+location.Host+location.Path != logoutRedirectURI || location.Query().Get("state") != tc.want {
				t.Fatalf("status=%d location=%v err=%v want state=%q", rec.Code, location, err, tc.want)
			}
			if tc.want == "" && location.Query().Has("state") {
				t.Fatal("absent state was added to the redirect")
			}
		})
	}
}

//spec:covers OIDC-LOGOUT-ENDPOINT: Discovery が広告する /end_session で RP の要求を受け付けることを固定する。
//spec:covers OIDC-LOGOUT-REDIRECT: 登録済みの完全一致 URI だけへ state を保ったままリダイレクトすることを固定する。
//spec:covers EX-OAUTH2-023-01: 登録済みの post_logout_redirect_uri へのログアウトでは、要求した state がそのリダイレクト先へ伝播する。
func TestEndSessionRedirectsToRegisteredURIWithStatePropagation(t *testing.T) {
	e := newEndSessionServer(t)
	q := url.Values{
		"client_id":                {logoutClientID},
		"post_logout_redirect_uri": {logoutRedirectURI},
		"state":                    {"opaque-state-123"},
	}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/end_session?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	if loc.Scheme+"://"+loc.Host+loc.Path != logoutRedirectURI {
		t.Fatalf("redirect URI mismatch: got %s want %s", loc, logoutRedirectURI)
	}
	if got := loc.Query().Get("state"); got != "opaque-state-123" {
		t.Fatalf("state propagation: got %q, want %q", got, "opaque-state-123")
	}
}

// その URI へのリダイレクトは発生しない。
//
// 拒否の本文を書いたうえで Location も付ける実装は、エラー本文だけを読むテストを
// 通してしまう。ブラウザーが従うのは Location なので、そこまで読み直す。
//
//spec:covers EX-OAUTH2-023-02: 未登録の post_logout_redirect_uri は拒否され、
//spec:covers OIDC-LOGOUT-REDIRECT: 登録されていない URI を応答先として使わないことを固定する。
func TestEndSessionRejectsUnregisteredPostLogoutURI(t *testing.T) {
	e := newEndSessionServer(t)
	const unregistered = "https://evil.example.com/cb"
	q := url.Values{
		"client_id":                {logoutClientID},
		"post_logout_redirect_uri": {unregistered},
	}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/end_session?"+q.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code == http.StatusFound ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"error":"invalid_request"`)) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if location := rec.Header().Get("Location"); location != "" {
		t.Fatalf("拒否された要求が Location=%q を返した", location)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(unregistered)) {
		t.Fatalf("拒否の本文が未登録 URI を反射している: %s", rec.Body.String())
	}
}
