package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AuthorizationRequest は OAuth2/OIDC 認可要求の状態を表す。
type AuthorizationRequest struct {
	ID                   string                          `json:"id"`
	TenantID             string                          `json:"tenant_id"`
	State                spec.AuthorizationCodeFlowState `json:"state"`
	ClientID             string                          `json:"client_id"`
	RedirectURI          string                          `json:"redirect_uri"`
	ResponseType         spec.ResponseType               `json:"response_type"`
	Scope                string                          `json:"scope"`
	StateParam           *string                         `json:"state_param,omitempty"`
	Nonce                *string                         `json:"nonce,omitempty"`
	CodeChallenge        string                          `json:"code_challenge"`
	CodeChallengeMethod  spec.CodeChallengeMethod        `json:"code_challenge_method"`
	Prompt               *string                         `json:"prompt,omitempty"`
	MaxAge               *int                            `json:"max_age,omitempty"`
	ParRequestURI        *string                         `json:"par_request_uri,omitempty"`
	UserID               *string                         `json:"user_id,omitempty"`
	AuthTime             *int64                          `json:"auth_time,omitempty"`
	AMR                  []string                        `json:"amr,omitempty"`
	ACR                  *string                         `json:"acr,omitempty"`
	ACRValues            *string                         `json:"acr_values,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail      `json:"authorization_details,omitempty"`
	CreatedAt            time.Time                       `json:"created_at"`
	ExpiresAt            time.Time                       `json:"expires_at"`
}

func (a AuthorizationRequest) Validate() error { return spec.ValidateAuthorizationRequest(&a) }

// AuthorizationCodeRecord は単回使用の認可コードを表す。
type AuthorizationCodeRecord struct {
	Code                   string                            `json:"code"`
	TenantID               string                            `json:"tenant_id"`
	AuthorizationRequestID string                            `json:"authorization_request_id"`
	ClientID               string                            `json:"client_id"`
	UserID                 string                            `json:"user_id"`
	Scopes                 []string                          `json:"scopes"`
	RedirectURI            string                            `json:"redirect_uri"`
	CodeChallenge          string                            `json:"code_challenge"`
	CodeChallengeMethod    spec.CodeChallengeMethod          `json:"code_challenge_method"`
	Nonce                  *string                           `json:"nonce,omitempty"`
	AuthTime               int64                             `json:"auth_time"`
	AMR                    []string                          `json:"amr,omitempty"`
	ACR                    *string                           `json:"acr,omitempty"`
	AuthorizationDetails   []spec.AuthorizationDetail        `json:"authorization_details,omitempty"`
	State                  spec.AuthorizationCodeRecordState `json:"state"`
	IssuedAt               time.Time                         `json:"issued_at"`
	ExpiresAt              time.Time                         `json:"expires_at"`
	RedeemedAt             *time.Time                        `json:"redeemed_at,omitempty"`
	IssuedFamilyID         *string                           `json:"issued_family_id,omitempty"`
}

func (a AuthorizationCodeRecord) Validate() error { return spec.ValidateAuthorizationCodeRecord(&a) }

// SenderConstraint は DPoP または mTLS の sender-constrained token 情報を表す。
type SenderConstraint struct {
	Type    spec.SenderConstraintType `json:"type"`
	JKT     string                    `json:"jkt,omitempty"`
	X5TS256 string                    `json:"x5t#S256,omitempty"`
}

const (
	SenderConstraintDPoP = spec.SenderConstraintDPoP
	SenderConstraintMTLS = spec.SenderConstraintMTLS
)

// RefreshTokenRecord は refresh token rotation の永続化レコードを表す。
type RefreshTokenRecord struct {
	ID                string            `json:"id"`
	TenantID          string            `json:"tenant_id"`
	Hash              string            `json:"hash"`
	FamilyID          string            `json:"family_id"`
	ParentID          *string           `json:"parent_id,omitempty"`
	ClientID          string            `json:"client_id"`
	UserID            string            `json:"user_id"`
	Scopes            []string          `json:"scopes"`
	IssuedAt          time.Time         `json:"issued_at"`
	ExpiresAt         time.Time         `json:"expires_at"`
	AbsoluteExpiresAt time.Time         `json:"absolute_expires_at"`
	Revoked           bool              `json:"revoked"`
	Rotated           bool              `json:"rotated"`
	SenderConstraint  *SenderConstraint `json:"sender_constraint,omitempty"`
}

func (r RefreshTokenRecord) Validate() error { return spec.ValidateRefreshTokenRecord(&r) }

// PARRecord は Pushed Authorization Request の消費状態を表す。
type PARRecord struct {
	RequestURI string            `json:"request_uri"`
	TenantID   string            `json:"tenant_id"`
	ClientID   string            `json:"client_id"`
	Parameters map[string]string `json:"parameters"`
	IssuedAt   time.Time         `json:"issued_at"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Used       bool              `json:"used"`
}

func (p PARRecord) Validate() error { return spec.ValidatePARRecord(&p) }

// DeviceAuthorization は RFC 8628 device authorization の状態を表す。
type DeviceAuthorization struct {
	DeviceCodeHash  string                   `json:"device_code_hash"`
	TenantID        string                   `json:"tenant_id"`
	UserCode        string                   `json:"user_code"`
	ClientID        string                   `json:"client_id"`
	Scopes          []string                 `json:"scopes"`
	State           spec.DeviceCodeFlowState `json:"state"`
	UserID          *string                  `json:"user_id,omitempty"`
	AuthTime        *int64                   `json:"auth_time,omitempty"`
	IntervalSeconds int                      `json:"interval_seconds"`
	LastPolledAt    *time.Time               `json:"last_polled_at,omitempty"`
	IssuedFamilyID  *string                  `json:"issued_family_id,omitempty"`
	IssuedAt        time.Time                `json:"issued_at"`
	ExpiresAt       time.Time                `json:"expires_at"`
}

func (d DeviceAuthorization) Validate() error { return spec.ValidateDeviceAuthorization(&d) }

// AccessTokenClaims は発行済み access token の claims を表す。
type AccessTokenClaims struct {
	Issuer               string                     `json:"iss"`
	Subject              string                     `json:"sub"`
	Audience             any                        `json:"aud"`
	ClientID             string                     `json:"client_id"`
	Scope                string                     `json:"scope"`
	Exp                  int64                      `json:"exp"`
	Iat                  int64                      `json:"iat"`
	Nbf                  int64                      `json:"nbf,omitempty"`
	JTI                  string                     `json:"jti"`
	AuthTime             int64                      `json:"auth_time,omitempty"`
	ACR                  string                     `json:"acr,omitempty"`
	AMR                  []string                   `json:"amr,omitempty"`
	CNF                  map[string]string          `json:"cnf,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
}

// IDTokenClaims は発行済み ID token の claims を表す。
type IDTokenClaims struct {
	Issuer            string   `json:"iss"`
	Subject           string   `json:"sub"`
	Audience          any      `json:"aud"`
	Exp               int64    `json:"exp"`
	Iat               int64    `json:"iat"`
	AuthTime          int64    `json:"auth_time"`
	Nonce             string   `json:"nonce,omitempty"`
	ACR               string   `json:"acr,omitempty"`
	AMR               []string `json:"amr,omitempty"`
	AZP               string   `json:"azp,omitempty"`
	AtHash            string   `json:"at_hash,omitempty"`
	Name              string   `json:"name,omitempty"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
	Email             string   `json:"email,omitempty"`
	EmailVerified     bool     `json:"email_verified,omitempty"`
}
