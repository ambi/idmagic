package cimd_http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTLSDocumentServer starts an httptest TLS server (client_id must be
// https) whose handler renders body with __SELF__ replaced by the server's
// own URL, and returns a Fetcher wired to the server's trusting client.
func newTLSDocumentServer(t *testing.T, body string) (*httptest.Server, *Fetcher) {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.ReplaceAll(body, "__SELF__", srv.URL)))
	}))
	t.Cleanup(srv.Close)
	return srv, newFetcherWithClient(srv.Client())
}

func TestFetcher_FetchReturnsSynthesizedClientOnSuccess(t *testing.T) {
	srv, f := newTLSDocumentServer(t, `{
		"client_id": "__SELF__/client.json",
		"client_name": "Example Client",
		"redirect_uris": ["http://127.0.0.1:3000/callback"]
	}`)
	clientIDURL := srv.URL + "/client.json"
	client, err := f.Fetch(t.Context(), clientIDURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.ClientID != clientIDURL {
		t.Errorf("ClientID = %q, want %q", client.ClientID, clientIDURL)
	}
}

func TestFetcher_FetchRejectsNon200Status(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	f := newFetcherWithClient(srv.Client())
	if _, err := f.Fetch(t.Context(), srv.URL+"/client.json"); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestFetcher_FetchRejectsOversizedResponse(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"client_id":"` + strings.Repeat("a", maxDocumentBytes+1) + `"}`))
	}))
	t.Cleanup(srv.Close)
	f := newFetcherWithClient(srv.Client())
	if _, err := f.Fetch(t.Context(), srv.URL+"/client.json"); err == nil {
		t.Fatal("expected error for oversized response")
	}
}

func TestFetcher_FetchCachesSuccessfulResolution(t *testing.T) {
	hits := 0
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"client_id":"%s/client.json","client_name":"Example","redirect_uris":["http://127.0.0.1:3000/cb"]}`, srv.URL)
	}))
	t.Cleanup(srv.Close)
	f := newFetcherWithClient(srv.Client())
	url := srv.URL + "/client.json"
	if _, err := f.Fetch(t.Context(), url); err != nil {
		t.Fatalf("unexpected error on first fetch: %v", err)
	}
	if _, err := f.Fetch(t.Context(), url); err != nil {
		t.Fatalf("unexpected error on second fetch: %v", err)
	}
	if hits != 1 {
		t.Errorf("server hit %d times, want 1 (second Fetch should hit cache)", hits)
	}
}

func TestFetcher_FetchRejectsUnsafeURLShapeWithoutNetworkCall(t *testing.T) {
	f := newFetcherWithClient(http.DefaultClient)
	if _, err := f.Fetch(t.Context(), "http://insecure.example.com/client.json"); err == nil {
		t.Fatal("expected error for non-https client_id URL")
	}
}

func TestFetcher_RealFetcherRejectsLoopbackViaSSRFDialer(t *testing.T) {
	// NewFetcher (not newFetcherWithClient) wires the SSRF-safe dialer; a
	// loopback target must be rejected before any request is attempted.
	f := NewFetcher()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if _, err := f.Fetch(ctx, "https://127.0.0.1:1/client.json"); err == nil {
		t.Fatal("expected error for loopback target via SSRF-safe dialer")
	}
}
