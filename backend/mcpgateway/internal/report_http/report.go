// Package report_http は、上流の MCP サーバーの tools/list を集め、公開契約 POST /resource-access/tools へ報告する。
// ワイヤの型は TypeSpec に合わせてここで持ち、サーバーの内部の型を共有しない。
package report_http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
)

// 上限は REQ-MCPGATEWAY-011 と、IdMagic が受け入れる REQ-OAUTH2-086 の上限に揃える。
const (
	maxTools            = 100
	maxPages            = 10
	maxTitle            = 256
	maxDescription      = 2048
	maxArguments        = 32
	maxPageBytes        = 4 << 20
	maxReportErrorBytes = 1 << 16
)

// Tool は報告する一つのツールである。
type Tool struct {
	Name        string   `json:"name"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Arguments   []string `json:"arguments,omitempty"`
}

// Result は報告した件数と、上流の一覧をすべて取得できたかである。
type Result struct {
	Tools    int
	Complete bool
}

// Reporter は上流から一覧を集め、評価用のクライアントの資格情報で報告する。
type Reporter struct {
	Upstream     string
	ReportURL    string
	ClientID     string
	ClientSecret string
	// UpstreamClient は上流へ、IdMagicClient は IdMagic へ送る。資格情報は IdMagic へだけ送る。
	UpstreamClient *http.Client
	IdMagicClient  *http.Client
}

// Report は一覧を集めて報告する。最初のページを取得できなければ報告しない。
func (r Reporter) Report(ctx context.Context) (Result, error) {
	tools, complete, err := r.collect(ctx)
	if err != nil {
		return Result{}, err
	}
	if err := r.send(ctx, tools, complete); err != nil {
		return Result{}, err
	}
	return Result{Tools: len(tools), Complete: complete}, nil
}

// collect は nextCursor をたどる。二ページ目以降の失敗と上限の超過は、取得できた分を不完全として返す。
func (r Reporter) collect(ctx context.Context) ([]Tool, bool, error) {
	var tools []Tool
	cursor := ""
	for page := range maxPages {
		listed, next, err := r.fetchPage(ctx, cursor)
		if err != nil {
			if page == 0 {
				return nil, false, err
			}
			return tools, false, nil
		}
		for _, tool := range listed {
			if len(tools) == maxTools {
				return tools, false, nil
			}
			tools = append(tools, tool)
		}
		if next == "" {
			return tools, true, nil
		}
		cursor = next
	}
	return tools, false, nil
}

type listedTool struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	InputSchema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	} `json:"inputSchema"`
}

func (r Reporter) fetchPage(ctx context.Context, cursor string) ([]Tool, string, error) {
	params := map[string]any{"_meta": map[string]any{
		"io.modelcontextprotocol/protocolVersion":    domain.ProtocolVersion,
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}}
	if cursor != "" {
		params["cursor"] = cursor
	}
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": params})
	if err != nil {
		return nil, "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.Upstream, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", domain.ProtocolVersion)
	request.Header.Set("Mcp-Method", "tools/list")
	response, err := r.UpstreamClient.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxPageBytes))
	if err != nil {
		return nil, "", err
	}
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("tools/list returned HTTP %d", response.StatusCode)
	}
	return parsePage(raw)
}

// parsePage は JSON-RPC の応答か、SSE の data 行の最後の応答を読む。
func parsePage(raw []byte) ([]Tool, string, error) {
	text := strings.TrimSpace(string(raw))
	for line := range strings.SplitSeq(text, "\n") {
		if data, ok := strings.CutPrefix(strings.TrimSpace(line), "data:"); ok {
			text = strings.TrimSpace(data)
		}
	}
	var message struct {
		Result *struct {
			Tools      []listedTool `json:"tools"`
			NextCursor string       `json:"nextCursor"`
		} `json:"result"`
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(text), &message); err != nil {
		return nil, "", fmt.Errorf("tools/list response is not JSON-RPC: %w", err)
	}
	if message.Error != nil || message.Result == nil {
		return nil, "", errors.New("tools/list returned no result")
	}
	tools := make([]Tool, 0, len(message.Result.Tools))
	for _, listed := range message.Result.Tools {
		arguments := make([]string, 0, len(listed.InputSchema.Properties))
		for name := range listed.InputSchema.Properties {
			arguments = append(arguments, name)
		}
		slices.Sort(arguments)
		tools = append(tools, Tool{
			Name: listed.Name, Title: truncate(listed.Title, maxTitle), Description: truncate(listed.Description, maxDescription),
			Arguments: arguments[:min(len(arguments), maxArguments)],
		})
	}
	return tools, message.Result.NextCursor, nil
}

func truncate(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}

func (r Reporter) send(ctx context.Context, tools []Tool, complete bool) error {
	if tools == nil {
		tools = []Tool{}
	}
	encoded, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	form := url.Values{"tools": {string(encoded)}, "complete": {fmt.Sprint(complete)}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.ReportURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(r.ClientID, r.ClientSecret)
	response, err := r.IdMagicClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusNoContent {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, maxReportErrorBytes))
		return fmt.Errorf("the tool report was refused: HTTP %d %s", response.StatusCode, strings.TrimSpace(string(detail)))
	}
	return nil
}
