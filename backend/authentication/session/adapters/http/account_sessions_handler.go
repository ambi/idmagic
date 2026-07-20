// /api/account/sessions — エンドユーザー自身の有効なセッションの一覧と失効 (self-service,
// wi-20 スライス 2)。actor.sub に固定し、本人のセッションのみ参照・失効できる。失効は
// LoginSession に revoked_at / revoke_reason を設定する tombstone であり、物理削除しない
// (wi-253 / ADR-126)。
package http

import (
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/authentication/adapters/http/httpdeps"
	authnports "github.com/ambi/idmagic/backend/authentication/session/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type accountSessionResponse struct {
	ID        string    `json:"id"`
	Current   bool      `json:"current"`
	AMR       []string  `json:"amr"`
	ACR       string    `json:"acr"`
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func toAccountSessionResponse(view authusecases.SessionView) accountSessionResponse {
	amr := view.AMR
	if amr == nil {
		amr = []string{}
	}
	return accountSessionResponse{
		ID: view.ID, Current: view.Current, AMR: amr, ACR: view.ACR,
		StartedAt: view.StartedAt, ExpiresAt: view.ExpiresAt,
	}
}

func sessionStore(d httpdeps.Deps) authnports.SessionStore {
	if d.SessionManager == nil {
		return nil
	}
	return d.SessionManager.Store
}

func accountSessionDeps(d httpdeps.Deps) authusecases.SessionDeps {
	return authusecases.SessionDeps{Store: sessionStore(d), Emit: d.Emit}
}

// revokeOAuthSessionTokens は sid (LoginSession.id) に紐づく RefreshTokenRecord を
// family/client を横断して失効させる (ADR-127 §3)。RefreshStore が配線されていない
// 環境 (offline_access 未使用など) では no-op とする。
func revokeOAuthSessionTokens(d httpdeps.Deps, c *echo.Context, sid string) error {
	if d.RefreshStore == nil {
		return nil
	}
	return oauthusecases.RevokeTokensBySid(
		c.Request().Context(), oauthusecases.RevokeDeps{RefreshStore: d.RefreshStore}, sid, time.Now().UTC(),
	)
}

// requireAuthenticatedSession は認証済み (pending でない) セッションの sub と sessionID を
// 返す。sessionID は "現在のセッション" の判定と revoke_others の除外に使う。
func requireAuthenticatedSession(d httpdeps.Deps, c *echo.Context) (sub, sessionID string, err error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return "", "", err
	}
	if authn == nil || authn.AuthenticationPending {
		return "", "", support.ErrAdminAuthenticationRequired
	}
	return authn.UserID, authn.SessionID, nil
}

func HandleListAccountSessions(d httpdeps.Deps, c *echo.Context) error {
	sub, sessionID, err := requireAuthenticatedSession(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	views, err := authusecases.ListSessions(c.Request().Context(), sessionStore(d), sub, sessionID)
	if err != nil {
		return err
	}
	sessions := make([]accountSessionResponse, len(views))
	for i, view := range views {
		sessions[i] = toAccountSessionResponse(view)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"sessions": sessions})
}

func HandleRevokeAccountSession(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, _, err := requireAuthenticatedSession(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	targetSessionID := c.Param("id")
	if err := authusecases.RevokeOwnSession(
		c.Request().Context(), accountSessionDeps(d), sub, targetSessionID, time.Now().UTC(),
	); err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	if err := revokeOAuthSessionTokens(d, c, targetSessionID); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

type adminSessionResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	AMR        []string  `json:"amr"`
	ACR        string    `json:"acr"`
	StartedAt  time.Time `json:"started_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func toAdminSessionResponse(view authusecases.AdminSessionView) adminSessionResponse {
	amr := view.AMR
	if amr == nil {
		amr = []string{}
	}
	return adminSessionResponse{
		ID: view.ID, UserID: view.UserID, AMR: amr, ACR: view.ACR,
		StartedAt: view.StartedAt, LastSeenAt: view.LastSeenAt, ExpiresAt: view.ExpiresAt,
	}
}

// HandleAdminListSessions は admin が対象ユーザーの有効なセッションを一覧する
// (wi-28 T007, ADR-127 決定9)。
func HandleAdminListSessions(d httpdeps.Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	views, err := authusecases.AdminListSessions(c.Request().Context(), sessionStore(d), c.Param("sub"))
	if err != nil {
		return err
	}
	sessions := make([]adminSessionResponse, len(views))
	for i, view := range views {
		sessions[i] = toAdminSessionResponse(view)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"sessions": sessions})
}

// HandleAdminRevokeSession は admin が対象ユーザーのセッション 1 件を失効する
// (wi-28 T007)。session revoke に続けて、同じ sid を共有する RefreshTokenRecord も
// family/client を横断して失効させる (ADR-127, T004 の RevokeTokensBySid を再利用)。
func HandleAdminRevokeSession(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	targetUserID := c.Param("sub")
	targetSessionID := c.Param("id")
	if err := authusecases.AdminRevokeSession(
		c.Request().Context(), accountSessionDeps(d), actor.ID, targetUserID, targetSessionID, time.Now().UTC(),
	); err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	if err := revokeOAuthSessionTokens(d, c, targetSessionID); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

// HandleAdminRevokeAllSessions は admin が対象ユーザーの全セッションを失効する
// (wi-28 T007、"全セッションを終了" 操作)。RevokeOtherSessions と異なり操作者自身の
// セッションではないため除外対象は無い。
func HandleAdminRevokeAllSessions(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	targetUserID := c.Param("sub")
	revokedIDs, err := authusecases.AdminRevokeUserSessions(
		c.Request().Context(), accountSessionDeps(d), actor.ID, targetUserID, time.Now().UTC(),
	)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	for _, revokedID := range revokedIDs {
		if err := revokeOAuthSessionTokens(d, c, revokedID); err != nil {
			return err
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleRevokeOtherAccountSessions(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// 他の全セッションの失効は高 sensitivity 操作。step-up 再認証を要求する (ADR-043)。
	sub, sessionID, err := httpdeps.RequireStepUpSession(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	revokedIDs, err := authusecases.RevokeOtherSessions(
		c.Request().Context(), accountSessionDeps(d), sub, sessionID, time.Now().UTC(),
	)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	for _, revokedID := range revokedIDs {
		if err := revokeOAuthSessionTokens(d, c, revokedID); err != nil {
			return err
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
