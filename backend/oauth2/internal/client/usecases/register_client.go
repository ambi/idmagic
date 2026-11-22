// /register (RFC 7591 Dynamic Client Registration)
package usecases

import (
	"context"
	"errors"
	"slices"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type (
	RegisterClientInput  = clientports.RegisterClientInput
	RegisterClientResult = clientports.RegisterClientResult
)

type RegisterClientDeps struct {
	ClientRepo oauthports.OAuth2ClientRepository
	Emit       func(spec.DomainEvent)
	// QuotaRepo enforces the tenant's Hard Quota on oauth2_clients (wi-160).
	// This single check covers both dynamic client registration (/register) and admin client creation
	// (CreateAdminOAuth2Client calls RegisterClient internally).
	// nil skips enforcement (wiring gaps in tests/tools); production bootstrap always sets it.
	QuotaRepo tenantports.QuotaRepository
}

func RegisterClient(ctx context.Context, deps RegisterClientDeps, in RegisterClientInput, now time.Time) (*RegisterClientResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
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
			in.TokenEndpointAuthMethod = oauthdomain.AuthMethodNone
		} else {
			in.TokenEndpointAuthMethod = oauthdomain.AuthMethodClientSecretBasic
		}
	}
	if in.TokenEndpointAuthMethod == oauthdomain.AuthMethodPrivateKeyJwt {
		candidate := oauthdomain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: "validation", ClientType: in.ClientType,
			RedirectURIs:             []string{"https://validation.invalid/callback"},
			GrantTypes:               []spec.GrantType{spec.GrantClientCredentials},
			TokenEndpointAuthMethod:  oauthdomain.AuthMethodPrivateKeyJwt,
			JWKS:                     in.JWKS,
			JwksURI:                  in.JwksURI,
			IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
			FapiProfile:              oauthdomain.FapiNone,
			CreatedAt:                now,
			// 検証用の候補にも UpdatedAt を入れる。スキーマがこれを必須にして
			// いるので、欠けていると鍵の有無によらず候補が必ず落ち、どんな
			// private_key_jwt クライアントも登録できない。
			UpdatedAt: now,
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
	usesSecret := in.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretBasic ||
		in.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretPost
	if in.ClientType == spec.ClientConfidential && usesSecret {
		s, err := generateOpaqueToken()
		if err != nil {
			return nil, err
		}
		secret = s
		ss := oauthdomain.HashClientSecret(s)
		secretHash = &ss
	}
	fapiProfile := in.FapiProfile
	if fapiProfile == "" {
		fapiProfile = oauthdomain.FapiNone
	}
	scope := in.Scope
	if scope == "" {
		scope = "openid profile email"
	}
	c := &oauthdomain.OAuth2Client{
		TenantID:                           tenantports.TenantID(ctx),
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
		RequirePKCE:                        in.RequirePKCE,
		DpopBoundAccessTokens:              in.DpopBoundAccessTokens,
		FapiProfile:                        fapiProfile,
		BackChannelLogoutURI:               in.BackChannelLogoutURI,
		BackChannelLogoutSessionRequired:   in.BackChannelLogoutSessionRequired,
		FrontChannelLogoutURI:              in.FrontChannelLogoutURI,
		FrontChannelLogoutSessionRequired:  in.FrontChannelLogoutSessionRequired,
		CreatedAt:                          now,
		UpdatedAt:                          now,
	}
	if in.ClientName != "" {
		name := in.ClientName
		c.ClientName = &name
	}
	if err := storeNewClient(ctx, deps, c); err != nil {
		return nil, err
	}
	if secretHash != nil {
		credentialID, err := spec.NewUUIDv4()
		if err != nil {
			return nil, err
		}
		if err := deps.ClientRepo.SaveClientSecretCredential(ctx, oauthdomain.ClientSecretCredential{
			ID: credentialID, ClientID: clientID, SecretHash: *secretHash, CreatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	emit(deps.Emit, &oauthdomain.ClientRegistered{At: now, TenantID: tenantports.TenantID(ctx), ClientID: clientID, ClientType: in.ClientType})
	return &RegisterClientResult{Client: c, ClientSecret: secret}, nil
}

// storeNewClient は登録経路に依存しないメタデータ検証と使用量の確保を所有する。
func storeNewClient(ctx context.Context, deps RegisterClientDeps, c *oauthdomain.OAuth2Client) error {
	if err := c.Validate(); err != nil {
		return NewOAuthError("invalid_client_metadata", err.Error())
	}
	reservation, err := tenantports.ReserveQuota(ctx, deps.QuotaRepo, c.TenantID, tenancydomain.ResourceOAuth2Clients, 1)
	if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
		emit(deps.Emit, &tenancydomain.QuotaExceeded{At: c.CreatedAt, TenantID: c.TenantID, Resource: qErr.Resource, HardLimit: true})
	}
	if err != nil {
		return err
	}
	defer reservation.Release(ctx)
	if err := deps.ClientRepo.Save(ctx, c); err != nil {
		return err
	}
	reservation.Keep()
	return nil
}

func CreateSeedClient(ctx context.Context, deps RegisterClientDeps, c *oauthdomain.OAuth2Client) error {
	existing, err := deps.ClientRepo.FindByID(ctx, c.TenantID, c.ClientID)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("seed client already exists")
	}
	if err := storeNewClient(ctx, deps, c); err != nil {
		return err
	}
	emit(deps.Emit, &oauthdomain.AdminOAuth2ClientCreated{At: c.CreatedAt, TenantID: c.TenantID, ActorUserID: "seed", ClientID: c.ClientID})
	return nil
}
