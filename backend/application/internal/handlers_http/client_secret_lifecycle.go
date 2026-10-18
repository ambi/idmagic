package handlers_http

import (
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
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
	result, err := d.Clients.RotateSecret(c.Request().Context(), appports.RotateClientSecretInput{ActorUserID: actor.ID, ClientID: clientID, GraceDays: graceDays, Now: time.Now().UTC()})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"client_secret": result.ClientSecret, "grace_until": result.GraceUntil, "credentials": result.Credentials})
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
	result, err := d.Clients.IssueSecret(c.Request().Context(), appports.IssueClientSecretInput{
		ActorUserID: actor.ID, ClientID: clientID, ExpiresInDays: expiresInDays, Now: now,
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, map[string]any{
		"client_secret": result.ClientSecret, "credential": result.Credential, "credentials": result.Credentials,
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
	result, err := d.Clients.RevokeSecret(c.Request().Context(), appports.RevokeClientSecretInput{
		ActorUserID: actor.ID, ClientID: clientID, CredentialID: c.Param("credential_id"), Now: now,
	})
	if err != nil {
		return d.writeApplicationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"credentials": result.Credentials,
	})
}
