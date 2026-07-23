// /register (RFC 7591 Dynamic Client Registration)
package handlers_http

import (
	"errors"
	"net/http"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	clientusecases "github.com/ambi/idmagic/backend/oauth2/client/usecases"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleRegisterClient(c *echo.Context) error {
	var req registerClientRequest
	if err := c.Bind(&req); err != nil {
		return writeOAuthError(c, clientusecases.NewOAuthError("invalid_request", err.Error()))
	}
	if err := validateRegisterClientRequest(&req); err != nil {
		return writeOAuthError(c, clientusecases.NewOAuthError("invalid_client_metadata", err.Error()))
	}
	if req.JwksURI != nil {
		if err := tokens_jose.ValidateJWKSURI(*req.JwksURI); err != nil {
			return writeOAuthError(c, clientusecases.NewOAuthError("invalid_client_metadata", err.Error()))
		}
	}
	in := clientusecases.RegisterClientInput{
		ClientName:              req.ClientName,
		ClientType:              spec.ClientType(req.ClientType),
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: oauthdomain.TokenEndpointAuthMethod(req.TokenEndpointAuthMethod),
		Scope:                   req.Scope,
		JWKS:                    req.JWKS,
		JwksURI:                 req.JwksURI,
		TlsClientAuthSubjectDN:  req.TlsClientAuthSubjectDN,
		RequirePAR:              req.RequirePAR,
		DpopBoundAccessTokens:   req.DpopBoundAccessTokens,
		FapiProfile:             oauthdomain.FapiProfile(req.FapiProfile),
	}
	for _, g := range req.GrantTypes {
		in.GrantTypes = append(in.GrantTypes, spec.GrantType(g))
	}
	for _, r := range req.ResponseTypes {
		in.ResponseTypes = append(in.ResponseTypes, spec.ResponseType(r))
	}
	result, err := clientusecases.RegisterClient(c.Request().Context(), clientusecases.RegisterClientDeps{
		ClientRepo: d.ClientRepo, Emit: d.Emit, QuotaRepo: d.QuotaRepo,
	}, in, time.Now().UTC())
	if err != nil {
		// QuotaExceededError bypasses the RFC 7591 error mapping (writeOAuthError
		// only understands *usecases.OAuthError and would otherwise flatten a
		// quota rejection into a generic 500) so it gets the same stable
		// quota_exceeded response, logging, and metrics as every other create
		// path (support_http.ErrorHandler, wi-160 ADR-134).
		if _, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
			return err
		}
		return writeOAuthError(c, err)
	}
	resp := map[string]any{
		"client_id":                             result.Client.ClientID,
		"client_type":                           result.Client.ClientType,
		"redirect_uris":                         result.Client.RedirectURIs,
		"grant_types":                           result.Client.GrantTypes,
		"response_types":                        result.Client.ResponseTypes,
		"token_endpoint_auth_method":            result.Client.TokenEndpointAuthMethod,
		"scope":                                 result.Client.Scope,
		"require_pushed_authorization_requests": result.Client.RequirePushedAuthorizationRequests,
		"dpop_bound_access_tokens":              result.Client.DpopBoundAccessTokens,
		"fapi_profile":                          result.Client.FapiProfile,
	}
	if result.Client.JWKS != nil {
		resp["jwks"] = result.Client.JWKS
	}
	if result.Client.JwksURI != nil {
		resp["jwks_uri"] = *result.Client.JwksURI
	}
	if result.Client.TlsClientAuthSubjectDN != nil {
		resp["tls_client_auth_subject_dn"] = *result.Client.TlsClientAuthSubjectDN
	}
	if result.ClientSecret != "" {
		resp["client_secret"] = result.ClientSecret
	}
	return c.JSON(http.StatusCreated, resp)
}
