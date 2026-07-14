// /authorize + browser authentication APIs + /end_session
package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

const (
	authorizationTransactionCookie = "idmagic_transaction"
)

type browserFlowResponse struct {
	Next       string `json:"next,omitempty"`
	RedirectTo string `json:"redirect_to,omitempty"`
}

type transactionResponse struct {
	Kind                 string              `json:"kind"`
	CSRFToken            string              `json:"csrf_token"`
	ClientName           string              `json:"client_name,omitempty"`
	Scopes               []string            `json:"scopes,omitempty"`
	AuthorizationDetails []consentDetailView `json:"authorization_details,omitempty"`
	// SecondFactorMethods は kind=totp (第二要素待ち) のときに利用できる method 一覧
	// (totp / webauthn / recovery_code)。UI が第二要素選択画面の選択肢に使う (wi-26)。
	SecondFactorMethods []string `json:"second_factor_methods,omitempty"`
}

// consentDetailView は同意画面に提示する authorization_details の人間可読表現 (ADR-050)。
type consentDetailView struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Summary     string   `json:"summary"`
	Lines       []string `json:"lines,omitempty"`
}

type loginAPIRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ReturnTo string `json:"return_to,omitempty"`
}

type consentAPIRequest struct {
	Action string `json:"action"`
}

type totpAPIRequest struct {
	Code     string `json:"code"`
	ReturnTo string `json:"return_to,omitempty"`
}

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
		d.Emit(&spec.AuthorizationDetailsRequested{
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

func (d Deps) handleTransaction(c *echo.Context) error {
	req, err := d.transactionRequest(c)
	if err != nil {
		if returnTo := c.QueryParam("return_to"); returnTo != "" {
			if !validReturnTo(c, returnTo) {
				return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "return_to が不正です")
			}
			csrf, csrfErr := d.EnsureCSRFCookie(c)
			if csrfErr != nil {
				return csrfErr
			}
			authn, _ := d.ResolveAuthentication(c)
			if authn != nil && authn.AuthenticationPending {
				return support.NoStoreJSON(c, http.StatusOK, d.secondFactorTransaction(c, csrf, authn))
			}
			return support.NoStoreJSON(c, http.StatusOK, transactionResponse{Kind: "login", CSRFToken: csrf})
		}
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", err.Error())
	}
	csrf, err := d.EnsureCSRFCookie(c)
	if err != nil {
		return err
	}
	if req.UserID == nil {
		authn, _ := d.ResolveAuthentication(c)
		if authn != nil && authn.AuthenticationPending {
			return support.NoStoreJSON(c, http.StatusOK, d.secondFactorTransaction(c, csrf, authn))
		}
		return support.NoStoreJSON(c, http.StatusOK, transactionResponse{Kind: "login", CSRFToken: csrf})
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn != nil && authn.AuthenticationPending {
		return support.NoStoreJSON(c, http.StatusOK, d.secondFactorTransaction(c, csrf, authn))
	}
	if authn == nil || authn.UserID != *req.UserID {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証セッションが一致しません")
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), req.ClientID)
	if err != nil {
		return err
	}
	if client == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_transaction", "クライアントが存在しません")
	}
	// 表示名は client_name → Application カタログ名 → client_id の順で解決する (wi-141)。
	// ADR-084 で client_id を UUID 化したため、同意画面での UUID 生表示を避ける。
	name := d.ClientDisplayNameResolver.Resolve(
		c.Request().Context(), support.RequestTenantID(c), req.ClientID,
	)
	return support.NoStoreJSON(c, http.StatusOK, transactionResponse{
		Kind: "consent", CSRFToken: csrf, ClientName: name, Scopes: strings.Fields(req.Scope),
		AuthorizationDetails: d.renderConsentDetails(c, req.AuthorizationDetails),
	})
}

func (d Deps) handleLoginAPI(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	var input loginAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if strings.TrimSpace(input.Username) == "" || input.Password == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "ユーザー名とパスワードが必要です")
	}
	req, transactionErr := d.transactionRequest(c)
	directAdminLogin := transactionErr != nil && input.ReturnTo != ""
	if directAdminLogin {
		if !validReturnTo(c, input.ReturnTo) {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "return_to が不正です")
		}
	} else if transactionErr != nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", transactionErr.Error())
	}

	normalizedUsername := strings.ToLower(input.Username)
	clientIP := extractClientIP(c.Request(), d.TrustedForwardedHops)
	if result, err := d.acquireLoginThrottle(c, authnports.LoginThrottleAccount, normalizedUsername); err != nil {
		return err
	} else if !result.Allowed {
		return writeLoginThrottled(c, result.RetryAfterSeconds)
	}
	if clientIP != "" {
		if result, err := d.acquireLoginThrottle(c, authnports.LoginThrottleIP, clientIP); err != nil {
			return err
		} else if !result.Allowed {
			return writeLoginThrottled(c, result.RetryAfterSeconds)
		}
	}

	user, err := d.UserRepo.FindByUsername(c.Request().Context(), support.RequestTenantID(c), input.Username)
	if err != nil {
		return err
	}
	hashToVerify := d.SentinelPasswordHash
	if user != nil {
		hashToVerify = user.PasswordHash
	}
	ok := false
	if hashToVerify != "" {
		ok, err = d.PasswordHasher.Verify(input.Password, hashToVerify)
	}
	if user == nil || err != nil || !ok {
		aggregated, ferr := d.recordLoginFailure(c, normalizedUsername, clientIP)
		if ferr != nil {
			return ferr
		}
		// 閾値超過後は AuthenticationEventAggregated に集約し、個別行を出さない (爆発抑制)。
		if !aggregated {
			d.emitAuthenticationFailure(c, input.Username, "invalid_credentials")
		}
		return support.WriteBrowserError(c, http.StatusUnauthorized, "invalid_credentials", "ユーザー名またはパスワードを確認してください。")
	}
	if !user.IsActive() {
		d.emitAuthenticationFailure(c, input.Username, "account_disabled")
		return support.WriteBrowserError(c, http.StatusUnauthorized, "invalid_credentials", "ユーザー名またはパスワードを確認してください。")
	}
	if result := authusecases.ValidatePassword(input.Password); !result.OK {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "password_policy", "パスワードがセキュリティ要件を満たしていません。")
	}
	if d.LoginAttemptThrottle != nil {
		if err := d.LoginAttemptThrottle.RecordSuccess(
			c.Request().Context(), authnports.LoginThrottleAccount, normalizedUsername,
		); err != nil {
			return err
		}
	}

	authTime := time.Now().UTC()
	authn, err := d.SessionManager.Create(
		c.Request().Context(),
		user.ID,
		[]string{"pwd"},
		authTime,
	)
	if err != nil {
		return err
	}
	d.setSessionCookie(c, authn.SessionID)
	if directAdminLogin {
		if redirected, err := d.enforceDefaultSignInPolicy(c, authn, true); err != nil {
			return err
		} else if redirected {
			return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{
				Next: d.pendingAuthPath(c, authn) + "?return_to=" + url.QueryEscape(input.ReturnTo),
			})
		}
		d.emitAuthenticationSuccess(c, authTime, user, authn, "")
		gateNext, err := d.recordLoginAndRequiredAction(c, user, authTime)
		if err != nil {
			return err
		}
		if gateNext != "" {
			return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: gateNext})
		}
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: input.ReturnTo})
	}
	req.UserID, req.AuthTime, req.AMR, req.ACR = &user.ID, &authn.AuthTime, authn.AMR, &authn.ACR
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), req.ClientID)
	if err != nil || client == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_transaction", "クライアントが存在しません")
	}
	if !client.FirstParty {
		decision, err := d.EvaluateApplicationAccess(
			c.Request().Context(), support.RequestTenantID(c), appdomain.ProtocolBindingOIDC, req.ClientID,
			authn.UserID, authn, d.ClientIP(c.Request()),
		)
		if err != nil {
			return err
		}
		if decision.StepUpRequired {
			if len(d.secondFactorMethods(c, authn.UserID)) == 0 {
				enrollment, policyErr := d.applicationMfaEnrollmentPolicy(c, decision.ApplicationID)
				if policyErr != nil {
					return policyErr
				}
				if begun, err := d.beginMfaEnrollment(c, authn, enrollment); err != nil {
					return err
				} else if begun {
					return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: support.TenantRoute(c, "/mfa-enrollment")})
				}
				return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{
					RedirectTo: authorizationErrorURL(req, support.RequestIssuer(c, d.Issuer), "access_denied", "アプリケーションのサインインポリシーを満たせません"),
				})
			}
			pending, err := d.SessionManager.RequireFactor(c.Request().Context(), authn.SessionID)
			if err != nil {
				return err
			}
			if pending == nil {
				return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "セッションが失効しました")
			}
			d.setSessionCookie(c, pending.SessionID)
			return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: d.pendingAuthPath(c, authn)})
		}
		if !decision.Allowed {
			return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{
				RedirectTo: authorizationErrorURL(req, support.RequestIssuer(c, d.Issuer), "access_denied", "この利用者はアプリケーションにアクセスできません"),
			})
		}
	}
	if client.FirstParty {
		if redirected, err := d.enforceDefaultSignInPolicy(c, authn, true); err != nil {
			return err
		} else if redirected {
			return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: d.pendingAuthPath(c, authn)})
		}
	}
	d.emitAuthenticationSuccess(c, authTime, user, authn, req.ClientID)
	// full authentication 完了。last_login_at 記録 + required action gate。
	gateNext, err := d.recordLoginAndRequiredAction(c, user, authTime)
	if err != nil {
		return err
	}
	if gateNext != "" {
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: gateNext})
	}
	next, err := d.completeAfterAuthn(c, req, client, authn)
	if err != nil {
		return err
	}
	if next.RedirectTo != "" {
		d.clearTransactionCookie(c)
	}
	return writeAuthorizationNext(c, next)
}

func (d Deps) enforceDefaultSignInPolicy(
	c *echo.Context,
	authn *authdomain.AuthenticationContext,
	allowChallenge bool,
) (bool, error) {
	if d.DefaultSignInPolicyRepo == nil {
		return false, nil
	}
	policy, err := d.DefaultSignInPolicyRepo.Get(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return false, err
	}
	if policy == nil || len(policy.Rules) == 0 {
		return false, nil
	}
	evaluation := appusecases.EvaluateSignInPolicy(
		&appdomain.AppSignInPolicy{TenantID: policy.TenantID, Rules: policy.Rules},
		authn,
		d.ClientIP(c.Request()),
		time.Now().UTC(),
	)
	switch evaluation.Decision {
	case appusecases.PolicyAllow:
		return false, nil
	case appusecases.PolicyStepUpRequired:
		if !allowChallenge {
			return true, nil
		}
		if len(d.secondFactorMethods(c, authn.UserID)) == 0 {
			enrollment := appusecases.MfaEnrollmentPolicyFromRules(policy.Rules)
			if begun, err := d.beginMfaEnrollment(c, authn, enrollment); err != nil {
				return false, err
			} else if begun {
				return true, nil
			}
			return false, support.WriteBrowserError(c, http.StatusForbidden, "access_denied", "MFA必須ですが、利用できる第二要素がありません")
		}
		pending, err := d.SessionManager.RequireFactor(c.Request().Context(), authn.SessionID)
		if err != nil {
			return false, err
		}
		if pending == nil {
			return false, support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "セッションが失効しました")
		}
		d.setSessionCookie(c, pending.SessionID)
		return true, nil
	default:
		return false, support.WriteBrowserError(c, http.StatusForbidden, "access_denied", "サインインポリシーを満たせません")
	}
}

func (d Deps) emitAuthenticationSuccess(
	c *echo.Context,
	at time.Time,
	user *idmdomain.User,
	authn *authdomain.AuthenticationContext,
	clientID string,
) {
	if d.Emit == nil || user == nil {
		return
	}
	d.Emit(&spec.UserAuthenticated{
		At: at, TenantID: user.TenantID, UserID: user.ID,
		AMR: authn.AMR, SessionID: authn.SessionID, ClientID: clientID, ACR: authn.ACR,
		IP: extractClientIP(c.Request(), d.TrustedForwardedHops), UserAgent: c.Request().UserAgent(),
	})
}

// recordLoginAndRequiredAction は full authentication 完了時に last_login_at を
// 記録し (wi-19)、未充足の required action があればログイン後の強制誘導先を返す。
// 返り値 gateNext が非空なら OAuth フローを完了させず、その画面へ誘導する。現状
// 専用画面のある update_password のみ change-password へ gate する (UI 拡張は wi-21)。
func (d Deps) recordLoginAndRequiredAction(c *echo.Context, user *idmdomain.User, now time.Time) (string, error) {
	updated := *user
	updated.Lifecycle.LastLoginAt = &now
	if err := d.UserRepo.Save(c.Request().Context(), &updated); err != nil {
		return "", err
	}
	*user = updated
	if slices.Contains(updated.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
		return support.TenantRoute(c, "/change_password"), nil
	}
	return "", nil
}

func (d Deps) handleTOTPAPI(c *echo.Context) error {
	if d.MfaFactorRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "mfa_unavailable", "MFA factor store is unavailable")
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn == nil || authn.SessionID == "" {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "TOTP検証セッションがありません")
	}
	if containsString(authn.AMR, "otp") && !authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusForbidden, "access_denied", "TOTPは既に検証済みです")
	}
	var input totpAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	req, transactionErr := d.transactionRequest(c)
	directAdminLogin := transactionErr != nil && input.ReturnTo != ""
	if directAdminLogin {
		if !validReturnTo(c, input.ReturnTo) {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "return_to が不正です")
		}
	} else if transactionErr != nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", transactionErr.Error())
	}
	result, err := authusecases.VerifyTOTPFactor(
		c.Request().Context(),
		d.MfaFactorRepo,
		authn.UserID,
		input.Code,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	if !result.OK {
		d.emitAuthenticationFailure(c, authn.UserID, result.Reason)
		return support.WriteBrowserError(c, http.StatusUnauthorized, "invalid_totp", "TOTPコードを確認してください。")
	}
	return d.finishSecondFactor(c, authn.SessionID, req, "otp", directAdminLogin, input.ReturnTo)
}

func (d Deps) handleConsentAPI(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	req, err := d.transactionRequest(c)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", err.Error())
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn == nil || req.UserID == nil || authn.UserID != *req.UserID {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証セッションが一致しません")
	}
	var input consentAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	if input.Action != "allow" {
		_ = d.RequestStore.UpdateState(ctx, req.ID, spec.AuthFlowRejected)
		d.clearTransactionCookie(c)
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: authorizationErrorURL(req, support.RequestIssuer(c, d.Issuer), "access_denied", "")})
	}

	scopes := strings.Fields(req.Scope)
	if d.ConsentRepo != nil {
		now := time.Now().UTC()
		if err := d.ConsentRepo.Save(ctx, support.RequestTenantID(c), &oauthdomain.Consent{
			UserID: authn.UserID, ClientID: req.ClientID,
			Scopes: scopes, State: oauthdomain.ConsentGranted,
			GrantedAt: now, ExpiresAt: now.Add(365 * 24 * time.Hour),
			AuthorizationDetails: req.AuthorizationDetails,
		}); err != nil {
			return err
		}
		if d.Emit != nil {
			d.Emit(&oauthdomain.ConsentGrantedEvent{At: now, TenantID: support.RequestTenantID(c), UserID: authn.UserID, ClientID: req.ClientID, Scopes: scopes})
			if len(req.AuthorizationDetails) > 0 {
				d.Emit(&spec.AuthorizationDetailsConsented{
					At: now, TenantID: support.RequestTenantID(c), UserID: authn.UserID, ClientID: req.ClientID,
					DetailTypes: oauthdomain.DetailTypes(req.AuthorizationDetails),
				})
			}
		}
	}
	redirectTo, err := d.issueCodeURL(ctx, c, req, authn, time.Unix(authn.AuthTime, 0))
	if err != nil {
		return err
	}
	d.clearTransactionCookie(c)
	return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: redirectTo})
}

type authorizationNext struct {
	Path       string
	RedirectTo string
}

func (d Deps) completeAfterAuthn(
	c *echo.Context,
	req *oauthdomain.AuthorizationRequest,
	client *oauthdomain.OAuth2Client,
	authn *authdomain.AuthenticationContext,
) (authorizationNext, error) {
	if authn.AuthenticationPending {
		return authorizationNext{Path: d.pendingAuthPath(c, authn)}, nil
	}
	// first-party クライアント (IdP 自身の管理コンソール / アカウントポータル) は
	// resource owner が IdP 利用者自身であるため consent をスキップする (ADR-061)。
	if d.ConsentRepo != nil && !client.FirstParty {
		consent, _ := d.ConsentRepo.Find(
			c.Request().Context(), support.RequestTenantID(c), authn.UserID, client.ClientID,
		)
		covered := consent != nil &&
			consent.State == oauthdomain.ConsentGranted &&
			consent.RevokedAt == nil &&
			time.Now().Before(consent.ExpiresAt)
		if covered {
			for scope := range strings.FieldsSeq(req.Scope) {
				if !containsString(consent.Scopes, scope) {
					covered = false
					break
				}
			}
		}
		if req.Prompt != nil && *req.Prompt == "consent" {
			covered = false
		}
		// RFC 9396 — 構造化された authorization_details は粗い scope 同意では代替できない。
		// 明示同意を要求し、過去 scope 同意での自動スキップを許さない (fail-closed, ADR-050)。
		if len(req.AuthorizationDetails) > 0 {
			covered = false
		}
		if !covered {
			ctx, cancel := d.OperationContext(c.Request().Context())
			defer cancel()
			if err := d.RequestStore.AttachAuthentication(
				ctx, req.ID, authn.UserID, authn.AuthTime, authn.AMR, authn.ACR,
			); err != nil {
				return authorizationNext{}, err
			}
			req.UserID, req.AuthTime = &authn.UserID, &authn.AuthTime
			return authorizationNext{Path: support.TenantRoute(c, "/consent")}, nil
		}
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	redirectTo, err := d.issueCodeURL(ctx, c, req, authn, time.Unix(authn.AuthTime, 0))
	return authorizationNext{RedirectTo: redirectTo}, err
}

func (d Deps) canUseTOTP(c *echo.Context, sub string) bool {
	if d.MfaFactorRepo == nil {
		return false
	}
	factor, err := d.MfaFactorRepo.Find(c.Request().Context(), sub, spec.MfaFactorTOTP)
	return err == nil && factor != nil && factor.Secret != nil && *factor.Secret != ""
}

// clientIsFirstParty は client_id が first-party クライアントかを返す。解決不能なら false
// (fail-closed で割当ゲートを適用する)。
func (d Deps) clientIsFirstParty(ctx context.Context, clientID string) bool {
	if d.ClientRepo == nil {
		return false
	}
	client, err := d.ClientRepo.FindByID(ctx, tenancy.TenantID(ctx), clientID)
	return err == nil && client != nil && client.FirstParty
}

func (d Deps) issueCodeURL(
	ctx context.Context,
	c *echo.Context,
	req *oauthdomain.AuthorizationRequest,
	authn *authdomain.AuthenticationContext,
	authTime time.Time,
) (string, error) {
	iss := tenancy.Issuer(ctx, d.Issuer)
	tenantID := tenancy.TenantID(ctx)
	// 割当ゲート (wi-69): client が Application binding に属する場合、未割当 subject には
	// 認可コードを発行せず access_denied で RP へ返す (fail-closed, AssignmentGatesProtocol)。
	// ただし first-party クライアント (IdP 自身の管理コンソール / アカウントポータル) は
	// resource owner が IdP 利用者自身であり、アプリ割当でログインをゲートしない (ADR-061)。
	if !d.clientIsFirstParty(ctx, req.ClientID) {
		decision, err := d.EvaluateApplicationAccess(
			ctx, tenantID, appdomain.ProtocolBindingOIDC, req.ClientID, authn.UserID, authn, d.ClientIP(c.Request()),
		)
		if err != nil {
			return "", err
		}
		if decision.StepUpRequired {
			if d.Emit != nil {
				d.Emit(&appdomain.AppStepUpRequired{
					At: time.Now().UTC(), TenantID: tenantID, ApplicationID: decision.ApplicationID,
					Protocol: string(appdomain.ProtocolBindingOIDC), Subject: authn.UserID,
				})
			}
			if len(d.secondFactorMethods(c, authn.UserID)) > 0 { //nolint:contextcheck // HTTP request context is required for factor lookup.
				pending, err := d.SessionManager.RequireFactor(ctx, authn.SessionID)
				if err != nil {
					return "", err
				}
				if pending == nil {
					return authorizationErrorURL(req, iss, "login_required", "既存セッションが認証要件を満たしません"), nil
				}
				d.setSessionCookie(c, pending.SessionID)    //nolint:contextcheck // Cookie path is derived from the Echo request.
				return support.TenantRoute(c, "/totp"), nil //nolint:contextcheck // Redirect URL is derived from the Echo request.
			}
			return authorizationErrorURL(req, iss, "access_denied", "アプリケーションのサインインポリシーを満たせません"), nil
		}
		if !decision.Allowed {
			reason := decision.Reason
			if reason == "" {
				reason = "subject not assigned to application"
			}
			if d.Emit != nil && decision.ApplicationID != "" {
				d.Emit(&appdomain.AppAccessDeniedByPolicy{
					At: time.Now().UTC(), TenantID: tenantID, ApplicationID: decision.ApplicationID,
					Protocol: string(appdomain.ProtocolBindingOIDC), Subject: authn.UserID, Reason: reason,
				})
			}
			return authorizationErrorURL(req, iss, "access_denied", "この利用者はアプリケーションにアクセスできません"), nil
		}
	}
	out, err := usecases.CompleteLogin(ctx, usecases.CompleteLoginDeps{
		RequestStore: d.RequestStore,
		CodeStore:    d.CodeStore,
	}, usecases.CompleteLoginInput{
		RequestID: req.ID,
		Sub:       authn.UserID,
		AuthTime:  authTime,
		AMR:       authn.AMR,
		ACR:       authn.ACR,
	})
	if err != nil {
		var oauthErr *usecases.OAuthError
		if errors.As(err, &oauthErr) {
			return authorizationErrorURL(req, iss, oauthErr.Code, oauthErr.Description), nil
		}
		return "", err
	}
	if d.Emit != nil {
		d.Emit(&spec.AuthorizationCodeIssued{
			At: time.Now().UTC(), TenantID: tenantID, ClientID: req.ClientID, UserID: authn.UserID,
			Scopes: out.Code.Scopes, CodeChallengeMethod: req.CodeChallengeMethod,
		})
	}
	u, _ := url.Parse(out.Request.RedirectURI)
	query := u.Query()
	query.Set("code", out.Code.Code)
	if out.Request.StateParam != nil {
		query.Set("state", *out.Request.StateParam)
	}
	// RFC 9207 §2: Authorization Server Issuer Identification (mix-up 攻撃対策)。
	query.Set("iss", iss)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func authorizationErrorURL(req *oauthdomain.AuthorizationRequest, iss, code, description string) string {
	u, _ := url.Parse(req.RedirectURI)
	query := u.Query()
	query.Set("error", code)
	if description != "" {
		query.Set("error_description", description)
	}
	if req.StateParam != nil {
		query.Set("state", *req.StateParam)
	}
	// RFC 9207 §2: error response も含めて iss を必須にする。
	if iss != "" {
		query.Set("iss", iss)
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func redirectAuthorizationNext(c *echo.Context, next authorizationNext) error {
	if next.RedirectTo != "" {
		return c.Redirect(http.StatusFound, next.RedirectTo)
	}
	return c.Redirect(http.StatusSeeOther, next.Path)
}

func writeAuthorizationNext(c *echo.Context, next authorizationNext) error {
	if next.RedirectTo != "" {
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: next.RedirectTo})
	}
	return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: next.Path})
}

func (d Deps) handleEndSession(c *echo.Context) error {
	if d.SessionManager != nil {
		_ = d.SessionManager.Revoke(c.Request().Context(), c.Request().Header.Get("Cookie"))
		d.clearSessionCookie(c)
	}
	post := c.QueryParam("post_logout_redirect_uri")
	if post == "" {
		post = c.Request().PostFormValue("post_logout_redirect_uri")
	}
	if post == "" {
		return c.Redirect(http.StatusSeeOther, "/status?state=signed-out")
	}
	clientID := c.QueryParam("client_id")
	if clientID == "" {
		clientID = c.Request().PostFormValue("client_id")
	}
	if clientID == "" {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "client_id が必要"))
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), clientID)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if client == nil {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "post_logout_redirect_uri が未登録"))
	}
	// Redirect only to a URI from the client's registered allowlist. Selecting the
	// stored value (rather than reusing the request parameter) keeps the redirect
	// target server-controlled and avoids open-redirect via user input.
	registered := ""
	for _, uri := range client.RedirectURIs {
		if uri == post {
			registered = uri
			break
		}
	}
	if registered == "" {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "post_logout_redirect_uri が未登録"))
	}
	u, err := url.Parse(registered)
	if err != nil {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "post_logout_redirect_uri が不正"))
	}
	query := u.Query()
	if state := c.QueryParam("state"); state != "" {
		query.Set("state", state)
	}
	u.RawQuery = query.Encode()
	return c.Redirect(http.StatusFound, u.String())
}

func (d Deps) emitAuthenticationFailure(c *echo.Context, username, reason string) {
	if d.Emit != nil {
		d.Emit(&spec.AuthenticationFailed{
			At: time.Now().UTC(), TenantID: support.RequestTenantID(c), Username: username, Reason: reason,
			IP: extractClientIP(c.Request(), d.TrustedForwardedHops), UserAgent: c.Request().UserAgent(),
		})
	}
}

func (d Deps) acquireLoginThrottle(
	c *echo.Context,
	kind authnports.LoginThrottleKind,
	key string,
) (authnports.LoginThrottleResult, error) {
	if d.LoginAttemptThrottle == nil {
		return authnports.LoginThrottleResult{Allowed: true}, nil
	}
	return d.LoginAttemptThrottle.TryAcquire(c.Request().Context(), kind, key, time.Now().UTC())
}

// recordLoginFailure は失敗を throttle に記録し、閾値超過 (Locked) の key については
// LoginThrottled を emit したうえで失敗を集約 bucket に積む。集約に切り替わった場合は
// aggregated=true を返し、呼び出し側は個別の AuthenticationFailed を抑制する
// (これが攻撃時の行爆発を止める要点 / wi-20 スライス 3)。
func (d Deps) recordLoginFailure(c *echo.Context, username, clientIP string) (bool, error) {
	if d.LoginAttemptThrottle == nil {
		return false, nil
	}
	now := time.Now().UTC()
	aggregated := false
	for _, attempt := range []struct {
		kind authnports.LoginThrottleKind
		key  string
	}{
		{authnports.LoginThrottleAccount, username},
		{authnports.LoginThrottleIP, clientIP},
	} {
		if attempt.key == "" {
			continue
		}
		result, err := d.LoginAttemptThrottle.RecordFailure(
			c.Request().Context(), attempt.kind, attempt.key, now,
		)
		if err != nil {
			return aggregated, err
		}
		if !result.Locked {
			continue
		}
		keyHash := d.correlationHash(c, attempt.key)
		if d.Emit != nil {
			d.Emit(&spec.LoginThrottled{
				At: now, TenantID: support.RequestTenantID(c), Kind: string(attempt.kind),
				KeyHash:           keyHash,
				RetryAfterSeconds: result.RetryAfterSeconds,
			})
		}
		if d.recordFailedLoginBucket(c, keyHash, now) {
			aggregated = true
		}
	}
	return aggregated, nil
}

// correlationHash は throttle / bucket の emit keyHash を tenant salt 付きで計算する
// (wi-145 / ADR-046)。username / IP の相関検索属性と同じ単一ヘルパ (spec.SaltedHash) を共有し、
// tenant salt により cross-tenant で相関を集約しない。salt store が無い構成 (一部テスト) では
// unsalted SHA-256 にフォールバックする。
func (d Deps) correlationHash(c *echo.Context, value string) string {
	if d.TenantSaltStore != nil {
		if salt, err := d.TenantSaltStore.GetSalt(c.Request().Context()); err == nil {
			return spec.SaltedHash(salt, value)
		}
	}
	return hashThrottleKey(value)
}

func hashThrottleKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// recordFailedLoginBucket は閾値超過後の失敗を 5 分窓の bucket に積み、その窓で最初の
// 記録だったときだけ AuthenticationEventAggregated を 1 件 emit する。bucket store が
// 無い構成では集約せず false を返し、呼び出し側は従来どおり個別イベントを残す。
func (d Deps) recordFailedLoginBucket(c *echo.Context, keyHash string, now time.Time) bool {
	if d.AuthEventBucketStore == nil {
		return false
	}
	result, err := d.AuthEventBucketStore.Record(
		c.Request().Context(), authnports.AuthEventBucketFailedLogin, support.RequestTenantID(c), keyHash, now,
	)
	if err != nil {
		return false
	}
	if result.FirstInWindow && d.Emit != nil {
		bucket := result.Bucket
		d.Emit(&spec.AuthenticationEventAggregated{
			At: now, TenantID: bucket.TenantID, Kind: string(bucket.Kind),
			BucketKey: failedLoginBucketKey(bucket),
			KeyHash:   bucket.KeyHash, Count: bucket.Count,
			FirstSeen: bucket.FirstSeen, LastSeen: bucket.LastSeen,
			TopKeys: []string{bucket.KeyHash},
		})
	}
	return true
}

func failedLoginBucketKey(bucket authnports.AuthEventBucket) string {
	return string(bucket.Kind) + ":" + bucket.KeyHash + ":" +
		strconv.FormatInt(bucket.WindowStart.Unix(), 10)
}

func extractClientIP(request *http.Request, trustedHops int) string {
	if request == nil || trustedHops <= 0 {
		return ""
	}
	parts := strings.Split(request.Header.Get("X-Forwarded-For"), ",")
	ips := make([]string, 0, len(parts))
	for _, part := range parts {
		if ip := strings.TrimSpace(part); ip != "" {
			ips = append(ips, ip)
		}
	}
	index := len(ips) - 1 - trustedHops
	if index < 0 || index >= len(ips) {
		return ""
	}
	return ips[index]
}

func writeLoginThrottled(c *echo.Context, retryAfterSeconds int) error {
	c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	return support.NoStoreJSON(c, http.StatusTooManyRequests, map[string]any{
		"error": "too_many_requests", "retry_after_seconds": retryAfterSeconds,
	})
}
