// /register (RFC 7591 Dynamic Client Registration)
package http

import (
	"net/http"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	clientusecases "github.com/ambi/idmagic/backend/oauth2/client/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"

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
		if err := crypto.ValidateJWKSURI(*req.JwksURI); err != nil {
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
		ClientRepo: d.ClientRepo, Emit: d.Emit,
	}, in, time.Now().UTC())
	if err != nil {
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
