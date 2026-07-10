package spec

// OAuth2 bounded context の双子定義のうち、client / consent / 認可詳細タイプ定義
// （OAuth2Client / Consent / AuthorizationDetailType 系）は internal/oauth2/domain へ
// 移設済み（wi-173, ADR-089）。本ファイルには authorization request / code / refresh /
// PAR / device / token claims（wi-181 で移設予定）と、それらが参照する実行時
// AuthorizationDetail（複数 context 予定の型から参照されるため shared に残置）が残る。

import "time"

// AuthorizationDetail は RFC 9396 authorization_details の 1 要素。type で識別される
// 構造化された細粒度権限を表し、登録済み AuthorizationDetailType に対し fail-closed に検証する。
// AuthorizationDetailType 自体は internal/oauth2/domain へ移設済み (wi-173)。
type AuthorizationDetail struct {
	Type       string         `json:"type"`
	Locations  []string       `json:"locations,omitempty"`
	Actions    []string       `json:"actions,omitempty"`
	Datatypes  []string       `json:"datatypes,omitempty"`
	Identifier string         `json:"identifier,omitempty"`
	Privileges []string       `json:"privileges,omitempty"`
	Fields     map[string]any `json:"fields,omitempty"`
}

// ===============================================================
// 認可リクエスト
// ===============================================================

type AuthorizationRequest struct {
	ID                   string                     `json:"id"`
	TenantID             string                     `json:"tenant_id"`
	State                AuthorizationCodeFlowState `json:"state"`
	ClientID             string                     `json:"client_id"`
	RedirectURI          string                     `json:"redirect_uri"`
	ResponseType         ResponseType               `json:"response_type"`
	Scope                string                     `json:"scope"`
	StateParam           *string                    `json:"state_param,omitempty"`
	Nonce                *string                    `json:"nonce,omitempty"`
	CodeChallenge        string                     `json:"code_challenge"`
	CodeChallengeMethod  CodeChallengeMethod        `json:"code_challenge_method"`
	Prompt               *string                    `json:"prompt,omitempty"`
	MaxAge               *int                       `json:"max_age,omitempty"`
	ParRequestURI        *string                    `json:"par_request_uri,omitempty"`
	UserID               *string                    `json:"user_id,omitempty"`
	AuthTime             *int64                     `json:"auth_time,omitempty"`
	AMR                  []string                   `json:"amr,omitempty"`
	ACR                  *string                    `json:"acr,omitempty"`
	ACRValues            *string                    `json:"acr_values,omitempty"`
	AuthorizationDetails []AuthorizationDetail      `json:"authorization_details,omitempty"`
	CreatedAt            time.Time                  `json:"created_at"`
	ExpiresAt            time.Time                  `json:"expires_at"`
}

func (a AuthorizationRequest) Validate() error {
	return validate(authorizationRequestSchema, &a)
}

// ===============================================================
// 認可コードレコード
// ===============================================================

type AuthorizationCodeRecord struct {
	Code                   string                       `json:"code"`
	TenantID               string                       `json:"tenant_id"`
	AuthorizationRequestID string                       `json:"authorization_request_id"`
	ClientID               string                       `json:"client_id"`
	UserID                 string                       `json:"user_id"`
	Scopes                 []string                     `json:"scopes"`
	RedirectURI            string                       `json:"redirect_uri"`
	CodeChallenge          string                       `json:"code_challenge"`
	CodeChallengeMethod    CodeChallengeMethod          `json:"code_challenge_method"`
	Nonce                  *string                      `json:"nonce,omitempty"`
	AuthTime               int64                        `json:"auth_time"`
	AMR                    []string                     `json:"amr,omitempty"`
	ACR                    *string                      `json:"acr,omitempty"`
	AuthorizationDetails   []AuthorizationDetail        `json:"authorization_details,omitempty"`
	State                  AuthorizationCodeRecordState `json:"state"`
	IssuedAt               time.Time                    `json:"issued_at"`
	ExpiresAt              time.Time                    `json:"expires_at"`
	RedeemedAt             *time.Time                   `json:"redeemed_at,omitempty"`
	IssuedFamilyID         *string                      `json:"issued_family_id,omitempty"`
}

func (a AuthorizationCodeRecord) Validate() error {
	return validate(authorizationCodeRecordSchema, &a)
}

// ===============================================================
// SenderConstraint (DPoP / mTLS)
// ===============================================================

type SenderConstraint struct {
	Type    SenderConstraintType `json:"type"`
	JKT     string               `json:"jkt,omitempty"`
	X5TS256 string               `json:"x5t#S256,omitempty"`
}

// ===============================================================
// リフレッシュトークン (ストアレコード)
// ===============================================================

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

func (r RefreshTokenRecord) Validate() error {
	return validate(refreshTokenRecordSchema, &r)
}

// ===============================================================
// PAR (Pushed Authorization Request) レコード
// ===============================================================

type PARRecord struct {
	RequestURI string            `json:"request_uri"`
	TenantID   string            `json:"tenant_id"`
	ClientID   string            `json:"client_id"`
	Parameters map[string]string `json:"parameters"`
	IssuedAt   time.Time         `json:"issued_at"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Used       bool              `json:"used"`
}

func (p PARRecord) Validate() error {
	return validate(parRecordSchema, &p)
}

// ===============================================================
// DeviceAuthorization (RFC 8628 ストアレコード)
// ===============================================================

type DeviceAuthorization struct {
	DeviceCodeHash  string              `json:"device_code_hash"`
	TenantID        string              `json:"tenant_id"`
	UserCode        string              `json:"user_code"`
	ClientID        string              `json:"client_id"`
	Scopes          []string            `json:"scopes"`
	State           DeviceCodeFlowState `json:"state"`
	UserID          *string             `json:"user_id,omitempty"`
	AuthTime        *int64              `json:"auth_time,omitempty"`
	IntervalSeconds int                 `json:"interval_seconds"`
	LastPolledAt    *time.Time          `json:"last_polled_at,omitempty"`
	IssuedFamilyID  *string             `json:"issued_family_id,omitempty"`
	IssuedAt        time.Time           `json:"issued_at"`
	ExpiresAt       time.Time           `json:"expires_at"`
}

func (d DeviceAuthorization) Validate() error {
	return validate(deviceAuthorizationSchema, &d)
}

// ===============================================================
// アクセストークン / ID トークン クレーム
// ===============================================================

type AccessTokenClaims struct {
	Issuer               string                `json:"iss"`
	Subject              string                `json:"sub"`
	Audience             any                   `json:"aud"`
	ClientID             string                `json:"client_id"`
	Scope                string                `json:"scope"`
	Exp                  int64                 `json:"exp"`
	Iat                  int64                 `json:"iat"`
	Nbf                  int64                 `json:"nbf,omitempty"`
	JTI                  string                `json:"jti"`
	AuthTime             int64                 `json:"auth_time,omitempty"`
	ACR                  string                `json:"acr,omitempty"`
	AMR                  []string              `json:"amr,omitempty"`
	CNF                  map[string]string     `json:"cnf,omitempty"`
	AuthorizationDetails []AuthorizationDetail `json:"authorization_details,omitempty"`
}

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
