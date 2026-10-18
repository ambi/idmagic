package ports

import (
	"context"
	"errors"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var (
	ErrClientSecretLimitExceeded = errors.New("active client secret limit exceeded")
	ErrOIDCClientNotFound        = errors.New("client does not exist")
)

// OIDCClientAdministration はカタログが要求する管理操作である。
// 保存表現と秘密のハッシュは渡さず、各操作の確定と不変条件は所有者が保つ。
type OIDCClientAdministration interface {
	Create(context.Context, CreateOIDCClientInput) (*CreatedOIDCClient, error)
	Update(context.Context, UpdateOIDCClientInput) error
	Configuration(context.Context, string, string, time.Time) (*OIDCConfiguration, error)
	BindApplication(context.Context, string, string, string) error
	UpdateClaims(context.Context, string, string, OIDCClaimUpdate) error
	Delete(context.Context, string, string) error
	RotateSecret(context.Context, RotateClientSecretInput) (*RotateClientSecretResult, error)
	IssueSecret(context.Context, IssueClientSecretInput) (*IssueClientSecretResult, error)
	RevokeSecret(context.Context, RevokeClientSecretInput) (*RevokeClientSecretResult, error)
}

type OIDCClientRegistration struct {
	ClientName              string
	ClientType              spec.ClientType
	RedirectURIs            []string
	GrantTypes              []spec.GrantType
	ResponseTypes           []spec.ResponseType
	TokenEndpointAuthMethod string
	Scope                   string
	JwksURI                 *string
	TlsClientAuthSubjectDN  *string
}

type CreateOIDCClientInput struct {
	ActorUserID  string
	Registration OIDCClientRegistration
	Now          time.Time
}

type CreatedOIDCClient struct{ ClientID, ClientSecret string }

type UpdateOIDCClientInput struct {
	ActorUserID, ClientID       string
	RedirectURIs                *[]string
	GrantTypes                  *[]spec.GrantType
	ResponseTypes               *[]spec.ResponseType
	Scope                       *string
	RequirePAR, DpopBoundTokens *bool
	Now                         time.Time
}

type OIDCClaimUpdate struct {
	SubSourceAttribute *string
	Rules              *[]claimdomain.ClaimMappingRule
	Now                time.Time
}

// OIDCConfiguration は管理画面へ返す値で、クライアントの保存型を含まない。
type OIDCConfiguration struct {
	ClientID                string                         `json:"client_id"`
	ClientType              spec.ClientType                `json:"client_type"`
	RedirectURIs            []string                       `json:"redirect_uris"`
	GrantTypes              []spec.GrantType               `json:"grant_types"`
	ResponseTypes           []spec.ResponseType            `json:"response_types"`
	TokenEndpointAuthMethod string                         `json:"token_endpoint_auth_method"`
	Scope                   string                         `json:"scope"`
	RequirePAR              bool                           `json:"require_pushed_authorization_requests"`
	DpopBoundAccessTokens   bool                           `json:"dpop_bound_access_tokens"`
	FapiProfile             string                         `json:"fapi_profile"`
	ClientSecretRotatable   bool                           `json:"client_secret_rotatable"`
	SecretCredentials       []ClientSecretMetadata         `json:"secret_credentials"`
	SubSourceAttribute      string                         `json:"sub_source_attribute"`
	Rules                   []claimdomain.ClaimMappingRule `json:"rules"`
}

type ClientSecretMetadata struct {
	CredentialID string     `json:"credential_id"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	Status       string     `json:"status"`
}

type RotateClientSecretInput struct {
	ActorUserID, ClientID string
	GraceDays             int
	Now                   time.Time
}
type RotateClientSecretResult struct {
	ClientSecret string
	GraceUntil   *time.Time
	Credentials  []ClientSecretMetadata
}
type IssueClientSecretInput struct {
	ActorUserID, ClientID string
	ExpiresInDays         int
	Now                   time.Time
}
type IssueClientSecretResult struct {
	ClientSecret string
	Credential   ClientSecretMetadata
	Credentials  []ClientSecretMetadata
}
type RevokeClientSecretInput struct {
	ActorUserID, ClientID, CredentialID string
	Now                                 time.Time
}
type RevokeClientSecretResult struct{ Credentials []ClientSecretMetadata }
