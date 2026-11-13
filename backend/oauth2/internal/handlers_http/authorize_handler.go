// /authorize エンドポイント。既存セッションの再利用可否判定とログイン誘導を行う。
// ブラウザ認証 API (login/totp/consent)・/end_session・login throttle は各concern別の
// authorize_*.go / end_session_handler.go / login_throttle.go に分割している。
package handlers_http

import (
	"net/http"
	"net/url"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	authorizationusecases "github.com/ambi/idmagic/backend/oauth2/internal/authorization/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleAuthorize(c *echo.Context) error {
	q := c.QueryParams()
	if err := validateAuthorizationParameterCardinality(q); err != nil {
		return writeOAuthError(c, authorizationusecases.NewOAuthError("invalid_request", err.Error()))
	}
	parUsed := false
	if requestURI := q.Get("request_uri"); requestURI != "" {
		// RFC 9101 §6.3: request_uri と client_id 以外にクエリへ重ねた値は拒否せず使わない。
		// 下で PAR に保存した値だけから q を作り直す。
		consumed, err := d.PARStore.Consume(c.Request().Context(), requestURI)
		if err != nil {
			return writeOAuthError(c, err)
		}
		if consumed == nil {
			return writeOAuthError(c, authorizationusecases.NewOAuthError("invalid_request_uri", "request_uri is invalid or already used"))
		}
		if consumed.TenantID != support.RequestTenantID(c) {
			return writeOAuthError(c, authorizationusecases.NewOAuthError("invalid_request_uri", "request_uri is invalid or already used"))
		}
		if cid := q.Get("client_id"); cid != "" && cid != consumed.ClientID {
			return writeOAuthError(c, authorizationusecases.NewOAuthError("invalid_request", "client_id does not match PAR"))
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
		return writeOAuthError(c, authorizationusecases.NewOAuthError("invalid_request", err.Error()))
	}
	clientIP := c.RealIP()
	if blocked, err := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "authorize", clientIP+"|"+request.ClientID); err != nil {
		return err
	} else if blocked {
		return nil
	}
	in := authorizationusecases.AuthorizeRequestInput{
		ClientID: request.ClientID, RedirectURI: request.RedirectURI,
		ResponseType: request.ResponseType, Scope: request.Scope,
		StateParam: request.StateParam, Nonce: request.Nonce,
		CodeChallenge: request.CodeChallenge, CodeChallengeMethod: request.CodeChallengeMethod,
		Prompt: request.Prompt, MaxAge: request.MaxAge, ACRValues: request.AcrValues, ParUsed: parUsed,
		AuthorizationDetailsRaw: q.Get("authorization_details"),
		Resource:                q["resource"],
	}
	if requestURI := c.QueryParam("request_uri"); requestURI != "" {
		in.ParRequestURI = requestURI
	}
	out, err := authorizationusecases.Authorize(c.Request().Context(), authorizationusecases.AuthorizeDeps{
		ClientRepo:            d.ClientRepo,
		RequestStore:          d.RequestStore,
		AuthzDetailTypeRepo:   d.AuthzDetailTypeRepo,
		McpResourceServerRepo: d.McpResourceServerRepo,
		Emit:                  d.Emit,
	}, in)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if len(out.Request.AuthorizationDetails) > 0 && d.Emit != nil {
		d.Emit(&oauthdomain.AuthorizationDetailsRequested{
			At: time.Now().UTC(), TenantID: support.RequestTenantID(c), ClientID: out.Request.ClientID,
			DetailTypes: oauthdomain.DetailTypes(out.Request.AuthorizationDetails),
		})
	}

	d.setTransactionCookie(c, out.Request.ID)
	if d.AuthnResolver != nil {
		authn, _ := d.AuthnResolver.Resolve(c.Request().Context(), authdomain.HTTPHeadersAdapter{H: c.Request().Header})
		if authn != nil {
			prompt, _ := oauthdomain.ParsePromptTokens(in.Prompt)
			if authn.AuthenticationPending {
				if prompt.None {
					return writeOAuthError(c, authorizationusecases.NewOAuthError("login_required", "additional factor verification required"))
				}
				return c.Redirect(http.StatusSeeOther, d.SignIn.PendingPath(c, authn))
			}
			policy := oauthdomain.ParsePrompt(out.Request)
			requested := out.Request.RequestedAssurance()
			acrUnmet := out.Request.ACRValues != nil &&
				!authdomain.ACRSatisfies(authn.ACR, *out.Request.ACRValues)
			// 新しさを満たさないセッションは、要素を足しても認証の時刻が古いままなので、ログインから
			// やり直させる。要求した水準はログインの後に満たさせる (REQ-OAUTH2-005)。
			if oauthdomain.NeedsReauthentication(policy, time.Unix(authn.AuthTime, 0), time.Now(), false) ||
				acrUnmet && !requested.MFA {
				if prompt.None {
					return d.redirectAuthorizationError(c, out.Request, "login_required", "The existing session does not satisfy the authentication requirements.")
				}
				return c.Redirect(http.StatusSeeOther, support.TenantRoute(c, "/login"))
			}
			// 新しさを満たし MFA の水準だけが足りないセッションには、登録済みのいずれかの第二要素を
			// 求める。再認証の要求では記憶済みの端末を MFA の充足と認めない (REQ-AUTHENTICATION-027)。
			if requested.MFA && (acrUnmet || requested.Reauthentication && !authn.SecondFactorAuthenticated()) {
				if prompt.None {
					return d.redirectAuthorizationError(c, out.Request, "login_required", "The existing session does not satisfy the authentication requirements.")
				}
				if len(d.SignIn.SecondFactorMethods(c, authn.UserID)) == 0 {
					return d.redirectAuthorizationError(c, out.Request, "unmet_authentication_requirements",
						"The requested authentication level cannot be met because no second factor is enrolled.")
				}
				pending, err := d.SignIn.RequireSecondFactor(c.Request().Context(), c, authn.SessionID)
				if err != nil {
					return err
				}
				if pending == nil {
					return writeOAuthError(c, authorizationusecases.NewOAuthError("login_required", "The existing session does not satisfy the authentication requirements."))
				}
				return c.Redirect(http.StatusSeeOther, d.SignIn.PendingPath(c, authn))
			}
			if out.Client.FirstParty {
				redirected, err := d.SignIn.EnforceTenantDefault(c, authn, !prompt.None)
				if err != nil {
					return err
				}
				if redirected {
					if prompt.None {
						return writeOAuthError(c, authorizationusecases.NewOAuthError("login_required", "The existing session does not satisfy the authentication requirements."))
					}
					return c.Redirect(http.StatusSeeOther, d.SignIn.PendingPath(c, authn))
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
	if out.Request.Prompt != nil {
		prompt, _ := oauthdomain.ParsePromptTokens(*out.Request.Prompt)
		if prompt.None {
			return d.redirectAuthorizationError(c, out.Request, "login_required", "Reauthentication cannot be performed when prompt=none.")
		}
	}
	return c.Redirect(http.StatusSeeOther, support.TenantRoute(c, "/login"))
}

func (d Deps) redirectAuthorizationError(c *echo.Context, req *oauthdomain.AuthorizationRequest, code, description string) error {
	return c.Redirect(http.StatusSeeOther, authorizationErrorURL(req, tenantports.Issuer(c.Request().Context(), d.Issuer), code, description))
}
