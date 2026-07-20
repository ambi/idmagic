package http

import (
	"errors"
	"net/http"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

func (d Deps) applicationMfaEnrollmentPolicy(c *echo.Context, applicationID string) (*appdomain.MfaEnrollmentPolicy, error) {
	if d.ApplicationGate == nil {
		return &appdomain.MfaEnrollmentPolicy{}, nil
	}
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	var appPolicy *appdomain.AppSignInPolicy
	var defaultPolicy *appdomain.TenantDefaultSignInPolicy
	var err error
	if d.ApplicationSignInPolicyRepo != nil {
		appPolicy, err = d.ApplicationSignInPolicyRepo.Get(ctx, tenantID, applicationID)
		if err != nil {
			return nil, err
		}
	}
	if d.DefaultSignInPolicyRepo != nil {
		defaultPolicy, err = d.DefaultSignInPolicyRepo.Get(ctx, tenantID)
		if err != nil {
			return nil, err
		}
	}
	return appusecases.MfaEnrollmentPolicyFromRules(appusecases.EffectiveSignInRules(defaultPolicy, appPolicy)), nil
}

type mfaEnrollmentConfirmRequest struct {
	Secret   string `json:"secret"`
	Code     string `json:"code"`
	ReturnTo string `json:"return_to,omitempty"`
}

func (d Deps) pendingAuthPath(c *echo.Context, authn *authdomain.AuthenticationContext) string {
	if authn != nil && authn.PendingPurpose == authdomain.LoginPendingEnrollment {
		return support.TenantRoute(c, "/mfa-enrollment")
	}
	return support.TenantRoute(c, "/totp")
}

func (d Deps) beginMfaEnrollment(c *echo.Context, authn *authdomain.AuthenticationContext, policy *appdomain.MfaEnrollmentPolicy) (bool, error) {
	if policy == nil || policy.EnforcementStartAt == nil || policy.GracePeriodSeconds == nil || d.MfaEnrollmentBypassRepo == nil {
		return false, nil
	}
	now := time.Now().UTC()
	bypass, err := d.MfaEnrollmentBypassRepo.FindActive(c.Request().Context(), support.RequestTenantID(c), authn.UserID, now)
	if err != nil {
		return false, err
	}
	if bypass == nil {
		expired, expireErr := d.MfaEnrollmentBypassRepo.ExpireOpen(c.Request().Context(), support.RequestTenantID(c), authn.UserID, now)
		if expireErr != nil {
			return false, expireErr
		}
		if expired != nil && d.Emit != nil {
			d.Emit(&authdomain.MfaEnrollmentBypassExpired{At: now, TenantID: tenancy.TenantID(c.Request().Context()), UserID: authn.UserID, BypassID: expired.ID})
		}
	}
	decision, deadline := mfadomain.EvaluateMfaEnrollment(now, policy.EnforcementStartAt, time.Duration(*policy.GracePeriodSeconds)*time.Second, policy.AllowAdminBypass, bypass)
	if decision != mfadomain.MfaEnrollmentRequired || deadline == nil {
		return false, nil
	}
	consumed, err := d.MfaEnrollmentBypassRepo.ConsumeActive(c.Request().Context(), support.RequestTenantID(c), authn.UserID, now)
	if err != nil {
		return false, err
	}
	if consumed == nil {
		return false, nil
	}
	pending, err := d.SessionManager.RequireEnrollment(c.Request().Context(), authn.SessionID, *deadline, consumed.ID)
	if err != nil {
		return false, err
	}
	if pending == nil {
		return false, nil
	}
	*authn = *pending
	d.setSessionCookie(c, pending.SessionID)
	if d.Emit != nil {
		d.Emit(&authdomain.MfaEnrollmentBypassConsumed{At: now, TenantID: tenancy.TenantID(c.Request().Context()), UserID: authn.UserID, BypassID: consumed.ID, SessionID: pending.SessionID})
		d.Emit(&authdomain.MfaEnrollmentRequiredEvent{At: now, TenantID: tenancy.TenantID(c.Request().Context()), UserID: authn.UserID, BypassID: consumed.ID, SessionID: pending.SessionID, Deadline: *deadline})
	}
	return true, nil
}

func (d Deps) requireEnrollmentSession(c *echo.Context) (*authdomain.AuthenticationContext, error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || !authn.AuthenticationPending || authn.PendingPurpose != authdomain.LoginPendingEnrollment || authn.EnrollmentDeadline == nil {
		return nil, authusecases.ErrMfaEnrollmentNotAllowed
	}
	if time.Now().UTC().After(*authn.EnrollmentDeadline) {
		return nil, authusecases.ErrMfaEnrollmentExpired
	}
	return authn, nil
}

func (d Deps) handleStartMfaEnrollmentAPI(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, err := d.requireEnrollmentSession(c)
	if err != nil {
		return writeBrowserEnrollmentError(c, err)
	}
	start, err := authusecases.StartTOTPEnrollment(c.Request().Context(), authusecases.AccountMfaDeps{
		UserRepo: d.UserRepo, MfaFactorRepo: d.MfaFactorRepo, Emit: d.Emit, Issuer: d.Issuer,
	}, authn.UserID)
	if err != nil {
		return writeBrowserEnrollmentError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"secret": start.Secret, "otpauth_uri": start.OTPAuthURI, "account_name": start.AccountName, "issuer": start.Issuer})
}

func (d Deps) handleConfirmMfaEnrollmentAPI(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, err := d.requireEnrollmentSession(c)
	if err != nil {
		return writeBrowserEnrollmentError(c, err)
	}
	var input mfaEnrollmentConfirmRequest
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
	now := time.Now().UTC()
	if err := authusecases.ConfirmTOTPEnrollment(c.Request().Context(), authusecases.AccountMfaDeps{
		UserRepo: d.UserRepo, MfaFactorRepo: d.MfaFactorRepo, Emit: d.Emit, Issuer: d.Issuer,
	}, authusecases.ConfirmTOTPEnrollmentInput{Sub: authn.UserID, Secret: input.Secret, Code: input.Code, Now: now}); err != nil {
		return writeBrowserEnrollmentError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&authdomain.MfaEnrollmentCompleted{At: now, TenantID: tenancy.TenantID(c.Request().Context()), UserID: authn.UserID, SessionID: authn.SessionID, FactorType: "Totp"})
	}
	return d.finishSecondFactor(c, authn.SessionID, req, "otp", directAdminLogin, input.ReturnTo)
}

func writeBrowserEnrollmentError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, authusecases.ErrMfaEnrollmentExpired):
		return support.WriteBrowserError(c, http.StatusForbidden, "mfa_enrollment_expired", "MFA登録期限が切れています。管理者に連絡してください。")
	case errors.Is(err, authusecases.ErrMfaEnrollmentNotAllowed), errors.Is(err, authusecases.ErrMfaAlreadyEnrolled):
		return support.WriteBrowserError(c, http.StatusForbidden, "mfa_enrollment_not_allowed", "MFA登録を開始できません。")
	case errors.Is(err, authusecases.ErrInvalidTOTPCode), errors.Is(err, authusecases.ErrInvalidTOTPSecret):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_totp", "認証コードを確認してください。")
	default:
		return err
	}
}
