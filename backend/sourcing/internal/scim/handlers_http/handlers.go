package handlers_http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	scimdomain "github.com/ambi/idmagic/backend/sourcing/internal/scim/domain"
	scimusecases "github.com/ambi/idmagic/backend/sourcing/internal/scim/usecases"
)

type Handler struct {
	deps Deps
}

func NewHandler(d Deps) *Handler {
	return &Handler{
		deps: d,
	}
}

func (h *Handler) authenticate(c *echo.Context, allowedScopes ...apitokendomain.Scope) (string, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return "", apitokendomain.ErrInvalidToken
	}
	tokenStr := authHeader[7:]

	principal, err := h.deps.ApiTokenAuthenticator.Authenticate(c.Request().Context(), tokenStr)
	if err != nil {
		return "", err
	}
	if !principal.Scopes.HasAny(allowedScopes...) {
		required := make([]string, 0, len(allowedScopes))
		for _, scope := range allowedScopes {
			required = append(required, string(scope))
		}
		return "", &support.InsufficientScopeError{Required: strings.Join(required, " ")}
	}

	reqTenantID := support.RequestTenantID(c)
	if reqTenantID != principal.TenantID {
		return "", apitokendomain.ErrInvalidToken
	}

	return reqTenantID, nil
}

func (h *Handler) writeScimAuthError(c *echo.Context, err error) error {
	if scopeErr, ok := errors.AsType[*support.InsufficientScopeError](err); ok {
		support.SetBearerChallenge(c, "insufficient_scope", scopeErr.Required)
		return h.writeScimError(c, http.StatusForbidden, err.Error(), "")
	}
	if !errors.Is(err, apitokendomain.ErrInvalidToken) {
		return h.writeInternalError(c, err)
	}
	support.SetBearerChallenge(c, "invalid_token", "")
	return h.writeScimError(c, http.StatusUnauthorized, apitokendomain.ErrInvalidToken.Error(), "")
}

func (h *Handler) writeInternalError(c *echo.Context, err error) error {
	request := c.Request()
	logging.Error(request.Context(), "SCIM request failed",
		"error", err.Error(), "stack", fmt.Sprintf("%+v", err),
		"method", request.Method, "path", request.URL.Path,
	)
	return h.writeScimError(c, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), "")
}

func (h *Handler) writeScimError(c *echo.Context, status int, detail, scimType string) error {
	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(status, scimdomain.NewScimError(strconv.Itoa(status), detail, scimType))
}

// writeMutationError maps CreateUser/UpdateUser/PatchUser/CreateGroup/
// UpdateGroup/PatchGroup errors to the SCIM protocol error RFC 7644 §3.12
// requires: not found as 404, a uniqueness conflict as 409, a
// *domain.MutationError as 400 with its carried scimType (invalidValue /
// invalidPath / mutability), anything else as an internal error.
func (h *Handler) writeMutationError(c *echo.Context, notFoundDetail string, err error) error {
	if quota, ok := errors.AsType[interface {
		error
		IsQuotaExceeded() bool
	}](err); ok && quota.IsQuotaExceeded() {
		return h.writeScimError(c, http.StatusUnprocessableEntity, err.Error(), "")
	}
	if errors.Is(err, scimusecases.ErrNotFound) {
		return h.writeScimError(c, http.StatusNotFound, notFoundDetail, "")
	}
	if errors.Is(err, scimusecases.ErrDuplicate) {
		return h.writeScimError(c, http.StatusConflict, err.Error(), "uniqueness")
	}
	if mutErr, ok := errors.AsType[*scimdomain.MutationError](err); ok {
		return h.writeScimError(c, http.StatusBadRequest, err.Error(), mutErr.ScimType)
	}
	return h.writeInternalError(c, err)
}

func (h *Handler) handleGetServiceProviderConfig(c *echo.Context) error {
	if _, err := h.authenticate(c, apitokendomain.ScopeScimUsersRead, apitokendomain.ScopeScimUsersWrite, apitokendomain.ScopeScimGroupsRead, apitokendomain.ScopeScimGroupsWrite); err != nil {
		return h.writeScimAuthError(c, err)
	}

	config := scimdomain.ServiceProviderConfig{
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		Patch: struct {
			Supported bool `json:"supported"`
		}{Supported: true},
		Bulk: scimdomain.BulkConfig{Supported: false},
		Filter: scimdomain.FilterConfig{
			Supported:  true,
			MaxResults: scimdomain.MaxResults,
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
		AuthenticationSchemes: []scimdomain.AuthenticationScheme{
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

// discoveryListResponse is the ListResponse that GET /Schemas and
// /ResourceTypes answer with (RFC 7644 §4). Discovery does not paginate, so
// the one page holds every resource: totalResults and itemsPerPage are the
// resource count and startIndex is 1. It is a literal type of its own, not
// scimdomain.ListResponse, so that each handler writes the body it returns
// where check-contract-drift can compare it with the contract.
type discoveryListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    any      `json:"Resources"`
}

// refuseDiscoveryFilter answers a discovery request that carries a filter,
// even an empty one, with 403 as RFC 7644 §4 recommends, so that a client
// never mistakes the full list for a filtered one (REQ-SOURCING-029). The
// other query parameters (sorting, pagination) are ignored as §4 requires.
func (h *Handler) refuseDiscoveryFilter(c *echo.Context) (bool, error) {
	if !c.Request().URL.Query().Has("filter") {
		return false, nil
	}
	return true, h.writeScimError(c, http.StatusForbidden, "filtering is not supported on discovery endpoints", "")
}

func (h *Handler) handleGetResourceTypes(c *echo.Context) error {
	if _, err := h.authenticate(c, apitokendomain.ScopeScimUsersRead, apitokendomain.ScopeScimUsersWrite, apitokendomain.ScopeScimGroupsRead, apitokendomain.ScopeScimGroupsWrite); err != nil {
		return h.writeScimAuthError(c, err)
	}
	if refused, err := h.refuseDiscoveryFilter(c); refused {
		return err
	}

	types := []scimdomain.ResourceType{
		{
			Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			ID:          "User",
			Name:        "User",
			Endpoint:    "/Users",
			Description: "User Account",
			Schema:      "urn:ietf:params:scim:schemas:core:2.0:User",
			SchemaExtensions: []scimdomain.SchemaExtension{
				{Schema: scimdomain.EnterpriseUserSchemaURN, Required: false},
			},
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
	return c.JSON(http.StatusOK, discoveryListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: len(types),
		StartIndex:   1,
		ItemsPerPage: len(types),
		Resources:    types,
	})
}

func (h *Handler) handleGetSchemas(c *echo.Context) error {
	if _, err := h.authenticate(c, apitokendomain.ScopeScimUsersRead, apitokendomain.ScopeScimUsersWrite, apitokendomain.ScopeScimGroupsRead, apitokendomain.ScopeScimGroupsWrite); err != nil {
		return h.writeScimAuthError(c, err)
	}
	if refused, err := h.refuseDiscoveryFilter(c); refused {
		return err
	}

	schemas := []scimdomain.Schema{scimdomain.UserCoreSchema(), scimdomain.EnterpriseUserSchema(), scimdomain.GroupCoreSchema()}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, discoveryListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: len(schemas),
		StartIndex:   1,
		ItemsPerPage: len(schemas),
		Resources:    schemas,
	})
}

// Users
func (h *Handler) handleCreateUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimUsersWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "invalidSyntax")
	}

	res, err := h.deps.Usecases.CreateUser(c.Request().Context(), tenantID, body)
	if err != nil {
		return h.writeMutationError(c, "", err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusCreated, res)
}

func (h *Handler) handleGetUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimUsersRead)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	res, err := h.deps.Usecases.GetUser(c.Request().Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, scimusecases.ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "user not found", "")
		}
		return h.writeInternalError(c, err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleUpdateUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimUsersWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "invalidSyntax")
	}

	res, err := h.deps.Usecases.UpdateUser(c.Request().Context(), tenantID, id, body)
	if err != nil {
		return h.writeMutationError(c, "user not found", err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handlePatchUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimUsersWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "invalidSyntax")
	}

	res, err := h.deps.Usecases.PatchUser(c.Request().Context(), tenantID, id, body)
	if err != nil {
		return h.writeMutationError(c, "user not found", err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleDeleteUser(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimUsersWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	if err := h.deps.Usecases.DeleteUser(c.Request().Context(), tenantID, id); err != nil {
		return h.writeMutationError(c, "SCIM resource not found", err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) handleListUsers(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimUsersRead)
	if err != nil {
		return h.writeScimAuthError(c, err)
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
func (h *Handler) parseListQuery(c *echo.Context) (scimusecases.ListQuery, error) {
	query := scimusecases.ListQuery{Filter: c.QueryParam("filter")}

	if raw := c.QueryParam("startIndex"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return scimusecases.ListQuery{}, fmt.Errorf("startIndex must be an integer")
		}
		query.StartIndex = &n
	}

	if c.QueryParams().Has("count") {
		n, err := strconv.Atoi(c.QueryParam("count"))
		if err != nil {
			return scimusecases.ListQuery{}, fmt.Errorf("count must be an integer")
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
	if _, ok := errors.AsType[*scimdomain.FilterError](err); ok {
		return h.writeScimError(c, http.StatusBadRequest, err.Error(), "invalidFilter")
	}
	if _, ok := errors.AsType[*scimdomain.PaginationError](err); ok {
		return h.writeScimError(c, http.StatusBadRequest, err.Error(), "invalidValue")
	}
	return h.writeInternalError(c, err)
}

func (h *Handler) writeListResponse(c *echo.Context, result scimusecases.ListResult) error {
	list := scimdomain.ListResponse{
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
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimGroupsWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "invalidSyntax")
	}

	res, err := h.deps.Usecases.CreateGroup(c.Request().Context(), tenantID, body)
	if err != nil {
		return h.writeMutationError(c, "", err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusCreated, res)
}

func (h *Handler) handleGetGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimGroupsRead)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	res, err := h.deps.Usecases.GetGroup(c.Request().Context(), tenantID, id)
	if err != nil {
		if errors.Is(err, scimusecases.ErrNotFound) {
			return h.writeScimError(c, http.StatusNotFound, "group not found", "")
		}
		return h.writeInternalError(c, err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleListGroups(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimGroupsRead)
	if err != nil {
		return h.writeScimAuthError(c, err)
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
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimGroupsWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "invalidSyntax")
	}

	res, err := h.deps.Usecases.UpdateGroup(c.Request().Context(), tenantID, id, body)
	if err != nil {
		return h.writeMutationError(c, "group not found", err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handlePatchGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimGroupsWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	var body map[string]any
	if err := support.DecodeJSON(c.Request(), &body); err != nil {
		return h.writeScimError(c, http.StatusBadRequest, "invalid body", "invalidSyntax")
	}

	res, err := h.deps.Usecases.PatchGroup(c.Request().Context(), tenantID, id, body)
	if err != nil {
		return h.writeMutationError(c, "group not found", err)
	}

	c.Response().Header().Set("Content-Type", "application/scim+json")
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) handleDeleteGroup(c *echo.Context) error {
	tenantID, err := h.authenticate(c, apitokendomain.ScopeScimGroupsWrite)
	if err != nil {
		return h.writeScimAuthError(c, err)
	}

	id := c.Param("id")
	if err := h.deps.Usecases.DeleteGroup(c.Request().Context(), tenantID, id); err != nil {
		return h.writeMutationError(c, "SCIM resource not found", err)
	}

	return c.NoContent(http.StatusNoContent)
}
