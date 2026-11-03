package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// ReportedTool は、リソースサーバーが上流の tools/list から報告したツールである。
// MCP サーバーの申告であり、公開、評価、発行の根拠にはしない。
type ReportedTool struct {
	Name        string   `json:"name"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Arguments   []string `json:"arguments,omitempty"`
}

// McpToolReport は、一つの登録について最新の報告だけを保つ。登録の更新時刻とは独立して置き換わる。
type McpToolReport struct {
	TenantID         string
	ResourceServerID string
	Tools            []ReportedTool
	Complete         bool
	ReportedAt       time.Time
}

// 報告の上限は REQ-OAUTH2-086 が定め、登録の規則数とツール名の上限に揃える。
const (
	maxReportedTools            = 100
	maxReportedToolName         = 128
	maxReportedToolTitle        = 256
	maxReportedToolDescription  = 2048
	maxReportedToolArguments    = 32
	maxReportedToolArgumentName = 128
)

var errInvalidToolReport = errors.New("invalid tool report")

// ParseReportedTools は報告の JSON の配列を読み、一件でも上限を越えれば全体を拒否する。
func ParseReportedTools(raw string) ([]ReportedTool, error) {
	var tools []ReportedTool
	if err := json.Unmarshal([]byte(raw), &tools); err != nil || tools == nil {
		return nil, errInvalidToolReport
	}
	if len(tools) > maxReportedTools {
		return nil, errInvalidToolReport
	}
	seen := make(map[string]bool, len(tools))
	for _, tool := range tools {
		if !validReportedTool(tool) || seen[tool.Name] {
			return nil, errInvalidToolReport
		}
		seen[tool.Name] = true
	}
	return tools, nil
}

func validReportedTool(tool ReportedTool) bool {
	if tool.Name == "" || utf8.RuneCountInString(tool.Name) > maxReportedToolName || strings.ContainsAny(tool.Name, ":#") {
		return false
	}
	if utf8.RuneCountInString(tool.Title) > maxReportedToolTitle ||
		utf8.RuneCountInString(tool.Description) > maxReportedToolDescription ||
		len(tool.Arguments) > maxReportedToolArguments {
		return false
	}
	for _, argument := range tool.Arguments {
		if utf8.RuneCountInString(argument) > maxReportedToolArgumentName {
			return false
		}
	}
	return true
}
