package http

import (
	"context"
	"net/http"
	"slices"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	oauthusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type adminConsentResponse struct {
	UserID            string                   `json:"user_id"`
	PreferredUsername string                   `json:"preferred_username,omitempty"`
	ClientID          string                   `json:"client_id"`
	ClientName        string                   `json:"client_name"`
	Scopes            []string                 `json:"scopes"`
	State             oauthdomain.ConsentState `json:"state"`
	GrantedAt         time.Time                `json:"granted_at"`
	ExpiresAt         time.Time                `json:"expires_at"`
	RevokedAt         *time.Time               `json:"revoked_at,omitempty"`
}

func (d Deps) handleListAdminConsents(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx := c.Request().Context()
	consents, err := oauthusecases.ListConsents(ctx, d.ConsentDeps())
	if err != nil {
		return err
	}
	tenantID := support.RequestTenantID(c)
	clientIDs := make([]string, len(consents))
	for i, consent := range consents {
		clientIDs[i] = consent.ClientID
	}
	clientNames := d.ClientDisplayNameResolver.ResolveAll(ctx, tenantID, clientIDs)
	usernames := map[string]string{}
	response := make([]adminConsentResponse, len(consents))
	for i, consent := range consents {
		response[i] = toAdminConsentResponse(
			consent, clientNames[consent.ClientID], d.resolveUsername(ctx, usernames, consent.UserID),
		)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"consents": response})
}

func (d Deps) handleGetAdminConsent(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx := c.Request().Context()
	consent, err := oauthusecases.GetConsent(
		ctx, d.ConsentDeps(), c.Param("sub"), c.Param("client_id"),
	)
	if err != nil {
		return d.WriteConsentError(c, err)
	}
	clientName := d.ClientDisplayNameResolver.Resolve(ctx, support.RequestTenantID(c), consent.ClientID)
	return support.NoStoreJSON(
		c, http.StatusOK, toAdminConsentResponse(consent, clientName, d.resolveUsername(ctx, map[string]string{}, consent.UserID)),
	)
}

// resolveUsername は user_id を preferred_username へ解決し、結果を cache に丸める (wi-141)。
// User が見つからない / エラー時は空文字を返し、応答では省略される。
func (d Deps) resolveUsername(ctx context.Context, cache map[string]string, userID string) string {
	if name, ok := cache[userID]; ok {
		return name
	}
	name := ""
	if d.UserRepo != nil {
		if user, err := d.UserRepo.FindBySub(ctx, userID); err == nil && user != nil {
			name = user.PreferredUsername
		}
	}
	cache[userID] = name
	return name
}

func (d Deps) handleRevokeAdminConsent(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := oauthusecases.RevokeConsent(
		c.Request().Context(), d.ConsentDeps(), actor.ID,
		c.Param("sub"), c.Param("client_id"), time.Now().UTC(),
	); err != nil {
		return d.WriteConsentError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func toAdminConsentResponse(consent *oauthdomain.Consent, clientName, preferredUsername string) adminConsentResponse {
	return adminConsentResponse{
		UserID: consent.UserID, PreferredUsername: preferredUsername,
		ClientID: consent.ClientID, ClientName: clientName,
		Scopes: slices.Clone(consent.Scopes), State: consent.State,
		GrantedAt: consent.GrantedAt, ExpiresAt: consent.ExpiresAt, RevokedAt: consent.RevokedAt,
	}
}
