// /resource-access/tools (REQ-OAUTH2-086)
package handlers_http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// 報告は最大 100 件で、各項目の上限の合計に余裕を持たせた大きさで本文を打ち切る。
const maxToolReportBytes = 512 << 10

func (d Deps) handleReportResourceTools(c *echo.Context) error {
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxToolReportBytes)
	if err := c.Request().ParseForm(); err != nil {
		return support.NoStoreJSON(c, http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	client, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if d.RateLimiter == nil || d.McpToolReportRepo == nil {
		return support.NoStoreJSON(c, http.StatusServiceUnavailable,
			support.OAuthErrorBody("temporarily_unavailable", "tool reports are not configured"))
	}
	tenantID := support.RequestTenantID(c)
	key := fmt.Sprintf("%q:%q", tenantID, client.ID)
	if blocked, err := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "resource_access", key); err != nil {
		return err
	} else if blocked {
		return nil
	}
	form := c.Request().PostFormValue
	err = tokenusecases.ReportResourceTools(c.Request().Context(), tokenusecases.ToolReportDeps{
		Servers: d.McpResourceServerRepo, Reports: d.McpToolReportRepo,
	}, tokenusecases.ToolReportRequest{
		TenantID: tenantID, ClientID: client.ID, Tools: form("tools"), Complete: form("complete"),
	}, time.Now().UTC())
	switch {
	case errors.Is(err, tokenusecases.ErrInvalidToolReport):
		return support.NoStoreJSON(c, http.StatusBadRequest,
			support.OAuthErrorBody("invalid_request", "tools or complete is not acceptable"))
	case errors.Is(err, tokenusecases.ErrResourceServerNotRegistered):
		return support.NoStoreJSON(c, http.StatusForbidden,
			support.OAuthErrorBody("access_denied", "the client does not represent an active resource server"))
	case err != nil:
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
