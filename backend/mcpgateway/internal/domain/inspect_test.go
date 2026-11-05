package domain

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func toolCallHeaders(name string) http.Header {
	headers := http.Header{}
	headers.Set("Mcp-Protocol-Version", ProtocolVersion)
	headers.Set("Mcp-Method", MethodToolsCall)
	headers.Set("Mcp-Name", name)
	return headers
}

const validToolCall = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{"id":"d1"},` +
	`"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`

func TestInspectReadsAToolCall(t *testing.T) {
	message, violation := Inspect(toolCallHeaders("read").Values, []byte(validToolCall))
	if violation != nil {
		t.Fatalf("violation: %+v", violation)
	}
	if message.Method != MethodToolsCall || message.ToolName != "read" || string(message.Arguments["id"]) != `"d1"` {
		t.Fatalf("message = %+v", message)
	}
}

func TestInspectDecodesAnEncodedName(t *testing.T) {
	name := "=?base64?" + base64.StdEncoding.EncodeToString([]byte("read")) + "?="
	if _, violation := Inspect(toolCallHeaders(name).Values, []byte(validToolCall)); violation != nil {
		t.Fatalf("violation: %+v", violation)
	}
}

// 実装ごとに読み方が割れる入力は、どれも拒否する。
//
//spec:covers REQ-MCPGATEWAY-002, EX-MCPGATEWAY-002-01: バッチ、重複キー、大文字と小文字の違うキー、不正な UTF-8、後続データ、ヘッダーの不一致、古い版の拒否と JSON-RPC のコード
func TestInspectRejectsAmbiguousRequests(t *testing.T) {
	meta := `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}`
	cases := []struct {
		name    string
		body    string
		headers func(http.Header)
		code    int
	}{
		{name: "batch", body: `[` + validToolCall + `]`, code: CodeInvalidRequest},
		{name: "duplicate method", body: `{"jsonrpc":"2.0","id":1,"method":"tools/list","method":"tools/call","params":{"name":"read",` + meta + `}}`, code: CodeInvalidRequest},
		{name: "case variant name", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","NAME":"delete",` + meta + `}}`, code: CodeInvalidRequest},
		{name: "case variant in arguments", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{"id":"a","ID":"b"},` + meta + `}}`, code: CodeInvalidRequest},
		{name: "invalid utf-8", body: "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"re\xffad\"}}", code: CodeParseError},
		{name: "trailing data", body: validToolCall + `{}`, code: CodeParseError},
		{name: "not json", body: `{`, code: CodeParseError},
		{name: "empty", body: ``, code: CodeParseError},
		{name: "notification", body: `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read",` + meta + `}}`, code: CodeInvalidRequest},
		{name: "null id", body: `{"jsonrpc":"2.0","id":null,"method":"tools/call","params":{"name":"read",` + meta + `}}`, code: CodeInvalidRequest},
		{name: "wrong jsonrpc", body: `{"jsonrpc":"1.0","id":1,"method":"tools/call","params":{"name":"read",` + meta + `}}`, code: CodeInvalidRequest},
		{name: "array params", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":[]}`, code: CodeInvalidRequest},
		{name: "no tool name", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{` + meta + `}}`, code: CodeInvalidRequest},
		{name: "method header differs", body: validToolCall, headers: func(h http.Header) { h.Set("Mcp-Method", "tools/list") }, code: CodeHeaderMismatch},
		{name: "name header differs", body: validToolCall, headers: func(h http.Header) { h.Set("Mcp-Name", "delete") }, code: CodeHeaderMismatch},
		{name: "two name headers", body: validToolCall, headers: func(h http.Header) { h.Add("Mcp-Name", "read") }, code: CodeHeaderMismatch},
		{name: "no version header", body: validToolCall, headers: func(h http.Header) { h.Del("Mcp-Protocol-Version") }, code: CodeHeaderMismatch},
		{name: "broken encoded name", body: validToolCall, headers: func(h http.Header) { h.Set("Mcp-Name", "=?base64?***?=") }, code: CodeHeaderMismatch},
		{name: "version header differs from body", body: validToolCall, headers: func(h http.Header) { h.Set("Mcp-Protocol-Version", "2025-11-25") }, code: CodeHeaderMismatch},
		{
			name:    "older version",
			body:    `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","_meta":{"io.modelcontextprotocol/protocolVersion":"2025-11-25"}}}`,
			headers: func(h http.Header) { h.Set("Mcp-Protocol-Version", "2025-11-25") },
			code:    CodeUnsupportedVersion,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			headers := toolCallHeaders("read")
			if tc.headers != nil {
				tc.headers(headers)
			}
			_, violation := Inspect(headers.Values, []byte(tc.body))
			if violation == nil {
				t.Fatal("accepted")
			}
			if violation.Code != tc.code {
				t.Fatalf("violation = %+v, want code %d", violation, tc.code)
			}
		})
	}
}

// FuzzInspect は、受理した要求の方法とツール名がヘッダーと一致し、重複キーの後勝ちで読み替えられないことを確かめる。
func FuzzInspect(f *testing.F) {
	f.Add([]byte(validToolCall), "tools/call", "read")
	f.Add([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","method":"tools/call"}`), "tools/call", "read")
	f.Fuzz(func(t *testing.T, body []byte, method, name string) {
		headers := http.Header{}
		headers.Set("Mcp-Protocol-Version", ProtocolVersion)
		headers.Set("Mcp-Method", method)
		headers.Set("Mcp-Name", name)
		message, violation := Inspect(headers.Values, body)
		if violation != nil {
			return
		}
		if message.Method != method {
			t.Fatalf("method %q accepted against header %q", message.Method, method)
		}
		if message.Method == MethodToolsCall {
			if decoded, ok := decodeMirrored(name); !ok || decoded != message.ToolName {
				t.Fatalf("tool %q accepted against header %q", message.ToolName, name)
			}
		}
	})
}

//spec:covers REQ-MCPGATEWAY-003: 宣言のある操作の選択、宣言のないツールの監査専用の評価、長い名前と引数の欠落の無評価の拒否、未知の方法
func TestDecideSelectsTheOperation(t *testing.T) {
	mapping := Mapping{
		Methods: map[string]OperationRule{"tools/list": {ResourceType: "mcp_server", Relation: "list", ResourceID: "main"}},
		Tools: map[string]OperationRule{
			"read":  {ResourceType: "document", Relation: "viewer", ResourceIDArgument: "id"},
			"fixed": {ResourceType: "document", Relation: "viewer", ResourceID: "d9"},
		},
	}
	long := make([]rune, 129)
	for i := range long {
		long[i] = 'x'
	}
	cases := []struct {
		name string
		in   Message
		kind EvaluationKind
		op   Operation
		tool string
	}{
		{"argument", Message{Method: MethodToolsCall, ToolName: "read", Arguments: idArgument(`"d1"`)}, EvaluateAndForward, Operation{"document", "viewer", "d1"}, "read"},
		{"fixed id", Message{Method: MethodToolsCall, ToolName: "fixed"}, EvaluateAndForward, Operation{"document", "viewer", "d9"}, "fixed"},
		{"missing argument", Message{Method: MethodToolsCall, ToolName: "read"}, DenyWithoutEvaluation, Operation{}, "read"},
		{"number argument", Message{Method: MethodToolsCall, ToolName: "read", Arguments: idArgument(`1`)}, DenyWithoutEvaluation, Operation{}, "read"},
		{"empty argument", Message{Method: MethodToolsCall, ToolName: "read", Arguments: idArgument(`""`)}, DenyWithoutEvaluation, Operation{}, "read"},
		{"unmapped tool", Message{Method: MethodToolsCall, ToolName: "drop"}, EvaluateForAuditOnly, Operation{"mcp_tool", "unmapped", "drop"}, UnmappedTool},
		{"128 runes is evaluated", Message{Method: MethodToolsCall, ToolName: string(long[:128])}, EvaluateForAuditOnly, Operation{"mcp_tool", "unmapped", string(long[:128])}, UnmappedTool},
		{"129 runes is not", Message{Method: MethodToolsCall, ToolName: string(long)}, DenyWithoutEvaluation, Operation{}, UnmappedTool},
		{"mapped method", Message{Method: "tools/list"}, EvaluateAndForward, Operation{"mcp_server", "list", "main"}, ""},
		{"unknown method", Message{Method: "initialize"}, MethodNotFound, Operation{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.in, mapping)
			if got.Kind != tc.kind || got.Operation != tc.op || got.ToolLabel != tc.tool {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func idArgument(raw string) map[string]json.RawMessage {
	return map[string]json.RawMessage{"id": json.RawMessage(raw)}
}

//spec:covers REQ-MCPGATEWAY-003: 名指したキーだけを値を変えずに attributes にすること、宣言のない操作は送らないこと、16 KiB の境界
func TestDecideSelectsDeclaredAttributes(t *testing.T) {
	mapping := Mapping{Tools: map[string]OperationRule{
		"pay":   {ResourceType: "account", Relation: "payer", ResourceID: "a1", AttributeArguments: []string{"amount", "currency"}},
		"plain": {ResourceType: "account", Relation: "payer", ResourceID: "a1"},
	}}
	call := func(tool string, arguments map[string]json.RawMessage) Message {
		return Message{Method: MethodToolsCall, ToolName: tool, Arguments: arguments}
	}
	cases := []struct {
		name       string
		in         Message
		kind       EvaluationKind
		attributes string
	}{
		{"declared keys only", call("pay", map[string]json.RawMessage{
			"amount": json.RawMessage(`1e2`), "currency": json.RawMessage(`"USD"`), "secret": json.RawMessage(`"s"`),
		}), EvaluateAndForward, `{"amount":1e2,"currency":"USD"}`},
		{"structured value is kept", call("pay", map[string]json.RawMessage{"amount": json.RawMessage(`{"value": 5}`)}), EvaluateAndForward, `{"amount":{"value":5}}`},
		{"missing keys are omitted", call("pay", map[string]json.RawMessage{"other": json.RawMessage(`1`)}), EvaluateAndForward, `{}`},
		{"no arguments", call("pay", nil), EvaluateAndForward, `{}`},
		{"undeclared tool sends none", call("plain", map[string]json.RawMessage{"amount": json.RawMessage(`1`)}), EvaluateAndForward, ""},
		{"16 KiB is evaluated", call("pay", map[string]json.RawMessage{"amount": sizedString(maxAttributesBytes - len(`{"amount":}`))}), EvaluateAndForward, ""},
		{"over 16 KiB is not", call("pay", map[string]json.RawMessage{"amount": sizedString(maxAttributesBytes - len(`{"amount":}`) + 1)}), DenyWithoutEvaluation, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(tc.in, mapping)
			if got.Kind != tc.kind {
				t.Fatalf("got %+v", got)
			}
			if tc.kind == DenyWithoutEvaluation {
				if got.Reason != "attributes_too_large" || got.Attributes != "" {
					t.Fatalf("got %+v", got)
				}
				return
			}
			if strings.HasPrefix(tc.name, "16 KiB") {
				if len(got.Attributes) != maxAttributesBytes {
					t.Fatalf("len=%d", len(got.Attributes))
				}
				return
			}
			if got.Attributes != tc.attributes {
				t.Fatalf("attributes=%s", got.Attributes)
			}
		})
	}
}

// sizedString は直列化すると n バイトになる JSON の文字列である。
func sizedString(n int) json.RawMessage {
	return json.RawMessage(`"` + strings.Repeat("x", n-2) + `"`)
}
