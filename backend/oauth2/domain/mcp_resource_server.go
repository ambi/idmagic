package domain

// McpResourceServer は MCP resource server (ツール/データソース) の tenant-scoped 登録
// (ADR-055)。canonical resource URI と許可 scope を所有し、Protected Resource Metadata
// (RFC 9728) と resource indicator (RFC 8707) 検証の基準になる。

import (
	"net/url"
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type McpResourceServerState string

const (
	McpResourceServerActive   McpResourceServerState = "Active"
	McpResourceServerDisabled McpResourceServerState = "Disabled"
)

func (s McpResourceServerState) Valid() bool {
	switch s {
	case McpResourceServerActive, McpResourceServerDisabled:
		return true
	}
	return false
}

type McpResourceServer struct {
	TenantID         string                 `json:"tenant_id"`
	ResourceServerID string                 `json:"resource_server_id"`
	Resource         string                 `json:"resource"`
	Name             string                 `json:"name"`
	Scopes           []string               `json:"scopes"`
	State            McpResourceServerState `json:"state"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

func (m McpResourceServer) IsActive() bool { return m.State == McpResourceServerActive }

func isAbsoluteURIWithoutFragment(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.IsAbs() && parsed.Fragment == ""
}

var mcpResourceServerSchema = z.Struct(z.Shape{
	"ResourceServerID": z.String().Min(1).Required(),
	"Resource": z.String().URL().TestFunc(
		func(value *string, _ z.Ctx) bool { return value != nil && isAbsoluteURIWithoutFragment(*value) },
		z.Message("resource must be an absolute URI without a fragment (RFC 8707 §2)"),
	).Required(),
	"Name":   z.String().Min(1).Max(200).Required(),
	"Scopes": z.Slice(z.String().Min(1)).Min(1).Required(),
	"State": z.StringLike[McpResourceServerState]().TestFunc(
		func(value *McpResourceServerState, _ z.Ctx) bool { return value.Valid() },
		z.Message("mcp resource server state is not in enum"),
	).Required(),
	"CreatedAt": z.Time().Required(),
	"UpdatedAt": z.Time().Required(),
})

func (m McpResourceServer) Validate() error {
	return spec.Validate(mcpResourceServerSchema, &m)
}
