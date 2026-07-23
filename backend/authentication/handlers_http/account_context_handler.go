// /api/auth/account: 認証済みセッション向けのアカウントコンテキスト取得。
// authentication コンテキスト所有。パスワード変更は password feature の handlers_http
// (ADR-130 Phase 2) へ分割されている。
package handlers_http

import (
	"net/http"

	"github.com/labstack/echo/v5"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"
)

type accountContextResponse struct {
	CSRFToken         string   `json:"csrf_token"`
	ID                string   `json:"id"`
	PreferredUsername string   `json:"preferred_username,omitempty"`
	TenantID          string   `json:"tenant_id,omitempty"`
	Realm             string   `json:"realm,omitempty"`
	Roles             []string `json:"roles,omitempty"`
}

func handleAccountContext(d Deps, c *echo.Context) error {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		if handled, result := support.WriteAccessTokenError(c, err); handled {
			return result
		}
		return err
	}
	if authn == nil || authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required.")
	}
	csrf, err := d.EnsureCSRFCookie(c)
	if err != nil {
		return err
	}
	resp := accountContextResponse{CSRFToken: csrf, ID: authn.UserID}
	// realm はリクエストが解決した現在テナントの公開 slug (ADR-085)。UI の system console
	// gating (realm == "default") に使う。
	if t := tenancy.Tenant(c.Request().Context()); t != nil {
		resp.Realm = t.Realm
	}
	if d.UserRepo != nil {
		if user, _ := d.UserRepo.FindBySub(c.Request().Context(), authn.UserID); user != nil {
			resp.PreferredUsername = user.PreferredUsername
			resp.TenantID = user.TenantID
			// グループ由来ロールを含む有効ロールを返す (ADR-038)。
			resp.Roles = d.EffectiveRoles(c.Request().Context(), user)
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, resp)
}
