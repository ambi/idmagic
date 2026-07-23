// MCP resource server 登録 (RFC 9728 / RFC 8707, ADR-055) の管理 API。
// AdminMcpResourcesManage で保護され、テナント境界に閉じる。
package handlers_http

import (
	"net/http"
	"slices"
	"strings"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type mcpResourceServerRequest struct {
	Resource string                             `json:"resource"`
	Name     string                             `json:"name"`
	Scopes   []string                           `json:"scopes"`
	State    oauthdomain.McpResourceServerState `json:"state"`
}

func toMcpResourceServerResponse(m *oauthdomain.McpResourceServer) map[string]any {
	return map[string]any{
		"tenant_id":          m.TenantID,
		"resource_server_id": m.ResourceServerID,
		"resource":           m.Resource,
		"name":               m.Name,
		"scopes":             m.Scopes,
		"state":              m.State,
		"created_at":         m.CreatedAt,
		"updated_at":         m.UpdatedAt,
	}
}

func (d Deps) handleListAdminMcpResourceServers(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	servers, err := d.McpResourceServerRepo.ListByTenant(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	slices.SortFunc(servers, func(a, b *oauthdomain.McpResourceServer) int { return strings.Compare(a.Resource, b.Resource) })
	out := make([]map[string]any, len(servers))
	for i, s := range servers {
		out[i] = toMcpResourceServerResponse(s)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"resource_servers": out})
}

func (d Deps) handleGetAdminMcpResourceServer(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	m, err := d.McpResourceServerRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), c.Param("resource_server_id"))
	if err != nil {
		return err
	}
	if m == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "resource_server_not_found", "The MCP resource server does not exist.")
	}
	return support.NoStoreJSON(c, http.StatusOK, toMcpResourceServerResponse(m))
}

func (d Deps) handleCreateAdminMcpResourceServer(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req mcpResourceServerRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if strings.TrimSpace(req.Resource) == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "resource is required.")
	}
	tenantID := support.RequestTenantID(c)
	existing, err := d.McpResourceServerRepo.FindByResource(c.Request().Context(), tenantID, req.Resource)
	if err != nil {
		return err
	}
	if existing != nil {
		return support.WriteBrowserError(c, http.StatusConflict, "resource_exists", "The same resource is already registered.")
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	state := req.State
	if state == "" {
		state = oauthdomain.McpResourceServerActive
	}
	m := &oauthdomain.McpResourceServer{
		TenantID: tenantID, ResourceServerID: id, Resource: req.Resource,
		Name: req.Name, Scopes: req.Scopes, State: state,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := d.saveValidatedMcpResourceServer(c, m); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, toMcpResourceServerResponse(m))
}

func (d Deps) handleUpdateAdminMcpResourceServer(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	tenantID := support.RequestTenantID(c)
	existing, err := d.McpResourceServerRepo.FindByID(c.Request().Context(), tenantID, c.Param("resource_server_id"))
	if err != nil {
		return err
	}
	if existing == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "resource_server_not_found", "The MCP resource server does not exist.")
	}
	var req mcpResourceServerRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	// resource (canonical URI) は不変 — 発行済みトークンの aud 意味が変わるため更新不可。
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Scopes != nil {
		existing.Scopes = req.Scopes
	}
	if req.State != "" {
		existing.State = req.State
	}
	existing.UpdatedAt = time.Now().UTC()
	if err := d.saveValidatedMcpResourceServer(c, existing); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toMcpResourceServerResponse(existing))
}

func (d Deps) handleDeleteAdminMcpResourceServer(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := d.McpResourceServerRepo.Delete(c.Request().Context(), support.RequestTenantID(c), c.Param("resource_server_id")); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) saveValidatedMcpResourceServer(c *echo.Context, m *oauthdomain.McpResourceServer) error {
	if err := m.Validate(); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_resource_server", err.Error())
	}
	if err := d.McpResourceServerRepo.Save(c.Request().Context(), m); err != nil {
		return err
	}
	return nil
}
