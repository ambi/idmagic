package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// CatalogTool は、リソースサーバーが上流の tools/list からカタログとして送ったツールである。
// MCP サーバーの申告であり、公開、評価、発行の根拠にはしない。
type CatalogTool struct {
	Name        string   `json:"name"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Arguments   []string `json:"arguments,omitempty"`
}

// McpToolCatalog は、一つの登録について最新のカタログだけを保つ。登録の更新時刻とは独立して置き換わる。
type McpToolCatalog struct {
	TenantID         string
	ResourceServerID string
	Tools            []CatalogTool
	Complete         bool
	ReceivedAt       time.Time
	// Previous と ChangedAt は、内容の変わった直前の一覧と、最新の内容を初めて受信した時刻である。
	// 保存先が保存済みのカタログとの比較で決め、保存の入力では読まない (REQ-OAUTH2-086)。
	Previous  *CatalogToolList
	ChangedAt time.Time
}

// カタログの上限は REQ-OAUTH2-086 が定め、登録の規則数とツール名の上限に揃える。
const (
	maxCatalogTools            = 100
	maxCatalogToolName         = 128
	maxCatalogToolTitle        = 256
	maxCatalogToolDescription  = 2048
	maxCatalogToolArguments    = 32
	maxCatalogToolArgumentName = 128
)

var errInvalidToolCatalog = errors.New("invalid tool catalog")

// ParseCatalogTools はカタログの JSON の配列を読み、一件でも上限を越えれば全体を拒否する。
func ParseCatalogTools(raw string) ([]CatalogTool, error) {
	var tools []CatalogTool
	if err := json.Unmarshal([]byte(raw), &tools); err != nil || tools == nil {
		return nil, errInvalidToolCatalog
	}
	if len(tools) > maxCatalogTools {
		return nil, errInvalidToolCatalog
	}
	seen := make(map[string]bool, len(tools))
	for _, tool := range tools {
		if !validCatalogTool(tool) || seen[tool.Name] {
			return nil, errInvalidToolCatalog
		}
		seen[tool.Name] = true
	}
	return tools, nil
}

func validCatalogTool(tool CatalogTool) bool {
	if tool.Name == "" || utf8.RuneCountInString(tool.Name) > maxCatalogToolName || strings.ContainsAny(tool.Name, ":#") {
		return false
	}
	if utf8.RuneCountInString(tool.Title) > maxCatalogToolTitle ||
		utf8.RuneCountInString(tool.Description) > maxCatalogToolDescription ||
		len(tool.Arguments) > maxCatalogToolArguments {
		return false
	}
	for _, argument := range tool.Arguments {
		if utf8.RuneCountInString(argument) > maxCatalogToolArgumentName {
			return false
		}
	}
	return true
}
