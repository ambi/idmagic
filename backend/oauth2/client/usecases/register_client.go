// /register (RFC 7591 Dynamic Client Registration)
package usecases

import (
	"context"
	"errors"
	"slices"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

type RegisterClientInput struct {
	ClientName              string
	ClientType              spec.ClientType
	RedirectURIs            []string
	GrantTypes              []spec.GrantType
	ResponseTypes           []spec.ResponseType
	TokenEndpointAuthMethod domain.TokenEndpointAuthMethod
	Scope                   string
	JWKS                    map[string]any
	JwksURI                 *string
	TlsClientAuthSubjectDN  *string
	RequirePAR              bool
	DpopBoundAccessTokens   bool
	FapiProfile             domain.FapiProfile
}

type RegisterClientResult struct {
	Client       *domain.OAuth2Client
	ClientSecret string // 平文。出力後は再表示されない (RFC 7591 §3.2.1)
}

type RegisterClientDeps struct {
	ClientRepo ports.OAuth2ClientRepository
	Emit       func(spec.DomainEvent)
	// QuotaRepo enforces the tenant's Hard Quota on oauth2_clients (wi-160,
	// ADR-134). This single check covers both dynamic client registration
	// (/register) and admin client creation (CreateAdminOAuth2Client calls
	// RegisterClient internally). nil skips enforcement (wiring gaps in
	// tests/tools); production bootstrap always sets it.
	QuotaRepo tenantports.QuotaRepository
}

func RegisterClient(ctx context.Context, deps RegisterClientDeps, in RegisterClientInput, now time.Time) (*RegisterClientResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if deps.QuotaRepo != nil {
		tenantID := tenancy.TenantID(ctx)
		err := tenancyusecases.CheckQuotaAndIncrement(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceOAuth2Clients, 1)
		if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
			emit(deps.Emit, &tenancydomain.QuotaExceeded{At: now, TenantID: tenantID, Resource: qErr.Resource, HardLimit: true})
		}
		if err != nil {
			return nil, err
		}
	}
	if in.ClientType == "" {
		in.ClientType = spec.ClientConfidential
	}
	if len(in.GrantTypes) == 0 {
		in.GrantTypes = []spec.GrantType{spec.GrantAuthorizationCode}
	}
	// redirect 系グラント (authorization_code) のみ redirect_uri / code response_type を要求する。
	// client_credentials のみの M2M クライアントは redirect を持たない (RFC 6749 §3.1.2)。
	interactive := slices.Contains(in.GrantTypes, spec.GrantAuthorizationCode)
	if interactive && len(in.RedirectURIs) == 0 {
		return nil, NewOAuthError("invalid_redirect_uri", "redirect_uris is required")
	}
	if interactive && len(in.ResponseTypes) == 0 {
		in.ResponseTypes = []spec.ResponseType{spec.ResponseTypeCode}
	}
	if in.TokenEndpointAuthMethod == "" {
		if in.ClientType == spec.ClientPublic {
			in.TokenEndpointAuthMethod = domain.AuthMethodNone
		} else {
			in.TokenEndpointAuthMethod = domain.AuthMethodClientSecretBasic
		}
	}
	if in.TokenEndpointAuthMethod == domain.AuthMethodPrivateKeyJwt {
		candidate := domain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: "validation", ClientType: in.ClientType,
			RedirectURIs:             []string{"https://validation.invalid/callback"},
			GrantTypes:               []spec.GrantType{spec.GrantClientCredentials},
			TokenEndpointAuthMethod:  domain.AuthMethodPrivateKeyJwt,
			JWKS:                     in.JWKS,
			JwksURI:                  in.JwksURI,
			IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
			FapiProfile:              domain.FapiNone,
			CreatedAt:                now,
		}
		if err := candidate.Validate(); err != nil {
			return nil, NewOAuthError("invalid_client_metadata", "private_key_jwt requires non-empty inline jwks")
		}
	}
	clientID, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	var secret string
	var secretHash *string
	usesSecret := in.TokenEndpointAuthMethod == domain.AuthMethodClientSecretBasic ||
		in.TokenEndpointAuthMethod == domain.AuthMethodClientSecretPost
	if in.ClientType == spec.ClientConfidential && usesSecret {
		s, err := generateOpaqueToken()
		if err != nil {
			return nil, err
		}
		secret = s
		ss := domain.HashClientSecret(s)
		secretHash = &ss
	}
	fapiProfile := in.FapiProfile
	if fapiProfile == "" {
		fapiProfile = domain.FapiNone
	}
	scope := in.Scope
	if scope == "" {
		scope = "openid profile email"
	}
	c := &domain.OAuth2Client{
		TenantID:                           tenancy.TenantID(ctx),
		ClientID:                           clientID,
		ClientSecretHash:                   secretHash,
		ClientType:                         in.ClientType,
		RedirectURIs:                       in.RedirectURIs,
		GrantTypes:                         in.GrantTypes,
		ResponseTypes:                      in.ResponseTypes,
		TokenEndpointAuthMethod:            in.TokenEndpointAuthMethod,
		Scope:                              scope,
		JWKS:                               in.JWKS,
		JwksURI:                            in.JwksURI,
		TlsClientAuthSubjectDN:             in.TlsClientAuthSubjectDN,
		IDTokenSignedResponseAlg:           signingdomain.SigAlgPS256,
		RequirePushedAuthorizationRequests: in.RequirePAR,
		DpopBoundAccessTokens:              in.DpopBoundAccessTokens,
		FapiProfile:                        fapiProfile,
		CreatedAt:                          now,
		UpdatedAt:                          now,
	}
	if in.ClientName != "" {
		name := in.ClientName
		c.ClientName = &name
	}
	if err := c.Validate(); err != nil {
		return nil, NewOAuthError("invalid_client_metadata", err.Error())
	}
	if err := deps.ClientRepo.Save(ctx, c); err != nil {
		return nil, err
	}
	if secretHash != nil {
		credentialID, err := spec.NewUUIDv4()
		if err != nil {
			return nil, err
		}
		if err := deps.ClientRepo.SaveClientSecretCredential(ctx, domain.ClientSecretCredential{
			CredentialID: credentialID, ClientID: clientID, SecretHash: *secretHash, CreatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	emit(deps.Emit, &domain.ClientRegistered{At: now, TenantID: tenancy.TenantID(ctx), ClientID: clientID, ClientType: in.ClientType})
	return &RegisterClientResult{Client: c, ClientSecret: secret}, nil
}
