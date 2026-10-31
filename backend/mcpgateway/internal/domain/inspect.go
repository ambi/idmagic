// Package domain は、MCP の要求の解釈と評価する操作の選択を、作用のない計算として持つ。
package domain

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"unicode/utf8"
)

// ProtocolVersion はゲートウェイが解釈する唯一の MCP の版である。
const ProtocolVersion = "2026-07-28"

const (
	MethodToolsCall = "tools/call"

	protocolVersionMetaKey = "io.modelcontextprotocol/protocolVersion"
)

// JSON-RPC のエラーコード。-32020 と -32022 は MCP の仕様が割り当てる。
const (
	CodeParseError         = -32700
	CodeInvalidRequest     = -32600
	CodeMethodNotFound     = -32601
	CodeHeaderMismatch     = -32020
	CodeUnsupportedVersion = -32022
)

// Headers はヘッダーの名前から値の一覧を返す。名前の大文字と小文字は区別しない。
type Headers func(name string) []string

// Message は解釈できた一つの JSON-RPC の要求である。
type Message struct {
	// ID は応答へそのまま返す JSON の値である。
	ID        json.RawMessage
	Method    string
	ToolName  string
	Arguments map[string]json.RawMessage
}

// Violation は Inspect の拒否で、どれも 400 で応答する。ID は読めたときだけ持つ。
type Violation struct {
	Code                    int
	Message                 string
	ID                      json.RawMessage
	UnsupportedVersionError bool
}

// Inspect は本文とヘッダーを MCP の要求として解釈する (REQ-MCPGATEWAY-002)。ここを通らない要求は
// 評価も転送もしないので、解釈が一意でない入力はすべて拒否する。
func Inspect(headers Headers, body []byte) (Message, *Violation) {
	if !utf8.Valid(body) {
		return Message{}, &Violation{Code: CodeParseError, Message: "the body is not UTF-8"}
	}
	if err := rejectAmbiguousKeys(body); err != nil {
		if errors.Is(err, errAmbiguousKey) {
			return Message{}, invalidRequest(nil, "the body has ambiguous keys")
		}
		return Message{}, &Violation{Code: CodeParseError, Message: "the body is not JSON"}
	}
	if first := bytes.TrimLeft(body, " \t\r\n"); len(first) == 0 || first[0] != '{' {
		return Message{}, invalidRequest(nil, "the body must be a single JSON-RPC request")
	}
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  *string         `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return Message{}, invalidRequest(nil, "the request has wrongly typed members")
	}
	id := validID(envelope.ID)
	if envelope.JSONRPC != "2.0" || id == nil || envelope.Method == nil {
		return Message{}, invalidRequest(id, "the body must be a JSON-RPC 2.0 request with an id")
	}
	var params struct {
		Name      *string                    `json:"name"`
		Arguments map[string]json.RawMessage `json:"arguments"`
		Meta      map[string]json.RawMessage `json:"_meta"`
	}
	if len(envelope.Params) > 0 {
		if err := json.Unmarshal(envelope.Params, &params); err != nil {
			return Message{}, invalidRequest(id, "params must be an object with well-typed members")
		}
	}
	message := Message{ID: id, Method: *envelope.Method, Arguments: params.Arguments}
	if message.Method == MethodToolsCall {
		if params.Name == nil || *params.Name == "" {
			return Message{}, invalidRequest(id, "tools/call requires params.name")
		}
		message.ToolName = *params.Name
	}
	if violation := checkMirroredHeaders(headers, message, params.Meta); violation != nil {
		return Message{}, violation
	}
	return message, nil
}

func invalidRequest(id json.RawMessage, text string) *Violation {
	return &Violation{Code: CodeInvalidRequest, Message: text, ID: id}
}

// validID は文字列か数の id だけを返す。
func validID(raw json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] == '{' || trimmed[0] == '[' || trimmed[0] == 't' || trimmed[0] == 'f' || trimmed[0] == 'n' {
		return nil
	}
	return trimmed
}

func checkMirroredHeaders(headers Headers, message Message, meta map[string]json.RawMessage) *Violation {
	mismatch := func(text string) *Violation {
		return &Violation{Code: CodeHeaderMismatch, Message: text, ID: message.ID}
	}
	headerVersion, ok := singleHeader(headers, "MCP-Protocol-Version")
	if !ok {
		return mismatch("exactly one MCP-Protocol-Version header is required")
	}
	if method, ok := singleHeader(headers, "Mcp-Method"); !ok || method != message.Method {
		return mismatch("Mcp-Method must match the body")
	}
	if message.Method == MethodToolsCall {
		name, ok := singleHeader(headers, "Mcp-Name")
		if !ok {
			return mismatch("exactly one Mcp-Name header is required")
		}
		decoded, ok := decodeMirrored(name)
		if !ok || decoded != message.ToolName {
			return mismatch("Mcp-Name must match the body")
		}
	}
	var bodyVersion string
	if err := json.Unmarshal(meta[protocolVersionMetaKey], &bodyVersion); err != nil || bodyVersion != headerVersion {
		return mismatch("MCP-Protocol-Version must match params._meta")
	}
	if bodyVersion != ProtocolVersion {
		return &Violation{
			Code: CodeUnsupportedVersion, Message: "unsupported protocol version",
			ID: message.ID, UnsupportedVersionError: true,
		}
	}
	return nil
}

func singleHeader(headers Headers, name string) (string, bool) {
	values := headers(name)
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

// decodeMirrored は MCP の =?base64?...?= の形を復号する。
func decodeMirrored(value string) (string, bool) {
	encoded, found := strings.CutPrefix(value, "=?base64?")
	if !found {
		return value, true
	}
	encoded, found = strings.CutSuffix(encoded, "?=")
	if !found {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || !utf8.Valid(decoded) {
		return "", false
	}
	return string(decoded), true
}

var errAmbiguousKey = errors.New("ambiguous key")

// rejectAmbiguousKeys は本文を一度だけ読み切り、構文の誤り、後続のデータ、大文字と小文字を区別せずに
// 同じになるキーの重複を拒否する。encoding/json は重複を後勝ちで読み、キーの大文字と小文字も区別しない。
// 別の実装の上流が別の値を読む余地を残さない。
func rejectAmbiguousKeys(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := walkValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing data")
	}
	return nil
}

func walkValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if delimiter == '[' {
		for decoder.More() {
			if err := walkValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	}
	seen := map[string]bool{}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := key.(string)
		if !ok {
			return errors.New("object key is not a string")
		}
		folded := strings.ToLower(name)
		if seen[folded] {
			return errAmbiguousKey
		}
		seen[folded] = true
		if err := walkValue(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
