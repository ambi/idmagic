package scim

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
)

type Handler struct {
	Usecases *Usecases
	support  support.Deps
}

func NewHandler(usecases *Usecases, sd support.Deps) *Handler {
	return &Handler{
		Usecases: usecases,
		support:  sd,
	}
}

func (h *Handler) authenticate(c *echo.Context) (string, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", errors.New("missing or invalid authorization header")
	}
	tokenStr := authHeader[7:]

	resolvedTenantID, err := h.Usecases.AuthenticateToken(c.Request().Context(), tokenStr)
	if err != nil {
		return "", err
	}

	reqTenantID := support.RequestTenantID(c)
	if reqTenantID != resolvedTenantID {
		return "", errors.New("tenant mismatch")
	}

	cfg, err := h.Usecases.GetConfig(c.Request().Context(), reqTenantID)
	if err != nil || !cfg.Enabled {
		return "", errors.New("SCIM is disabled for this tenant")
	}

	return reqTenantID, nil
}

func (h *Handler) writeScimError(c *echo.Context, status int, detail, scimType string) error {
	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(status, NewScimError(strconv.Itoa(status), detail, scimType))
}

func (h *Handler) handleGetServiceProviderConfig(c *echo.Context) error {
	if _, err := h.authenticate(c); err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	config := ServiceProviderConfig{
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		Patch: struct {
			Supported bool `json:"supported"`
		}{Supported: true},
		Bulk: BulkConfig{Supported: false},
		Filter: FilterConfig{
			Supported:  true,
			MaxResults: 100,
		},
		ChangePassword: struct {
			Supported bool `json:"supported"`
		}{Supported: false},
		Sort: struct {
			Supported bool `json:"supported"`
		}{Supported: false},
		Etag: struct {
			Supported bool `json:"supported"`
		}{Supported: false},
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, config)
}

func (h *Handler) handleGetResourceTypes(c *echo.Context) error {
	if _, err := h.authenticate(c); err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	types := []ResourceType{
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			ID:          "User",
			Name:        "User",
			Endpoint:    "/Users",
			Description: "User Account",
			Schema:      "urn:ietf:params:scim:schemas:core:2.0:User",
		},
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			ID:          "Group",
			Name:        "Group",
			Endpoint:    "/Groups",
			Description: "Group",
			Schema:      "urn:ietf:params:scim:schemas:core:2.0:Group",
		},
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, types)
}

func (h *Handler) handleGetSchemas(c *echo.Context) error {
	if _, err := h.authenticate(c); err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	// 最小限の schemas
	schemas := []Schema{
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Schema"},
			ID:          "urn:ietf:params:scim:schemas:core:2.0:User",
			Name:        "User",
			Description: "User core schema",
			Attributes:  []SchemaAttribute{},
		},
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, schemas)
}

// Users
func (h *Handler) handleCreateUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "")
	}

	res, err := h.Usecases.CreateUser(c.Request().Context(), tenantID, body)
	if err != nil {
		return h.writeScimError(c, http.StatusConflict, err.Error(), "uniqueness")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusCreated, res)
}

func (h *Handler) handleGetUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	res, err := h.Usecases.GetUser(c.Request().Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "user not found", "")
		}
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleUpdateUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "")
	}

	res, err := h.Usecases.UpdateUser(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "user not found", "")
		}
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handlePatchUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "")
	}

	res, err := h.Usecases.PatchUser(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "user not found", "")
		}
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleDeleteUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	if err := h.Usecases.DeleteUser(c.Request().Context(), tenantID, id); err != nil {
		return h.writeScimError(c, http.StatusNotFound, err.Error(), "")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleListUsers(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	filter := c.QueryParam("filter")
	users, err := h.Usecases.ListUsers(c.Request().Context(), tenantID, filter)
	if err != nil {
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	// SCIM ListResponse
	list := ListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: len(users),
		StartIndex:   1,
		ItemsPerPage: len(users),
		Resources:    make([]any, len(users)),
	}
	for i, u := range users {
		list.Resources[i] = u
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, list)
}

// Groups
func (h *Handler) handleCreateGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "")
	}

	res, err := h.Usecases.CreateGroup(c.Request().Context(), tenantID, body)
	if err != nil {
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusCreated, res)
}

func (h *Handler) handleGetGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	res, err := h.Usecases.GetGroup(c.Request().Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "group not found", "")
		}
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleListGroups(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	groups, err := h.Usecases.ListGroups(c.Request().Context(), tenantID)
	if err != nil {
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	list := ListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: len(groups),
		StartIndex:   1,
		ItemsPerPage: len(groups),
		Resources:    make([]any, len(groups)),
	}
	for i, g := range groups {
		list.Resources[i] = g
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, list)
}

func (h *Handler) handleUpdateGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "")
	}

	res, err := h.Usecases.UpdateGroup(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "group not found", "")
		}
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handlePatchGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "")
	}

	res, err := h.Usecases.PatchGroup(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "group not found", "")
		}
		return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleDeleteGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	id := c.Param("id")
	if err := h.Usecases.DeleteGroup(c.Request().Context(), tenantID, id); err != nil {
		return h.writeScimError(c, http.StatusNotFound, err.Error(), "")
	}

	return c.NoContent(http.StatusNoContent)
}

// Admin API for SCIM configuration and tokens

type scimConfigResponse struct {
	TenantID  string    `json:"tenant_id"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type scimConfigUpdateRequest struct {
	Enabled bool `json:"enabled"`
}

type scimTokenResponse struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type scimTokenCreateRequest struct {
	Description string `json:"description"`
	ExpiryDays  int    `json:"expiry_days"`
}

func (h *Handler) handleGetAdminConfig(c *echo.Context) error {
	if _, err := h.support.RequireAdmin(c); err != nil {
		return h.support.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	cfg, err := h.Usecases.GetConfig(c.Request().Context(), tenantID)
	if err != nil {
		return err
	}

	return support.NoStoreJSON(c, http.StatusOK, scimConfigResponse{
		TenantID:  cfg.TenantID,
		Enabled:   cfg.Enabled,
		CreatedAt: cfg.CreatedAt,
		UpdatedAt: cfg.UpdatedAt,
	})
}

func (h *Handler) handleUpdateAdminConfig(c *echo.Context) error {
	if _, err := h.support.RequireAdmin(c); err != nil {
		return h.support.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	var body scimConfigUpdateRequest
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return err
	}

	cfg, err := h.Usecases.UpdateConfig(c.Request().Context(), tenantID, body.Enabled)
	if err != nil {
		return err
	}

	return support.NoStoreJSON(c, http.StatusOK, scimConfigResponse{
		TenantID:  cfg.TenantID,
		Enabled:   cfg.Enabled,
		CreatedAt: cfg.CreatedAt,
		UpdatedAt: cfg.UpdatedAt,
	})
}

func (h *Handler) handleListAdminTokens(c *echo.Context) error {
	if _, err := h.support.RequireAdmin(c); err != nil {
		return h.support.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	tokens, err := h.Usecases.ListTokens(c.Request().Context(), tenantID)
	if err != nil {
		return err
	}

	res := make([]scimTokenResponse, len(tokens))
	for i, tok := range tokens {
		res[i] = scimTokenResponse{
			ID:          tok.ID,
			Description: tok.Description,
			CreatedAt:   tok.CreatedAt,
			ExpiresAt:   tok.ExpiresAt,
		}
	}

	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"tokens": res})
}

func (h *Handler) handleCreateAdminToken(c *echo.Context) error {
	if _, err := h.support.RequireAdmin(c); err != nil {
		return h.support.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	var body scimTokenCreateRequest
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return err
	}

	tokenStr, tok, err := h.Usecases.GenerateToken(c.Request().Context(), tenantID, body.Description, body.ExpiryDays)
	if err != nil {
		return err
	}

	return support.NoStoreJSON(c, http.StatusCreated, map[string]any{
		"token": tokenStr,
		"meta": scimTokenResponse{
			ID:          tok.ID,
			Description: tok.Description,
			CreatedAt:   tok.CreatedAt,
			ExpiresAt:   tok.ExpiresAt,
		},
	})
}

func (h *Handler) handleRevokeAdminToken(c *echo.Context) error {
	if _, err := h.support.RequireAdmin(c); err != nil {
		return h.support.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	id := c.Param("id")
	if err := h.Usecases.RevokeToken(c.Request().Context(), tenantID, id); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
