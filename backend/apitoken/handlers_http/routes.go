package handlers_http

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/apitoken/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	Service *usecases.Service
}

type issueRequest struct {
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`
	ExpiryDays  int      `json:"expiry_days"`
}

type metadataResponse struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Scopes      []string   `json:"scopes"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

func metadataJSON(metadata domain.Metadata) metadataResponse {
	return metadataResponse{
		ID: metadata.ID, Description: metadata.Description, Scopes: metadata.Scopes.Strings(),
		CreatedAt: metadata.CreatedAt, ExpiresAt: metadata.ExpiresAt,
	}
}

func RegisterRoutes(group *echo.Group, deps Deps) {
	handler := handler{deps: deps}
	group.GET("/api/admin/api-tokens", handler.list)
	group.POST("/api/admin/api-tokens", handler.issue)
	group.DELETE("/api/admin/api-tokens/:id", handler.revoke)
}

type handler struct{ deps Deps }

func (h handler) requireAdmin(c *echo.Context) error {
	if _, err := h.deps.RequireAdmin(c); err != nil {
		return h.deps.WriteAdminAccessError(c, err)
	}
	return nil
}

func (h handler) list(c *echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	metadata, err := h.deps.Service.List(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	result := make([]metadataResponse, len(metadata))
	for i, item := range metadata {
		result[i] = metadataJSON(item)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tokens": result})
}

func (h handler) issue(c *echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	var request issueRequest
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "invalid API token request")
	}
	literal, metadata, err := h.deps.Service.Issue(
		c.Request().Context(), support.RequestTenantID(c), request.Description, request.Scopes, request.ExpiryDays,
	)
	if errors.Is(err, usecases.ErrInvalidRequest) {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, map[string]any{
		"token": literal,
		"meta":  metadataJSON(metadata),
	})
}

func (h handler) revoke(c *echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}
	if err := h.deps.Service.Revoke(c.Request().Context(), support.RequestTenantID(c), c.Param("id")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
