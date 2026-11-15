package handlers_http

import (
	"errors"
	"net/http"

	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

func (d Deps) authorizationOrDirectLogin(c *echo.Context, returnTo string) (signinports.Authorization, bool, error) {
	authorization, err := d.Authorizations.Find(c)
	if err == nil {
		return authorization, false, nil
	}
	var unavailable *signinports.AuthorizationUnavailableError
	if !errors.As(err, &unavailable) {
		return nil, false, err
	}
	if returnTo != "" && validReturnTo(c, returnTo) {
		return nil, true, nil
	}
	if returnTo != "" {
		err = support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "return_to is invalid.")
	} else {
		err = support.WriteProblem(c, http.StatusUnauthorized, "transaction_unavailable", unavailable.Error())
	}
	if err != nil {
		return nil, false, err
	}
	return nil, false, support.ErrResponseWritten
}
