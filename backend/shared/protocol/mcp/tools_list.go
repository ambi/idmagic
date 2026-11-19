package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// MethodToolsList は一覧の取得の JSON-RPC のメソッドである。
const MethodToolsList = "tools/list"

// ListedTool は tools/list の一つのツールのうち、カタログに使う項目である。
// Arguments は inputSchema.properties のキーを辞書順に並べたもので、件数と長さは切り詰めない。
type ListedTool struct {
	Name        string
	Title       string
	Description string
	Arguments   []string
}

// ToolsListRequest は一ページ分の tools/list の要求の本文を作る。cursor が空なら最初のページである。
func ToolsListRequest(cursor string) ([]byte, error) {
	params := map[string]any{"_meta": map[string]any{
		protocolVersionMetaKey:                       ProtocolVersion,
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}}
	if cursor != "" {
		params["cursor"] = cursor
	}
	return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": MethodToolsList, "params": params})
}

// ParseToolsListPage は JSON-RPC の応答か、SSE の data 行のうち最後の応答から、ツールと nextCursor を読む。
// エラーの応答と result のない応答は失敗にする。
func ParseToolsListPage(raw []byte) ([]ListedTool, string, error) {
	text := strings.TrimSpace(string(raw))
	for line := range strings.SplitSeq(text, "\n") {
		if data, ok := strings.CutPrefix(strings.TrimSpace(line), "data:"); ok {
			text = strings.TrimSpace(data)
		}
	}
	var message struct {
		Result *struct {
			Tools []struct {
				Name        string `json:"name"`
				Title       string `json:"title"`
				Description string `json:"description"`
				InputSchema struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
			NextCursor string `json:"nextCursor"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal([]byte(text), &message); err != nil {
		return nil, "", fmt.Errorf("tools/list response is not JSON-RPC: %w", err)
	}
	if len(message.Error) > 0 || message.Result == nil {
		return nil, "", errors.New("tools/list returned no result")
	}
	tools := make([]ListedTool, 0, len(message.Result.Tools))
	for _, listed := range message.Result.Tools {
		arguments := make([]string, 0, len(listed.InputSchema.Properties))
		for name := range listed.InputSchema.Properties {
			arguments = append(arguments, name)
		}
		slices.Sort(arguments)
		tools = append(tools, ListedTool{Name: listed.Name, Title: listed.Title, Description: listed.Description, Arguments: arguments})
	}
	return tools, message.Result.NextCursor, nil
}
