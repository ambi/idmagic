package proxy_http

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
	"github.com/ambi/idmagic/backend/mcpgateway/internal/evaluation_http"
)

const approvalURI = "https://id.example.test/realms/default/account/approvals?request=a1"

var approvalKey = domain.RequestStateKey("gateway-secret")

func withApprovalKey(options *Options) { options.RequestStateKey = approvalKey }

// approvalCall は URL の elicitation の宣言と、再送の requestState と inputResponses を持つ tools/call である。
func approvalCall(t *testing.T, target, arguments, capabilities, retry string) *http.Request {
	t.Helper()
	request := toolRequest(t, target, "read")
	body := `{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"read","arguments":` + arguments + retry +
		`,"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":` + capabilities + `}}}`
	request.Body = io.NopCloser(strings.NewReader(body))
	request.ContentLength = int64(len(body))
	return request
}

const urlCapability = `{"elicitation":{"url":{}}}`

func pendingResult() evaluation_http.Result {
	return evaluation_http.Result{
		Available: true, Reasons: []string{"approval_pending"},
		ApprovalRequest: &evaluation_http.ApprovalRequest{ID: "a1", ExpiresAt: time.Now().Add(10 * time.Minute), URI: approvalURI},
	}
}

// sealFor は引数 {"amount":100} の呼び出しに対して封じた状態を返す。
func sealFor(t *testing.T, expires time.Time) string {
	t.Helper()
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(`{"amount":100}`), &decoded); err != nil {
		t.Fatal(err)
	}
	digest, err := domain.OperationDigest("read", decoded)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := domain.SealRequestState(approvalKey, resourceURL, domain.RequestState{ApprovalRequestID: "a1", OperationDigest: digest, ExpiresAt: expires}, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func retryWith(state, action string) string {
	return `,"requestState":"` + state + `","inputResponses":{"idmagic_approval":{"action":"` + action + `"},"upstream_question":{"action":"accept"}}`
}

// URL を宣言したクライアントには、承認を起票させ、承認の URL と封じた状態を InputRequiredResult で返す。
//
//spec:covers REQ-MCPGATEWAY-014, REQ-MCPGATEWAY-006: 起票の要求、InputRequiredResult の形、上流へ送らないこと、input_required の監査
func TestApprovalPendingIsReturnedAsInputRequired(t *testing.T) {
	h := newHarness(t, accepted, withApprovalKey)
	h.evaluator.result = pendingResult()
	response, body := do(t, approvalCall(t, h.gateway.URL, `{"amount":100}`, urlCapability, ""))
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/json" || h.calls.Load() != 0 {
		t.Fatalf("status=%d type=%q upstream=%d body=%v", response.StatusCode, response.Header.Get("Content-Type"), h.calls.Load(), body)
	}
	if !h.evaluator.last.ApprovalStart || h.evaluator.last.ApprovalRequestID != "" {
		t.Fatalf("evaluation=%+v", h.evaluator.last)
	}
	result, _ := body["result"].(map[string]any)
	requests, _ := result["inputRequests"].(map[string]any)
	approval, _ := requests["idmagic_approval"].(map[string]any)
	params, _ := approval["params"].(map[string]any)
	state, _ := result["requestState"].(string)
	if body["id"] != "call-1" || result["resultType"] != "input_required" || approval["method"] != "elicitation/create" ||
		params["mode"] != "url" || params["url"] != approvalURI || params["message"] == "" || !strings.HasPrefix(state, domain.RequestStatePrefix) {
		t.Fatalf("body=%v", body)
	}
	log := h.auditLog()
	if !strings.Contains(log, `"outcome":"input_required"`) || !strings.Contains(log, `"approval_request_id":"a1"`) || strings.Contains(log, state) {
		t.Fatalf("audit=%s", log)
	}
}

// URL を宣言しないクライアントと Envoy の判定には起票を求めず、従来の応答のままにする。
//
//spec:covers REQ-MCPGATEWAY-014, REQ-MCPGATEWAY-003: 宣言のない要求と envoy_ext_authz では approval_start も参照も送らないこと
func TestApprovalIsNotStartedWithoutURLElicitation(t *testing.T) {
	for name, capabilities := range map[string]string{"empty": `{}`, "form only": `{"elicitation":{"form":{}}}`} {
		h := newHarness(t, accepted, withApprovalKey)
		h.evaluator.result = evaluation_http.Result{Available: true, Remediation: json.RawMessage(`{"authorization_details":[]}`)}
		response, body := do(t, approvalCall(t, h.gateway.URL, `{"amount":100}`, capabilities, ""))
		if h.evaluator.last.ApprovalStart || response.StatusCode != http.StatusForbidden || errorCode(body) != codeForbidden {
			t.Fatalf("%s: start=%v status=%d body=%v", name, h.evaluator.last.ApprovalStart, response.StatusCode, body)
		}
	}
	h := newHarness(t, accepted, func(options *Options) { envoyMode(options); withApprovalKey(options) })
	request := approvalCall(t, h.gateway.URL, `{"amount":100}`, urlCapability, retryWith(sealFor(t, time.Now().Add(time.Minute)), "accept"))
	request.Header.Set(envoyPartialBodyHeader, "false")
	if response, _ := do(t, request); response.StatusCode != http.StatusOK || h.evaluator.last.ApprovalStart || h.evaluator.last.ApprovalRequestID != "" {
		t.Fatalf("envoy: status=%d evaluation=%+v", response.StatusCode, h.evaluator.last)
	}
}

// 再送は封じた参照を評価へ渡し、許可されたら自分の状態だけを除いて一回転送する。
//
//spec:covers REQ-MCPGATEWAY-014: 参照の受け渡し、転送する本文から requestState と承認の応答だけを除くこと
func TestApprovedRetryForwardsWithoutTheGatewayState(t *testing.T) {
	var forwardedBody string
	h := newHarness(t, capturing(&forwardedBody), withApprovalKey)
	h.evaluator.result = evaluation_http.Result{Available: true, Permitted: true}
	state := sealFor(t, time.Now().Add(time.Minute))
	response, _ := do(t, approvalCall(t, h.gateway.URL, `{"amount":100}`, urlCapability, retryWith(state, "accept")))
	if response.StatusCode != http.StatusOK || h.calls.Load() != 1 {
		t.Fatalf("status=%d upstream=%d", response.StatusCode, h.calls.Load())
	}
	if !h.evaluator.last.ApprovalStart || h.evaluator.last.ApprovalRequestID != "a1" {
		t.Fatalf("evaluation=%+v", h.evaluator.last)
	}
	raw := []byte(forwardedBody)
	var envelope struct {
		Params map[string]json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if _, kept := envelope.Params["requestState"]; kept || strings.Contains(string(raw), "idmagic_approval") ||
		!strings.Contains(string(envelope.Params["inputResponses"]), "upstream_question") ||
		string(envelope.Params["arguments"]) != `{"amount":100}` || string(envelope.Params["name"]) != `"read"` {
		t.Fatalf("forwarded body=%s", raw)
	}
}

// 開けない状態、引数を変えた再送、辞退した elicitation は評価せずに拒否する。期限切れの状態は参照のない要求として評価する。
//
//spec:covers REQ-MCPGATEWAY-014: 改ざん、引数の変更、辞退と取り消しの 403 と無評価、期限切れの状態の扱い、他者の requestState の素通し
func TestRetryStateIsCheckedBeforeEvaluation(t *testing.T) {
	valid := sealFor(t, time.Now().Add(time.Minute))
	cases := []struct {
		name      string
		arguments string
		retry     string
		evaluated bool
		reference string
	}{
		{name: "tampered", arguments: `{"amount":100}`, retry: retryWith(valid[:len(valid)-2]+"AA", "accept")},
		{name: "another key", arguments: `{"amount":100}`, retry: retryWith(func() string {
			sealed, _ := domain.SealRequestState(domain.RequestStateKey("other"), resourceURL, domain.RequestState{ApprovalRequestID: "a1", OperationDigest: "d", ExpiresAt: time.Now().Add(time.Minute)}, rand.Reader)
			return sealed
		}(), "accept")},
		{name: "changed arguments", arguments: `{"amount":1000}`, retry: retryWith(valid, "accept")},
		{name: "declined", arguments: `{"amount":100}`, retry: retryWith(valid, "decline")},
		{name: "cancelled", arguments: `{"amount":100}`, retry: retryWith(valid, "cancel")},
		{name: "expired", arguments: `{"amount":100}`, retry: retryWith(sealFor(t, time.Now().Add(-time.Second)), "accept"), evaluated: true},
		{name: "no response", arguments: `{"amount":100}`, retry: `,"requestState":"` + valid + `"`, evaluated: true, reference: "a1"},
		{name: "upstream state", arguments: `{"amount":100}`, retry: `,"requestState":"upstream-opaque"`, evaluated: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, accepted, withApprovalKey)
			h.evaluator.result = evaluation_http.Result{Available: true, Permitted: true}
			response, body := do(t, approvalCall(t, h.gateway.URL, tc.arguments, urlCapability, tc.retry))
			evaluated := h.evaluator.calls.Load() == 1
			if evaluated != tc.evaluated {
				t.Fatalf("evaluated=%v status=%d body=%v", evaluated, response.StatusCode, body)
			}
			if !tc.evaluated && (response.StatusCode != http.StatusForbidden || errorCode(body) != codeForbidden || h.calls.Load() != 0) {
				t.Fatalf("status=%d body=%v upstream=%d", response.StatusCode, body, h.calls.Load())
			}
			if tc.evaluated && h.evaluator.last.ApprovalRequestID != tc.reference {
				t.Fatalf("reference=%q, want %q", h.evaluator.last.ApprovalRequestID, tc.reference)
			}
		})
	}
	// 他者の状態は読まずに、本文を変えずに転送する。
	var foreignBody string
	h := newHarness(t, capturing(&foreignBody), withApprovalKey)
	do(t, approvalCall(t, h.gateway.URL, `{"amount":100}`, urlCapability, `,"requestState":"upstream-opaque"`))
	if !strings.Contains(foreignBody, `"requestState":"upstream-opaque"`) {
		t.Fatalf("forwarded body=%s", foreignBody)
	}
}

// 拒否と上限は 403 で返し、承認の URL を示さない。
//
//spec:covers REQ-MCPGATEWAY-014: approval_denied と approval_request_limit_exceeded を -32000 の 403 にすること
func TestDeniedAndLimitedApprovalsAreForbidden(t *testing.T) {
	for _, reason := range []string{"approval_denied", "approval_request_limit_exceeded"} {
		h := newHarness(t, accepted, withApprovalKey)
		h.evaluator.result = evaluation_http.Result{Available: true, Reasons: []string{reason}}
		response, body := do(t, approvalCall(t, h.gateway.URL, `{"amount":100}`, urlCapability, ""))
		if response.StatusCode != http.StatusForbidden || errorCode(body) != codeForbidden || h.calls.Load() != 0 {
			t.Fatalf("%s: status=%d body=%v", reason, response.StatusCode, body)
		}
	}
}

// capturing は上流が受け取った本文を記録する。
func capturing(body *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		*body = string(raw)
		accepted(w, r)
	}
}
