// MCP resource server 登録 (RFC 9728 / RFC 8707) の管理 API。
// AdminMcpResourcesManage で保護され、テナント境界に閉じる。
package handlers_http

import (
	"context"
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
	Resource string   `json:"resource"`
	Name     string   `json:"name"`
	Scopes   []string `json:"scopes"`
	// ClientID と AccessRules は、更新で省くと変えない (REQ-OAUTH2-064)。
	ClientID    *string                            `json:"client_id"`
	AccessRules []oauthdomain.ResourceAccessRule   `json:"access_rules"`
	State       oauthdomain.McpResourceServerState `json:"state"`
}

func toMcpResourceServerResponse(m *oauthdomain.McpResourceServer) map[string]any {
	rules := m.AccessRules
	if rules == nil {
		rules = []oauthdomain.ResourceAccessRule{}
	}
	// 保存表現のキー規約を画面へ持ち出さず、割当先を不透明な参照として返す。
	rules = slices.Clone(rules)
	for i := range rules {
		if rules[i].Tool != nil {
			tool := *rules[i].Tool
			tool.AssignmentResourceID = m.OperationResourceID(rules[i])
			rules[i].Tool = &tool
		}
	}
	response := map[string]any{
		"tenant_id":    m.TenantID,
		"id":           m.ID,
		"resource":     m.Resource,
		"name":         m.Name,
		"scopes":       m.Scopes,
		"access_rules": rules,
		"state":        m.State,
		"created_at":   m.CreatedAt,
		"updated_at":   m.UpdatedAt,
	}
	if m.ClientID != "" {
		response["client_id"] = m.ClientID
	}
	return response
}

func (d Deps) handleListAdminMcpResourceServers(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	servers, err := d.McpResourceServerRepo.ListAll(c.Request().Context(), support.RequestTenantID(c))
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
	m, err := d.McpResourceServerRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), c.Param("id"))
	if err != nil {
		return err
	}
	if m == nil {
		return support.WriteProblem(c, http.StatusNotFound, "resource_server_not_found", "The MCP resource server does not exist.")
	}
	return support.NoStoreJSON(c, http.StatusOK, toMcpResourceServerResponse(m))
}

// handleGetAdminMcpToolCatalog は最新のカタログを公開中のツールと別に返す (REQ-OAUTH2-087)。
func (d Deps) handleGetAdminMcpToolCatalog(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx, tenantID := c.Request().Context(), support.RequestTenantID(c)
	m, err := d.McpResourceServerRepo.FindByID(ctx, tenantID, c.Param("id"))
	if err != nil {
		return err
	}
	if m == nil {
		return support.WriteProblem(c, http.StatusNotFound, "resource_server_not_found", "The MCP resource server does not exist.")
	}
	response := map[string]any{"tools": []oauthdomain.CatalogTool{}, "complete": false}
	catalog, err := d.McpToolCatalogRepo.Find(ctx, tenantID, m.ID)
	if err != nil {
		return err
	}
	if catalog != nil {
		response["tools"], response["complete"], response["received_at"] = catalog.Tools, catalog.Complete, catalog.ReceivedAt
		if changes := catalog.Changes(); changes != nil {
			response["changes"] = toToolCatalogChangesResponse(changes)
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, response)
}

// toolCatalogChangesResponse は TypeSpec の ToolCatalogChanges の形である。
type toolCatalogChangesResponse struct {
	ChangedAt time.Time                         `json:"changed_at"`
	Added     []string                          `json:"added"`
	Removed   []string                          `json:"removed"`
	Modified  []catalogToolModificationResponse `json:"modified"`
}

type catalogToolModificationResponse struct {
	Name     string                  `json:"name"`
	Fields   []string                `json:"fields"`
	Previous oauthdomain.CatalogTool `json:"previous"`
}

func toToolCatalogChangesResponse(changes *oauthdomain.ToolCatalogChanges) toolCatalogChangesResponse {
	modified := make([]catalogToolModificationResponse, len(changes.Modified))
	for index, tool := range changes.Modified {
		modified[index] = catalogToolModificationResponse{Name: tool.Name, Fields: tool.Fields, Previous: tool.Previous}
	}
	return toolCatalogChangesResponse{ChangedAt: changes.ChangedAt, Added: changes.Added, Removed: changes.Removed, Modified: modified}
}

// defaultMcpScope は、スコープを省いた登録に補う唯一のスコープである。Gateway が執行する構成では
// スコープは IdMagic の中だけの認証上限なので、管理者に名前を考えさせない (REQ-OAUTH2-064)。
const defaultMcpScope = "mcp"

func (d Deps) handleCreateAdminMcpResourceServer(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req mcpResourceServerRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if strings.TrimSpace(req.Resource) == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "resource is required.")
	}
	tenantID := support.RequestTenantID(c)
	existing, err := d.McpResourceServerRepo.FindByResource(c.Request().Context(), tenantID, req.Resource)
	if err != nil {
		return err
	}
	if existing != nil {
		return support.WriteProblem(c, http.StatusConflict, "resource_exists", "The same resource is already registered.")
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
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{defaultMcpScope}
	}
	m := &oauthdomain.McpResourceServer{
		TenantID: tenantID, ID: id, Resource: req.Resource,
		Name: req.Name, Scopes: scopes, AccessRules: req.AccessRules, State: state,
		CreatedAt: now, UpdatedAt: now,
	}
	if req.ClientID != nil {
		m.ClientID = *req.ClientID
	}
	if err := prepareMcpToolIDs(m.AccessRules, nil); err != nil {
		return err
	}
	if err := d.saveValidatedMcpResourceServer(c, m, nil); err != nil {
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
	existing, err := d.McpResourceServerRepo.FindByID(c.Request().Context(), tenantID, c.Param("id"))
	if err != nil {
		return err
	}
	if existing == nil {
		return support.WriteProblem(c, http.StatusNotFound, "resource_server_not_found", "The MCP resource server does not exist.")
	}
	var req mcpResourceServerRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	// resource (canonical URI) は不変 — 発行済みトークンの aud 意味が変わるため更新不可。
	expectedUpdatedAt := existing.UpdatedAt
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Scopes != nil {
		existing.Scopes = req.Scopes
	}
	if req.ClientID != nil {
		existing.ClientID = *req.ClientID
	}
	if req.AccessRules != nil {
		if err := prepareMcpToolIDs(req.AccessRules, existing.AccessRules); err != nil {
			return err
		}
		existing.AccessRules = req.AccessRules
	}
	if req.State != "" {
		existing.State = req.State
	}
	existing.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
	if !existing.UpdatedAt.After(expectedUpdatedAt) {
		existing.UpdatedAt = expectedUpdatedAt.Add(time.Microsecond)
	}
	if err := d.saveValidatedMcpResourceServer(c, existing, &expectedUpdatedAt); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toMcpResourceServerResponse(existing))
}

// 同名の公開中ツールは識別を保つ。削除・改名後の再登録には必ず新しい識別を発行し、
// 古い割当を再有効化しない。本文の ID と割当参照は信頼しない。
func prepareMcpToolIDs(rules, previous []oauthdomain.ResourceAccessRule) error {
	ids := map[string]string{}
	for _, rule := range previous {
		if rule.Tool != nil {
			ids[rule.Tool.Name] = rule.Tool.ID
		}
	}
	for i := range rules {
		if rules[i].Tool == nil {
			continue
		}
		rules[i].Tool.AssignmentResourceID = ""
		id := ids[rules[i].Tool.Name]
		if id == "" {
			var err error
			id, err = spec.NewUUIDv4()
			if err != nil {
				return err
			}
		}
		rules[i].Tool.ID = id
	}
	return nil
}

func (d Deps) handleDeleteAdminMcpResourceServer(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx, tenantID := c.Request().Context(), support.RequestTenantID(c)
	if err := d.McpResourceServerRepo.Delete(ctx, tenantID, c.Param("id")); err != nil {
		return err
	}
	// PostgreSQL は外部キーで消すが、メモリの保存先にも同じ結果を保証する。
	// カタログの保存先を配線しない構成には消すカタログもない。
	if d.McpToolCatalogRepo != nil {
		if err := d.McpToolCatalogRepo.Delete(ctx, tenantID, c.Param("id")); err != nil {
			return err
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) saveValidatedMcpResourceServer(c *echo.Context, m *oauthdomain.McpResourceServer, expectedUpdatedAt *time.Time) error {
	if err := m.Validate(); err != nil {
		_ = support.WriteProblem(c, http.StatusBadRequest, "invalid_resource_server", err.Error())
		return support.ErrResponseWritten
	}
	if problem, err := d.resourceServerClientProblem(c.Request().Context(), m); err != nil {
		return err
	} else if problem != "" {
		_ = support.WriteProblem(c, http.StatusBadRequest, "invalid_resource_server", problem)
		return support.ErrResponseWritten
	}
	if expectedUpdatedAt == nil {
		return d.McpResourceServerRepo.Save(c.Request().Context(), m)
	}
	saved, err := d.McpResourceServerRepo.SaveIfUnchanged(c.Request().Context(), m, *expectedUpdatedAt)
	if err != nil {
		return err
	}
	if !saved {
		_ = support.WriteProblem(c, http.StatusConflict, "resource_server_conflict", "The resource server changed. Reload and retry.")
		return support.ErrResponseWritten
	}
	return nil
}

// resourceServerClientProblem は、評価用に束縛するクライアントがテナントの機密クライアントで、
// 別の登録に束縛されていないことを確かめる (REQ-OAUTH2-064)。
func (d Deps) resourceServerClientProblem(ctx context.Context, m *oauthdomain.McpResourceServer) (string, error) {
	if m.ClientID == "" {
		return "", nil
	}
	client, err := d.ClientRepo.FindByID(ctx, m.TenantID, m.ClientID)
	if err != nil {
		return "", err
	}
	if client == nil || client.ClientType != spec.ClientConfidential {
		return "client_id must name a confidential client of this tenant.", nil
	}
	bound, err := d.McpResourceServerRepo.FindByClientID(ctx, m.TenantID, m.ClientID)
	if err != nil {
		return "", err
	}
	if bound != nil && bound.ID != m.ID {
		return "client_id is already bound to another resource server.", nil
	}
	return "", nil
}
