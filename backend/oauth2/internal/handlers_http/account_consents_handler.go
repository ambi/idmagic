// /api/account/v1/consents — エンドユーザー自身の接続済みアプリ (Consent) の参照と取り消し
// (self-service, wi-21)。
package handlers_http

import (
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	consentusecases "github.com/ambi/idmagic/backend/oauth2/internal/consent/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type accountConsentResponse struct {
	ClientID   string              `json:"client_id"`
	ClientName string              `json:"client_name"`
	Scopes     []string            `json:"scopes"`
	State      domain.ConsentState `json:"state"`
	GrantedAt  time.Time           `json:"granted_at"`
	ExpiresAt  time.Time           `json:"expires_at"`
}

func toAccountConsentResponse(consent *domain.Consent, clientName string) accountConsentResponse {
	return accountConsentResponse{
		ClientID: consent.ClientID, ClientName: clientName,
		Scopes: slices.Clone(consent.Scopes), State: consent.State,
		GrantedAt: consent.GrantedAt, ExpiresAt: consent.ExpiresAt,
	}
}

// accountConsents は本人の操作に使う同意の usecase である。
func (d Deps) accountConsents() consentusecases.AccountConsents {
	return consentusecases.AccountConsents{ConsentRepo: d.ConsentRepo, Emit: d.Emit, QuotaRepo: d.QuotaRepo}
}

func (d Deps) handleListMyConsents(c *echo.Context) error {
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	ctx := c.Request().Context()
	consents, err := d.accountConsents().ListGranted(ctx, sub)
	if err != nil {
		return err
	}
	clientIDs := make([]string, len(consents))
	for i, consent := range consents {
		clientIDs[i] = consent.ClientID
	}
	names := d.ClientDisplayNameResolver.ResolveAll(ctx, support.RequestTenantID(c), clientIDs)
	response := make([]accountConsentResponse, len(consents))
	for i, consent := range consents {
		response[i] = toAccountConsentResponse(consent, names[consent.ClientID])
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"consents": response})
}

func (d Deps) handleRevokeMyConsent(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	// 操作者も対象も自分の sub に固定する。URL の client_id 以外は信用しない。
	if err := d.accountConsents().Revoke(c.Request().Context(), sub, c.Param("client_id"), time.Now().UTC()); err != nil {
		return writeConsentError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

// requireAccountSub は認証済みの本人の sub を返す。認証が完了していなければ
// support.ErrAdminAuthenticationRequired を返す。
func (d Deps) requireAccountSub(c *echo.Context) (string, error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return "", err
	}
	if authn == nil || authn.AuthenticationPending {
		return "", support.ErrAdminAuthenticationRequired
	}
	return authn.UserID, nil
}

// writeAccountSubError は本人の解決の失敗を、アクセストークンの拒否か未認証の応答へ変換する。
func writeAccountSubError(c *echo.Context, err error) error {
	if handled, result := support.WriteAccessTokenError(c, err); handled {
		return result
	}
	if errors.Is(err, support.ErrAdminAuthenticationRequired) {
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required.")
	}
	return err
}
