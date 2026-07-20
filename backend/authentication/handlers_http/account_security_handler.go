// /api/account/security — エンドユーザー自身のセキュリティ概要 (password/totp/webauthn/
// recovery を横断した集計、wi-21 / ADR-042)。TOTP self-service 登録・解除は mfa feature の
// handlers_http (ADR-130 Phase 2) へ分割されている。
package handlers_http

import (
	"net/http"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	recoveryusecases "github.com/ambi/idmagic/backend/authentication/recovery/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type accountMfaFactorResponse struct {
	Type       spec.MfaFactorType `json:"type"`
	Label      *string            `json:"label,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	LastUsedAt *time.Time         `json:"last_used_at,omitempty"`
}

type webAuthnCredentialSummaryResponse struct {
	CredentialID string     `json:"credential_id"`
	Label        *string    `json:"label,omitempty"`
	Transports   []string   `json:"transports"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
}

type recoveryCodeStatusResponse struct {
	GeneratedAt *time.Time `json:"generated_at,omitempty"`
	Total       int        `json:"total"`
	Remaining   int        `json:"remaining"`
}

type accountSecurityResponse struct {
	PasswordChangedAt   *time.Time                          `json:"password_changed_at,omitempty"`
	TotpEnrolled        bool                                `json:"totp_enrolled"`
	Factors             []accountMfaFactorResponse          `json:"factors"`
	WebAuthnCredentials []webAuthnCredentialSummaryResponse `json:"webauthn_credentials"`
	RecoveryCodes       recoveryCodeStatusResponse          `json:"recovery_codes"`
}

func handleGetAccountSecurity(d Deps, c *echo.Context) error {
	sub, err := httpdeps.RequireAuthenticatedSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	user, _, err := userusecases.GetUserProfile(c.Request().Context(), httpdeps.AccountProfileDeps(d), sub)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	factors, err := d.MfaFactorRepo.ListBySub(c.Request().Context(), sub)
	if err != nil {
		return err
	}
	responses := make([]accountMfaFactorResponse, 0, len(factors))
	totpEnrolled := false
	for _, factor := range factors {
		if factor.Type == spec.MfaFactorTOTP {
			totpEnrolled = true
		}
		responses = append(responses, accountMfaFactorResponse{
			Type: factor.Type, Label: factor.Label,
			CreatedAt: factor.CreatedAt, LastUsedAt: factor.LastUsedAt,
		})
	}
	credentials := []webAuthnCredentialSummaryResponse{}
	if d.WebAuthnCredentialRepo != nil {
		stored, err := d.WebAuthnCredentialRepo.ListBySub(c.Request().Context(), sub)
		if err != nil {
			return err
		}
		for _, cred := range stored {
			transports := cred.Transports
			if transports == nil {
				transports = []string{}
			}
			credentials = append(credentials, webAuthnCredentialSummaryResponse{
				CredentialID: cred.CredentialID, Label: cred.Label, Transports: transports,
				CreatedAt: cred.CreatedAt, LastUsedAt: cred.LastUsedAt,
			})
		}
	}
	recovery := recoveryCodeStatusResponse{}
	if d.RecoveryCodeRepo != nil {
		status, err := recoveryusecases.RecoveryCodeStatusFor(c.Request().Context(), d.RecoveryCodeRepo, sub)
		if err != nil {
			return err
		}
		recovery = recoveryCodeStatusResponse{
			GeneratedAt: status.GeneratedAt, Total: status.Total, Remaining: status.Remaining,
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, accountSecurityResponse{
		PasswordChangedAt:   user.Lifecycle.PasswordChangedAt,
		TotpEnrolled:        totpEnrolled,
		Factors:             responses,
		WebAuthnCredentials: credentials,
		RecoveryCodes:       recovery,
	})
}
