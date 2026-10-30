package domain

// Consent の双子定義。internal/shared/spec/oauth2.go から移設 (wi-173)。
// AuthorizationDetail（実行時インスタンス）は [[wi-181]] 側の AuthorizationRequest /
// AuthorizationCodeRecord からも参照されるため shared に残置し、本パッケージからは
// spec 経由で参照する。

import (
	"errors"
	"slices"
	"time"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type ConsentState string

const (
	ConsentGranted ConsentState = "granted"
	ConsentRevoked ConsentState = "revoked"
	ConsentExpired ConsentState = "expired"
)

func (s ConsentState) Valid() bool {
	switch s {
	case ConsentGranted, ConsentRevoked, ConsentExpired:
		return true
	}
	return false
}

// Consent は、クライアントへの同意か Agent への代行の付与である。ClientID と AgentID の
// どちらか一方だけを持つ。
type Consent struct {
	UserID               string                     `json:"user_id"`
	ClientID             string                     `json:"client_id,omitempty"`
	AgentID              string                     `json:"agent_id,omitempty"`
	Scopes               []string                   `json:"scopes"`
	State                ConsentState               `json:"state"`
	GrantedAt            time.Time                  `json:"granted_at"`
	ExpiresAt            time.Time                  `json:"expires_at"`
	RevokedAt            *time.Time                 `json:"revoked_at,omitempty"`
	LastUsedAt           *time.Time                 `json:"last_used_at,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
}

// PermitsDelegation は、この付与が now の時点で scopes の代行を許すかを判定する。
func (c Consent) PermitsDelegation(scopes []string, now time.Time) bool {
	if c.State != ConsentGranted || !now.Before(c.ExpiresAt) {
		return false
	}
	for _, scope := range scopes {
		if !slices.Contains(c.Scopes, scope) {
			return false
		}
	}
	return true
}

var consentSchema = z.Struct(z.Shape{
	"UserID": z.String().Required(),
	"Scopes": z.Slice(z.String()).Min(1).Required(),
	"State": z.StringLike[ConsentState]().TestFunc(
		func(value *ConsentState, _ z.Ctx) bool { return value.Valid() },
		z.Message("state is not in enum"),
	).Required(),
	"GrantedAt": z.Time().Required(),
	"ExpiresAt": z.Time().Required(),
})

func (c Consent) Validate() error {
	if (c.ClientID == "") == (c.AgentID == "") {
		return errors.New("consent must have exactly one of client_id and agent_id")
	}
	return spec.Validate(consentSchema, &c)
}

// ===============================================================
// イベント
// ===============================================================

type ConsentGrantedEvent struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	ClientID string    `json:"clientId"`
	Scopes   []string  `json:"scopes"`
}

func (e *ConsentGrantedEvent) EventType() string     { return "ConsentGranted" }
func (e *ConsentGrantedEvent) OccurredAt() time.Time { return e.At }

type ConsentRevokedEvent struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId,omitempty"`
	UserID      string    `json:"userId"`
	ClientID    string    `json:"clientId"`
}

func (e *ConsentRevokedEvent) EventType() string     { return "ConsentRevoked" }
func (e *ConsentRevokedEvent) OccurredAt() time.Time { return e.At }

// AgentDelegationGranted は、利用者が Agent に自分の代行を許したことを表す。
type AgentDelegationGranted struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	AgentID   string    `json:"agentId"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (e *AgentDelegationGranted) EventType() string     { return "AgentDelegationGranted" }
func (e *AgentDelegationGranted) OccurredAt() time.Time { return e.At }

// AgentDelegationRevoked は、利用者が Agent への代行の付与を撤回したことを表す。
type AgentDelegationRevoked struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	UserID   string    `json:"userId"`
	AgentID  string    `json:"agentId"`
}

func (e *AgentDelegationRevoked) EventType() string     { return "AgentDelegationRevoked" }
func (e *AgentDelegationRevoked) OccurredAt() time.Time { return e.At }
