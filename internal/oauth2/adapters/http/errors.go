package http

import (
	"errors"
	"net/http"
	"slices"

	"idmagic/internal/oauth2/usecases"
	"idmagic/internal/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

func writeOAuthError(c *echo.Context, err error) error {
	var oe *usecases.OAuthError
	if !errors.As(err, &oe) {
		return c.JSON(http.StatusInternalServerError, support.OAuthErrorBody("server_error", err.Error()))
	}
	status := http.StatusBadRequest
	switch oe.Code {
	case "invalid_client":
		status = http.StatusUnauthorized
	case "server_error":
		status = http.StatusInternalServerError
	}
	return c.JSON(status, support.OAuthErrorBody(oe.Code, oe.Description))
}

func containsString(ss []string, s string) bool {
	return slices.Contains(ss, s)
}
