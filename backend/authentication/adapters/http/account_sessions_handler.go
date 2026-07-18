// /api/account/sessions — エンドユーザー自身の有効なセッションの一覧と失効 (self-service,
// wi-20 スライス 2)。actor.sub に固定し、本人のセッションのみ参照・失効できる。失効は
// LoginSession に revoked_at / revoke_reason を設定する tombstone であり、物理削除しない
// (wi-253 / ADR-126)。
package http

import (
	"net/http"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
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

func (d Deps) sessionStore() authnports.SessionStore {
	if d.SessionManager == nil {
		return nil
	}
	return d.SessionManager.Store
}

func (d Deps) accountSessionDeps() authusecases.SessionDeps {
	return authusecases.SessionDeps{Store: d.sessionStore(), Emit: d.Emit}
}

// revokeOAuthSessionTokens は sid (LoginSession.id) に紐づく RefreshTokenRecord を
// family/client を横断して失効させる (ADR-127 §3)。RefreshStore が配線されていない
// 環境 (offline_access 未使用など) では no-op とする。
func (d Deps) revokeOAuthSessionTokens(c *echo.Context, sid string) error {
	if d.RefreshStore == nil {
		return nil
	}
	return oauthusecases.RevokeTokensBySid(
		c.Request().Context(), oauthusecases.RevokeDeps{RefreshStore: d.RefreshStore}, sid, time.Now().UTC(),
	)
}

// requireAuthenticatedSession は認証済み (pending でない) セッションの sub と sessionID を
// 返す。sessionID は "現在のセッション" の判定と revoke_others の除外に使う。
func (d Deps) requireAuthenticatedSession(c *echo.Context) (sub, sessionID string, err error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return "", "", err
	}
	if authn == nil || authn.AuthenticationPending {
		return "", "", support.ErrAdminAuthenticationRequired
	}
	return authn.UserID, authn.SessionID, nil
}

func (d Deps) handleListAccountSessions(c *echo.Context) error {
	sub, sessionID, err := d.requireAuthenticatedSession(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	views, err := authusecases.ListSessions(c.Request().Context(), d.sessionStore(), sub, sessionID)
	if err != nil {
		return err
	}
	sessions := make([]accountSessionResponse, len(views))
	for i, view := range views {
		sessions[i] = toAccountSessionResponse(view)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"sessions": sessions})
}

func (d Deps) handleRevokeAccountSession(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, _, err := d.requireAuthenticatedSession(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	targetSessionID := c.Param("id")
	if err := authusecases.RevokeOwnSession(
		c.Request().Context(), d.accountSessionDeps(), sub, targetSessionID, time.Now().UTC(),
	); err != nil {
		return d.writeAccountError(c, err)
	}
	if err := d.revokeOAuthSessionTokens(c, targetSessionID); err != nil {
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

// handleAdminListSessions は admin が対象ユーザーの有効なセッションを一覧する
// (wi-28 T007, ADR-127 決定9)。
func (d Deps) handleAdminListSessions(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	views, err := authusecases.AdminListSessions(c.Request().Context(), d.sessionStore(), c.Param("sub"))
	if err != nil {
		return err
	}
	sessions := make([]adminSessionResponse, len(views))
	for i, view := range views {
		sessions[i] = toAdminSessionResponse(view)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"sessions": sessions})
}

// handleAdminRevokeSession は admin が対象ユーザーのセッション 1 件を失効する
// (wi-28 T007)。session revoke に続けて、同じ sid を共有する RefreshTokenRecord も
// family/client を横断して失効させる (ADR-127, T004 の RevokeTokensBySid を再利用)。
func (d Deps) handleAdminRevokeSession(c *echo.Context) error {
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
		c.Request().Context(), d.accountSessionDeps(), actor.ID, targetUserID, targetSessionID, time.Now().UTC(),
	); err != nil {
		return d.writeAccountError(c, err)
	}
	if err := d.revokeOAuthSessionTokens(c, targetSessionID); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

// handleAdminRevokeAllSessions は admin が対象ユーザーの全セッションを失効する
// (wi-28 T007、"全セッションを終了" 操作)。RevokeOtherSessions と異なり操作者自身の
// セッションではないため除外対象は無い。
func (d Deps) handleAdminRevokeAllSessions(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	targetUserID := c.Param("sub")
	revokedIDs, err := authusecases.AdminRevokeUserSessions(
		c.Request().Context(), d.accountSessionDeps(), actor.ID, targetUserID, time.Now().UTC(),
	)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	for _, revokedID := range revokedIDs {
		if err := d.revokeOAuthSessionTokens(c, revokedID); err != nil {
			return err
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleRevokeOtherAccountSessions(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// 他の全セッションの失効は高 sensitivity 操作。step-up 再認証を要求する (ADR-043)。
	sub, sessionID, err := d.requireStepUpSession(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	revokedIDs, err := authusecases.RevokeOtherSessions(
		c.Request().Context(), d.accountSessionDeps(), sub, sessionID, time.Now().UTC(),
	)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	for _, revokedID := range revokedIDs {
		if err := d.revokeOAuthSessionTokens(c, revokedID); err != nil {
			return err
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
