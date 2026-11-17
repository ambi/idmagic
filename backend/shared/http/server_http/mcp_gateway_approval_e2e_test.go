package server_http_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/mcpgateway"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type approvalGatewayAudit struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (a *approvalGatewayAudit) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buffer.Write(p)
}

func (a *approvalGatewayAudit) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buffer.String()
}

type mcpApprovalFixture struct {
	*approvedOperationFixture
	config         string
	audit          approvalGatewayAudit
	effects        atomic.Int32
	dropResponse   atomic.Bool
	dropEvaluation atomic.Bool
	inflight       sync.WaitGroup
}

type approvedToolInput struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

func newMcpApprovalFixture(t *testing.T) *mcpApprovalFixture {
	t.Helper()
	f := &mcpApprovalFixture{approvedOperationFixture: newApprovedOperationFixture(t)}
	f.s.Echo.Use(support.RequestIDMiddleware(false))
	idmagic := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.dropEvaluation.Load() && strings.HasSuffix(r.URL.Path, "/resource-access/evaluate") {
			f.s.Echo.ServeHTTP(httptest.NewRecorder(), r)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		f.s.Echo.ServeHTTP(w, r)
	}))
	t.Cleanup(idmagic.Close)
	server := mcp.NewServer(&mcp.Implementation{Name: "ordinary-mcp-server", Version: "1"}, nil)
	for _, name := range []string{"execute", "other"} {
		mcp.AddTool(server, &mcp.Tool{Name: name}, func(context.Context, *mcp.CallToolRequest, approvedToolInput) (*mcp.CallToolResult, readDocumentOutput, error) {
			f.effects.Add(1)
			return nil, readDocumentOutput{Title: "executed"}, nil
		})
	}
	sdk := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.dropResponse.Load() {
			// SDK がツールを実行してから、クライアントへ応答を一バイトも返さずに接続を閉じる。
			sdk.ServeHTTP(httptest.NewRecorder(), r)
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = connection.Close()
			return
		}
		sdk.ServeHTTP(w, r)
	}))
	t.Cleanup(upstream.Close)
	dir := t.TempDir()
	write := func(name string, content []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	f.config = write("gateway.yaml", fmt.Appendf(nil, `listen: 127.0.0.1:0
resource: %s
upstream: %s/
idmagic:
  realm_url: %s/realms/default
  ca_file: %s
  client_id: %s
  client_secret_file: %s
mapping:
  methods:
    server/discover: {resource_type: document, relation: viewer, resource_id: d1}
  tools:
    execute: {resource_type: document, relation: viewer, resource_id: d1, attribute_arguments: [amount]}
    other: {resource_type: document, relation: viewer, resource_id: d2}
`, agentRevocationResource, upstream.URL, idmagic.URL,
		write("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idmagic.Certificate().Raw})),
		evalResourceServerClient, write("secret", []byte(evalResourceServerSecret))))
	return f
}

func (f *mcpApprovalFixture) gateway(t *testing.T) *httptest.Server {
	t.Helper()
	loaded, err := mcpgateway.Load(f.config, &f.audit)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.inflight.Add(1)
		defer f.inflight.Done()
		loaded.Handler().ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func (f *mcpApprovalFixture) call(t *testing.T, gateway *httptest.Server, token, tool, parameters string, id int) (int, string) {
	t.Helper()
	status, _, body := f.callWithHeader(t, gateway, token, tool, parameters, id)
	return status, body
}

func (f *mcpApprovalFixture) callWithHeader(t *testing.T, gateway *httptest.Server, token, tool, parameters string, id int) (int, http.Header, string) {
	t.Helper()
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":%q,"arguments":%s,"_meta":{"io.modelcontextprotocol/protocolVersion":%q,"io.modelcontextprotocol/clientCapabilities":{}}}}`, id, tool, parameters, mcpProtocolVersion)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, gateway.URL+"/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Mcp-Protocol-Version", mcpProtocolVersion)
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", tool)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, response.Header, string(raw)
}

func requireApprovalGatewayError(t *testing.T, status int, body string, wantStatus, wantCode int) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("error body=%s: %v", body, err)
	}
	if status != wantStatus || envelope.Error.Code != wantCode {
		t.Fatalf("status=%d body=%s; want status=%d code=%d", status, body, wantStatus, wantCode)
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-MCPGATEWAY-006, REQ-OAUTH2-065: 対象と引数が一致する承認を一回だけ転送し、再送は拒否し、承認の相関だけを記録すること
func TestMcpGatewayForwardsAnApprovalAtMostOnce(t *testing.T) {
	f := newMcpApprovalFixture(t)
	gateway := f.gateway(t)
	token := f.approvedToken(t)
	status, body := f.call(t, gateway, token, "execute", `{"amount":101,"currency":"JPY"}`, 1)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32000)
	status, body = f.call(t, gateway, token, "other", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32000)
	if f.effects.Load() != 0 {
		t.Fatalf("mismatches executed: %d", f.effects.Load())
	}
	status, body = f.call(t, gateway, token, "execute", `{"currency":"JPY", "amount":100}`, 3)
	if status != http.StatusOK || !strings.Contains(body, "executed") || f.effects.Load() != 1 {
		t.Fatalf("approved call: status=%d body=%s effects=%d", status, body, f.effects.Load())
	}
	for _, id := range []int{3, 4} {
		status, body = f.call(t, gateway, token, "execute", approvedParameters, id)
		requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	}
	// 同じ引数でも、新たな人の承認があれば別の一回を実行できる。
	secondToken := f.approvedToken(t)
	status, body = f.call(t, gateway, secondToken, "execute", approvedParameters, 5)
	if status != http.StatusOK || f.effects.Load() != 2 {
		t.Fatalf("new approval: %d %s effects=%d", status, body, f.effects.Load())
	}
	f.inflight.Wait()
	log := f.audit.String()
	if !strings.Contains(log, `"approval_request_id":`) || !strings.Contains(log, `"operation_digest":`) {
		t.Fatalf("missing approval correlation: %s", log)
	}
	if strings.Contains(log, token) || strings.Contains(log, `"amount"`) || strings.Contains(log, `"operation_key"`) {
		t.Fatalf("credentials or arguments logged: %s", log)
	}
}

//spec:covers REQ-MCPGATEWAY-013: 別プロセス相当のゲートウェイへの並行要求と再作成で承認を再利用しないこと
func TestMcpGatewaySharesApprovalUseAcrossInstances(t *testing.T) {
	f := newMcpApprovalFixture(t)
	first, second := f.gateway(t), f.gateway(t)
	token := f.approvedToken(t)
	start := make(chan struct{})
	statuses := make(chan int, 2)
	for i, gateway := range []*httptest.Server{first, second} {
		go func() {
			status := 0
			defer func() { statuses <- status }()
			<-start
			status, _ = f.call(t, gateway, token, "execute", approvedParameters, i+1)
		}()
	}
	close(start)
	counts := map[int]int{}
	counts[<-statuses]++
	counts[<-statuses]++
	if counts[http.StatusOK] != 1 || counts[http.StatusForbidden] != 1 || f.effects.Load() != 1 {
		t.Fatalf("parallel calls: statuses=%v effects=%d", counts, f.effects.Load())
	}
	first.Close()
	second.Close()
	restarted := f.gateway(t)
	status, body := f.call(t, restarted, token, "execute", approvedParameters, 3)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 1 {
		t.Fatalf("restart executed: %d", f.effects.Load())
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-MCPGATEWAY-006: 実行後に上流の応答が失われても使用を戻さず、結果不明を返して再送を拒否すること
func TestMcpGatewayDoesNotRetryAnUnknownApprovedOutcome(t *testing.T) {
	f := newMcpApprovalFixture(t)
	gateway := f.gateway(t)
	token := f.approvedToken(t)
	f.dropResponse.Store(true)
	status, body := f.call(t, gateway, token, "execute", approvedParameters, 1)
	requireApprovalGatewayError(t, status, body, http.StatusBadGateway, -32005)
	if !strings.Contains(body, `"id":1`) {
		t.Fatalf("lost RPC correlation: %s", body)
	}
	if f.effects.Load() != 1 {
		t.Fatalf("lost response effects=%d", f.effects.Load())
	}
	f.dropResponse.Store(false)
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 1 {
		t.Fatalf("retry executed: %d", f.effects.Load())
	}
	f.inflight.Wait()
	if !strings.Contains(f.audit.String(), `"outcome":"approval_outcome_unknown"`) {
		t.Fatalf("audit=%s", f.audit.String())
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-OAUTH2-065: 使用を記録した評価の応答を失っても許可を推測せず、承認を再発行しないこと
func TestMcpGatewayDoesNotReleaseApprovalWhenEvaluationResponseIsLost(t *testing.T) {
	f := newMcpApprovalFixture(t)
	gateway := f.gateway(t)
	token := f.approvedToken(t)
	f.dropEvaluation.Store(true)
	status, body := f.call(t, gateway, token, "execute", approvedParameters, 1)
	requireApprovalGatewayError(t, status, body, http.StatusServiceUnavailable, -32002)
	f.dropEvaluation.Store(false)
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 0 {
		t.Fatalf("unconfirmed permit was forwarded: %d", f.effects.Load())
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-OAUTH2-065: 条件が承認を求める場合だけ承認を消費し、閾値以下は通常の認可で実行できること
func TestMcpGatewayClaimsApprovalOnlyWhenTheConditionRequiresIt(t *testing.T) {
	f := newMcpApprovalFixture(t)
	if status, body := f.putCondition(t, map[string]any{
		"require_approval_when": "attributes.amount >= 100", "approval_detail_type": approvedDetailType,
	}); status != http.StatusOK {
		t.Fatalf("condition: status=%d body=%v", status, body)
	}
	gateway := f.gateway(t)
	ordinaryToken := f.resourceToken(t, f.clientID, agentRevocationResource, nil)
	status, body := f.call(t, gateway, ordinaryToken, "execute", `{"amount":80,"currency":"JPY"}`, 1)
	if status != http.StatusOK {
		t.Fatalf("ordinary call: %d %s", status, body)
	}
	status, body = f.call(t, gateway, ordinaryToken, "execute", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32000)
	token := f.approvedToken(t)
	status, body = f.call(t, gateway, token, "execute", `{"amount":80,"currency":"JPY"}`, 3)
	if status != http.StatusOK {
		t.Fatalf("approval not needed: %d %s", status, body)
	}
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 4)
	if status != http.StatusOK {
		t.Fatalf("approval was consumed below threshold: %d %s", status, body)
	}
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 5)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 3 {
		t.Fatalf("effects=%d", f.effects.Load())
	}
}

// remediationChallenge は 403 の応答から、チャレンジのヘッダーと JSON-RPC の error.data の明細を読む。
// ヘッダーの形は draft-ietf-oauth-rar-metadata-remediation-00 の §4 の例に従って独立に解析する。
func remediationChallenge(t *testing.T, header http.Header, body string) (headerJSON, dataJSON string) {
	t.Helper()
	challenge := header.Get("WWW-Authenticate")
	if !strings.HasPrefix(challenge, "Bearer ") || !strings.Contains(challenge, `error="insufficient_authorization"`) ||
		!strings.Contains(challenge, `resource_metadata="`) {
		t.Fatalf("challenge=%q", challenge)
	}
	_, after, found := strings.Cut(challenge, "authorization_remediation=")
	if !found {
		t.Fatalf("challenge without authorization_remediation: %q", challenge)
	}
	encoded, _, _ := strings.Cut(strings.Trim(after, `"`), `"`)
	encoded, _, _ = strings.Cut(encoded, ",")
	decoded, err := base64.RawURLEncoding.DecodeString(strings.Trim(encoded, `" `))
	if err != nil {
		t.Fatalf("authorization_remediation is not base64url: %q: %v", encoded, err)
	}
	var envelope struct {
		Error struct {
			Code int `json:"code"`
			Data struct {
				AuthorizationRemediation json.RawMessage `json:"authorization_remediation"`
			} `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil || envelope.Error.Code != -32000 {
		t.Fatalf("body=%s: %v", body, err)
	}
	return string(decoded), string(envelope.Error.Data.AuthorizationRemediation)
}

// 承認のない Agent が、ゲートウェイの示す明細をそのまま CIBA で要求し、本人の承認で得たトークンで
// 同じ呼び出しを一回だけ実行できる。期待する明細は登録と要求から独立に書く。
//
//spec:covers REQ-OAUTH2-065, REQ-MCPGATEWAY-003: 承認の不足に明細を示し、その明細の承認で一回だけ実行すること
//spec:covers RARREM-ERROR-CODE, RARREM-REMEDIATION, RARREM-STATUS, RARREM-REFERENCE: 403 の insufficient_authorization と、authorization_reference のない明細
func TestMcpGatewayGuidesAnAgentFromRemediationToOneApprovedCall(t *testing.T) {
	f := newMcpApprovalFixture(t)
	gateway := f.gateway(t)
	ordinaryToken := f.resourceToken(t, f.clientID, agentRevocationResource, nil)

	status, header, body := f.callWithHeader(t, gateway, ordinaryToken, "execute", `{"currency":"JPY","amount":100}`, 1)
	if status != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", status, body)
	}
	headerJSON, dataJSON := remediationChallenge(t, header, body)
	want := `{"authorization_details":[{"type":"payment","locations":["` + agentRevocationResource + `"],"actions":["viewer"],"identifier":"d1","fields":{"amount":100,"currency":"JPY"}}]}`
	for name, got := range map[string]string{"header": headerJSON, "data": dataJSON} {
		var gotValue, wantValue any
		if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
			t.Fatalf("%s: %s: %v", name, got, err)
		}
		_ = json.Unmarshal([]byte(want), &wantValue)
		if fmt.Sprint(gotValue) != fmt.Sprint(wantValue) {
			t.Fatalf("%s remediation=%s, want %s", name, got, want)
		}
		if strings.Contains(got, "authorization_reference") {
			t.Fatalf("%s carries a reference for a single-use approval: %s", name, got)
		}
	}
	if f.effects.Load() != 0 {
		t.Fatalf("executed without approval: %d", f.effects.Load())
	}

	// Agent は示された明細を書き換えずに CIBA で要求する。
	var remediation struct {
		AuthorizationDetails json.RawMessage `json:"authorization_details"`
	}
	if err := json.Unmarshal([]byte(headerJSON), &remediation); err != nil {
		t.Fatal(err)
	}
	user, err := f.s.Users.FindBySub(context.Background(), stack.UserID)
	if err != nil || user == nil {
		t.Fatalf("user: %v %v", user, err)
	}
	status, started := f.post(t, "bc-authorize", f.clientID, url.Values{
		"login_hint": {user.PreferredUsername}, "scope": {"openid tools:read"},
		"authorization_details": {string(remediation.AuthorizationDetails)},
	})
	authReqID, _ := started["auth_req_id"].(string)
	if status != http.StatusOK || authReqID == "" {
		t.Fatalf("bc-authorize status=%d body=%v", status, started)
	}
	ctx := f.tenantContext()
	rec, err := f.s.ApprovalRequests.FindByAuthReqIDHash(ctx, oauthdomain.HashAuthReqID(authReqID))
	if err != nil || rec == nil {
		t.Fatalf("approval: %v %v", rec, err)
	}
	if decided, err := f.s.ApprovalRequests.Decide(ctx, rec.ID, stack.UserID, spec.ApprovalEventApprove, time.Now().UTC()); err != nil || decided == nil {
		t.Fatalf("decide: %v %v", decided, err)
	}
	approvedToken := f.exchange(t, authReqID)

	status, body = f.call(t, gateway, approvedToken, "execute", `{"currency":"JPY","amount":100}`, 2)
	if status != http.StatusOK || !strings.Contains(body, "executed") || f.effects.Load() != 1 {
		t.Fatalf("approved call: status=%d body=%s effects=%d", status, body, f.effects.Load())
	}
	status, body = f.call(t, gateway, approvedToken, "execute", `{"currency":"JPY","amount":100}`, 3)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 1 {
		t.Fatalf("effects=%d", f.effects.Load())
	}
	f.inflight.Wait()
	if strings.Contains(f.audit.String(), "authorization_remediation") || strings.Contains(f.audit.String(), `"currency"`) {
		t.Fatalf("remediation logged: %s", f.audit.String())
	}
}

// genericClientToken は、CIBA もトークンの差し替えもしない汎用の MCP クライアントが認可コードで得た
// 利用者のトークンに当たる。発行の経路は今回の故障と無関係なので、製品の署名器で直接発行する。
func (f *mcpApprovalFixture) genericClientToken(t *testing.T) string {
	t.Helper()
	ctx := f.s.RealmContext(t, "default")
	client, err := f.s.Clients.FindByID(ctx, tenancydomain.DefaultTenantID, f.clientID)
	if err != nil || client == nil {
		t.Fatalf("client: %v %v", client, err)
	}
	token, _, err := f.s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: client, Sub: stack.UserID, Scopes: []string{"tools:read"}, Audiences: []string{agentRevocationResource},
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// requireApprovalOverThreshold は閾値以上の金額にだけ人の承認を求め、利用者にも d1 の関係を与える。
func (f *mcpApprovalFixture) requireApprovalOverThreshold(t *testing.T) {
	t.Helper()
	if status, body := f.putCondition(t, map[string]any{
		"require_approval_when": "has(attributes.amount) && attributes.amount >= 100", "approval_detail_type": approvedDetailType,
	}); status != http.StatusOK {
		t.Fatalf("condition: status=%d body=%v", status, body)
	}
	tuple := d1ViewerTuple()
	tuple["subject_type"], tuple["subject_id"] = "user", stack.UserID
	f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []map[string]any{tuple}})
}

// portal は本人のアカウントポータルの要求を送る。セッションはパスワードと TOTP で認証済みとし、
// 判断に要るステップアップ認証を満たす。
func (f *mcpApprovalFixture) portal(ctx context.Context, t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	tenant := tenantports.WithTenant(ctx, &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, stack.Issuer, "/realms/default")
	authn, err := f.s.Sessions.Create(tenant, stack.UserID, []string{"pwd", "otp"}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var raw bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&raw).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequestWithContext(ctx, method, stack.Issuer+"/realms/default"+path, &raw)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", stack.Issuer)
	request.Header.Set(support.CSRFHeader, "portal-csrf")
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "portal-csrf"})
	request.AddCookie(&http.Cookie{Name: authentication.SessionCookie, Value: authn.SessionID})
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	result := map[string]any{}
	if recorder.Body.Len() > 0 {
		_ = json.Unmarshal(recorder.Body.Bytes(), &result)
	}
	return recorder.Code, result
}

// 汎用の MCP クライアントは、承認の不足で URL の elicitation を受け、本人がポータルで承認した後の
// 再送で一回だけ実行する。トークンは最後まで同じで、同じ操作の次の呼び出しは新しい承認を求める。
//
//spec:covers REQ-MCPGATEWAY-014, REQ-OAUTH2-093, REQ-OAUTH2-043: URL の elicitation、本人の承認、再送の一回の転送、起票者の表示、辞退の拒否
func TestMcpGatewayLeadsAGenericClientThroughHumanApproval(t *testing.T) {
	f := newMcpApprovalFixture(t)
	f.requireApprovalOverThreshold(t)
	gateway := f.gateway(t)
	token := f.genericClientToken(t)

	var elicited []*mcp.ElicitParams
	decision := "approve"
	client := mcp.NewClient(&mcp.Implementation{Name: "generic-client", Version: "1"}, &mcp.ClientOptions{
		Capabilities: &mcp.ClientCapabilities{Elicitation: &mcp.ElicitationCapabilities{URL: &mcp.URLElicitationCapabilities{}}},
		ElicitationHandler: func(ctx context.Context, request *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			elicited = append(elicited, request.Params)
			if decision == "decline" {
				return &mcp.ElicitResult{Action: "decline"}, nil
			}
			target, err := url.Parse(request.Params.URL)
			if err != nil {
				return nil, err
			}
			id := target.Query().Get("request")
			// 本人はポータルで、起票した登録の表示名とともに保留中の承認リクエストを見てから承認する。
			status, listed := f.portal(ctx, t, http.MethodGet, "/api/account/v1/approval-requests", nil)
			requests, _ := listed["approval_requests"].([]any)
			if status != http.StatusOK || len(requests) != 1 {
				t.Errorf("pending approvals: status=%d body=%v", status, listed)
				return &mcp.ElicitResult{Action: "cancel"}, nil
			}
			pending, _ := requests[0].(map[string]any)
			if pending["id"] != id || pending["resource_server_name"] != "agent revocation" {
				t.Errorf("listed approval does not identify the request and its resource server: %v (url id %q)", pending, id)
			}
			if status, body := f.portal(ctx, t, http.MethodPost, "/api/account/v1/approval-requests/"+id+"/decision", map[string]any{"decision": decision}); status != http.StatusNoContent {
				t.Errorf("decision: status=%d body=%v", status, body)
			}
			return &mcp.ElicitResult{Action: "accept"}, nil
		},
	})
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint: gateway.URL + "/", HTTPClient: &http.Client{Transport: presentToken{token}},
	}, nil)
	if err != nil {
		f.inflight.Wait()
		t.Fatalf("connect: %v\naudit=%s", err, f.audit.String())
	}
	t.Cleanup(func() { _ = session.Close() })

	arguments := map[string]any{"amount": 100, "currency": "JPY"}
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "execute", Arguments: arguments})
	if err != nil || result.IsError || f.effects.Load() != 1 {
		t.Fatalf("approved call: result=%v err=%v effects=%d", result, err, f.effects.Load())
	}
	if len(elicited) != 1 || elicited[0].Mode != "url" ||
		!strings.HasSuffix(strings.Split(elicited[0].URL, "?")[0], "/realms/default/account/approvals") {
		t.Fatalf("elicitations=%+v, want one URL elicitation to the account portal", elicited)
	}

	// 同じ承認は二度目の呼び出しに使えない。新しい承認を本人が辞退すれば実行されない。
	decision = "decline"
	if _, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "execute", Arguments: arguments}); err == nil {
		t.Fatal("declined call succeeded")
	}
	if f.effects.Load() != 1 || len(elicited) != 2 || elicited[1].URL == elicited[0].URL {
		t.Fatalf("second call: effects=%d elicitations=%d", f.effects.Load(), len(elicited))
	}

	f.inflight.Wait()
	log := f.audit.String()
	if !strings.Contains(log, `"outcome":"input_required"`) || !strings.Contains(log, `"approval_request_id":`) {
		t.Fatalf("missing input_required audit: %s", log)
	}
	if strings.Contains(log, token) || strings.Contains(log, `"currency"`) || strings.Contains(log, "requestState") {
		t.Fatalf("credentials, arguments or state logged: %s", log)
	}
}

//spec:covers REQ-OAUTH2-093: approval_start の真偽値以外と UUID でない approval_request_id を 400 の invalid_request で拒否すること
func TestEvaluationRejectsMalformedApprovalInputs(t *testing.T) {
	f := newMcpApprovalFixture(t)
	token := f.genericClientToken(t)
	for name, change := range map[string]func(url.Values){
		"approval_start yes":       func(form url.Values) { form.Set("approval_start", "yes") },
		"approval_start upper":     func(form url.Values) { form.Set("approval_start", "TRUE") },
		"approval_request_id text": func(form url.Values) { form.Set("approval_request_id", "approval-1") },
		"approval_request_id upper": func(form url.Values) {
			form.Set("approval_request_id", "0B6F2C8E-6A59-4F1E-9A0F-3F1D6C2B7A10")
		},
	} {
		form := approvedForm(token, "k", approvedParameters)
		change(form)
		w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form)
		if w.Code != http.StatusBadRequest || body["error"] != "invalid_request" {
			t.Errorf("%s: status=%d body=%v", name, w.Code, body)
		}
	}
	form := approvedForm(token, "k", approvedParameters)
	form.Set("approval_start", "false")
	form.Set("approval_request_id", "0b6f2c8e-6a59-4f1e-9a0f-3f1d6c2b7a10")
	if w, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form); w.Code != http.StatusOK || body["decision"] != false {
		t.Fatalf("well-formed inputs: status=%d body=%v", w.Code, body)
	}
}
