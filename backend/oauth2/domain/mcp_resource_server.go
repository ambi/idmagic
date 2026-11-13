package domain

// McpResourceServer は MCP resource server (ツール/データソース) の tenant-scoped 登録。
// canonical resource URI と許可 scope を所有し、Protected Resource Metadata
// (RFC 9728) と resource indicator (RFC 8707) 検証の基準になる。

import (
	"errors"
	"net/url"
	"time"

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
	TenantID string   `json:"tenant_id"`
	ID       string   `json:"id"`
	Resource string   `json:"resource"`
	Name     string   `json:"name"`
	Scopes   []string `json:"scopes"`
	// ClientID は /resource-access/evaluate を呼べるクライアント。空なら評価できない。
	ClientID string                  `json:"client_id,omitempty"`
	Hosting  *HostedMcpConfiguration `json:"hosting,omitempty"`
	// AccessRules は評価できる操作。ここにない操作は不許可になる。
	AccessRules []ResourceAccessRule   `json:"access_rules"`
	State       McpResourceServerState `json:"state"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (m McpResourceServer) IsActive() bool { return m.State == McpResourceServerActive }

func isAbsoluteURIWithoutFragment(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.IsAbs() && parsed.Fragment == ""
}

func (m McpResourceServer) Validate() error {
	if m.Hosting != nil {
		if m.ClientID == "" {
			return errors.New("client_id is required for a managed gateway")
		}
		if err := m.Hosting.Validate(); err != nil {
			return err
		}
	}
	if err := spec.All(
		spec.Required("ID", m.ID),
		spec.Required("Resource", m.Resource, spec.URL(), spec.Must(isAbsoluteURIWithoutFragment,
			"resource must be an absolute URI without a fragment (RFC 8707 §2)")),
		spec.Required("Name", m.Name, spec.Chars(1, spec.LengthDisplayName)),
		spec.NonEmpty("Scopes", m.Scopes),
		spec.Required("State", m.State, spec.Must(McpResourceServerState.Valid, "mcp resource server state is not in enum")),
		spec.Required("CreatedAt", m.CreatedAt),
		spec.Required("UpdatedAt", m.UpdatedAt),
	); err != nil {
		return err
	}
	return m.validateAccessRules()
}
