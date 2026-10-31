package domain

import "time"

// 以下は TokenVault のドメインイベントである。トークン、本文、URL のパスとクエリを載せない
// (REQ-TOKENVAULT-015)。

// UpstreamRevocation の値。
const (
	UpstreamRevocationSucceeded     = "succeeded"
	UpstreamRevocationFailed        = "failed"
	UpstreamRevocationNotConfigured = "not_configured"
)

type TokenVaultConnectorCreated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
}

func (e *TokenVaultConnectorCreated) EventType() string     { return "TokenVaultConnectorCreated" }
func (e *TokenVaultConnectorCreated) OccurredAt() time.Time { return e.At }

type TokenVaultConnectorUpdated struct {
	At          time.Time       `json:"-"`
	TenantID    string          `json:"tenantId"`
	ConnectorID string          `json:"connectorId"`
	Status      ConnectorStatus `json:"status"`
}

func (e *TokenVaultConnectorUpdated) EventType() string     { return "TokenVaultConnectorUpdated" }
func (e *TokenVaultConnectorUpdated) OccurredAt() time.Time { return e.At }

type TokenVaultConnectorDeleted struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
}

func (e *TokenVaultConnectorDeleted) EventType() string     { return "TokenVaultConnectorDeleted" }
func (e *TokenVaultConnectorDeleted) OccurredAt() time.Time { return e.At }

type TokenVaultConnectionLinked struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
	UserID      string    `json:"userId"`
}

func (e *TokenVaultConnectionLinked) EventType() string     { return "TokenVaultConnectionLinked" }
func (e *TokenVaultConnectionLinked) OccurredAt() time.Time { return e.At }

type TokenVaultConnectionLinkFailed struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
	UserID      string    `json:"userId"`
}

func (e *TokenVaultConnectionLinkFailed) EventType() string     { return "TokenVaultConnectionLinkFailed" }
func (e *TokenVaultConnectionLinkFailed) OccurredAt() time.Time { return e.At }

type TokenVaultConnectionUnlinked struct {
	At                 time.Time `json:"-"`
	TenantID           string    `json:"tenantId"`
	ConnectorID        string    `json:"connectorId"`
	UserID             string    `json:"userId"`
	UpstreamRevocation string    `json:"upstreamRevocation"`
}

func (e *TokenVaultConnectionUnlinked) EventType() string     { return "TokenVaultConnectionUnlinked" }
func (e *TokenVaultConnectionUnlinked) OccurredAt() time.Time { return e.At }

type TokenVaultConnectionReauthorizationRequired struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
	UserID      string    `json:"userId"`
}

func (e *TokenVaultConnectionReauthorizationRequired) EventType() string {
	return "TokenVaultConnectionReauthorizationRequired"
}
func (e *TokenVaultConnectionReauthorizationRequired) OccurredAt() time.Time { return e.At }

type TokenVaultExecutionPerformed struct {
	At             time.Time `json:"-"`
	TenantID       string    `json:"tenantId"`
	ConnectorID    string    `json:"connectorId"`
	UserID         string    `json:"userId"`
	AgentID        string    `json:"agentId"`
	RuleID         string    `json:"ruleId"`
	Method         string    `json:"method"`
	Origin         string    `json:"origin"`
	UpstreamStatus int       `json:"upstreamStatus"`
}

func (e *TokenVaultExecutionPerformed) EventType() string     { return "TokenVaultExecutionPerformed" }
func (e *TokenVaultExecutionPerformed) OccurredAt() time.Time { return e.At }

type TokenVaultExecutionDenied struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ConnectorID string    `json:"connectorId"`
	UserID      string    `json:"userId,omitempty"`
	AgentID     string    `json:"agentId,omitempty"`
	Reason      string    `json:"reason"`
}

func (e *TokenVaultExecutionDenied) EventType() string     { return "TokenVaultExecutionDenied" }
func (e *TokenVaultExecutionDenied) OccurredAt() time.Time { return e.At }
