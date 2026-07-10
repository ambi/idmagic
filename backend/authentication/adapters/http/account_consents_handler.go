// /api/account/consents — エンドユーザー自身の接続済みアプリ (Consent) の参照と取り消し
// (self-service, wi-21)。
package http

import (
	"net/http"
	"slices"
	"time"

	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type accountConsentResponse struct {
	ClientID   string            `json:"client_id"`
	ClientName string            `json:"client_name"`
	Scopes     []string          `json:"scopes"`
	State      spec.ConsentState `json:"state"`
	GrantedAt  time.Time         `json:"granted_at"`
	ExpiresAt  time.Time         `json:"expires_at"`
}

func toAccountConsentResponse(consent *spec.Consent, clientName string) accountConsentResponse {
	return accountConsentResponse{
		ClientID: consent.ClientID, ClientName: clientName,
		Scopes: slices.Clone(consent.Scopes), State: consent.State,
		GrantedAt: consent.GrantedAt, ExpiresAt: consent.ExpiresAt,
	}
}

func (d Deps) handleListAccountConsents(c *echo.Context) error {
	sub, err := d.requireAuthenticatedSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	ctx := c.Request().Context()
	consents, err := oauthusecases.ListConsentsForSub(ctx, d.ConsentDeps(), sub)
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

func (d Deps) handleRevokeAccountConsent(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAuthenticatedSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	// actor も target も自分の sub に固定する。URL の client_id 以外は信用しない。
	if err := oauthusecases.RevokeConsent(
		c.Request().Context(), d.ConsentDeps(), sub, sub, c.Param("client_id"), time.Now().UTC(),
	); err != nil {
		return d.WriteConsentError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
