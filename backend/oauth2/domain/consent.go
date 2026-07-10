package domain

// Consent の双子定義。internal/shared/spec/oauth2.go から移設 (wi-173, ADR-089)。
// AuthorizationDetail（実行時インスタンス）は [[wi-181]] 側の AuthorizationRequest /
// AuthorizationCodeRecord からも参照されるため shared に残置し、本パッケージからは
// spec 経由で参照する。

import (
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

type Consent struct {
	UserID               string                     `json:"user_id"`
	ClientID             string                     `json:"client_id"`
	Scopes               []string                   `json:"scopes"`
	State                ConsentState               `json:"state"`
	GrantedAt            time.Time                  `json:"granted_at"`
	ExpiresAt            time.Time                  `json:"expires_at"`
	RevokedAt            *time.Time                 `json:"revoked_at,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
}

var consentSchema = z.Struct(z.Shape{
	"UserID":   z.String().Required(),
	"ClientID": z.String().Required(),
	"Scopes":   z.Slice(z.String()).Min(1).Required(),
	"State": z.StringLike[ConsentState]().TestFunc(
		func(value *ConsentState, _ z.Ctx) bool { return value.Valid() },
		z.Message("state is not in enum"),
	).Required(),
	"GrantedAt": z.Time().Required(),
	"ExpiresAt": z.Time().Required(),
})

func (c Consent) Validate() error {
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
