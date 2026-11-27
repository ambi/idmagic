package upstream_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	tokenvaultdomain "github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	tokenvaultports "github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

//spec:covers REQ-TOKENVAULT-010: 内部のアドレスへ接続しないこと、リダイレクトを追わないこと
func TestUpstreamClientRefusesInternalAddressesAndRedirects(t *testing.T) {
	var reached, redirected int
	internal := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached++ }))
	defer internal.Close()

	_, err := New().Send(context.Background(), tokenvaultports.OutboundRequest{Method: "GET", URL: internal.URL, AccessToken: "secret"})
	if !errors.Is(err, tokenvaultports.ErrUpstreamUnreachable) {
		t.Fatalf("err = %v", err)
	}
	if reached != 0 {
		t.Fatal("the hardened client reached a loopback server")
	}

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/next" {
			redirected++
			return
		}
		http.Redirect(w, r, "/next", http.StatusFound)
	}))
	defer target.Close()
	client := &Client{HTTP: withoutRedirects(target.Client())}
	response, err := client.Send(context.Background(), tokenvaultports.OutboundRequest{Method: "GET", URL: target.URL + "/start", AccessToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != http.StatusFound || redirected != 0 {
		t.Fatalf("status %d, redirected %d", response.Status, redirected)
	}
}

//spec:covers REQ-TOKENVAULT-011: 上限を超える応答本文を返さないこと
func TestSendRefusesOversizedResponse(t *testing.T) {
	large := strings.Repeat("x", tokenvaultdomain.MaxResponseBodyBytes+1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(large)) }))
	defer target.Close()
	client := &Client{HTTP: withoutRedirects(target.Client())}
	_, err := client.Send(context.Background(), tokenvaultports.OutboundRequest{Method: "GET", URL: target.URL, AccessToken: "secret"})
	if !errors.Is(err, tokenvaultports.ErrUpstreamResponseTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

//spec:covers REQ-TOKENVAULT-008: 保管したトークンで呼び出し元の Authorization を上書きすること
func TestSendAttachesStoredToken(t *testing.T) {
	var authorization string
	target := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { authorization = r.Header.Get("Authorization") }))
	defer target.Close()
	client := &Client{HTTP: withoutRedirects(target.Client())}
	_, err := client.Send(context.Background(), tokenvaultports.OutboundRequest{
		Method: "GET", URL: target.URL, AccessToken: "stored", Headers: map[string]string{"Authorization": "Bearer caller"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer stored" {
		t.Fatalf("authorization = %q", authorization)
	}
}

// 成功の応答は、更新トークンと有効期間を応答にある分だけ返し、有効期間はドメインの上限で頭打ちにする。
//
//spec:covers REQ-TOKENVAULT-012, REQ-TOKENVAULT-013: invalid_grant と不正な応答を区別して返すこと、応答にない更新トークンと有効期間を空にし、365 日を超える有効期間を 365 日にすること
func TestRefreshParsesTokenEndpointOutcomes(t *testing.T) {
	cases := []struct {
		name, contentType, body string
		status                  int
		wantOAuthError          string
		// want が nil なら、TokenEndpointError を期待する。
		want *tokenvaultports.TokenResponse
	}{
		{
			"success", "application/json", `{"access_token":"a","token_type":"bearer","refresh_token":"r","expires_in":60}`, 200, "",
			&tokenvaultports.TokenResponse{AccessToken: "a", RefreshToken: "r", ExpiresIn: new(time.Minute)},
		},
		{
			"no refresh token and no lifetime", "application/json", `{"access_token":"a","token_type":"Bearer"}`, 200, "",
			&tokenvaultports.TokenResponse{AccessToken: "a"},
		},
		{
			"lifetime over a year", "application/json", `{"access_token":"a","token_type":"Bearer","expires_in":31536001}`, 200, "",
			&tokenvaultports.TokenResponse{AccessToken: "a", ExpiresIn: new(tokenvaultdomain.MaxTokenLifetime)},
		},
		{"invalid_grant", "application/json", `{"error":"invalid_grant"}`, 400, "invalid_grant", nil},
		{"error with 200", "application/json", `{"error":"invalid_grant"}`, 200, "invalid_grant", nil},
		{"html", "text/html", `<html>`, 200, "", nil},
		{"no access token", "application/json", `{"token_type":"bearer"}`, 200, "", nil},
		{"mac token", "application/json", `{"access_token":"a","token_type":"mac"}`, 200, "", nil},
		{"server error", "text/plain", `boom`, 500, "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept") != "application/json" {
					t.Error("Accept is not application/json")
				}
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer target.Close()
			client := &Client{HTTP: withoutRedirects(target.Client())}
			response, err := client.Refresh(context.Background(), tokenvaultports.OAuthClient{TokenEndpoint: target.URL, ClientID: "c", ClientSecret: "s"}, "old")
			if tc.want != nil {
				if err != nil || !reflect.DeepEqual(response, *tc.want) {
					t.Fatalf("response %+v, err %v, want %+v", response, err, *tc.want)
				}
				return
			}
			refused, ok := errors.AsType[*tokenvaultports.TokenEndpointError](err)
			if !ok || refused.OAuthError != tc.wantOAuthError {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestPostAuthSendsCredentialsInForm(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if _, _, basic := r.BasicAuth(); basic || r.PostForm.Get("client_secret") != "s" || r.PostForm.Get("client_id") != "c" {
			t.Errorf("credentials: %v", r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a","token_type":"Bearer"}`))
	}))
	defer target.Close()
	client := &Client{HTTP: withoutRedirects(target.Client())}
	if _, err := client.Refresh(context.Background(), tokenvaultports.OAuthClient{
		TokenEndpoint: target.URL, ClientID: "c", ClientSecret: "s", AuthMethod: tokenvaultdomain.AuthMethodClientSecretPost,
	}, "old"); err != nil {
		t.Fatal(err)
	}
}
