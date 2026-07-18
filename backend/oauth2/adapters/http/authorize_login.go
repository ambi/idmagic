package http

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type loginAPIRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ReturnTo string `json:"return_to,omitempty"`
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
		d.recordLoginOutcome("throttled", "", "password")
		return writeLoginThrottled(c, result.RetryAfterSeconds)
	}
	if clientIP != "" {
		if result, err := d.acquireLoginThrottle(c, authnports.LoginThrottleIP, clientIP); err != nil {
			return err
		} else if !result.Allowed {
			d.recordLoginOutcome("throttled", "", "password")
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
		// 監査イベントは集約時に抑制されるが (下記)、golden signal のカウンタは
		// 集約の有無に関わらず確定した失敗ごとに記録する (攻撃バーストで無音化しない)。
		d.recordLoginOutcome("failure", "invalid_credentials", "password")
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
		d.recordLoginOutcome("failure", "account_disabled", "password")
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
	if user == nil {
		return
	}
	d.recordLoginOutcome("success", "", loginMethod(authn))
	if d.Emit == nil {
		return
	}
	d.Emit(&authdomain.UserAuthenticated{
		At: at, TenantID: user.TenantID, UserID: user.ID,
		AMR: authn.AMR, SessionID: authn.SessionID, ClientID: clientID, ACR: authn.ACR,
		IP: extractClientIP(c.Request(), d.TrustedForwardedHops), UserAgent: c.Request().UserAgent(),
	})
}

// loginMethod returns the just-completed authentication factor (the last AMR
// entry) as the bounded "method" label for the login golden signal, or
// "unknown" when AMR is empty.
func loginMethod(authn *authdomain.AuthenticationContext) string {
	if authn == nil || len(authn.AMR) == 0 {
		return "unknown"
	}
	return authn.AMR[len(authn.AMR)-1]
}

// recordLoginOutcome records one confirmed login decision point (independent
// of whether the corresponding audit event was aggregated/suppressed, so the
// golden signal stays accurate under a credential-stuffing burst).
func (d Deps) recordLoginOutcome(outcome, reasonClass, method string) {
	if d.Metrics != nil {
		d.Metrics.RecordLoginOutcome(outcome, reasonClass, method)
	}
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
