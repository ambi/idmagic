package push_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientDeliverPostsLogoutToken(t *testing.T) {
	var received string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.FormValue("logout_token")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err := NewClient(server.Client()).Deliver(context.Background(), server.URL, "signed-token"); err != nil {
		t.Fatal(err)
	}
	if received != "signed-token" {
		t.Fatalf("logout_token=%q", received)
	}
}

func TestClientDeliverRejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer server.Close()
	if err := NewClient(server.Client()).Deliver(context.Background(), server.URL, "signed-token"); err == nil {
		t.Fatal("expected non-success status error")
	}
}
