package handlers_http

import (
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	clientusecases "github.com/ambi/idmagic/backend/oauth2/client/usecases"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type rotateOIDCClientSecretRequest struct {
	GraceDays *int `json:"grace_days"`
}

func (d Deps) handleRotateOIDCClientSecret(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := d.requireApp(c)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	clientID := bindingKeyOf(app, domain.ApplicationProtocolOIDC)
	if clientID == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The OIDC binding does not exist.")
	}
	var req rotateOIDCClientSecretRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	graceDays := 7
	if req.GraceDays != nil {
		graceDays = *req.GraceDays
	}
	result, err := clientusecases.RotateClientSecret(c.Request().Context(), clientusecases.AdminOAuth2ClientDeps{ClientRepo: d.ClientRepo, Emit: d.Emit}, clientusecases.RotateClientSecretInput{ActorUserID: actor.ID, ClientID: clientID, GraceDays: graceDays, Now: time.Now().UTC()})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	metadata := clientSecretMetadata(result.Credentials, time.Now().UTC())
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"client_secret": result.ClientSecret, "grace_until": result.GraceUntil, "credentials": metadata})
}

type issueOIDCClientSecretRequest struct {
	ExpiresInDays *int `json:"expires_in_days"`
}

func (d Deps) handleIssueOIDCClientSecret(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := d.requireApp(c)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	clientID := bindingKeyOf(app, domain.ApplicationProtocolOIDC)
	if clientID == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The OIDC binding does not exist.")
	}
	var req issueOIDCClientSecretRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	expiresInDays := 90
	if req.ExpiresInDays != nil {
		expiresInDays = *req.ExpiresInDays
	}
	now := time.Now().UTC()
	result, err := clientusecases.IssueClientSecret(c.Request().Context(), clientusecases.AdminOAuth2ClientDeps{
		ClientRepo: d.ClientRepo, Emit: d.Emit,
	}, clientusecases.IssueClientSecretInput{
		ActorUserID: actor.ID, ClientID: clientID, ExpiresInDays: expiresInDays, Now: now,
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	metadata := clientSecretMetadata(result.Credentials, now)
	issued := clientSecretMetadata([]oauthdomain.ClientSecretCredential{result.Credential}, now)[0]
	return support.NoStoreJSON(c, http.StatusCreated, map[string]any{
		"client_secret": result.ClientSecret, "credential": issued, "credentials": metadata,
	})
}

func (d Deps) handleRevokeOIDCClientSecret(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	app, err := d.requireApp(c)
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	clientID := bindingKeyOf(app, domain.ApplicationProtocolOIDC)
	if clientID == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The OIDC binding does not exist.")
	}
	now := time.Now().UTC()
	result, err := clientusecases.RevokeClientSecret(c.Request().Context(), clientusecases.AdminOAuth2ClientDeps{
		ClientRepo: d.ClientRepo, Emit: d.Emit,
	}, clientusecases.RevokeClientSecretInput{
		ActorUserID: actor.ID, ClientID: clientID, CredentialID: c.Param("credential_id"), Now: now,
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"credentials": clientSecretMetadata(result.Credentials, now),
	})
}
