package upstream_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

//spec:covers REQ-TOKENVAULT-010: 内部のアドレスへ接続しないこと、リダイレクトを追わないこと
func TestUpstreamClientRefusesInternalAddressesAndRedirects(t *testing.T) {
	var reached, redirected int
	internal := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached++ }))
	defer internal.Close()

	_, err := New().Send(context.Background(), ports.OutboundRequest{Method: "GET", URL: internal.URL, AccessToken: "secret"})
	if !errors.Is(err, ports.ErrUpstreamUnreachable) {
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
	response, err := client.Send(context.Background(), ports.OutboundRequest{Method: "GET", URL: target.URL + "/start", AccessToken: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != http.StatusFound || redirected != 0 {
		t.Fatalf("status %d, redirected %d", response.Status, redirected)
	}
}

//spec:covers REQ-TOKENVAULT-011: 上限を超える応答本文を返さないこと
func TestSendRefusesOversizedResponse(t *testing.T) {
	large := strings.Repeat("x", domain.MaxResponseBodyBytes+1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(large)) }))
	defer target.Close()
	client := &Client{HTTP: withoutRedirects(target.Client())}
	_, err := client.Send(context.Background(), ports.OutboundRequest{Method: "GET", URL: target.URL, AccessToken: "secret"})
	if !errors.Is(err, ports.ErrUpstreamResponseTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

//spec:covers REQ-TOKENVAULT-008: 保管したトークンで呼び出し元の Authorization を上書きすること
func TestSendAttachesStoredToken(t *testing.T) {
	var authorization string
	target := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { authorization = r.Header.Get("Authorization") }))
	defer target.Close()
	client := &Client{HTTP: withoutRedirects(target.Client())}
	_, err := client.Send(context.Background(), ports.OutboundRequest{
		Method: "GET", URL: target.URL, AccessToken: "stored", Headers: map[string]string{"Authorization": "Bearer caller"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer stored" {
		t.Fatalf("authorization = %q", authorization)
	}
}

//spec:covers REQ-TOKENVAULT-013: invalid_grant と不正な応答を区別して返すこと
func TestRefreshParsesTokenEndpointOutcomes(t *testing.T) {
	cases := []struct {
		name, contentType, body string
		status                  int
		wantOAuthError          string
		wantErr                 bool
	}{
		{"success", "application/json", `{"access_token":"a","token_type":"bearer","refresh_token":"r","expires_in":60}`, 200, "", false},
		{"invalid_grant", "application/json", `{"error":"invalid_grant"}`, 400, "invalid_grant", true},
		{"error with 200", "application/json", `{"error":"invalid_grant"}`, 200, "invalid_grant", true},
		{"html", "text/html", `<html>`, 200, "", true},
		{"no access token", "application/json", `{"token_type":"bearer"}`, 200, "", true},
		{"mac token", "application/json", `{"access_token":"a","token_type":"mac"}`, 200, "", true},
		{"server error", "text/plain", `boom`, 500, "", true},
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
			response, err := client.Refresh(context.Background(), ports.OAuthClient{TokenEndpoint: target.URL, ClientID: "c", ClientSecret: "s"}, "old")
			if !tc.wantErr {
				if err != nil || response.AccessToken != "a" || response.RefreshToken != "r" || response.ExpiresIn == nil {
					t.Fatalf("response %+v, err %v", response, err)
				}
				return
			}
			refused, ok := errors.AsType[*ports.TokenEndpointError](err)
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
	if _, err := client.Refresh(context.Background(), ports.OAuthClient{
		TokenEndpoint: target.URL, ClientID: "c", ClientSecret: "s", AuthMethod: domain.AuthMethodClientSecretPost,
	}, "old"); err != nil {
		t.Fatal(err)
	}
}
