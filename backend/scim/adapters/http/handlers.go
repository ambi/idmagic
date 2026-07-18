package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/scim/domain"
	"github.com/ambi/idmagic/backend/scim/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/kernel"
)

type Handler struct {
	deps Deps
}

func NewHandler(d Deps) *Handler {
	return &Handler{
		deps: d,
	}
}

func (h *Handler) authenticate(c *echo.Context) (string, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", errors.New("missing or invalid authorization header")
	}
	tokenStr := authHeader[7:]

	resolvedTenantID, err := h.deps.Usecases.AuthenticateToken(c.Request().Context(), tokenStr)
	if err != nil {
		return "", err
	}

	reqTenantID := support.RequestTenantID(c)
	if reqTenantID != resolvedTenantID {
		return "", errors.New("tenant mismatch")
	}

	return reqTenantID, nil
}

func (h *Handler) writeScimError(c *echo.Context, status int, detail, scimType string) error {
	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(status, domain.NewScimError(strconv.Itoa(status), kernel.EnglishErrorText(detail), scimType))
}

func (h *Handler) handleGetServiceProviderConfig(c *echo.Context) error {
	if _, err := h.authenticate(c); err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	config := domain.ServiceProviderConfig{
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		Patch: struct {
			Supported bool `json:"supported"`
		}{Supported: true},
		Bulk: domain.BulkConfig{Supported: false},
		Filter: domain.FilterConfig{
			Supported:  true,
			MaxResults: domain.MaxResults,
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
		AuthenticationSchemes: []domain.AuthenticationScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "OAuth Bearer Token",
				Description: "Authentication scheme using the OAuth Bearer Token Standard",
				SpecUri:     "https://www.rfc-editor.org/info/rfc6750",
			},
		},
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, config)
}

func (h *Handler) handleGetResourceTypes(c *echo.Context) error {
	if _, err := h.authenticate(c); err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	types := []domain.ResourceType{
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
	schemas := []domain.Schema{
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Schema"},
			ID:          "urn:ietf:params:scim:schemas:core:2.0:User",
			Name:        "User",
			Description: "User core schema",
			Attributes:  []domain.SchemaAttribute{},
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

	res, err := h.deps.Usecases.CreateUser(c.Request().Context(), tenantID, body)
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
	res, err := h.deps.Usecases.GetUser(c.Request().Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
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

	res, err := h.deps.Usecases.UpdateUser(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
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

	res, err := h.deps.Usecases.PatchUser(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
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
	if err := h.deps.Usecases.DeleteUser(c.Request().Context(), tenantID, id); err != nil {
		return h.writeScimError(c, http.StatusNotFound, err.Error(), "")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleListUsers(c *echo.Context) error {
	tenantID, err := h.authenticate(c)
	if err != nil {
		return h.writeScimError(c, http.StatusUnauthorized, err.Error(), "")
	}

	query, err := h.parseListQuery(c)
	if err != nil {
		return h.writeScimError(c, http.StatusBadRequest, err.Error(), "invalidValue")
	}

	result, err := h.deps.Usecases.ListUsers(c.Request().Context(), tenantID, query)
	if err != nil {
		return h.writeListError(c, err)
	}

	return h.writeListResponse(c, result)
}

// parseListQuery decodes the filter/startIndex/count query parameters shared
// by ListScimUsers and ListScimGroups (SCL bindings request_form: query).
// Values that fail to parse as integers are a binding-level invalidValue
// error; startIndex/count business rules (RFC 7644 §3.4.2.4) are enforced by
// domain.NormalizePage inside the usecase.
func (h *Handler) parseListQuery(c *echo.Context) (usecases.ListQuery, error) {
	query := usecases.ListQuery{Filter: c.QueryParam("filter")}

	if raw := c.QueryParam("startIndex"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return usecases.ListQuery{}, fmt.Errorf("startIndex must be an integer")
		}
		query.StartIndex = &n
	}

	if c.QueryParams().Has("count") {
		n, err := strconv.Atoi(c.QueryParam("count"))
		if err != nil {
			return usecases.ListQuery{}, fmt.Errorf("count must be an integer")
		}
		query.Count = &n
		query.HasCount = true
	}

	return query, nil
}

// writeListError maps ListUsers/ListGroups errors to the SCIM protocol error
// RFC 7644 §3.12 requires: invalid filters as invalidFilter, invalid
// pagination as invalidValue, anything else as an internal error.
func (h *Handler) writeListError(c *echo.Context, err error) error {
	if _, ok := errors.AsType[*domain.FilterError](err); ok {
		return h.writeScimError(c, http.StatusBadRequest, err.Error(), "invalidFilter")
	}
	if _, ok := errors.AsType[*domain.PaginationError](err); ok {
		return h.writeScimError(c, http.StatusBadRequest, err.Error(), "invalidValue")
	}
	return h.writeScimError(c, http.StatusInternalServerError, err.Error(), "")
}

func (h *Handler) writeListResponse(c *echo.Context, result usecases.ListResult) error {
	list := domain.ListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: result.Total,
		StartIndex:   result.StartIndex,
		ItemsPerPage: result.ItemsPerPage,
		Resources:    make([]any, len(result.Items)),
	}
	for i, item := range result.Items {
		list.Resources[i] = item
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

	res, err := h.deps.Usecases.CreateGroup(c.Request().Context(), tenantID, body)
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
	res, err := h.deps.Usecases.GetGroup(c.Request().Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
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

	query, err := h.parseListQuery(c)
	if err != nil {
		return h.writeScimError(c, http.StatusBadRequest, err.Error(), "invalidValue")
	}

	result, err := h.deps.Usecases.ListGroups(c.Request().Context(), tenantID, query)
	if err != nil {
		return h.writeListError(c, err)
	}

	return h.writeListResponse(c, result)
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

	res, err := h.deps.Usecases.UpdateGroup(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
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

	res, err := h.deps.Usecases.PatchGroup(c.Request().Context(), tenantID, id, body)
	if err != nil {
		if errors.Is(err, usecases.ErrNotFound) {
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
	if err := h.deps.Usecases.DeleteGroup(c.Request().Context(), tenantID, id); err != nil {
		return h.writeScimError(c, http.StatusNotFound, err.Error(), "")
	}

	return c.NoContent(http.StatusNoContent)
}

// Admin API for SCIM access tokens

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

func (h *Handler) handleListAdminTokens(c *echo.Context) error {
	if _, err := h.deps.RequireAdmin(c); err != nil {
		return h.deps.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	tokens, err := h.deps.Usecases.ListTokens(c.Request().Context(), tenantID)
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
	if _, err := h.deps.RequireAdmin(c); err != nil {
		return h.deps.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	var body scimTokenCreateRequest
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return err
	}

	tokenStr, tok, err := h.deps.Usecases.GenerateToken(c.Request().Context(), tenantID, body.Description, body.ExpiryDays)
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
	if _, err := h.deps.RequireAdmin(c); err != nil {
		return h.deps.WriteAdminAccessError(c, err)
	}

	tenantID := support.RequestTenantID(c)
	id := c.Param("id")
	if err := h.deps.Usecases.RevokeToken(c.Request().Context(), tenantID, id); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
