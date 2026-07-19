// /session/check (OIDC Session Management 1.0 check_session_iframe, Draft 28,
// adoption: optional per ADR-127 決定8)。session_state の salted hash 相関
// アルゴリズムは実装せず、現在の browser cookie が有効な LoginSession に解決できるかを
// ページロード時点で判定し、静的な HTML/JS に埋め込んで返す。
package http

import (
	"net/http"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleCheckSessionIframe(c *echo.Context) error {
	status := "changed"
	if d.AuthnResolver != nil {
		authn, err := d.AuthnResolver.Resolve(c.Request().Context(), authdomain.HTTPHeadersAdapter{H: c.Request().Header})
		if err == nil && authn != nil && !authn.AuthenticationPending {
			status = "unchanged"
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.HTML(http.StatusOK, checkSessionIframeHTML(status))
}

// checkSessionIframeHTML は RP の hidden iframe から読み込まれる静的ページを返す。
// RP からの postMessage を受け取ったら、埋め込み済みの status ("changed"|"unchanged") を
// そのまま送信元 origin へ postMessage で返す。
func checkSessionIframeHTML(status string) string {
	return `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>check_session_iframe</title></head>
<body><script>
(function () {
  var STATUS = "` + status + `";
  window.addEventListener("message", function (e) {
    if (!e.source) { return; }
    e.source.postMessage(STATUS, e.origin);
  }, false);
})();
</script></body></html>`
}
