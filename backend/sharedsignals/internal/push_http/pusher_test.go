package push_http_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
	"github.com/ambi/idmagic/backend/sharedsignals/internal/push_http"
)

// rfc8935Recording は、RFC 8935 の 2.1 節と 2.2 節の例を転記した記録を読む。
func rfc8935Recording(t *testing.T) testing_httprecord.Recording {
	t.Helper()
	data, err := os.ReadFile("testdata/rfc8935.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatalf("decode RFC 8935 recording: %v", err)
	}
	return rec
}

// routedTo は、https の配送先への要求を、再生サーバーの http の URL へ向ける。
// pusher は https の配送先しか受けないが、再生サーバーは TLS を使わないためである。
func routedTo(server *httptest.Server) *http.Client {
	target, _ := url.Parse(server.URL)
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		routed := r.Clone(r.Context())
		routed.URL.Scheme, routed.URL.Host = target.Scheme, target.Host
		return http.DefaultTransport.RoundTrip(routed)
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// 受信者は Figure 1 の要求を受け取り、Figure 2 の 202 の空の応答で配送が成功する。
//
//spec:covers RFC8935-PUSH-REQUEST: SET の compact serialization だけを本文にして、Content-Type: application/secevent+jwt と Accept: application/json の POST で送り、202 の応答を成功とする。
func TestHTTPSecurityEventPusher_PushMatchesRFC8935Example(t *testing.T) {
	ex := rfc8935Recording(t).Exchange(t, "push-accepted")
	server := testing_httprecord.Replay{
		CheckBody: func(recorded, got []byte) []string {
			if !bytes.Equal(recorded, got) {
				return []string{fmt.Sprintf("body = %q, recorded %q", got, recorded)}
			}
			return nil
		},
	}.Serve(t, ex)

	pusher := &push_http.HTTPSecurityEventPusher{Client: routedTo(server)}
	if err := pusher.Push(context.Background(), "https://notify.rp.example.com/Events", "", string(testing_httprecord.BodyBytes(ex.Request.Body))); err != nil {
		t.Fatalf("Push: %v", err)
	}
}

// 記録の例は Authorization を持たないので、配送の設定の Authorization をそのまま送ることは手書きの応答で確かめる。
func TestHTTPSecurityEventPusher_Push_SendsConfiguredAuthorization(t *testing.T) {
	var gotAuth string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	pusher := &push_http.HTTPSecurityEventPusher{Client: server.Client()}
	if err := pusher.Push(context.Background(), server.URL, "Bearer secret-token", "compact.jwt.value"); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization = %q, want Bearer secret-token", gotAuth)
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
