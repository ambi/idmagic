// /par (RFC 9126 Pushed Authorization Request)
package handlers_http

import (
	"net/http"
	"time"

	authorizationusecases "github.com/ambi/idmagic/backend/oauth2/authorization/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

func (d Deps) handlePAR(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	if err := validateAuthorizationParameterCardinality(c.Request().PostForm); err != nil {
		return c.JSON(http.StatusBadRequest, support.OAuthErrorBody("invalid_request", err.Error()))
	}
	clientStub, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	params := map[string]string{}
	for k, v := range c.Request().PostForm {
		if k == "client_id" || k == "client_secret" ||
			k == "client_assertion" || k == "client_assertion_type" {
			continue
		}
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	res, err := authorizationusecases.PushAuthorizationRequest(ctx, authorizationusecases.PARDeps{
		ClientRepo: d.ClientRepo, Store: d.PARStore, AuthzDetailTypeRepo: d.AuthzDetailTypeRepo,
		McpResourceServerRepo: d.McpResourceServerRepo, Emit: d.Emit,
	}, authorizationusecases.PARInput{ClientID: clientStub.ID, Parameters: params, Resource: c.Request().PostForm["resource"]}, time.Now().UTC())
	if err != nil {
		return writeOAuthError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{
		"request_uri": res.RequestURI, "expires_in": res.ExpiresIn,
	})
}
