package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// AuthorizationRequest は OAuth2/OIDC 認可要求の状態を表す。
type AuthorizationRequest struct {
	ID                  string                          `json:"id"`
	TenantID            string                          `json:"tenant_id"`
	State               spec.AuthorizationCodeFlowState `json:"state"`
	ClientID            string                          `json:"client_id"`
	RedirectURI         string                          `json:"redirect_uri"`
	ResponseType        spec.ResponseType               `json:"response_type"`
	Scope               string                          `json:"scope"`
	StateParam          *string                         `json:"state_param,omitempty"`
	Nonce               *string                         `json:"nonce,omitempty"`
	CodeChallenge       string                          `json:"code_challenge"`
	CodeChallengeMethod spec.CodeChallengeMethod        `json:"code_challenge_method"`
	Prompt              *string                         `json:"prompt,omitempty"`
	MaxAge              *int                            `json:"max_age,omitempty"`
	ParRequestURI       *string                         `json:"par_request_uri,omitempty"`
	UserID              *string                         `json:"user_id,omitempty"`
	AuthTime            *int64                          `json:"auth_time,omitempty"`
	AMR                 []string                        `json:"amr,omitempty"`
	ACR                 *string                         `json:"acr,omitempty"`
	ACRValues           *string                         `json:"acr_values,omitempty"`
	// Sid は authenticate_user 完了時に AuthenticationContext.session_id から一度だけ
	// 伝搬する OIDC session id (Authentication の LoginSession.id と同値)。
	// AuthorizationCodeRecord / RefreshTokenRecord / IdTokenClaims へそのまま引き継ぐ (ADR-127)。
	Sid                  *string                    `json:"sid,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
	// Resource は RFC 8707 resource indicator (ADR-055)。非nil のとき、発行トークンの
	// aud をこの McpResourceServer の resource URI に厳格限定する。
	Resource  *string   `json:"resource,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (a AuthorizationRequest) Validate() error { return spec.ValidateAuthorizationRequest(&a) }

// AuthorizationCodeRecord は単回使用の認可コードを表す。
type AuthorizationCodeRecord struct {
	Code                   string                   `json:"code"`
	TenantID               string                   `json:"tenant_id"`
	AuthorizationRequestID string                   `json:"authorization_request_id"`
	ClientID               string                   `json:"client_id"`
	UserID                 string                   `json:"user_id"`
	Scopes                 []string                 `json:"scopes"`
	RedirectURI            string                   `json:"redirect_uri"`
	CodeChallenge          string                   `json:"code_challenge"`
	CodeChallengeMethod    spec.CodeChallengeMethod `json:"code_challenge_method"`
	Nonce                  *string                  `json:"nonce,omitempty"`
	AuthTime               int64                    `json:"auth_time"`
	AMR                    []string                 `json:"amr,omitempty"`
	ACR                    *string                  `json:"acr,omitempty"`
	// Sid は認可リクエストから引き継いだ OIDC session id。id_token の sid claim と
	// 発行 RefreshTokenRecord.sid に伝播する (ADR-127)。
	Sid                  *string                    `json:"sid,omitempty"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
	// Resource は認可リクエストから引き継いだ resource indicator (ADR-055)。
	Resource       *string                           `json:"resource,omitempty"`
	State          spec.AuthorizationCodeRecordState `json:"state"`
	IssuedAt       time.Time                         `json:"issued_at"`
	ExpiresAt      time.Time                         `json:"expires_at"`
	RedeemedAt     *time.Time                        `json:"redeemed_at,omitempty"`
	IssuedFamilyID *string                           `json:"issued_family_id,omitempty"`
}

func (a AuthorizationCodeRecord) Validate() error { return spec.ValidateAuthorizationCodeRecord(&a) }

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
