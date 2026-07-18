// /authorize エンドポイント。既存セッションの再利用可否判定とログイン誘導を行う。
// ブラウザ認証 API (login/totp/consent)・/end_session・login throttle は各concern別の
// authorize_*.go / end_session_handler.go / login_throttle.go に分割している。
package http

import (
	"net/http"
	"net/url"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleAuthorize(c *echo.Context) error {
	q := c.QueryParams()
	parUsed := false
	if requestURI := q.Get("request_uri"); requestURI != "" {
		consumed, err := d.PARStore.Consume(c.Request().Context(), requestURI)
		if err != nil {
			return writeOAuthError(c, err)
		}
		if consumed == nil {
			return writeOAuthError(c, usecases.NewOAuthError("invalid_request_uri", "request_uri 無効または使用済み"))
		}
		if consumed.TenantID != support.RequestTenantID(c) {
			return writeOAuthError(c, usecases.NewOAuthError("invalid_request_uri", "request_uri 無効または使用済み"))
		}
		if cid := q.Get("client_id"); cid != "" && cid != consumed.ClientID {
			return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "client_id が PAR と不一致"))
		}
		q = url.Values{}
		for k, v := range consumed.Parameters {
			q.Set(k, v)
		}
		q.Set("client_id", consumed.ClientID)
		parUsed = true
	}

	request, err := parseAuthorizeRequest(q)
	if err != nil {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", err.Error()))
	}
	details, err := usecases.ParseAuthorizationDetails(q.Get("authorization_details"))
	if err != nil {
		return writeOAuthError(c, err)
	}
	in := usecases.AuthorizeRequestInput{
		ClientID: request.ClientID, RedirectURI: request.RedirectURI,
		ResponseType: request.ResponseType, Scope: request.Scope,
		StateParam: request.StateParam, Nonce: request.Nonce,
		CodeChallenge: request.CodeChallenge, CodeChallengeMethod: request.CodeChallengeMethod,
		Prompt: request.Prompt, MaxAge: request.MaxAge, ACRValues: request.AcrValues, ParUsed: parUsed,
		AuthorizationDetails: details,
	}
	if requestURI := c.QueryParam("request_uri"); requestURI != "" {
		in.ParRequestURI = requestURI
	}
	out, err := usecases.Authorize(c.Request().Context(), usecases.AuthorizeDeps{
		ClientRepo:          d.ClientRepo,
		RequestStore:        d.RequestStore,
		AuthzDetailTypeRepo: d.AuthzDetailTypeRepo,
	}, in)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if len(details) > 0 && d.Emit != nil {
		d.Emit(&oauthdomain.AuthorizationDetailsRequested{
			At: time.Now().UTC(), TenantID: support.RequestTenantID(c), ClientID: out.Request.ClientID,
			DetailTypes: oauthdomain.DetailTypes(details),
		})
	}

	d.setTransactionCookie(c, out.Request.ID)
	if d.AuthnResolver != nil {
		authn, _ := d.AuthnResolver.Resolve(c.Request().Context(), authdomain.HTTPHeadersAdapter{H: c.Request().Header})
		if authn != nil {
			if authn.AuthenticationPending {
				if in.Prompt == "none" {
					return writeOAuthError(c, usecases.NewOAuthError("login_required", "追加factor検証が必要です"))
				}
				return c.Redirect(http.StatusSeeOther, d.pendingAuthPath(c, authn))
			}
			policy := oauthdomain.ParsePrompt(out.Request)
			needsStepUp := out.Request.ACRValues != nil &&
				!authusecases.ACRSatisfies(authn.ACR, *out.Request.ACRValues)
			if oauthdomain.NeedsReauthentication(policy, time.Unix(authn.AuthTime, 0), time.Now(), false) ||
				needsStepUp {
				if in.Prompt == "none" {
					return writeOAuthError(c, usecases.NewOAuthError("login_required", "既存セッションが認証要件を満たしません"))
				}
				if needsStepUp && d.canUseTOTP(c, authn.UserID) {
					pending, err := d.SessionManager.RequireFactor(c.Request().Context(), authn.SessionID)
					if err != nil {
						return err
					}
					if pending == nil {
						return writeOAuthError(c, usecases.NewOAuthError("login_required", "既存セッションが認証要件を満たしません"))
					}
					d.setSessionCookie(c, pending.SessionID)
					return c.Redirect(http.StatusSeeOther, d.pendingAuthPath(c, authn))
				}
				return c.Redirect(http.StatusSeeOther, support.TenantRoute(c, "/login"))
			}
			if out.Client.FirstParty {
				redirected, err := d.enforceDefaultSignInPolicy(c, authn, in.Prompt != "none")
				if err != nil {
					return err
				}
				if redirected {
					if in.Prompt == "none" {
						return writeOAuthError(c, usecases.NewOAuthError("login_required", "既存セッションが認証要件を満たしません"))
					}
					return c.Redirect(http.StatusSeeOther, d.pendingAuthPath(c, authn))
				}
			}
			next, err := d.completeAfterAuthn(c, out.Request, out.Client, authn)
			if err != nil {
				return err
			}
			if next.RedirectTo != "" {
				d.clearTransactionCookie(c)
			}
			return redirectAuthorizationNext(c, next)
		}
	}
	if out.Request.Prompt != nil && *out.Request.Prompt == "none" {
		return writeOAuthError(c, usecases.NewOAuthError("login_required", "prompt=none では再認証不可"))
	}
	return c.Redirect(http.StatusSeeOther, support.TenantRoute(c, "/login"))
}
