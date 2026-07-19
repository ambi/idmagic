package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type AuthorizationDetailsRequested struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ClientID    string    `json:"clientId"`
	UserID      string    `json:"userId,omitempty"`
	DetailTypes []string  `json:"detailTypes"`
}

func (e *AuthorizationDetailsRequested) EventType() string     { return "AuthorizationDetailsRequested" }
func (e *AuthorizationDetailsRequested) OccurredAt() time.Time { return e.At }

type AuthorizationDetailsConsented struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	UserID      string    `json:"userId"`
	ClientID    string    `json:"clientId"`
	DetailTypes []string  `json:"detailTypes"`
}

func (e *AuthorizationDetailsConsented) EventType() string     { return "AuthorizationDetailsConsented" }
func (e *AuthorizationDetailsConsented) OccurredAt() time.Time { return e.At }

type AuthorizationDetailsRejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Reason   string    `json:"reason"`
}

func (e *AuthorizationDetailsRejected) EventType() string     { return "AuthorizationDetailsRejected" }
func (e *AuthorizationDetailsRejected) OccurredAt() time.Time { return e.At }

type AuthorizationCodeIssued struct {
	At                  time.Time                `json:"-"`
	TenantID            string                   `json:"tenantId"`
	ClientID            string                   `json:"clientId"`
	UserID              string                   `json:"userId"`
	Scopes              []string                 `json:"scopes"`
	CodeChallengeMethod spec.CodeChallengeMethod `json:"codeChallengeMethod"`
}

func (e *AuthorizationCodeIssued) EventType() string     { return "AuthorizationCodeIssued" }
func (e *AuthorizationCodeIssued) OccurredAt() time.Time { return e.At }

type AuthorizationCodeRedeemed struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *AuthorizationCodeRedeemed) EventType() string     { return "AuthorizationCodeRedeemed" }
func (e *AuthorizationCodeRedeemed) OccurredAt() time.Time { return e.At }

type AccessTokenIssued struct {
	At               time.Time `json:"-"`
	TenantID         string    `json:"tenantId"`
	JTI              string    `json:"jti"`
	ClientID         string    `json:"clientId"`
	UserID           string    `json:"userId"`
	Scopes           []string  `json:"scopes"`
	SenderConstraint string    `json:"senderConstraint"` // "none" | "dpop" | "mtls"
}

func (e *AccessTokenIssued) EventType() string     { return "AccessTokenIssued" }
func (e *AccessTokenIssued) OccurredAt() time.Time { return e.At }

type RefreshTokenIssued struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	TokenID  string    `json:"tokenId"`
	FamilyID string    `json:"familyId"`
	ParentID string    `json:"parentId,omitempty"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *RefreshTokenIssued) EventType() string     { return "RefreshTokenIssued" }
func (e *RefreshTokenIssued) OccurredAt() time.Time { return e.At }

type RefreshTokenRotated struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	OldTokenID string    `json:"oldTokenId"`
	NewTokenID string    `json:"newTokenId"`
	FamilyID   string    `json:"familyId"`
}

func (e *RefreshTokenRotated) EventType() string     { return "RefreshTokenRotated" }
func (e *RefreshTokenRotated) OccurredAt() time.Time { return e.At }

type TokenRevoked struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	TokenType string    `json:"tokenType"` // "access_token" | "refresh_token"
	TokenID   string    `json:"tokenId"`
	Reason    string    `json:"reason"`
}

func (e *TokenRevoked) EventType() string     { return "TokenRevoked" }
func (e *TokenRevoked) OccurredAt() time.Time { return e.At }

type TokenIntrospected struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	RSClientID string    `json:"rsClientId"`
	TokenID    string    `json:"tokenId"`
	Active     bool      `json:"active"`
}

func (e *TokenIntrospected) EventType() string     { return "TokenIntrospected" }
func (e *TokenIntrospected) OccurredAt() time.Time { return e.At }

type TokenExchanged struct {
	At              time.Time `json:"-"`
	TenantID        string    `json:"tenantId"`
	ActorUserID     string    `json:"actorUserId"`
	SubjectUserID   string    `json:"subjectUserId"`
	Audience        string    `json:"audience"`
	DelegationDepth int       `json:"delegationDepth"`
}

func (e *TokenExchanged) EventType() string     { return "TokenExchanged" }
func (e *TokenExchanged) OccurredAt() time.Time { return e.At }

type TokenExchangeRejected struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId,omitempty"`
	Reason      string    `json:"reason"`
}

func (e *TokenExchangeRejected) EventType() string     { return "TokenExchangeRejected" }
func (e *TokenExchangeRejected) OccurredAt() time.Time { return e.At }

// ProtectedResourceMetadataServed は RFC 9728 — /.well-known/oauth-protected-resource
// が登録済み McpResourceServer の metadata を配信した (ADR-055)。
type ProtectedResourceMetadataServed struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Resource string    `json:"resource"`
}

func (e *ProtectedResourceMetadataServed) EventType() string {
	return "ProtectedResourceMetadataServed"
}
func (e *ProtectedResourceMetadataServed) OccurredAt() time.Time { return e.At }

// ResourceScopedTokenIssued は RFC 8707 — resource indicator に基づき audience を単一
// McpResourceServer へ厳格限定した Access Token を発行した (ADR-055)。Authorize/PAR/
// Token(authorization_code) 経路が対象。token-exchange は TokenExchanged.audience で表現する。
type ResourceScopedTokenIssued struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Resource string    `json:"resource"`
	Scopes   []string  `json:"scopes"`
}

func (e *ResourceScopedTokenIssued) EventType() string     { return "ResourceScopedTokenIssued" }
func (e *ResourceScopedTokenIssued) OccurredAt() time.Time { return e.At }

// ResourceAudienceRejected は RFC 8707 — resource indicator が未登録・Disabled・複数指定
// のため fail-closed で拒否した (ADR-055)。Authorize/PAR/Token(authorization_code) 経路が
// 対象。token-exchange の拒否は TokenExchangeRejected.reason で表現する。
type ResourceAudienceRejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Reason   string    `json:"reason"`
}

func (e *ResourceAudienceRejected) EventType() string     { return "ResourceAudienceRejected" }
func (e *ResourceAudienceRejected) OccurredAt() time.Time { return e.At }

type RefreshTokenReuseDetected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	FamilyID string    `json:"familyId"`
	TokenID  string    `json:"tokenId"`
	ClientID string    `json:"clientId"`
}

func (e *RefreshTokenReuseDetected) EventType() string     { return "RefreshTokenReuseDetected" }
func (e *RefreshTokenReuseDetected) OccurredAt() time.Time { return e.At }

type PARStored struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	RequestURI string    `json:"requestUri"`
	ClientID   string    `json:"clientId"`
}

func (e *PARStored) EventType() string     { return "PARStored" }
func (e *PARStored) OccurredAt() time.Time { return e.At }

type DeviceAuthorizationRequested struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Scopes   []string  `json:"scopes"`
}

func (e *DeviceAuthorizationRequested) EventType() string     { return "DeviceAuthorizationRequested" }
func (e *DeviceAuthorizationRequested) OccurredAt() time.Time { return e.At }

type DeviceAuthorizationApproved struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *DeviceAuthorizationApproved) EventType() string     { return "DeviceAuthorizationApproved" }
func (e *DeviceAuthorizationApproved) OccurredAt() time.Time { return e.At }

type DeviceAuthorizationDenied struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *DeviceAuthorizationDenied) EventType() string     { return "DeviceAuthorizationDenied" }
func (e *DeviceAuthorizationDenied) OccurredAt() time.Time { return e.At }
