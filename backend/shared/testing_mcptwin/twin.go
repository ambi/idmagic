package testing_mcptwin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// Fault は、模倣が次の要求に返す応答である。Status が 0 なら、その要求には模倣の通常の応答を返す。
type Fault struct {
	Status      int
	ContentType string
	Body        string
}

// Twin は MCP 2026-07-28 の Streamable HTTP の上流の模倣である。
//
// POST はパスに依らず MCP の入口として扱い、tools/list と subscriptions/listen に応える。
// GET は Resource から RFC 9728 §3.1 で導いた URL にだけ Protected Resource Metadata を返し、ほかは 404 にする。
// 設定の欄は要求を送る前に置き、要求の間に変えるときはメソッドを使う。
type Twin struct {
	// PageSize は tools/list の一ページのツールの数である。0 以下ならすべてを一ページで返す。
	PageSize int
	// Cursors は i 番目のページの後に返す nextCursor である。足りない分は "page-<n>" にする。空文字列も有効なカーソルである。
	Cursors []string
	// TTLMs と CacheScope は tools/list の結果に載せるキャッシュの指示である。0 と空なら載せない。
	TTLMs      int
	CacheScope string
	// Supported は受け付ける版である。
	Supported []string
	// Tokens は受け付けるアクセストークンと、そのスコープである。nil ならトークンを求めない。
	Tokens map[string][]string
	// RequiredScope は Tokens を求めるときに tools/list と subscriptions/listen が要るスコープである。
	RequiredScope string
	// ListChanged が偽なら、subscriptions/listen の受信確認に toolsListChanged を載せず、変更を通知しない。
	ListChanged bool
	// Resource と以下の欄は Protected Resource Metadata の値である。Resource が空なら PRM を返さない。
	Resource               string
	AuthorizationServers   []string
	BearerMethodsSupported []string
	ScopesSupported        []string
	ResourceDocumentation  string
	DPoPSigningAlgs        []string

	mu        sync.Mutex
	tools     []map[string]any
	faults    []Fault
	listeners map[*listener]struct{}
	script    *listenScript
	requests  []string
}

// listenScript は、次の subscriptions/listen が受信確認の後に送る変更通知の数と、続けて正常に終えるかである。
type listenScript struct {
	changes int
	end     bool
}

type listener struct {
	changes chan struct{}
	end     chan struct{}
}

// NewTwin は、ツールを持たず、MCP 2026-07-28 だけを受け付け、トークンを求めず、変更を通知する模倣を返す。
func NewTwin() *Twin {
	return &Twin{Supported: []string{RecordedProtocolVersion}, ListChanged: true, listeners: map[*listener]struct{}{}}
}

// setup は記録の往復の前に模倣へ置く状態である。
type setup struct {
	Tools                  []map[string]any    `json:"tools"`
	PageSize               int                 `json:"page_size"`
	Cursors                []string            `json:"cursors"`
	Supported              []string            `json:"supported"`
	Tokens                 map[string][]string `json:"tokens"`
	RequiredScope          string              `json:"required_scope"`
	ListChanged            *bool               `json:"list_changed"`
	ListChanges            int                 `json:"list_changes"`
	End                    bool                `json:"end"`
	Resource               string              `json:"resource"`
	AuthorizationServers   []string            `json:"authorization_servers"`
	BearerMethodsSupported []string            `json:"bearer_methods_supported"`
	ScopesSupported        []string            `json:"scopes_supported"`
	ResourceDocumentation  string              `json:"resource_documentation"`
}

// apply は設定を NewTwin の既定へ戻し、記録の setup を置く。キャッシュの指示は記録の例の値にする。
func (t *Twin) apply(raw json.RawMessage) error {
	var s setup
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.PageSize, t.Cursors, t.TTLMs, t.CacheScope = s.PageSize, s.Cursors, 300000, "public"
	t.Supported = []string{RecordedProtocolVersion}
	if s.Supported != nil {
		t.Supported = s.Supported
	}
	t.Tokens, t.RequiredScope = s.Tokens, s.RequiredScope
	t.ListChanged = s.ListChanged == nil || *s.ListChanged
	t.Resource, t.AuthorizationServers, t.BearerMethodsSupported = s.Resource, s.AuthorizationServers, s.BearerMethodsSupported
	t.ScopesSupported, t.ResourceDocumentation, t.DPoPSigningAlgs = s.ScopesSupported, s.ResourceDocumentation, nil
	t.tools = s.Tools
	t.script = nil
	if s.ListChanges > 0 || s.End {
		t.script = &listenScript{changes: s.ListChanges, end: s.End}
	}
	return nil
}

// AddTool はツールを一覧の末尾に足し、開いている購読へ変更を通知する。tool は tools/list の要素の JSON である。
func (t *Twin) AddTool(tool map[string]any) {
	t.mu.Lock()
	t.tools = append(t.tools, tool)
	t.mu.Unlock()
	t.notify()
}

// RemoveTool は名前の一致するツールを除き、開いている購読へ変更を通知する。
func (t *Twin) RemoveTool(name string) {
	t.mu.Lock()
	t.tools = slices.DeleteFunc(t.tools, func(tool map[string]any) bool { return tool["name"] == name })
	t.mu.Unlock()
	t.notify()
}

// EndSubscriptions は開いている購読を、subscriptions/listen の成功の応答で正常に終える。
func (t *Twin) EndSubscriptions() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for l := range t.listeners {
		close(l.end)
		delete(t.listeners, l)
	}
}

// Subscribers は開いている購読の数である。
func (t *Twin) Subscribers() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.listeners)
}

// InjectFaults は、次の要求から順に一つずつ返す応答を足す。
func (t *Twin) InjectFaults(faults ...Fault) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.faults = append(t.faults, faults...)
}

// Requests は受け取った要求を "<HTTP の方式> <JSON-RPC の方式または パス>" として順に返す。
func (t *Twin) Requests() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.requests)
}

func (t *Twin) notify() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ListChanged {
		return
	}
	for l := range t.listeners {
		select {
		case l.changes <- struct{}{}:
		default:
		}
	}
}

func (t *Twin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fault, ok := t.nextFault(); ok {
		t.record(r.Method + " " + r.URL.Path)
		if fault.ContentType != "" {
			w.Header().Set("Content-Type", fault.ContentType)
		}
		w.WriteHeader(fault.Status)
		_, _ = w.Write([]byte(fault.Body))
		return
	}
	switch r.Method {
	case http.MethodGet:
		t.record(r.Method + " " + r.URL.Path)
		t.serveMetadata(w, r)
	case http.MethodPost:
		t.serveMCP(w, r)
	default:
		t.record(r.Method + " " + r.URL.Path)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (t *Twin) nextFault() (Fault, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.faults) == 0 {
		return Fault{}, false
	}
	fault := t.faults[0]
	t.faults = t.faults[1:]
	return fault, fault.Status != 0
}

func (t *Twin) record(request string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.requests = append(t.requests, request)
}

// metadataPath は RFC 9728 §3.1 に従い、資源の path の末尾の / を除いて well-known の後へ継ぐ。
func metadataPath(resource string) (string, bool) {
	parsed, err := url.Parse(resource)
	if err != nil || parsed.Host == "" {
		return "", false
	}
	return "/.well-known/oauth-protected-resource" + strings.TrimSuffix(parsed.EscapedPath(), "/"), true
}

func (t *Twin) serveMetadata(w http.ResponseWriter, r *http.Request) {
	t.mu.Lock()
	defer t.mu.Unlock()
	path, ok := metadataPath(t.Resource)
	if !ok || r.URL.EscapedPath() != path {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	document := map[string]any{"resource": t.Resource}
	for name, values := range map[string][]string{
		"authorization_servers": t.AuthorizationServers, "bearer_methods_supported": t.BearerMethodsSupported,
		"scopes_supported": t.ScopesSupported, "dpop_signing_alg_values_supported": t.DPoPSigningAlgs,
	} {
		if len(values) > 0 { // RFC 9728 §3.2：値のない欄は省く。
			document[name] = values
		}
	}
	if t.ResourceDocumentation != "" {
		document["resource_documentation"] = t.ResourceDocumentation
	}
	writeJSON(w, http.StatusOK, document)
}

// rpcRequest は模倣が読む JSON-RPC の要求である。
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  struct {
		Meta struct {
			ProtocolVersion    *string         `json:"io.modelcontextprotocol/protocolVersion"`
			ClientCapabilities json.RawMessage `json:"io.modelcontextprotocol/clientCapabilities"`
		} `json:"_meta"`
		Cursor        *string `json:"cursor"`
		Notifications struct {
			ToolsListChanged bool `json:"toolsListChanged"`
		} `json:"notifications"`
	} `json:"params"`
}

func (t *Twin) serveMCP(w http.ResponseWriter, r *http.Request) {
	var call rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&call); err != nil || call.JSONRPC != "2.0" || len(call.ID) == 0 {
		t.record("POST (unreadable)")
		writeRPCError(w, http.StatusBadRequest, json.RawMessage("null"), -32700, "Parse error", nil)
		return
	}
	t.record("POST " + call.Method)
	if !t.checkVersion(w, r, call) || !t.authorize(w, r) {
		return
	}
	switch call.Method {
	case "tools/list":
		t.serveToolsList(w, call)
	case "subscriptions/listen":
		t.serveListen(w, r, call)
	default:
		writeRPCError(w, http.StatusNotFound, call.ID, -32601, "Method not found", nil)
	}
}

// checkVersion は basic/index の必須の _meta と、Streamable HTTP の版のヘッダーとの一致と、対応する版を確かめる。
func (t *Twin) checkVersion(w http.ResponseWriter, r *http.Request, call rpcRequest) bool {
	meta := call.Params.Meta
	if meta.ProtocolVersion == nil || len(meta.ClientCapabilities) == 0 {
		writeRPCError(w, http.StatusBadRequest, call.ID, -32602, "Invalid params: missing required _meta fields", nil)
		return false
	}
	if header := r.Header.Get("Mcp-Protocol-Version"); header != *meta.ProtocolVersion {
		writeRPCError(w, http.StatusBadRequest, call.ID, -32020,
			fmt.Sprintf("Header mismatch: MCP-Protocol-Version header value '%s' does not match body value '%s'", header, *meta.ProtocolVersion), nil)
		return false
	}
	t.mu.Lock()
	supported := slices.Clone(t.Supported)
	t.mu.Unlock()
	if !slices.Contains(supported, *meta.ProtocolVersion) {
		writeRPCError(w, http.StatusBadRequest, call.ID, -32022, "Unsupported protocol version",
			map[string]any{"supported": supported, "requested": *meta.ProtocolVersion})
		return false
	}
	return true
}

// authorize は Tokens があるときだけ Bearer のトークンとスコープを確かめる（MCP の Authorization の節）。
func (t *Twin) authorize(w http.ResponseWriter, r *http.Request) bool {
	t.mu.Lock()
	tokens, required := t.Tokens, t.RequiredScope
	t.mu.Unlock()
	if tokens == nil {
		return true
	}
	metadata := "http://" + r.Host + "/.well-known/oauth-protected-resource"
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	scopes, known := tokens[token]
	if !known {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata=%q, scope=%q`, metadata, required))
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	if required != "" && !slices.Contains(scopes, required) {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q, resource_metadata=%q`, required, metadata))
		w.WriteHeader(http.StatusForbidden)
		return false
	}
	return true
}

func (t *Twin) serveToolsList(w http.ResponseWriter, call rpcRequest) {
	t.mu.Lock()
	defer t.mu.Unlock()
	offset := 0
	if call.Params.Cursor != nil {
		page, ok := t.pageOf(*call.Params.Cursor)
		if !ok {
			writeRPCError(w, http.StatusBadRequest, call.ID, -32602, "Invalid params: unknown cursor", nil)
			return
		}
		offset = page * t.pageSize()
	}
	end := min(offset+t.pageSize(), len(t.tools))
	tools := slices.Clone(t.tools[min(offset, len(t.tools)):end])
	if tools == nil {
		tools = []map[string]any{}
	}
	result := map[string]any{"resultType": "complete", "tools": tools}
	if end < len(t.tools) {
		result["nextCursor"] = t.cursorAfter(offset / t.pageSize())
	}
	if t.TTLMs > 0 {
		result["ttlMs"] = t.TTLMs
	}
	if t.CacheScope != "" {
		result["cacheScope"] = t.CacheScope
	}
	writeJSON(w, http.StatusOK, map[string]any{"jsonrpc": "2.0", "id": call.ID, "result": result})
}

func (t *Twin) pageSize() int {
	if t.PageSize <= 0 {
		return max(len(t.tools), 1)
	}
	return t.PageSize
}

// cursorAfter は page 番目（0 から）のページの後に返すカーソルである。
func (t *Twin) cursorAfter(page int) string {
	if page < len(t.Cursors) {
		return t.Cursors[page]
	}
	return "page-" + strconv.Itoa(page+1)
}

// pageOf はカーソルが指すページの番号を返す。
func (t *Twin) pageOf(cursor string) (int, bool) {
	for page := range len(t.tools) {
		if t.cursorAfter(page) == cursor {
			return page + 1, true
		}
	}
	return 0, false
}

// serveListen は basic/patterns/subscriptions の購読を SSE で返す。受信確認を最初に送り、
// toolsListChanged を受け付けたときだけ変更を通知し、正常に終えるときは要求の id で成功の応答を送る。
func (t *Twin) serveListen(w http.ResponseWriter, r *http.Request, call rpcRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeRPCError(w, http.StatusInternalServerError, call.ID, -32603, "Internal error", nil)
		return
	}
	t.mu.Lock()
	granted := call.Params.Notifications.ToolsListChanged && t.ListChanged
	script := t.script
	t.script = nil
	l := &listener{changes: make(chan struct{}, 1), end: make(chan struct{})}
	if script == nil {
		t.listeners[l] = struct{}{}
	}
	t.mu.Unlock()
	defer func() {
		t.mu.Lock()
		delete(t.listeners, l)
		t.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	meta := map[string]any{"io.modelcontextprotocol/subscriptionId": call.ID}
	notifications := map[string]any{}
	if granted {
		notifications["toolsListChanged"] = true
	}
	send := func(message map[string]any) {
		encoded, err := json.Marshal(message)
		if err != nil {
			return
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", encoded)
		flusher.Flush()
	}
	changed := func() {
		send(map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed", "params": map[string]any{"_meta": meta}})
	}
	closeGracefully := func() {
		send(map[string]any{"jsonrpc": "2.0", "id": call.ID, "result": map[string]any{"resultType": "complete", "_meta": meta}})
	}
	send(map[string]any{"jsonrpc": "2.0", "method": "notifications/subscriptions/acknowledged", "params": map[string]any{"_meta": meta, "notifications": notifications}})

	if script != nil {
		for range script.changes {
			if granted {
				changed()
			}
		}
		if script.end {
			closeGracefully()
		}
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-l.end:
			closeGracefully()
			return
		case <-l.changes:
			if granted {
				changed()
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	encoded, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}

func writeRPCError(w http.ResponseWriter, status int, id json.RawMessage, code int, message string, data any) {
	failure := map[string]any{"code": code, "message": message}
	if data != nil {
		failure["data"] = data
	}
	writeJSON(w, status, map[string]any{"jsonrpc": "2.0", "id": id, "error": failure})
}
