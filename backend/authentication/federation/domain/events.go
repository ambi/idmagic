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

type FederatedLoginRejected struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	ProviderID string    `json:"providerId"`
	Reason     string    `json:"reason"`
}

func (e *FederatedLoginRejected) EventType() string     { return "FederatedLoginRejected" }
func (e *FederatedLoginRejected) OccurredAt() time.Time { return e.At }
