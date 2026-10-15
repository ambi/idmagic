package ports

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ErrClientSecretLimitExceeded は、有効なシークレットが上限に達したクライアントへシークレットを発行しようとしたことを表す。
var ErrClientSecretLimitExceeded = errors.New("active client secret limit exceeded")

// ClientAdministration は、ほかのモジュールの管理画面が OAuth2 クライアントを作成、更新し、
// シークレットを管理する操作である。シークレットの上限、クォータ、発行するイベントは OAuth2 が保つ。
type ClientAdministration interface {
	Create(ctx context.Context, in CreateAdminOAuth2ClientInput) (*RegisterClientResult, error)
	Update(ctx context.Context, in UpdateAdminOAuth2ClientInput) (*domain.OAuth2Client, error)
	RotateSecret(ctx context.Context, in RotateClientSecretInput) (*RotateClientSecretResult, error)
	IssueSecret(ctx context.Context, in IssueClientSecretInput) (*IssueClientSecretResult, error)
	RevokeSecret(ctx context.Context, in RevokeClientSecretInput) (*RevokeClientSecretResult, error)
}

type RegisterClientInput struct {
	ClientName                        string
	ClientType                        spec.ClientType
	RedirectURIs                      []string
	GrantTypes                        []spec.GrantType
	ResponseTypes                     []spec.ResponseType
	TokenEndpointAuthMethod           domain.TokenEndpointAuthMethod
	Scope                             string
	JWKS                              map[string]any
	JwksURI                           *string
	TlsClientAuthSubjectDN            *string
	RequirePAR                        bool
	DpopBoundAccessTokens             bool
	FapiProfile                       domain.FapiProfile
	BackChannelLogoutURI              *string
	BackChannelLogoutSessionRequired  bool
	FrontChannelLogoutURI             *string
	FrontChannelLogoutSessionRequired bool
}

type RegisterClientResult struct {
	Client       *domain.OAuth2Client
	ClientSecret string // 平文。出力後は再表示されない (RFC 7591 §3.2.1)
}

type CreateAdminOAuth2ClientInput struct {
	ActorUserID  string
	Registration RegisterClientInput
	Now          time.Time
}

type UpdateAdminOAuth2ClientInput struct {
	ActorUserID                       string
	ClientID                          string
	ClientName                        *string
	RedirectURIs                      *[]string
	GrantTypes                        *[]spec.GrantType
	ResponseTypes                     *[]spec.ResponseType
	Scope                             *string
	RequirePAR                        *bool
	DpopBoundTokens                   *bool
	BackChannelLogoutURI              *string
	BackChannelLogoutSessionRequired  *bool
	FrontChannelLogoutURI             *string
	FrontChannelLogoutSessionRequired *bool
	Now                               time.Time
}

type RotateClientSecretInput struct {
	ActorUserID string
	ClientID    string
	GraceDays   int
	Now         time.Time
}

type RotateClientSecretResult struct {
	ClientSecret string
	GraceUntil   *time.Time
	Credentials  []domain.ClientSecretCredential
}

type IssueClientSecretInput struct {
	ActorUserID   string
	ClientID      string
	ExpiresInDays int
	Now           time.Time
}

type IssueClientSecretResult struct {
	ClientSecret string
	Credential   domain.ClientSecretCredential
	Credentials  []domain.ClientSecretCredential
}

type RevokeClientSecretInput struct {
	ActorUserID  string
	ClientID     string
	CredentialID string
	Now          time.Time
}

type RevokeClientSecretResult struct {
	Credentials []domain.ClientSecretCredential
}
