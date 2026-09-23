package domain

import "time"

type FederatedAuthenticated struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	UserID     string    `json:"userId"`
	ProviderID string    `json:"providerId"`
	SessionID  string    `json:"sessionId"`
}

func (e *FederatedAuthenticated) EventType() string     { return "FederatedAuthenticated" }
func (e *FederatedAuthenticated) OccurredAt() time.Time { return e.At }

type FederatedIdentityLinked struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	UserID        string    `json:"userId"`
	ProviderID    string    `json:"providerId"`
	LinkingMethod string    `json:"linkingMethod"`
}

func (e *FederatedIdentityLinked) EventType() string     { return "FederatedIdentityLinked" }
func (e *FederatedIdentityLinked) OccurredAt() time.Time { return e.At }

type FederatedIdentityUnlinked struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	UserID     string    `json:"userId"`
	ProviderID string    `json:"providerId"`
}

func (e *FederatedIdentityUnlinked) EventType() string     { return "FederatedIdentityUnlinked" }
func (e *FederatedIdentityUnlinked) OccurredAt() time.Time { return e.At }

// FederatedLoginRejected の Reason。監査が拒否の種類を区別するための値である。
const (
	RejectionProtocolValidationFailed = "protocol_validation_failed"
	// 未発行、消費済み、期限切れの state を区別しない。どれも生きている attempt と一致しない。
	RejectionStateMismatch = "state_mismatch"
)

type FederatedLoginRejected struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	ProviderID string    `json:"providerId"`
	Reason     string    `json:"reason"`
}

func (e *FederatedLoginRejected) EventType() string     { return "FederatedLoginRejected" }
func (e *FederatedLoginRejected) OccurredAt() time.Time { return e.At }
