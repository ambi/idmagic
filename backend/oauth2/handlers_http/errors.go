package handlers_http

import (
	"errors"
	"net/http"
	"slices"

	"github.com/ambi/idmagic/backend/oauth2/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

func writeOAuthError(c *echo.Context, err error) error {
	var oe *usecases.OAuthError
	if !errors.As(err, &oe) {
		return c.JSON(http.StatusInternalServerError, support.OAuthErrorBody("server_error", "The authorization server encountered an unexpected condition."))
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

// writeUserInfoError は OAuth 認可サーバー用のエラー写像と、RFC 6750 の
// 保護リソース用エラー写像を分ける。invalid_token と insufficient_scope は
// Bearer challenge を伴う 401/403 であり、token endpoint の 400 規則へ流さない。
func writeUserInfoError(c *echo.Context, err error) error {
	var oe *usecases.OAuthError
	if !errors.As(err, &oe) {
		return writeOAuthError(c, err)
	}
	var status int
	requiredScope := ""
	switch oe.Code {
	case "invalid_token":
		status = http.StatusUnauthorized
	case "insufficient_scope":
		status = http.StatusForbidden
		requiredScope = "openid"
	default:
		return writeOAuthError(c, err)
	}
	support.SetBearerChallenge(c, oe.Code, requiredScope)
	return c.JSON(status, support.OAuthErrorBody(oe.Code, oe.Description))
}

func containsString(ss []string, s string) bool {
	return slices.Contains(ss, s)
}
