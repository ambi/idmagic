package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

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
	// Sid は発行元 AuthorizationCodeRecord.sid。Rotate では親の値をそのまま引き継ぐ。
	// nil は client_credentials 等 browser session を持たない発行 (ADR-127)。
	Sid *string `json:"sid,omitempty"`
	// Resource は発行元の resource indicator (ADR-055)。Rotate では親の値をそのまま
	// 引き継ぎ、ローテーション後も同じ McpResourceServer へ audience を厳格限定する
	// (wi-262)。nil は resource 未指定の発行。
	Resource *string `json:"resource,omitempty"`
}

func (r RefreshTokenRecord) Validate() error { return spec.ValidateRefreshTokenRecord(&r) }

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
	Issuer   string `json:"iss"`
	Subject  string `json:"sub"`
	Audience any    `json:"aud"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
	AuthTime int64  `json:"auth_time"`
	Nonce    string `json:"nonce,omitempty"`
	// Sid は browser session に束縛された発行のときだけ含める OIDC session id
	// (OpenID Connect Front/Back-Channel Logout 1.0, ADR-127)。
	Sid               string   `json:"sid,omitempty"`
	ACR               string   `json:"acr,omitempty"`
	AMR               []string `json:"amr,omitempty"`
	AZP               string   `json:"azp,omitempty"`
	AtHash            string   `json:"at_hash,omitempty"`
	Name              string   `json:"name,omitempty"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
	Email             string   `json:"email,omitempty"`
	EmailVerified     bool     `json:"email_verified,omitempty"`
}
