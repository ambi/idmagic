package evaluation_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
)

// 判断があると読めるのは decision を持つ 200 だけである。
//
//spec:covers REQ-MCPGATEWAY-003: 200 以外と解釈できない応答を判断なしにし、429 の Retry-After を返すこと
func TestEvaluatePermitsOnlyOnAnExplicitDecision(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		available bool
		permitted bool
		retry     string
	}{
		{name: "permit", status: 200, body: `{"decision":true}`, available: true, permitted: true},
		{name: "deny", status: 200, body: `{"decision":false,"reasons":["x"]}`, available: true},
		{name: "missing decision", status: 200, body: `{}`},
		{name: "malformed", status: 200, body: `{`},
		{name: "unavailable", status: 503, body: `{"decision":true}`},
		{name: "unbound", status: 403, body: `{"decision":true}`},
		{name: "redirect", status: 302, body: `{"decision":true}`},
		{name: "rate limited", status: 429, body: `{"decision":true}`, retry: "9"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if user, password, ok := r.BasicAuth(); !ok || user != "rs" || password != "secret" {
					t.Errorf("credentials = %q %q", user, password)
				}
				if r.PostFormValue("resource_id") != "d1" || r.PostFormValue("htu") != "https://mcp/" || r.PostFormValue("htm") != "POST" || r.PostFormValue("dpop_proof") != "p" {
					t.Errorf("form = %v", r.PostForm)
				}
				w.Header().Set("X-Request-ID", "server-id")
				if tc.retry != "" {
					w.Header().Set("Retry-After", tc.retry)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client := Client{
				EvaluateURL: server.URL, ClientID: "rs", ClientSecret: "secret",
				HTTPClient: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
			}
			result, err := client.Evaluate(t.Context(), Request{
				Token: "t", CorrelationID: "c", DPoPProof: "p", HTU: "https://mcp/",
				Operation: domain.Operation{ResourceType: "document", Relation: "viewer", ResourceID: "d1"},
			})
			if result.Available != tc.available || result.Permitted != tc.permitted || result.RetryAfter != tc.retry || result.RequestID != "server-id" {
				t.Fatalf("result = %+v err=%v", result, err)
			}
			if (err == nil) != tc.available {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
