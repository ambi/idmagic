package domain

import (
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/shared/protocol/mcp"
)

// 一覧取得用の主体と、そのトークンの印である (REQ-OAUTH2-096)。主体は資格情報を持たず、
// OAuth2Client の記録もない。client_id は UUID なので登録済みのクライアントと衝突しない。
const (
	McpToolDiscoveryClientID = "urn:idmagic:mcp-tool-discovery"
	McpToolDiscoveryTokenUse = "mcp_tool_discovery"
)

// MetadataStatus は Protected Resource Metadata の取得の結果である (REQ-OAUTH2-095)。
type MetadataStatus string

const (
	MetadataFound            MetadataStatus = "found"
	MetadataNotFound         MetadataStatus = "not_found"
	MetadataInvalid          MetadataStatus = "invalid"
	MetadataResourceMismatch MetadataStatus = "resource_mismatch"
	MetadataUnreachable      MetadataStatus = "unreachable"
)

// ToolsStatus は tools/list の取得の結果である (REQ-OAUTH2-095)。
type ToolsStatus string

const (
	ToolsFetched         ToolsStatus = "fetched"
	ToolsSkipped         ToolsStatus = "skipped"
	ToolsUnauthorized    ToolsStatus = "unauthorized"
	ToolsForbidden       ToolsStatus = "forbidden"
	ToolsUnreachable     ToolsStatus = "unreachable"
	ToolsInvalidResponse ToolsStatus = "invalid_response"
)

// ProtectedResourceMetadata は検査を通ったメタデータのうち、取得に使う値である。
type ProtectedResourceMetadata struct {
	AuthorizationServers []string
	// ScopesSupported と dpopAlgs は形が違えば無視し、取得の可否を左右しない。
	ScopesSupported []string
	dpopAlgs        []string
}

// Trusts は MCP サーバーが issuer を認可サーバーとして宣言しているかを返す。
func (m ProtectedResourceMetadata) Trusts(issuer string) bool {
	return slices.Contains(m.AuthorizationServers, issuer)
}

// SupportsES256DPoP は ES256 の DPoP 証明を受け付けると宣言しているかを返す (REQ-OAUTH2-096)。
func (m ProtectedResourceMetadata) SupportsES256DPoP() bool {
	return slices.Contains(m.dpopAlgs, "ES256")
}

const wellKnownProtectedResource = "/.well-known/oauth-protected-resource"

// ProtectedResourceMetadataURLs は RFC 9728 §3.1 の path を継いだ URL と、MCP の Authorization 仕様が
// 次に試す origin 直下の URL を、試す順に返す。path が空なら一件である。
func ProtectedResourceMetadataURLs(resource string) ([]string, error) {
	parsed, err := url.Parse(resource)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("resource must be an absolute URL")
	}
	origin := parsed.Scheme + "://" + parsed.Host
	path := strings.TrimSuffix(parsed.EscapedPath(), "/")
	if path == "" {
		return []string{origin + wellKnownProtectedResource}, nil
	}
	return []string{origin + wellKnownProtectedResource + path, origin + wellKnownProtectedResource}, nil
}

// ParseProtectedResourceMetadata は 200 の応答の本文を検査する。resource は RFC 9728 §3.3 に従い、
// 登録の resource と正規化せずに比べる。
func ParseProtectedResourceMetadata(raw []byte, resource string) (*ProtectedResourceMetadata, MetadataStatus) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil || document == nil {
		return nil, MetadataInvalid
	}
	var declared string
	if err := json.Unmarshal(document["resource"], &declared); err != nil || declared != resource {
		return nil, MetadataResourceMismatch
	}
	servers, ok := stringArray(document["authorization_servers"])
	if !ok {
		return nil, MetadataInvalid
	}
	scopes, _ := stringArray(document["scopes_supported"])
	algs, _ := stringArray(document["dpop_signing_alg_values_supported"])
	return &ProtectedResourceMetadata{AuthorizationServers: servers, ScopesSupported: scopes, dpopAlgs: algs}, MetadataFound
}

func stringArray(raw json.RawMessage) ([]string, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil, false
	}
	return values, true
}

// CatalogToolsFromListed は取得した一覧をカタログにする。項目と切り詰めは REQ-MCPGATEWAY-011、
// 件数と名前の上限は REQ-OAUTH2-086 に揃える。100件を超えた分は捨てて complete を偽にし、
// 名前が上限を満たさないか重複すれば、Gateway の報告と同じく全体を拒否する。
func CatalogToolsFromListed(listed []mcp.ListedTool) ([]CatalogTool, bool, error) {
	complete := len(listed) <= maxCatalogTools
	listed = listed[:min(len(listed), maxCatalogTools)]
	tools := make([]CatalogTool, 0, len(listed))
	seen := make(map[string]bool, len(listed))
	for _, item := range listed {
		tool := CatalogTool{
			Name: item.Name, Title: truncateRunes(item.Title, maxCatalogToolTitle),
			Description: truncateRunes(item.Description, maxCatalogToolDescription),
			Arguments:   item.Arguments[:min(len(item.Arguments), maxCatalogToolArguments)],
		}
		if !validCatalogTool(tool) || seen[tool.Name] {
			return nil, false, errInvalidToolCatalog
		}
		seen[tool.Name] = true
		tools = append(tools, tool)
	}
	return tools, complete, nil
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit])
}
