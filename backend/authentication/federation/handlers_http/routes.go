package handlers_http

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/federation/protocol_oidc"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type Deps struct {
	Broker federationusecases.BrokerDeps
	Auth   httpdeps.Deps
	OIDC   *oidcprotocol.Client
}

func RegisterRoutes(group *echo.Group, deps Deps) {
	group.GET("/api/auth/federation/providers", deps.listPublicProviders)
	group.GET("/api/auth/federation/start", func(c *echo.Context) error { return deps.start(c, "", false) })
	group.GET("/api/auth/federation/oidc/callback", func(c *echo.Context) error {
		return deps.complete(c, c.QueryParam("state"), c.QueryParam("code"), "/api/auth/federation/oidc/callback")
	})
	group.POST("/api/auth/federation/saml/callback", func(c *echo.Context) error {
		return deps.complete(c, c.FormValue("RelayState"), c.FormValue("SAMLResponse"), "/api/auth/federation/saml/callback")
	})
	group.GET("/api/account/linked-identities", deps.listLinked)
	group.POST("/api/account/linked-identities/:provider_id", func(c *echo.Context) error { return deps.start(c, c.Param("provider_id"), true) })
	group.DELETE("/api/account/linked-identities/:provider_id", deps.unlink)
	group.GET("/api/admin/identity-providers", deps.listAdmin)
	group.POST("/api/admin/identity-providers", deps.createAdmin)
	group.PUT("/api/admin/identity-providers/:provider_id", deps.updateAdmin)
	group.DELETE("/api/admin/identity-providers/:provider_id", deps.deleteAdmin)
	group.POST("/api/admin/identity-providers/:provider_id/activate", deps.activate)
	group.POST("/api/admin/identity-providers/:provider_id/disable", deps.disable)
	group.POST("/api/admin/identity-providers/:provider_id/refresh", deps.refresh)
	group.POST("/api/admin/identity-providers/:provider_id/test", deps.test)
	group.POST("/api/admin/identity-providers/:provider_id/mapping-preview", deps.previewMapping)
}

type providerSummary struct {
	ID          string                    `json:"id"`
	DisplayName string                    `json:"display_name"`
	Protocol    federationdomain.Protocol `json:"protocol"`
}

type connectionInput struct {
	DisplayName             string                         `json:"display_name"`
	Protocol                federationdomain.Protocol      `json:"protocol"`
	Issuer                  string                         `json:"issuer"`
	ClientID                string                         `json:"client_id"`
	SecretReference         string                         `json:"secret_reference"`
	AuthorizationEndpoint   string                         `json:"authorization_endpoint"`
	TokenEndpoint           string                         `json:"token_endpoint"`
	JWKSURI                 string                         `json:"jwks_uri"`
	SAMLSSOURL              string                         `json:"saml_sso_url"`
	SAMLEntityID            string                         `json:"saml_entity_id"`
	SAMLSigningCertificates []string                       `json:"saml_signing_certificates"`
	ClaimMapping            federationdomain.ClaimMapping  `json:"claim_mapping"`
	LinkingPolicy           federationdomain.LinkingPolicy `json:"linking_policy"`
	JITProvisioning         bool                           `json:"jit_provisioning"`
	AllowedEmailDomains     []string                       `json:"allowed_email_domains"`
}

func (input connectionInput) apply(connection *federationdomain.IdentityProviderConnection) {
	connection.DisplayName = input.DisplayName
	connection.Protocol = input.Protocol
	connection.Issuer = input.Issuer
	connection.ClientID = input.ClientID
	connection.SecretReference = input.SecretReference
	connection.AuthorizationEndpoint = input.AuthorizationEndpoint
	connection.TokenEndpoint = input.TokenEndpoint
	connection.JWKSURI = input.JWKSURI
	connection.SAMLSSOURL = input.SAMLSSOURL
	connection.SAMLEntityID = input.SAMLEntityID
	connection.SAMLSigningCertificates = input.SAMLSigningCertificates
	connection.ClaimMapping = input.ClaimMapping
	connection.LinkingPolicy = input.LinkingPolicy
	connection.JITProvisioning = input.JITProvisioning
	connection.AllowedEmailDomains = input.AllowedEmailDomains
}

func (d Deps) listPublicProviders(c *echo.Context) error {
	connections, err := d.Broker.Connections.List(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	providers := make([]providerSummary, 0, len(connections))
	for _, connection := range connections {
		if connection.Active() {
			providers = append(providers, providerSummary{
				ID: connection.ID, DisplayName: connection.DisplayName, Protocol: connection.Protocol,
			})
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"providers": providers})
}

func (d Deps) start(c *echo.Context, providerID string, linking bool) error {
	if providerID == "" {
		providerID = c.QueryParam("provider_id")
	}
	returnTo := c.QueryParam("return_to")
	if returnTo != "" && !validReturnTo(c, returnTo) {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "return_to is invalid.")
	}
	linkUserID := ""
	if linking {
		if err := d.Auth.VerifyBrowserRequest(c); err != nil {
			return err
		}
		sub, _, err := httpdeps.RequireStepUpSession(d.Auth, c)
		if err != nil {
			return httpdeps.WriteAccountError(c, err)
		}
		linkUserID = sub
	}
	connection, err := d.Broker.Connections.Find(c.Request().Context(), support.RequestTenantID(c), providerID)
	if err != nil || connection == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The identity provider is unavailable.")
	}
	callbackPath := "/api/auth/federation/oidc/callback"
	if connection.Protocol == federationdomain.ProtocolSAML {
		callbackPath = "/api/auth/federation/saml/callback"
	}
	callbackURL := support.TenantURL(c, callbackPath, d.Auth.Issuer)
	start, err := federationusecases.StartLogin(
		c.Request().Context(), d.Broker, providerID, returnTo, linkUserID, callbackURL, time.Now().UTC(),
	)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "federation_failed", "The identity provider login could not be started.")
	}
	return c.Redirect(http.StatusSeeOther, start.RedirectTo)
}

func (d Deps) complete(c *echo.Context, state, response, callbackPath string) error {
	if state == "" || response == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The federation response is incomplete.")
	}
	completion, err := federationusecases.CompleteLogin(
		c.Request().Context(), d.Broker, state, response,
		support.TenantURL(c, callbackPath, d.Auth.Issuer), time.Now().UTC(),
	)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "federation_failed", "The identity provider response was rejected.")
	}
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure is enabled for HTTPS issuers; local HTTP development intentionally disables it.
		Name:  support.TenantCookieName(c, sessionusecases.SessionCookie),
		Value: completion.Authentication.SessionID, Path: support.TenantCookiePath(c),
		Secure:   d.Auth.SecureCookies() || support.TenantCookieSecure(c),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: sessionusecases.SessionTTLSeconds,
	})
	if completion.LinkingMethod == federationusecases.LinkingMethodExplicit {
		return c.Redirect(http.StatusSeeOther, support.TenantRoute(c, "/account/security"))
	}
	if completion.ReturnTo != "" {
		return c.Redirect(http.StatusSeeOther, completion.ReturnTo)
	}
	if completion.User != nil && completion.Authentication != nil {
		// The OAuth2 resume endpoint applies the existing transaction's application policy,
		// consent, and required-action gates after the Authentication-owned callback.
		return c.Redirect(http.StatusSeeOther, support.TenantRoute(c, "/authorize/resume"))
	}
	return errors.New("federation completion is incomplete")
}

func (d Deps) listLinked(c *echo.Context) error {
	authn, err := httpdeps.RequireAuthenticatedAuthn(d.Auth, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	identities, err := d.Broker.Identities.ListByUser(
		c.Request().Context(), support.RequestTenantID(c), authn.UserID,
	)
	if err != nil {
		return err
	}
	for _, identity := range identities {
		identity.ExternalSubject = ""
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"identities": identities})
}

func (d Deps) unlink(c *echo.Context) error {
	if err := d.Auth.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, _, err := httpdeps.RequireStepUpSession(d.Auth, c); err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	authn, err := httpdeps.RequireAuthenticatedAuthn(d.Auth, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	if err := federationusecases.UnlinkIdentity(
		c.Request().Context(), d.Broker, authn, c.Param("provider_id"), time.Now().UTC(),
	); err != nil {
		return support.WriteBrowserError(c, http.StatusForbidden, "unlink_denied", "The linked identity cannot be removed.")
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) requireAdmin(c *echo.Context, csrf bool) error {
	if csrf {
		if err := d.Auth.VerifyBrowserRequest(c); err != nil {
			return err
		}
	}
	if _, err := d.Auth.RequireAdmin(c); err != nil {
		return d.Auth.WriteAdminAccessError(c, err)
	}
	return nil
}

func (d Deps) listAdmin(c *echo.Context) error {
	if err := d.requireAdmin(c, false); err != nil {
		return err
	}
	connections, err := d.Broker.Connections.List(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"connections": connections})
}

func (d Deps) createAdmin(c *echo.Context) error {
	if err := d.requireAdmin(c, true); err != nil {
		return err
	}
	var input connectionInput
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	var connection federationdomain.IdentityProviderConnection
	input.apply(&connection)
	now := time.Now().UTC()
	if connection.ID == "" {
		connection.ID = uuid.NewString()
	}
	connection.TenantID, connection.Status = support.RequestTenantID(c), federationdomain.ConnectionDraft
	connection.CreatedAt, connection.UpdatedAt = now, now
	if err := d.Broker.Connections.Save(c.Request().Context(), &connection); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	connection.SecretReference = ""
	return support.NoStoreJSON(c, http.StatusCreated, connection)
}

func (d Deps) updateAdmin(c *echo.Context) error {
	if err := d.requireAdmin(c, true); err != nil {
		return err
	}
	existing, err := d.Broker.Connections.Find(c.Request().Context(), support.RequestTenantID(c), c.Param("provider_id"))
	if err != nil || existing == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The identity provider does not exist.")
	}
	var input connectionInput
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	var connection federationdomain.IdentityProviderConnection
	input.apply(&connection)
	connection.ID, connection.TenantID = existing.ID, existing.TenantID
	connection.Status, connection.CreatedAt = federationdomain.ConnectionDraft, existing.CreatedAt
	connection.UpdatedAt = time.Now().UTC()
	if connection.SecretReference == "" {
		connection.SecretReference = existing.SecretReference
	}
	if err := d.Broker.Connections.Save(c.Request().Context(), &connection); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	connection.SecretReference = ""
	return support.NoStoreJSON(c, http.StatusOK, connection)
}

func (d Deps) deleteAdmin(c *echo.Context) error {
	if err := d.requireAdmin(c, true); err != nil {
		return err
	}
	connection, err := d.Broker.Connections.Find(c.Request().Context(), support.RequestTenantID(c), c.Param("provider_id"))
	if err != nil || connection == nil {
		return c.NoContent(http.StatusNoContent)
	}
	if connection.Status != federationdomain.ConnectionDisabled {
		return support.WriteBrowserError(c, http.StatusConflict, "invalid_state", "Disable the identity provider before deleting it.")
	}
	if err := d.Broker.Connections.Delete(c.Request().Context(), connection.TenantID, connection.ID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) activate(c *echo.Context) error {
	return d.changeStatus(c, true)
}

func (d Deps) disable(c *echo.Context) error {
	return d.changeStatus(c, false)
}

func (d Deps) changeStatus(c *echo.Context, activate bool) error {
	if err := d.requireAdmin(c, true); err != nil {
		return err
	}
	connection, err := d.Broker.Connections.Find(c.Request().Context(), support.RequestTenantID(c), c.Param("provider_id"))
	if err != nil || connection == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The identity provider does not exist.")
	}
	now := time.Now().UTC()
	if activate {
		err = connection.Activate(now)
	} else {
		connection.Disable(now)
	}
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_state", err.Error())
	}
	if err := d.Broker.Connections.Save(c.Request().Context(), connection); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) refresh(c *echo.Context) error {
	if err := d.requireAdmin(c, true); err != nil {
		return err
	}
	connection, err := d.Broker.Connections.Find(c.Request().Context(), support.RequestTenantID(c), c.Param("provider_id"))
	if err != nil || connection == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The identity provider does not exist.")
	}
	if connection.Protocol != federationdomain.ProtocolOIDC || d.OIDC == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "unsupported", "Automatic metadata refresh is unavailable for this provider.")
	}
	if err := d.OIDC.RefreshDiscovery(c.Request().Context(), connection, time.Now().UTC()); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "metadata_invalid", "The identity provider metadata was rejected.")
	}
	if err := d.Broker.Connections.Save(c.Request().Context(), connection); err != nil {
		return err
	}
	connection.SecretReference = ""
	return support.NoStoreJSON(c, http.StatusOK, connection)
}

func (d Deps) test(c *echo.Context) error {
	if err := d.requireAdmin(c, true); err != nil {
		return err
	}
	connection, err := d.Broker.Connections.Find(c.Request().Context(), support.RequestTenantID(c), c.Param("provider_id"))
	if err != nil || connection == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The identity provider does not exist.")
	}
	if err := connection.Validate(); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "connection_invalid", err.Error())
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"result": "valid"})
}

func (d Deps) previewMapping(c *echo.Context) error {
	if err := d.requireAdmin(c, true); err != nil {
		return err
	}
	connection, err := d.Broker.Connections.Find(
		c.Request().Context(), support.RequestTenantID(c), c.Param("provider_id"),
	)
	if err != nil || connection == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "not_found", "The identity provider does not exist.")
	}
	var input struct {
		Claims map[string]any `json:"claims"`
	}
	if err := support.DecodeJSON(c.Request(), &input); err != nil || input.Claims == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "A claims object is required.")
	}
	preview, err := oidcprotocol.NormalizeClaims(connection.ClaimMapping, input.Claims)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_mapping", err.Error())
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"preview": preview})
}

func validReturnTo(c *echo.Context, raw string) bool {
	if strings.Contains(raw, "\\") {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" || path.Clean(parsed.Path) != parsed.Path {
		return false
	}
	for _, root := range []string{support.TenantRoute(c, "/admin"), support.TenantRoute(c, "/account")} {
		if parsed.Path == root || strings.HasPrefix(parsed.Path, root+"/") {
			return true
		}
	}
	return false
}
