package handlers_http

import (
	"errors"
	"net/http"

	"github.com/ambi/idmagic/backend/apitoken/internal/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

func (h handler) personalSubject(c *echo.Context) (string, error) {
	authn, err := h.deps.Authenticate(c)
	if err != nil {
		return "", err
	}
	if authn == nil || authn.Pending() || h.deps.Principals == nil {
		return "", support.ErrAdminAuthenticationRequired
	}
	return authn.Subject(), nil
}

func (h handler) listPersonal(c *echo.Context) error {
	subject, err := h.personalSubject(c)
	if err != nil {
		return h.deps.WriteAdminAccessError(c, err)
	}
	metadata, err := h.deps.Service.ListPersonal(c.Request().Context(), support.RequestTenantID(c), subject)
	if err != nil {
		return err
	}
	result := make([]metadataResponse, len(metadata))
	for i, item := range metadata {
		result[i] = metadataJSON(item)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tokens": result})
}

func (h handler) issuePersonal(c *echo.Context) error {
	if err := h.deps.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if h.deps.PersonalIdentity == nil {
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required.")
	}
	subject, err := h.deps.PersonalIdentity.RequireStepUpSub(c)
	if err != nil {
		if handled, result := h.deps.PersonalIdentity.WriteAuthenticationError(c, err); handled {
			return result
		}
		return h.deps.WriteAdminAccessError(c, err)
	}
	var input issueRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "invalid API token request")
	}
	literal, metadata, err := h.deps.Service.IssuePersonal(c.Request().Context(), support.RequestTenantID(c), subject, input.Description, input.Scopes, input.ExpiryDays, input.DPoPJKT)
	if errors.Is(err, usecases.ErrInvalidRequest) {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "invalid personal API token request")
	}
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, map[string]any{"token": literal, "meta": metadataJSON(metadata)})
}

func (h handler) revokePersonal(c *echo.Context) error {
	if err := h.deps.VerifyBrowserRequest(c); err != nil {
		return err
	}
	subject, err := h.personalSubject(c)
	if err != nil {
		return h.deps.WriteAdminAccessError(c, err)
	}
	if err := h.deps.Service.RevokePersonal(c.Request().Context(), support.RequestTenantID(c), subject, c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
