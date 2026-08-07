package push_http_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/sharedsignals/push_http"
)

// TestHTTPSecurityEventPusher_Push_Succeeds — RED: 2xx を返す receiver への push は
// エラーを返さず、SSF push-based delivery が期待する Content-Type/body/Authorization
// で送信される。
func TestHTTPSecurityEventPusher_Push_Succeeds(t *testing.T) {
	var gotMethod, gotContentType, gotAuth, gotBody string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	pusher := &push_http.HTTPSecurityEventPusher{Client: server.Client()}
	err := pusher.Push(context.Background(), server.URL, "Bearer secret-token", "compact.jwt.value")
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", gotMethod)
	}
	if gotContentType != "application/secevent+jwt" {
		t.Fatalf("Content-Type = %q, want application/secevent+jwt", gotContentType)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization = %q, want Bearer secret-token", gotAuth)
	}
	if gotBody != "compact.jwt.value" {
		t.Fatalf("body = %q, want compact.jwt.value", gotBody)
	}
}

// TestHTTPSecurityEventPusher_Push_NonSuccessStatusIsError — RED: 2xx 以外の
// レスポンスは error として返る (呼び出し側の retry/backoff/dead-letter 判定の入力)。
func TestHTTPSecurityEventPusher_Push_NonSuccessStatusIsError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	pusher := &push_http.HTTPSecurityEventPusher{Client: server.Client()}
	if err := pusher.Push(context.Background(), server.URL, "", "compact.jwt.value"); err == nil {
		t.Fatal("expected an error for a 500 response, got nil")
	}
}

// TestHTTPSecurityEventPusher_Push_RejectsNonHTTPS — RED: delivery_endpoint が
// https でなければ、レシーバーに到達する前に拒否する。
func TestHTTPSecurityEventPusher_Push_RejectsNonHTTPS(t *testing.T) {
	pusher := &push_http.HTTPSecurityEventPusher{Client: http.DefaultClient}
	if err := pusher.Push(context.Background(), "http://receiver.example/stream", "", "compact.jwt.value"); err == nil {
		t.Fatal("expected an error for a non-https delivery_endpoint, got nil")
	}
}
