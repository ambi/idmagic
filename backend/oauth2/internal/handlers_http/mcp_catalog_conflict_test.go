package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

type conflictingMcpRepo struct {
	ports.McpResourceServerRepository
	change func()
}

func (r *conflictingMcpRepo) SaveIfUnchanged(ctx context.Context, m *domain.McpResourceServer, expected time.Time) (bool, error) {
	if r.change != nil {
		change := r.change
		r.change = nil
		change()
	}
	return r.McpResourceServerRepository.SaveIfUnchanged(ctx, m, expected)
}

//spec:covers REQ-OAUTH2-085: 読み取り後の競合で、古い操作IDとメタデータを保存しない。
func TestMcpCatalogConcurrentUpdateCannotRestoreRemovedOperation(t *testing.T) {
	repo := &conflictingMcpRepo{McpResourceServerRepository: db_memory.NewMcpResourceServerRepository()}
	e := newAdminMcpResourceServerHandlerWith(oauth2.NewMemoryModule().ClientRepo, repo)
	csrf, cookie := adminCSRF(t, e)
	payload := validMcpResourceServerPayload()
	payload["access_rules"] = []any{map[string]any{"resource_type": "idmagic_mcp_operation", "relation": "invoke", "scopes": []string{"mcp.read"}, "tool": map[string]any{"name": "read", "display_name": "閲覧"}}}
	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, payload)
	var server domain.McpResourceServer
	if err := json.Unmarshal(created.Body.Bytes(), &server); err != nil {
		t.Fatal(err)
	}
	repo.change = func() {
		current, err := repo.FindByID(context.Background(), server.TenantID, server.ID)
		if err != nil {
			t.Fatal(err)
		}
		current.Name = "concurrent edit"
		current.AccessRules = nil
		current.UpdatedAt = current.UpdatedAt.Add(time.Second)
		if err := repo.Save(context.Background(), current); err != nil {
			t.Fatal(err)
		}
	}
	response := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/v1/mcp-resource-servers/"+server.ID, csrf, cookie, map[string]any{"name": "stale edit"})
	if response.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	current, err := repo.FindByID(context.Background(), server.TenantID, server.ID)
	if err != nil || current.Name != "concurrent edit" || len(current.AccessRules) != 0 {
		t.Fatalf("current=%+v err=%v", current, err)
	}
}
