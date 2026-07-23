package domain

// OAuth2Client の双子定義。internal/shared/spec/oauth2.go から移設 (wi-173, ADR-089)。
// ClientType / GrantType / ResponseType は shared/spec/policy.go の SCL permissions
// 評価エンジンおよび [[wi-181]] 側に残る AuthorizationRequest からも参照されるため
// shared に残置し、本パッケージからは spec 経由で参照する (wi-173 Plan 参照)。

import (
	"slices"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	z "github.com/Oudwins/zog"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type TokenEndpointAuthMethod string

const (
	AuthMethodClientSecretBasic TokenEndpointAuthMethod = "client_secret_basic"
	AuthMethodClientSecretPost  TokenEndpointAuthMethod = "client_secret_post"
	AuthMethodPrivateKeyJwt     TokenEndpointAuthMethod = "private_key_jwt"
	AuthMethodTlsClientAuth     TokenEndpointAuthMethod = "tls_client_auth"
	AuthMethodNone              TokenEndpointAuthMethod = "none"
)

func (m TokenEndpointAuthMethod) Valid() bool {
	switch m {
	case AuthMethodClientSecretBasic, AuthMethodClientSecretPost,
		AuthMethodPrivateKeyJwt, AuthMethodTlsClientAuth, AuthMethodNone:
		return true
	}
	return false
}

type FapiProfile string

const (
	FapiNone              FapiProfile = "none"
	FapiSecurityProfileV2 FapiProfile = "fapi_2_security_profile"
)

func (f FapiProfile) Valid() bool { return f == FapiNone || f == FapiSecurityProfileV2 }

type OAuth2Client struct {
	TenantID                           string                           `json:"tenant_id"`
	ClientID                           string                           `json:"client_id"`
	ApplicationID                      string                           `json:"application_id,omitempty"`
	ClientSecretHash                   *string                          `json:"client_secret_hash,omitempty"`
	ClientName                         *string                          `json:"client_name,omitempty"`
	ClientType                         spec.ClientType                  `json:"client_type"`
	RedirectURIs                       []string                         `json:"redirect_uris"`
	GrantTypes                         []spec.GrantType                 `json:"grant_types"`
	ResponseTypes                      []spec.ResponseType              `json:"response_types"`
	TokenEndpointAuthMethod            TokenEndpointAuthMethod          `json:"token_endpoint_auth_method"`
	Scope                              string                           `json:"scope"`
	JWKS                               map[string]any                   `json:"jwks,omitempty"`
	JwksURI                            *string                          `json:"jwks_uri,omitempty"`
	TlsClientAuthSubjectDN             *string                          `json:"tls_client_auth_subject_dn,omitempty"`
	IDTokenSignedResponseAlg           signingdomain.SignatureAlgorithm `json:"id_token_signed_response_alg"`
	RequirePushedAuthorizationRequests bool                             `json:"require_pushed_authorization_requests"`
	DpopBoundAccessTokens              bool                             `json:"dpop_bound_access_tokens"`
	FapiProfile                        FapiProfile                      `json:"fapi_profile"`
	// FirstParty は IdP 自身が所有する信頼済みクライアント (管理コンソール /
	// アカウントポータル) を表す。resource owner が IdP 利用者自身であるため、
	// authorization_code フローで consent 画面をスキップする (ADR-061)。
	FirstParty bool      `json:"first_party"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var oauth2ClientSchema = z.Struct(z.Shape{
	"ClientID": z.String().Min(1).Max(128).Required(),
	"ClientName": z.Ptr(
		z.String().Min(1).Max(200),
	),
	"ClientType": z.StringLike[spec.ClientType]().TestFunc(
		func(value *spec.ClientType, _ z.Ctx) bool { return value.Valid() },
		z.Message("client_type is not in enum"),
	).Required(),
	"RedirectURIs": z.Slice(
		z.String().URL(),
	),
	"GrantTypes": z.Slice(
		z.StringLike[spec.GrantType]().TestFunc(
			func(value *spec.GrantType, _ z.Ctx) bool { return value.Valid() },
			z.Message("grant_type is not in enum"),
		),
	).Min(1).Required(),
	"ResponseTypes": z.Slice(
		z.StringLike[spec.ResponseType]().TestFunc(
			func(value *spec.ResponseType, _ z.Ctx) bool { return value.Valid() },
			z.Message("response_type is not in enum"),
		),
	),
	"TokenEndpointAuthMethod": z.StringLike[TokenEndpointAuthMethod]().TestFunc(
		func(value *TokenEndpointAuthMethod, _ z.Ctx) bool { return value.Valid() },
		z.Message("token_endpoint_auth_method is not in enum"),
	).Required(),
	"IDTokenSignedResponseAlg": z.StringLike[signingdomain.SignatureAlgorithm]().TestFunc(
		func(value *signingdomain.SignatureAlgorithm, _ z.Ctx) bool { return value.Valid() },
		z.Message("id_token_signed_response_alg is not in enum"),
	).Required(),
	"FapiProfile": z.StringLike[FapiProfile]().TestFunc(
		func(value *FapiProfile, _ z.Ctx) bool { return value.Valid() },
		z.Message("fapi_profile is not in enum"),
	).Required(),
	"CreatedAt": z.Time().Required(),
	"UpdatedAt": z.Time().Required(),
}).TestFunc(func(value any, _ z.Ctx) bool {
	client, ok := value.(*OAuth2Client)
	if !ok {
		return false
	}
	switch client.TokenEndpointAuthMethod {
	case AuthMethodPrivateKeyJwt:
		return hasJWKs(client.JWKS) || client.JwksURI != nil && *client.JwksURI != ""
	case AuthMethodTlsClientAuth:
		return client.TlsClientAuthSubjectDN != nil && *client.TlsClientAuthSubjectDN != ""
	default:
		return true
	}
}, z.Message("client authentication method requires matching credentials")).
	TestFunc(func(value any, _ z.Ctx) bool {
		client, ok := value.(*OAuth2Client)
		if !ok {
			return false
		}
		// redirect 系グラント (authorization_code) は redirect_uri を必須とする
		// (RFC 6749 §3.1.2)。client_credentials のみの M2M クライアントは redirect を持たない。
		if clientUsesRedirect(client) {
			return len(client.RedirectURIs) > 0
		}
		return true
	}, z.Message("redirect_uris is required for redirect-based grants"))

func (c OAuth2Client) Validate() error {
	return spec.Validate(oauth2ClientSchema, &c)
}

func hasJWKs(jwks map[string]any) bool {
	switch keys := jwks["keys"].(type) {
	case []any:
		return len(keys) > 0
	case []map[string]any:
		return len(keys) > 0
	default:
		return false
	}
}

// clientUsesRedirect は client が redirect 系グラント (authorization_code) または
// code response_type を使うかを返す。これらは redirect_uri を必要とする。
func clientUsesRedirect(client *OAuth2Client) bool {
	return slices.Contains(client.GrantTypes, spec.GrantAuthorizationCode) ||
		slices.Contains(client.ResponseTypes, spec.ResponseTypeCode)
}

// ===============================================================
// イベント
// ===============================================================

type ClientRegistered struct {
	At         time.Time       `json:"-"`
	TenantID   string          `json:"tenantId"`
	ClientID   string          `json:"clientId"`
	ClientType spec.ClientType `json:"clientType"`
}

func (e *ClientRegistered) EventType() string     { return "ClientRegistered" }
func (e *ClientRegistered) OccurredAt() time.Time { return e.At }

type AdminOAuth2ClientCreated struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	ClientID    string    `json:"clientId"`
}

func (e *AdminOAuth2ClientCreated) EventType() string     { return "AdminOAuth2ClientCreated" }
func (e *AdminOAuth2ClientCreated) OccurredAt() time.Time { return e.At }

type AdminOAuth2ClientUpdated struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	ActorUserID   string    `json:"actorUserId"`
	ClientID      string    `json:"clientId"`
	ChangedFields []string  `json:"changedFields"`
}

func (e *AdminOAuth2ClientUpdated) EventType() string     { return "AdminOAuth2ClientUpdated" }
func (e *AdminOAuth2ClientUpdated) OccurredAt() time.Time { return e.At }

type AdminOAuth2ClientDeleted struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId"`
	ClientID    string    `json:"clientId"`
}

func (e *AdminOAuth2ClientDeleted) EventType() string     { return "AdminOAuth2ClientDeleted" }
func (e *AdminOAuth2ClientDeleted) OccurredAt() time.Time { return e.At }

type ClientSecretRotated struct {
	At                 time.Time  `json:"-"`
	TenantID           string     `json:"tenantId"`
	ActorUserID        string     `json:"actorUserId"`
	ClientID           string     `json:"clientId"`
	GraceUntil         *time.Time `json:"graceUntil,omitempty"`
	RevokedImmediately bool       `json:"revokedImmediately"`
}

func (e *ClientSecretRotated) EventType() string     { return "ClientSecretRotated" }
func (e *ClientSecretRotated) OccurredAt() time.Time { return e.At }
