package evaluation_http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mcpgatewaydomain "github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
)

// 判断があると読めるのは decision を持つ 200 だけである。
//
//spec:covers REQ-MCPGATEWAY-003, REQ-MCPGATEWAY-013: 200 以外と解釈できない承認の応答と明細を判断なしにし、429 の Retry-After を返すこと
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
		{name: "approval without replay flag", status: 200, body: `{"decision":true,"approved_operation":{"approval_request_id":"a1","operation_digest":"` + strings.Repeat("a", 64) + `"}}`},
		{name: "approval without id", status: 200, body: `{"decision":true,"approved_operation":{"operation_digest":"` + strings.Repeat("a", 64) + `","replayed":false}}`},
		{name: "approval with invalid digest", status: 200, body: `{"decision":true,"approved_operation":{"approval_request_id":"a1","operation_digest":"bad","replayed":false}}`},
		{name: "remediation", status: 200, body: `{"decision":false,"authorization_remediation":{"authorization_details":[{"type":"payment"}]}}`, available: true},
		{name: "remediation without details", status: 200, body: `{"decision":false,"authorization_remediation":{}}`},
		{name: "remediation with non-array details", status: 200, body: `{"decision":false,"authorization_remediation":{"authorization_details":{"type":"payment"}}}`},
		{name: "remediation that is not an object", status: 200, body: `{"decision":false,"authorization_remediation":"x"}`},
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
				Operation: mcpgatewaydomain.Operation{ResourceType: "document", Relation: "viewer", ResourceID: "d1"},
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

//spec:covers REQ-MCPGATEWAY-003: 選んだ attributes だけを評価の form に載せ、選んでいない評価には載せないこと
func TestEvaluateSendsAttributesOnlyWhenSelected(t *testing.T) {
	for _, attributes := range []string{`{"amount":80}`, ""} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			if got, sent := r.PostForm["attributes"]; (attributes == "") == sent || (sent && got[0] != attributes) {
				t.Errorf("attributes = %v (want %q)", got, attributes)
			}
			_, _ = w.Write([]byte(`{"decision":true}`))
		}))
		client := Client{EvaluateURL: server.URL, ClientID: "rs", ClientSecret: "secret", HTTPClient: server.Client()}
		if _, err := client.Evaluate(t.Context(), Request{Token: "t", Attributes: attributes}); err != nil {
			t.Fatal(err)
		}
		server.Close()
	}
}

//spec:covers REQ-MCPGATEWAY-013: 実行の鍵と引数を form で送り、承認の相関と明示した再送状態を読むこと
func TestEvaluateSendsTheOperationAndReadsApprovalUse(t *testing.T) {
	const parameters = `{"amount":100,"currency":"JPY"}`
	for _, replayed := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.PostFormValue("operation_key") != "attempt-1" || r.PostFormValue("operation_parameters") != parameters {
				t.Errorf("form=%v", r.PostForm)
			}
			_, _ = fmt.Fprintf(w, `{"decision":true,"approved_operation":{"approval_request_id":"approval-1","operation_digest":%q,"replayed":%t}}`, strings.Repeat("a", 64), replayed)
		}))
		client := Client{EvaluateURL: server.URL, HTTPClient: server.Client()}
		result, err := client.Evaluate(t.Context(), Request{OperationKey: "attempt-1", OperationParameters: parameters})
		server.Close()
		if err != nil || !result.Available || !result.Permitted || result.ApprovedOperation == nil {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		if use := result.ApprovedOperation; use.ApprovalRequestID != "approval-1" || use.OperationDigest != strings.Repeat("a", 64) || use.Replayed != replayed {
			t.Fatalf("approval=%+v", use)
		}
	}
}

//spec:covers REQ-MCPGATEWAY-014: 起票と参照を求めるときだけ form に載せ、保留中の承認リクエストを読み、形の誤った値を判断なしにすること
func TestEvaluateStartsAndReferencesApprovals(t *testing.T) {
	const pending = `{"decision":false,"reasons":["approval_pending"],"approval_request":{"id":"a1","status":"pending","expires_at":"2026-11-18T09:10:00Z","approval_uri":"https://idp.example/realms/r/account/approvals?request=a1"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("approval_start") != "true" || r.PostFormValue("approval_request_id") != "a0" {
			t.Errorf("form=%v", r.PostForm)
		}
		_, _ = w.Write([]byte(pending))
	}))
	defer server.Close()
	client := Client{EvaluateURL: server.URL, HTTPClient: server.Client()}
	result, err := client.Evaluate(t.Context(), Request{ApprovalStart: true, ApprovalRequestID: "a0"})
	if err != nil || !result.Available || result.Permitted || result.ApprovalRequest == nil ||
		result.ApprovalRequest.ID != "a1" || result.ApprovalRequest.URI != "https://idp.example/realms/r/account/approvals?request=a1" ||
		!result.ApprovalRequest.ExpiresAt.Equal(time.Date(2026, 11, 18, 9, 10, 0, 0, time.UTC)) {
		t.Fatalf("result=%+v err=%v", result, err)
	}

	quiet := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if _, sent := r.PostForm["approval_start"]; sent {
			t.Errorf("approval_start sent without a request: %v", r.PostForm)
		}
		if _, sent := r.PostForm["approval_request_id"]; sent {
			t.Errorf("approval_request_id sent without a reference: %v", r.PostForm)
		}
		_, _ = w.Write([]byte(`{"decision":true}`))
	}))
	defer quiet.Close()
	if _, err := (Client{EvaluateURL: quiet.URL, HTTPClient: quiet.Client()}).Evaluate(t.Context(), Request{}); err != nil {
		t.Fatal(err)
	}

	for name, body := range map[string]string{
		"no id":        `{"decision":false,"approval_request":{"status":"pending","expires_at":"2026-11-18T09:10:00Z","approval_uri":"https://idp.example/a"}}`,
		"no uri":       `{"decision":false,"approval_request":{"id":"a1","status":"pending","expires_at":"2026-11-18T09:10:00Z"}}`,
		"relative uri": `{"decision":false,"approval_request":{"id":"a1","status":"pending","expires_at":"2026-11-18T09:10:00Z","approval_uri":"/account"}}`,
		"http uri":     `{"decision":false,"approval_request":{"id":"a1","status":"pending","expires_at":"2026-11-18T09:10:00Z","approval_uri":"http://idp.example/a"}}`,
		"no expiry":    `{"decision":false,"approval_request":{"id":"a1","status":"pending","approval_uri":"https://idp.example/a"}}`,
		"on a permit":  `{"decision":true,"approval_request":{"id":"a1","status":"pending","expires_at":"2026-11-18T09:10:00Z","approval_uri":"https://idp.example/a"}}`,
	} {
		malformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
		result, err := (Client{EvaluateURL: malformed.URL, HTTPClient: malformed.Client()}).Evaluate(t.Context(), Request{})
		malformed.Close()
		if err == nil || result.Available {
			t.Errorf("%s: result=%+v err=%v", name, result, err)
		}
	}
}
