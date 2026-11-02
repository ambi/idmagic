// agent-demo は、AI エージェントの統制を画面で見せるローカル専用のデモである。
// 台本で動くエージェント、改修していない MCP サーバー、実在の MCP ゲートウェイを一つのプログラムに持つ。
// 秘密鍵はメモリだけに置く。本番の構成の例として流用しない。
package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ambi/idmagic/backend/mcpgateway"
)

const (
	gatewayResource = "https://docs-mcp.agent-demo.example/"
	protocolVersion = "2026-07-28"
	demoScope       = "tools:read"
	publicDocument  = "handbook"
	secretDocument  = "salary-2026"
	maxResponse     = 1 << 20
)

//go:embed index.html
var indexHTML []byte

// step はエージェントの一つの行動と、その根拠の仕様を画面へ伝える。
type step struct {
	Title    string   `json:"title"`
	Outcome  string   `json:"outcome"` // allowed、denied、info
	Detail   string   `json:"detail"`
	Evidence string   `json:"evidence,omitempty"`
	Specs    []string `json:"specs,omitempty"`
	Executed bool     `json:"executed"`
}

type documentInput struct {
	DocumentID string `json:"document_id"`
}

type documentOutput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// documents は保護される側の MCP サーバーである。認可を知らず、呼ばれれば実行する。
type documents struct {
	mu       sync.Mutex
	deleted  map[string]bool
	executed int
}

var documentBodies = map[string]documentOutput{
	publicDocument: {Title: "就業規則 2026 年版", Body: "勤務時間は 9:30 から 18:00 です。"},
	secretDocument: {Title: "2026 年度 給与テーブル", Body: "役員 A: 2,400 万円 / 部長 B: 1,350 万円"},
}

func (d *documents) read(_ context.Context, _ *mcp.CallToolRequest, input documentInput) (*mcp.CallToolResult, documentOutput, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.executed++
	document, ok := documentBodies[input.DocumentID]
	if !ok || d.deleted[input.DocumentID] {
		return nil, documentOutput{}, fmt.Errorf("document %q not found", input.DocumentID)
	}
	return nil, document, nil
}

func (d *documents) delete(_ context.Context, _ *mcp.CallToolRequest, input documentInput) (*mcp.CallToolResult, documentOutput, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.executed++
	d.deleted[input.DocumentID] = true
	return nil, documentOutput{Title: input.DocumentID, Body: "削除しました。"}, nil
}

func (d *documents) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.executed
}

func (d *documents) restore() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.deleted = map[string]bool{}
}

func (d *documents) handler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "documents", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "read_document", Description: "文書を読む"}, d.read)
	mcp.AddTool(server, &mcp.Tool{Name: "delete_document", Description: "文書を削除する"}, d.delete)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
}

type demo struct {
	realm       string
	key         *rsa.PrivateKey
	docs        *documents
	upstreamURL string
	gatewayURL  string
	caFile      string
	proxyURL    string
	workDir     string
	http        *http.Client

	mu        sync.Mutex
	clientID  string
	gateway   http.Handler
	token     string
	authReqID string
}

// publicPEM は管理画面へ貼り付ける公開鍵を返す。
func (d *demo) publicPEM() (string, error) {
	der, err := x509.MarshalPKIXPublicKey(&d.key.PublicKey)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
}

// assertion は RFC 7523 のクライアントアサーションを要求ごとに作る。
func (d *demo) assertion(clientID string) (string, error) {
	encode := base64.RawURLEncoding.EncodeToString
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return "", err
	}
	now := time.Now().Unix()
	// 登録する鍵は一つなので kid を付けない。
	header, err := json.Marshal(map[string]string{"alg": "PS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(map[string]any{
		"iss": clientID, "sub": clientID, "aud": d.realm + "/token",
		"iat": now, "exp": now + 60, "jti": encode(jti),
	})
	if err != nil {
		return "", err
	}
	input := encode(header) + "." + encode(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, d.key, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		return "", err
	}
	return input + "." + encode(signature), nil
}

// authenticated はエージェントの鍵で認証して IdMagic のエンドポイントを呼ぶ。
func (d *demo) authenticated(ctx context.Context, clientID, endpoint string, form url.Values) (int, map[string]any, error) {
	signed, err := d.assertion(clientID)
	if err != nil {
		return 0, nil, err
	}
	form.Set("client_id", clientID)
	form.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	form.Set("client_assertion", signed)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.realm+"/"+endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := d.http.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = response.Body.Close() }()
	result := map[string]any{}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponse)).Decode(&result); err != nil {
		return response.StatusCode, nil, fmt.Errorf("%s の応答を読めません: %w", endpoint, err)
	}
	return response.StatusCode, result, nil
}

func (d *demo) issueToken(ctx context.Context) (string, step) {
	d.mu.Lock()
	clientID := d.clientID
	d.mu.Unlock()
	base := step{
		Title: "秘密鍵で署名して、文書サーバー宛のトークンを取得",
		Specs: []string{"OAuth 2.0 Client Credentials (RFC 6749)", "private_key_jwt (RFC 7523)", "Resource Indicators (RFC 8707)"},
	}
	status, result, err := d.authenticated(ctx, clientID, "token", url.Values{
		"grant_type": {"client_credentials"}, "scope": {demoScope}, "resource": {gatewayResource},
	})
	if err != nil {
		base.Outcome, base.Detail = "denied", err.Error()
		return "", base
	}
	token, _ := result["access_token"].(string)
	if status != http.StatusOK || token == "" {
		base.Outcome = "denied"
		base.Detail = fmt.Sprintf("IdMagic はトークンを発行しませんでした（HTTP %d, %v）。", status, result["error"])
		return "", base
	}
	base.Outcome = "allowed"
	base.Detail = fmt.Sprintf("有効期間 %v 秒のトークンを受け取りました。共有の API キーはどこにもありません。", result["expires_in"])
	return token, base
}

// callTool は Streamable HTTP の tools/call を一つ送り、上流で実行されたかを観測する。
func (d *demo) callTool(ctx context.Context, target, credentialHeader, credential, tool, documentID string) (int, string, bool, error) {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{
			"name": tool, "arguments": map[string]string{"document_id": documentID},
			"_meta": map[string]any{
				"io.modelcontextprotocol/protocolVersion":    protocolVersion,
				"io.modelcontextprotocol/clientCapabilities": map[string]any{},
			},
		},
	})
	if err != nil {
		return 0, "", false, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(body)))
	if err != nil {
		return 0, "", false, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", protocolVersion)
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", tool)
	if credential != "" {
		request.Header.Set(credentialHeader, credential)
	}
	before := d.docs.count()
	response, err := d.http.Do(request)
	if err != nil {
		return 0, "", false, err
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponse))
	if err != nil {
		return response.StatusCode, "", false, err
	}
	return response.StatusCode, toolText(raw), d.docs.count() > before, nil
}

// toolText は JSON-RPC の応答か SSE のイベントから、ツールの結果の文章を取り出す。
func toolText(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	for line := range strings.SplitSeq(text, "\n") {
		if data, ok := strings.CutPrefix(strings.TrimSpace(line), "data:"); ok {
			text = strings.TrimSpace(data)
		}
	}
	var message struct {
		Result struct {
			StructuredContent documentOutput `json:"structuredContent"`
		} `json:"result"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(text), &message); err != nil {
		return text
	}
	if message.Error.Message != "" {
		return message.Error.Message
	}
	return strings.TrimSpace(message.Result.StructuredContent.Title + " " + message.Result.StructuredContent.Body)
}

func (d *demo) toolStep(ctx context.Context, title, target, header, credential, tool, documentID string, specs []string) step {
	status, text, executed, err := d.callTool(ctx, target, header, credential, tool, documentID)
	result := step{Title: title, Specs: specs, Executed: executed, Detail: text, Evidence: fmt.Sprintf("HTTP %d / 文書サーバーでの実行: %s", status, yesNo(executed))}
	switch {
	case err != nil:
		result.Outcome, result.Detail = "denied", err.Error()
	case status == http.StatusOK && executed:
		result.Outcome = "allowed"
	default:
		result.Outcome = "denied"
		result.Detail = "ゲートウェイが止めました。文書サーバーには要求が届いていません。"
	}
	return result
}

func yesNo(value bool) string {
	if value {
		return "あり"
	}
	return "なし"
}

func (d *demo) currentToken(ctx context.Context) (string, []step) {
	d.mu.Lock()
	token := d.token
	d.mu.Unlock()
	if token != "" {
		return token, nil
	}
	token, issued := d.issueToken(ctx)
	if token != "" {
		d.mu.Lock()
		d.token = token
		d.mu.Unlock()
	}
	return token, []step{issued}
}

var gatewaySpecs = []string{"Model Context Protocol 2026-07-28", "OAuth 2.0 Protected Resource Metadata (RFC 9728)", "関係ベースのアクセス制御 (ReBAC)"}

func (d *demo) run(ctx context.Context, scenario string) ([]step, error) {
	switch scenario {
	case "unmanaged":
		return d.runUnmanaged(ctx), nil
	case "read-public", "read-secret", "delete":
		if !d.connected() {
			return nil, errors.New("先に手順 7 で接続してください。")
		}
		token, steps := d.currentToken(ctx)
		if token == "" {
			return steps, nil
		}
		title, tool, documentID := "「就業規則を読んで」", "read_document", publicDocument
		switch scenario {
		case "read-secret":
			title, documentID = "「給与テーブルを読んで」", secretDocument
		case "delete":
			title, tool = "「就業規則を削除して」", "delete_document"
		}
		return append(steps, d.toolStep(ctx, title, d.gatewayURL, "Authorization", "Bearer "+token, tool, documentID, gatewaySpecs)), nil
	case "after-stop":
		return d.runAfterStop(ctx)
	}
	return nil, fmt.Errorf("未知のシナリオです: %s", scenario)
}

func (d *demo) runUnmanaged(ctx context.Context) []step {
	defer d.docs.restore()
	const sharedKey = "shared-api-key-everyone-uses"
	specs := []string{"共有の API キー"}
	return []step{
		d.toolStep(ctx, "「就業規則を読んで」", d.upstreamURL, "X-Api-Key", sharedKey, "read_document", publicDocument, specs),
		d.toolStep(ctx, "「給与テーブルを読んで」", d.upstreamURL, "X-Api-Key", sharedKey, "read_document", secretDocument, specs),
		d.toolStep(ctx, "「就業規則を削除して」", d.upstreamURL, "X-Api-Key", sharedKey, "delete_document", publicDocument, specs),
	}
}

// runAfterStop は、停止の前に得たトークンと新しい発行の両方を試す。
func (d *demo) runAfterStop(ctx context.Context) ([]step, error) {
	if !d.connected() {
		return nil, errors.New("先に手順 7 で接続してください。")
	}
	d.mu.Lock()
	previous := d.token
	d.mu.Unlock()
	var steps []step
	if previous != "" {
		used := d.toolStep(ctx, "手元のトークンで「就業規則を読んで」", d.gatewayURL, "Authorization", "Bearer "+previous, "read_document", publicDocument, gatewaySpecs)
		steps = append(steps, used)
	}
	token, issued := d.issueToken(ctx)
	steps = append(steps, issued)
	if token == "" {
		return steps, nil
	}
	d.mu.Lock()
	d.token = token
	d.mu.Unlock()
	return append(steps, d.toolStep(ctx, "新しいトークンで「就業規則を読んで」", d.gatewayURL, "Authorization", "Bearer "+token, "read_document", publicDocument, gatewaySpecs)), nil
}

func (d *demo) connected() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.gateway != nil
}

func (d *demo) startApproval(ctx context.Context) (map[string]any, error) {
	if !d.connected() {
		return nil, errors.New("先に手順 7 で接続してください。")
	}
	d.mu.Lock()
	clientID := d.clientID
	d.mu.Unlock()
	specs := []string{"OpenID Connect CIBA"}
	refused := step{Title: "承認なしでトークンを要求", Specs: []string{"監督下の Agent の発行制限"}}
	status, result, err := d.authenticated(ctx, clientID, "token", url.Values{
		"grant_type": {"client_credentials"}, "scope": {demoScope}, "resource": {gatewayResource},
	})
	if err != nil {
		return nil, err
	}
	if status == http.StatusOK {
		return nil, errors.New("この Agent はまだ自律の区分です。IdMagic の画面で「監督下」へ変えてください。")
	}
	refused.Outcome = "denied"
	refused.Detail = fmt.Sprintf("IdMagic は拒否しました（%v）。", result["error"])
	status, result, err = d.authenticated(ctx, clientID, "bc-authorize", url.Values{
		"scope": {"openid profile"}, "login_hint": {"alice"}, "binding_message": {"就業規則の更新を承認してください"},
	})
	if err != nil {
		return nil, err
	}
	authReqID, _ := result["auth_req_id"].(string)
	if status != http.StatusOK || authReqID == "" {
		return nil, fmt.Errorf("承認を依頼できません（HTTP %d, %v）", status, result["error"])
	}
	d.mu.Lock()
	d.authReqID = authReqID
	d.mu.Unlock()
	return map[string]any{
		"steps": []step{refused, {
			Title: "所有者 alice に帯域外で承認を依頼", Outcome: "info", Specs: specs,
			Detail: "承認の画面で alice が判断するまで、エージェントは待ちます。",
		}},
		"interval": result["interval"],
	}, nil
}

func (d *demo) pollApproval(ctx context.Context) (map[string]any, error) {
	d.mu.Lock()
	clientID, authReqID := d.clientID, d.authReqID
	d.mu.Unlock()
	if authReqID == "" {
		return nil, errors.New("承認の依頼がありません。")
	}
	status, result, err := d.authenticated(ctx, clientID, "token", url.Values{
		"grant_type": {"urn:openid:params:grant-type:ciba"}, "auth_req_id": {authReqID},
	})
	if err != nil {
		return nil, err
	}
	if code, _ := result["error"].(string); code == "authorization_pending" || code == "slow_down" {
		return map[string]any{"pending": true}, nil
	}
	d.mu.Lock()
	d.authReqID = ""
	d.mu.Unlock()
	done := step{Title: "承認の結果を受け取る", Specs: []string{"OpenID Connect CIBA"}}
	if status == http.StatusOK {
		done.Outcome = "allowed"
		done.Detail = "alice の承認を経てトークンが発行されました。"
	} else {
		done.Outcome = "denied"
		done.Detail = fmt.Sprintf("トークンは発行されませんでした（%v）。", result["error"])
	}
	return map[string]any{"pending": false, "steps": []step{done}}, nil
}

type connectInput struct {
	AgentClientID   string `json:"agent_client_id"`
	GatewayClientID string `json:"gateway_client_id"`
	GatewaySecret   string `json:"gateway_client_secret"`
}

func (d *demo) connect(ctx context.Context, input connectInput) ([]step, error) {
	input.AgentClientID = strings.TrimSpace(input.AgentClientID)
	input.GatewayClientID = strings.TrimSpace(input.GatewayClientID)
	input.GatewaySecret = strings.TrimSpace(input.GatewaySecret)
	if input.AgentClientID == "" || input.GatewayClientID == "" || input.GatewaySecret == "" {
		return nil, errors.New("三つの値をすべて入力してください。")
	}
	secretFile := filepath.Join(d.workDir, "gateway-secret")
	if err := os.WriteFile(secretFile, []byte(input.GatewaySecret+"\n"), 0o600); err != nil {
		return nil, err
	}
	configFile := filepath.Join(d.workDir, "gateway.yaml")
	config := fmt.Sprintf(`listen: 127.0.0.1:0
resource: %s
upstream: %s
idmagic:
  realm_url: %s/realms/default
  ca_file: %s
  client_id: %q
  client_secret_file: %s
mapping:
  tools:
    read_document: { resource_type: document, relation: viewer, resource_id_argument: document_id }
`, gatewayResource, d.upstreamURL, d.proxyURL, d.caFile, input.GatewayClientID, secretFile)
	if err := os.WriteFile(configFile, []byte(config), 0o600); err != nil {
		return nil, err
	}
	loaded, err := mcpgateway.Load(configFile, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("ゲートウェイの設定を読めません: %w", err)
	}
	d.mu.Lock()
	d.clientID, d.gateway, d.token = input.AgentClientID, loaded.Handler(), ""
	d.mu.Unlock()
	token, issued := d.issueToken(ctx)
	if token == "" {
		return []step{issued}, nil
	}
	d.mu.Lock()
	d.token = token
	d.mu.Unlock()
	return []step{issued}, nil
}

func (d *demo) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, _ *http.Request) {
		origin := strings.TrimSuffix(d.realm, "/realms/default")
		publicKey, err := d.publicPEM()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"public_key": publicKey, "resource": gatewayResource, "scope": demoScope,
			"console": origin + "/realms/default", "connected": d.connected(),
			"public_document": publicDocument, "secret_document": secretDocument,
		})
	})
	mux.HandleFunc("POST /api/connect", d.jsonAction(func(ctx context.Context, body []byte) (any, error) {
		var input connectInput
		if err := json.Unmarshal(body, &input); err != nil {
			return nil, err
		}
		steps, err := d.connect(ctx, input)
		return map[string]any{"steps": steps}, err
	}))
	mux.HandleFunc("POST /api/run", d.jsonAction(func(ctx context.Context, body []byte) (any, error) {
		var input struct {
			Scenario string `json:"scenario"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			return nil, err
		}
		steps, err := d.run(ctx, input.Scenario)
		return map[string]any{"steps": steps}, err
	}))
	mux.HandleFunc("POST /api/approval/start", d.jsonAction(func(ctx context.Context, _ []byte) (any, error) { return d.startApproval(ctx) }))
	mux.HandleFunc("POST /api/approval/poll", d.jsonAction(func(ctx context.Context, _ []byte) (any, error) { return d.pollApproval(ctx) }))
	return mux
}

// jsonAction は JSON の要求だけを受ける。他のサイトのフォームからの送信を CORS の事前確認で止める。
func (d *demo) jsonAction(action func(context.Context, []byte) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "application/json が必要です。"})
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, maxResponse))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		result, err := action(r.Context(), body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("応答を書けません: %v", err)
	}
}

// gatewayHandler は接続前には 503 を返し、接続後は実在のゲートウェイへ渡す。
func (d *demo) gatewayHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d.mu.Lock()
		gateway := d.gateway
		d.mu.Unlock()
		if gateway == nil {
			http.Error(w, "gateway is not connected", http.StatusServiceUnavailable)
			return
		}
		gateway.ServeHTTP(w, r)
	})
}

func newDemo(realm, workDir string, key *rsa.PrivateKey, upstreamURL, gatewayURL, proxyURL, caFile string) *demo {
	return &demo{
		realm: strings.TrimRight(realm, "/"), key: key, workDir: workDir,
		docs: &documents{deleted: map[string]bool{}}, upstreamURL: upstreamURL, gatewayURL: gatewayURL,
		proxyURL: proxyURL, caFile: caFile, http: &http.Client{Timeout: 15 * time.Second},
	}
}

func serve(listener net.Listener, handler http.Handler) {
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func main() {
	realm := flag.String("realm", "http://localhost:5173/realms/default", "IdMagic のテナントの URL")
	listen := flag.String("listen", "127.0.0.1:5180", "デモの画面の待ち受け")
	flag.Parse()
	if err := start(*realm, *listen); err != nil {
		log.Fatal(err)
	}
}

func start(realm, listen string) error {
	target, err := url.Parse(realm)
	if err != nil || target.Host == "" {
		return fmt.Errorf("realm の URL が不正です: %q", realm)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	workDir, err := os.MkdirTemp("", "idmagic-agent-demo")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	// ゲートウェイは https の realm_url だけを受け付けるので、デモ専用の TLS の前段を置く。
	origin := &url.URL{Scheme: target.Scheme, Host: target.Host}
	proxy := &httputil.ReverseProxy{Rewrite: func(request *httputil.ProxyRequest) {
		request.SetURL(origin)
		request.SetXForwarded()
	}}
	tlsFront := httptest.NewTLSServer(proxy)
	defer tlsFront.Close()
	caFile := filepath.Join(workDir, "ca.pem")
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tlsFront.Certificate().Raw})
	if err := os.WriteFile(caFile, certificate, 0o600); err != nil {
		return err
	}

	var listenConfig net.ListenConfig
	upstreamListener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	gatewayListener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	consoleListener, err := listenConfig.Listen(context.Background(), "tcp", listen)
	if err != nil {
		return err
	}
	d := newDemo(realm, workDir, key, "http://"+upstreamListener.Addr().String()+"/", "http://"+gatewayListener.Addr().String()+"/", tlsFront.URL, caFile)
	go serve(upstreamListener, d.docs.handler())
	go serve(gatewayListener, d.gatewayHandler())
	log.Printf("デモの画面: http://%s/", consoleListener.Addr())
	serve(consoleListener, d.routes())
	return nil
}
