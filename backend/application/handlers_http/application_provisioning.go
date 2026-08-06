// アプリケーションの一括プロビジョニング (wi-69)。Okta / Entra のように、種別を選んで
// プロトコル設定もまとめて入力すると、backend が OAuth2 client / WS-Fed RP を作成し、
// Application と単一 protocol relation を一括で作る。OAuth2/WS-Fed の wire 設定は各 protocol
// context が所有し、本ハンドラは adapter として両者を合成する。
package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	clientusecases "github.com/ambi/idmagic/backend/oauth2/client/usecases"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"

	"github.com/labstack/echo/v5"
)

const (
	defaultOIDCScope    = "openid profile email"
	defaultServiceScope = "openid"
	defaultNameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
	defaultNameIDSource = "user_id"
)

type createApplicationRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // oidc | wsfed | weblink | service
	LaunchURL string `json:"launch_url"`
	// OIDC
	RedirectURIs []string `json:"redirect_uris"`
	// OIDC / service の生成 client 設定。auth 方式は作成時に確定し以後不変。
	Scope                   string                              `json:"scope"`
	ClientType              spec.ClientType                     `json:"client_type"`
	TokenEndpointAuthMethod oauthdomain.TokenEndpointAuthMethod `json:"token_endpoint_auth_method"`
	JwksURI                 string                              `json:"jwks_uri"`
	TLSClientAuthSubjectDN  string                              `json:"tls_client_auth_subject_dn"`
	// WS-Federation
	Wtrealm      string   `json:"wtrealm"`
	ReplyURLs    []string `json:"reply_urls"`
	NameIDFormat string   `json:"name_id_format"`
	NameIDSource string   `json:"name_id_source"`
	// SAML 2.0
	IDPProfileID                      string   `json:"idp_profile_id"`
	EntityID                          string   `json:"entity_id"`
	ACSURLs                           []string `json:"acs_urls"`
	SLOURL                            string   `json:"slo_url"`
	SignResponse                      bool     `json:"sign_response"`
	WantAuthnRequestsSigned           bool     `json:"want_authn_requests_signed"`
	AuthnRequestSigningCertificatePEM string   `json:"authn_request_signing_certificate_pem"`
}

// oidcConfig / wsfedConfig はアプリ詳細に解決して返す protocol 設定。
// advanced 項目を含めてアプリ編集画面に集約する (wi-76, ADR-066)。
// ClientType / TokenEndpointAuthMethod / FapiProfile は更新契約上の不変項目で表示専用。
type oidcConfig struct {
	ClientID                string                              `json:"client_id"`
	ClientType              spec.ClientType                     `json:"client_type"`
	RedirectURIs            []string                            `json:"redirect_uris"`
	GrantTypes              []spec.GrantType                    `json:"grant_types"`
	ResponseTypes           []spec.ResponseType                 `json:"response_types"`
	TokenEndpointAuthMethod oauthdomain.TokenEndpointAuthMethod `json:"token_endpoint_auth_method"`
	Scope                   string                              `json:"scope"`
	RequirePAR              bool                                `json:"require_pushed_authorization_requests"`
	DpopBoundAccessTokens   bool                                `json:"dpop_bound_access_tokens"`
	FapiProfile             oauthdomain.FapiProfile             `json:"fapi_profile"`
	ClientSecretRotatable   bool                                `json:"client_secret_rotatable"`
	SecretCredentials       []clientSecretCredentialMetadata    `json:"secret_credentials"`
	// SubSourceAttribute and Rules are the claim release override (wi-73, ADR-151).
	SubSourceAttribute string                         `json:"sub_source_attribute"`
	Rules              []claimdomain.ClaimMappingRule `json:"rules"`
}

type clientSecretCredentialMetadata struct {
	CredentialID string     `json:"credential_id"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	Status       string     `json:"status"`
}

func clientSecretMetadata(credentials []oauthdomain.ClientSecretCredential, now time.Time) []clientSecretCredentialMetadata {
	metadata := make([]clientSecretCredentialMetadata, 0, len(credentials))
	for _, credential := range credentials {
		metadata = append(metadata, clientSecretCredentialMetadata{
			CredentialID: credential.ID, CreatedAt: credential.CreatedAt,
			ExpiresAt: credential.ExpiresAt, RevokedAt: credential.RevokedAt,
			Status: string(credential.StatusAt(now)),
		})
	}
	return metadata
}

type wsfedConfig struct {
	Wtrealm      string                         `json:"wtrealm"`
	ReplyURLs    []string                       `json:"reply_urls"`
	Audience     string                         `json:"audience"`
	TokenType    feddomain.WsFedTokenType       `json:"token_type"`
	NameIDFormat string                         `json:"name_id_format"`
	NameIDSource string                         `json:"name_id_source"`
	Rules        []claimdomain.ClaimMappingRule `json:"rules"`
}

type samlConfig struct {
	IDPProfileID                      string                         `json:"idp_profile_id"`
	EntityID                          string                         `json:"entity_id"`
	ACSURLs                           []string                       `json:"acs_urls"`
	SLOURL                            string                         `json:"slo_url"`
	Audience                          string                         `json:"audience"`
	NameIDFormat                      string                         `json:"name_id_format"`
	NameIDSource                      string                         `json:"name_id_source"`
	SignAssertion                     bool                           `json:"sign_assertion"`
	SignResponse                      bool                           `json:"sign_response"`
	WantAuthnRequestsSigned           bool                           `json:"want_authn_requests_signed"`
	AuthnRequestSigningCertificatePEM string                         `json:"authn_request_signing_certificate_pem"`
	Rules                             []claimdomain.ClaimMappingRule `json:"rules"`
}

// nonNilRules は nil スライスを空スライスに正規化する。claim 規則を持たない RP/SP の
// JSON が null ではなく [] になり、UI 側の .length 参照が安全になる。
func nonNilRules(rules []claimdomain.ClaimMappingRule) []claimdomain.ClaimMappingRule {
	if rules == nil {
		return []claimdomain.ClaimMappingRule{}
	}
	return rules
}

// oidcSubSourceAttribute は OAuth2Client.ClaimPolicy から sub の source 属性を取り出す
// (wi-73, ADR-151)。policy 未設定なら既定 (defaultNameIDSource = "user_id") を返す。
func oidcSubSourceAttribute(policy *claimdomain.ClaimMappingPolicy) string {
	if policy == nil || policy.NameID.SourceAttribute == "" {
		return defaultNameIDSource
	}
	return policy.NameID.SourceAttribute
}

// oidcClaimPolicyRules は OAuth2Client.ClaimPolicy から claim release 上書き rule を取り出す。
func oidcClaimPolicyRules(policy *claimdomain.ClaimMappingPolicy) []claimdomain.ClaimMappingRule {
	if policy == nil {
		return nil
	}
	return policy.Rules
}

// resolveClaimAttributeDefs はこのテナントの属性可視性 floor (ADR-151) 判定用に builtin +
// custom 属性定義を解決する。OIDC / WS-Fed / SAML の claim release 上書き検証が共有する。
func (d Deps) resolveClaimAttributeDefs(ctx context.Context, tenantID string) ([]userdomain.UserAttributeDef, error) {
	return claimusecases.ResolveTenantAttributeDefs(ctx, tenantID, d.AttrSchemaRepo)
}

func (d Deps) handleCreateApplication(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req createApplicationRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	ctx := c.Request().Context()
	now := time.Now().UTC()

	switch req.Type {
	case "weblink":
		app, err := appusecases.CreateApplication(ctx, d.applicationDeps(), appusecases.CreateApplicationInput{
			ActorUserID: actor.ID, Name: req.Name, Kind: domain.ApplicationWeblink,
			LaunchURL: req.LaunchURL, Now: now,
		})
		if err != nil {
			return d.writeApplicationError(c, err)
		}
		return support.NoStoreJSON(c, http.StatusCreated, map[string]any{"application": d.buildApplicationResponse(ctx, support.RequestTenantID(c), app)})

	case "oidc":
		if len(req.RedirectURIs) == 0 {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "Specify at least one redirect URI.")
		}
		registration := clientusecases.RegisterClientInput{
			ClientName: req.Name, ClientType: req.ClientType, RedirectURIs: req.RedirectURIs,
			GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken},
			ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
			TokenEndpointAuthMethod: req.TokenEndpointAuthMethod, Scope: nonEmpty(req.Scope, defaultOIDCScope),
		}
		if dn := strings.TrimSpace(req.TLSClientAuthSubjectDN); dn != "" {
			registration.TlsClientAuthSubjectDN = &dn
		}
		if uri := strings.TrimSpace(req.JwksURI); uri != "" {
			registration.JwksURI = &uri
		}
		result, err := clientusecases.CreateAdminOAuth2Client(ctx, clientusecases.AdminOAuth2ClientDeps{ClientRepo: d.ClientRepo, Emit: d.Emit, QuotaRepo: d.QuotaRepo}, clientusecases.CreateAdminOAuth2ClientInput{
			ActorUserID:  actor.ID,
			Registration: registration,
			Now:          now,
		})
		if err != nil {
			return d.writeApplicationError(c, err)
		}
		app, err := d.createCatalogApp(ctx, actor.ID, req, now, domain.ApplicationFederated,
			domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: result.Client.ClientID})
		if err != nil {
			return d.writeApplicationError(c, err)
		}
		return support.NoStoreJSON(c, http.StatusCreated, map[string]any{
			"application": d.buildApplicationResponse(ctx, support.RequestTenantID(c), app), "client_id": result.Client.ClientID, "client_secret": result.ClientSecret,
		})

	case "service":
		// M2M / サービスクライアント (client_credentials)。redirect を持たず、ポータルにも
		// 出さない service kind の Application として登録する (Okta の API Services 相当)。
		result, err := clientusecases.CreateAdminOAuth2Client(ctx, clientusecases.AdminOAuth2ClientDeps{ClientRepo: d.ClientRepo, Emit: d.Emit, QuotaRepo: d.QuotaRepo}, clientusecases.CreateAdminOAuth2ClientInput{
			ActorUserID: actor.ID,
			Registration: clientusecases.RegisterClientInput{
				ClientName: req.Name, ClientType: spec.ClientConfidential,
				GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
				TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic, Scope: nonEmpty(req.Scope, defaultServiceScope),
			},
			Now: now,
		})
		if err != nil {
			return d.writeApplicationError(c, err)
		}
		app, err := d.createCatalogApp(ctx, actor.ID, req, now, domain.ApplicationService,
			domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: result.Client.ClientID})
		if err != nil {
			return d.writeApplicationError(c, err)
		}
		return support.NoStoreJSON(c, http.StatusCreated, map[string]any{
			"application": d.buildApplicationResponse(ctx, support.RequestTenantID(c), app), "client_id": result.Client.ClientID, "client_secret": result.ClientSecret,
		})

	case "wsfed":
		if strings.TrimSpace(req.Wtrealm) == "" || len(req.ReplyURLs) == 0 {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "Specify wtrealm and a reply URL.")
		}
		rp := &feddomain.WsFedRelyingParty{
			TenantID: support.RequestTenantID(c), Wtrealm: req.Wtrealm, DisplayName: req.Name, ReplyURLs: req.ReplyURLs,
			ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
				Format: nonEmpty(req.NameIDFormat, defaultNameIDFormat), SourceAttribute: nonEmpty(req.NameIDSource, defaultNameIDSource),
			}},
			CreatedAt: now,
		}
		if d.WsFedRPRepo == nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "WS-Federation is unavailable.")
		}
		if err := d.WsFedRPRepo.Save(ctx, rp); err != nil {
			return err
		}
		app, err := d.createCatalogApp(ctx, actor.ID, req, now, domain.ApplicationFederated,
			domain.ApplicationProtocol{Type: domain.ApplicationProtocolWsFed, Wtrealm: req.Wtrealm})
		if err != nil {
			return d.writeApplicationError(c, err)
		}
		return support.NoStoreJSON(c, http.StatusCreated, map[string]any{"application": d.buildApplicationResponse(ctx, support.RequestTenantID(c), app)})

	case "saml":
		if strings.TrimSpace(req.EntityID) == "" || len(req.ACSURLs) == 0 {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "Specify an entity ID and an ACS URL.")
		}
		if d.SamlSPRepo == nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "SAML is unavailable.")
		}
		if req.WantAuthnRequestsSigned {
			if _, err := samldomain.ParseCertificatePEM(req.AuthnRequestSigningCertificatePEM); err != nil {
				return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "Specify a certificate for AuthnRequest signature verification.")
			}
		}
		sp := &samldomain.SamlServiceProvider{
			TenantID: support.RequestTenantID(c), EntityID: req.EntityID, DisplayName: req.Name,
			IDPProfileID: nonEmpty(req.IDPProfileID, samldomain.DefaultIDPProfileID),
			ACSURLs:      req.ACSURLs, SLOURL: strings.TrimSpace(req.SLOURL),
			ClaimPolicy: claimdomain.ClaimMappingPolicy{NameID: claimdomain.NameIdConfiguration{
				Format: nonEmpty(req.NameIDFormat, samldomain.SamlNameIDFormatPersistent), SourceAttribute: nonEmpty(req.NameIDSource, defaultNameIDSource),
			}},
			SignAssertion: true, SignResponse: req.SignResponse,
			WantAuthnRequestsSigned:           req.WantAuthnRequestsSigned,
			AuthnRequestSigningCertificatePEM: strings.TrimSpace(req.AuthnRequestSigningCertificatePEM),
			CreatedAt:                         now,
		}
		if err := d.SamlSPRepo.Save(ctx, sp); err != nil {
			if errors.Is(err, samldomain.ErrInvalidIDPProfile) || errors.Is(err, samldomain.ErrDedicatedIDPProfileCardinality) {
				return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
			}
			return err
		}
		app, err := d.createCatalogApp(ctx, actor.ID, req, now, domain.ApplicationFederated,
			domain.ApplicationProtocol{Type: domain.ApplicationProtocolSAML, EntityID: req.EntityID})
		if err != nil {
			return d.writeApplicationError(c, err)
		}
		return support.NoStoreJSON(c, http.StatusCreated, map[string]any{"application": d.buildApplicationResponse(ctx, support.RequestTenantID(c), app)})

	default:
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The type must be oidc, wsfed, saml, or weblink.")
	}
}

// createCatalogApp は指定 kind の Application と不変な単一 protocol relation を作成する。
func (d Deps) createCatalogApp(ctx context.Context, actorUserID string, req createApplicationRequest, now time.Time, kind domain.ApplicationKind, protocol domain.ApplicationProtocol) (*domain.Application, error) {
	app, err := appusecases.CreateApplication(ctx, d.applicationDeps(), appusecases.CreateApplicationInput{
		ActorUserID: actorUserID, Name: req.Name, Kind: kind, LaunchURL: req.LaunchURL,
		Protocol: &protocol, Now: now,
	})
	if err != nil {
		return nil, err
	}
	// 通常 protocol CRUD は ownership を変更しない。Application 作成経路だけが
	// memory adapter の domain projection に application_id を同期する。
	switch protocol.Type {
	case domain.ApplicationProtocolOIDC:
		client, findErr := d.ClientRepo.FindByID(ctx, app.TenantID, protocol.ClientID)
		if findErr != nil {
			return nil, findErr
		}
		if client != nil {
			client.ApplicationID = app.ApplicationID
			if saveErr := d.ClientRepo.Save(ctx, client); saveErr != nil {
				return nil, saveErr
			}
		}
	case domain.ApplicationProtocolSAML:
		sp, findErr := d.SamlSPRepo.FindByEntityID(ctx, app.TenantID, protocol.EntityID)
		if findErr != nil {
			return nil, findErr
		}
		if sp != nil {
			sp.ApplicationID = app.ApplicationID
			if saveErr := d.SamlSPRepo.Save(ctx, sp); saveErr != nil {
				return nil, saveErr
			}
		}
	case domain.ApplicationProtocolWsFed:
		rp, findErr := d.WsFedRPRepo.FindByWtrealm(ctx, app.TenantID, protocol.Wtrealm)
		if findErr != nil {
			return nil, findErr
		}
		if rp != nil {
			rp.ApplicationID = app.ApplicationID
			if saveErr := d.WsFedRPRepo.Save(ctx, rp); saveErr != nil {
				return nil, saveErr
			}
		}
	}
	return app, nil
}

// resolveProtocolConfig は Application の relation から OAuth2 client / WS-Fed RP の
// 実設定を解決して返す (アプリ詳細表示用)。
func (d Deps) resolveProtocolConfig(c *echo.Context, app *domain.Application) (*oidcConfig, *wsfedConfig, *samlConfig) {
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	var oidc *oidcConfig
	var wsfed *wsfedConfig
	var saml *samlConfig
	if app.Protocol != nil {
		protocol := app.Protocol
		switch protocol.Type {
		case domain.ApplicationProtocolOIDC:
			if d.ClientRepo == nil {
				break
			}
			if client, err := d.ClientRepo.FindByID(ctx, tenantID, protocol.ClientID); err == nil && client != nil {
				credentials, _ := d.ClientRepo.ListClientSecretCredentials(ctx, client.ClientID)
				metadata := clientSecretMetadata(credentials, time.Now().UTC())
				oidc = &oidcConfig{
					ClientID: client.ClientID, ClientType: client.ClientType, RedirectURIs: client.RedirectURIs,
					GrantTypes: client.GrantTypes, ResponseTypes: client.ResponseTypes,
					TokenEndpointAuthMethod: client.TokenEndpointAuthMethod, Scope: client.Scope,
					RequirePAR:            client.RequirePushedAuthorizationRequests,
					DpopBoundAccessTokens: client.DpopBoundAccessTokens, FapiProfile: client.FapiProfile,
					ClientSecretRotatable: client.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretBasic || client.TokenEndpointAuthMethod == oauthdomain.AuthMethodClientSecretPost,
					SecretCredentials:     metadata,
					SubSourceAttribute:    oidcSubSourceAttribute(client.ClaimPolicy),
					Rules:                 nonNilRules(oidcClaimPolicyRules(client.ClaimPolicy)),
				}
			}
		case domain.ApplicationProtocolWsFed:
			if d.WsFedRPRepo == nil {
				break
			}
			if rp, err := d.WsFedRPRepo.FindByWtrealm(ctx, tenantID, protocol.Wtrealm); err == nil && rp != nil {
				wsfed = &wsfedConfig{
					Wtrealm: rp.Wtrealm, ReplyURLs: rp.ReplyURLs,
					Audience: rp.Audience, TokenType: rp.EffectiveTokenType(),
					NameIDFormat: rp.ClaimPolicy.NameID.Format, NameIDSource: rp.ClaimPolicy.NameID.SourceAttribute,
					Rules: nonNilRules(rp.ClaimPolicy.Rules),
				}
			}
		case domain.ApplicationProtocolSAML:
			if d.SamlSPRepo == nil {
				break
			}
			if sp, err := d.SamlSPRepo.FindByEntityID(ctx, tenantID, protocol.EntityID); err == nil && sp != nil {
				saml = &samlConfig{
					IDPProfileID: sp.EffectiveIDPProfileID(),
					EntityID:     sp.EntityID, ACSURLs: sp.ACSURLs, SLOURL: sp.SLOURL,
					Audience: sp.Audience, NameIDFormat: sp.ClaimPolicy.NameID.Format,
					NameIDSource:  sp.ClaimPolicy.NameID.SourceAttribute,
					SignAssertion: sp.SignAssertion, SignResponse: sp.SignResponse,
					WantAuthnRequestsSigned:           sp.WantAuthnRequestsSigned,
					AuthnRequestSigningCertificatePEM: sp.AuthnRequestSigningCertificatePEM,
					Rules:                             nonNilRules(sp.ClaimPolicy.Rules),
				}
			}
		}
	}
	return oidc, wsfed, saml
}

type updateOIDCRequest struct {
	RedirectURIs       *[]string                       `json:"redirect_uris"`
	GrantTypes         *[]spec.GrantType               `json:"grant_types"`
	ResponseTypes      *[]spec.ResponseType            `json:"response_types"`
	Scope              *string                         `json:"scope"`
	RequirePAR         *bool                           `json:"require_pushed_authorization_requests"`
	DpopBoundTokens    *bool                           `json:"dpop_bound_access_tokens"`
	SubSourceAttribute *string                         `json:"sub_source_attribute"`
	Rules              *[]claimdomain.ClaimMappingRule `json:"rules"`
}

func (d Deps) handleUpdateOIDCConfig(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := d.requireApp(c)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	clientID := bindingKeyOf(app, domain.ApplicationProtocolOIDC)
	if clientID == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The OIDC binding does not exist.")
	}
	var req updateOIDCRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	if req.Rules != nil {
		defs, err := d.resolveClaimAttributeDefs(ctx, tenantID)
		if err != nil {
			return err
		}
		if err := claimusecases.ValidateClaimReleaseRules(*req.Rules, defs); err != nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
		}
	}
	if _, err := clientusecases.UpdateAdminOAuth2Client(ctx, clientusecases.AdminOAuth2ClientDeps{ClientRepo: d.ClientRepo, Emit: d.Emit}, clientusecases.UpdateAdminOAuth2ClientInput{
		ActorUserID: actor.ID, ClientID: clientID,
		RedirectURIs: req.RedirectURIs, GrantTypes: req.GrantTypes, ResponseTypes: req.ResponseTypes,
		Scope: req.Scope, RequirePAR: req.RequirePAR, DpopBoundTokens: req.DpopBoundTokens,
		Now: time.Now().UTC(),
	}); err != nil {
		return d.writeApplicationError(c, err)
	}
	if req.SubSourceAttribute != nil || req.Rules != nil {
		client, err := d.ClientRepo.FindByID(ctx, tenantID, clientID)
		if err != nil {
			return err
		}
		if client == nil {
			return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The client does not exist.")
		}
		policy := claimdomain.ClaimMappingPolicy{}
		if client.ClaimPolicy != nil {
			policy = *client.ClaimPolicy
		}
		if policy.NameID.Format == "" {
			policy.NameID.Format = defaultNameIDSource
		}
		if req.SubSourceAttribute != nil {
			sub := strings.TrimSpace(*req.SubSourceAttribute)
			if sub == "" {
				sub = defaultNameIDSource
			}
			policy.NameID.SourceAttribute = sub
		} else if policy.NameID.SourceAttribute == "" {
			policy.NameID.SourceAttribute = defaultNameIDSource
		}
		if req.Rules != nil {
			policy.Rules = *req.Rules
		}
		client.ClaimPolicy = &policy
		client.UpdatedAt = time.Now().UTC()
		if err := d.ClientRepo.Save(ctx, client); err != nil {
			return err
		}
		d.Emit(&domain.ApplicationClaimMappingUpdated{
			At: client.UpdatedAt, TenantID: tenantID, ActorUserID: actor.ID, ApplicationID: app.ApplicationID, Protocol: string(domain.ApplicationProtocolOIDC),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

type updateWsFedRequest struct {
	ReplyURLs    *[]string                       `json:"reply_urls"`
	Audience     *string                         `json:"audience"`
	TokenType    *feddomain.WsFedTokenType       `json:"token_type"`
	NameIDFormat *string                         `json:"name_id_format"`
	NameIDSource *string                         `json:"name_id_source"`
	Rules        *[]claimdomain.ClaimMappingRule `json:"rules"`
}

func (d Deps) handleUpdateWsFedConfig(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := d.requireApp(c)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	wtrealm := bindingKeyOf(app, domain.ApplicationProtocolWsFed)
	if wtrealm == "" || d.WsFedRPRepo == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The WS-Federation binding does not exist.")
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	rp, err := d.WsFedRPRepo.FindByWtrealm(ctx, tenantID, wtrealm)
	if err != nil || rp == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The relying party does not exist.")
	}
	var req updateWsFedRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if req.Rules != nil {
		defs, err := d.resolveClaimAttributeDefs(ctx, tenantID)
		if err != nil {
			return err
		}
		if err := claimusecases.ValidateClaimReleaseRules(*req.Rules, defs); err != nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
		}
	}
	if req.ReplyURLs != nil {
		rp.ReplyURLs = *req.ReplyURLs
	}
	if req.Audience != nil {
		rp.Audience = strings.TrimSpace(*req.Audience)
	}
	if req.TokenType != nil {
		if *req.TokenType != "" && !req.TokenType.Valid() {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The token_type is invalid.")
		}
		rp.TokenType = *req.TokenType
	}
	if req.NameIDFormat != nil {
		rp.ClaimPolicy.NameID.Format = *req.NameIDFormat
	}
	if req.NameIDSource != nil {
		rp.ClaimPolicy.NameID.SourceAttribute = *req.NameIDSource
	}
	if req.Rules != nil {
		rp.ClaimPolicy.Rules = *req.Rules
	}
	now := time.Now().UTC()
	rp.UpdatedAt = now
	if err := d.WsFedRPRepo.Save(ctx, rp); err != nil {
		return err
	}
	if req.NameIDSource != nil || req.Rules != nil {
		d.Emit(&domain.ApplicationClaimMappingUpdated{
			At: now, TenantID: tenantID, ActorUserID: actor.ID, ApplicationID: app.ApplicationID, Protocol: string(domain.ApplicationProtocolWsFed),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

type updateSamlRequest struct {
	IDPProfileID                      *string                         `json:"idp_profile_id"`
	ACSURLs                           *[]string                       `json:"acs_urls"`
	SLOURL                            *string                         `json:"slo_url"`
	Audience                          *string                         `json:"audience"`
	NameIDFormat                      *string                         `json:"name_id_format"`
	NameIDSource                      *string                         `json:"name_id_source"`
	SignAssertion                     *bool                           `json:"sign_assertion"`
	SignResponse                      *bool                           `json:"sign_response"`
	WantAuthnRequestsSigned           *bool                           `json:"want_authn_requests_signed"`
	AuthnRequestSigningCertificatePEM *string                         `json:"authn_request_signing_certificate_pem"`
	Rules                             *[]claimdomain.ClaimMappingRule `json:"rules"`
}

func (d Deps) handleUpdateSamlConfig(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := d.requireApp(c)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	entityID := bindingKeyOf(app, domain.ApplicationProtocolSAML)
	if entityID == "" || d.SamlSPRepo == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The SAML binding does not exist.")
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	sp, err := d.SamlSPRepo.FindByEntityID(ctx, tenantID, entityID)
	if err != nil || sp == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The service provider does not exist.")
	}
	var req updateSamlRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if req.Rules != nil {
		defs, err := d.resolveClaimAttributeDefs(ctx, tenantID)
		if err != nil {
			return err
		}
		if err := claimusecases.ValidateClaimReleaseRules(*req.Rules, defs); err != nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
		}
	}
	if req.ACSURLs != nil {
		sp.ACSURLs = *req.ACSURLs
	}
	if req.IDPProfileID != nil {
		sp.IDPProfileID = strings.TrimSpace(*req.IDPProfileID)
	}
	if req.SLOURL != nil {
		sp.SLOURL = strings.TrimSpace(*req.SLOURL)
	}
	if req.Audience != nil {
		sp.Audience = strings.TrimSpace(*req.Audience)
	}
	if req.NameIDFormat != nil {
		sp.ClaimPolicy.NameID.Format = *req.NameIDFormat
	}
	if req.NameIDSource != nil {
		sp.ClaimPolicy.NameID.SourceAttribute = *req.NameIDSource
	}
	if req.SignAssertion != nil {
		sp.SignAssertion = *req.SignAssertion
	}
	if req.SignResponse != nil {
		sp.SignResponse = *req.SignResponse
	}
	if req.WantAuthnRequestsSigned != nil {
		sp.WantAuthnRequestsSigned = *req.WantAuthnRequestsSigned
	}
	if req.AuthnRequestSigningCertificatePEM != nil {
		sp.AuthnRequestSigningCertificatePEM = strings.TrimSpace(*req.AuthnRequestSigningCertificatePEM)
	}
	if sp.WantAuthnRequestsSigned {
		if _, err := samldomain.ParseCertificatePEM(sp.AuthnRequestSigningCertificatePEM); err != nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "Specify a certificate for AuthnRequest signature verification.")
		}
	}
	if req.Rules != nil {
		sp.ClaimPolicy.Rules = *req.Rules
	}
	now := time.Now().UTC()
	sp.UpdatedAt = now
	if err := d.SamlSPRepo.Save(ctx, sp); err != nil {
		if errors.Is(err, samldomain.ErrInvalidIDPProfile) || errors.Is(err, samldomain.ErrDedicatedIDPProfileCardinality) {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
		}
		return err
	}
	if req.NameIDSource != nil || req.Rules != nil {
		d.Emit(&domain.ApplicationClaimMappingUpdated{
			At: now, TenantID: tenantID, ActorUserID: actor.ID, ApplicationID: app.ApplicationID, Protocol: string(domain.ApplicationProtocolSAML),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) requireApp(c *echo.Context) (*domain.Application, error) {
	app, err := d.ApplicationRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), c.Param("application_id"))
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, appusecases.ErrApplicationNotFound
	}
	return app, nil
}

func bindingKeyOf(app *domain.Application, bindingType domain.ApplicationProtocolType) string {
	if app.Protocol == nil || app.Protocol.Type != bindingType {
		return ""
	}
	switch bindingType {
	case domain.ApplicationProtocolWsFed:
		return app.Protocol.Wtrealm
	case domain.ApplicationProtocolSAML:
		return app.Protocol.EntityID
	default:
		return app.Protocol.ClientID
	}
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
