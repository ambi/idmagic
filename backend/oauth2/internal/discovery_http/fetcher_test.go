package discovery_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//spec:covers REQ-OAUTH2-095: 取得は非公開アドレスへ接続せず、リダイレクトを追わず、上限を超えた本文を失敗にする。
func TestFetcherRejectsUnsafeDestinationsAndResponses(t *testing.T) {
	redirected := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/target", http.StatusFound)
		case "/target":
			redirected = true
		case "/large":
			_, _ = w.Write([]byte(strings.Repeat("x", 65)))
		case "/exact":
			_, _ = w.Write([]byte(strings.Repeat("x", 64)))
		}
	}))
	defer server.Close()
	ctx := context.Background()

	production := NewFetcher(nil)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/exact", http.NoBody)
	// 証明書の検証より前に、接続の時点で宛先のアドレスを拒否することを確かめる。
	if _, _, err := production.Do(request, 64); err == nil || !strings.Contains(err.Error(), "non-public address") {
		t.Fatalf("loopback destination: err=%v; want a refusal of the non-public address", err)
	}

	fetcher := NewFetcher(server.Client())
	request, _ = http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/redirect", http.NoBody)
	status, _, err := fetcher.Do(request, 64)
	if err != nil || status != http.StatusFound || redirected {
		t.Fatalf("redirect: status=%d err=%v followed=%v", status, err, redirected)
	}

	request, _ = http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/large", http.NoBody)
	if _, _, err := fetcher.Do(request, 64); !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("oversized body: err=%v", err)
	}
	request, _ = http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/exact", http.NoBody)
	if status, body, err := fetcher.Do(request, 64); err != nil || status != http.StatusOK || len(body) != 64 {
		t.Fatalf("body at the limit: status=%d len=%d err=%v", status, len(body), err)
	}
}
