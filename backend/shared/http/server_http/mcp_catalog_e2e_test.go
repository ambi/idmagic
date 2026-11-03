package server_http_test

import (
	"context"
	"net/http"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-MCPGATEWAY-010, REQ-OAUTH2-083: 配布する設定を読み、カタログの対応と割当・対象の拒否を実サーバーの作用で確認する。
func TestMcpCatalogGatewayEnforcesCurrentAssignments(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	server, err := f.s.McpResourceServers.FindByResource(context.Background(), tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("server=%v err=%v", server, err)
	}
	server.AccessRules = []oauthdomain.ResourceAccessRule{
		{ResourceType: "idmagic_mcp_operation", Relation: "invoke", Scopes: []string{"tools:read"}, Tool: &oauthdomain.ResourceAccessTool{ID: "operation-" + "read_document", Name: "read_document", DisplayName: "閲覧"}},
		{ResourceType: "idmagic_mcp_operation", Relation: "invoke", Scopes: []string{"tools:read"}, Tool: &oauthdomain.ResourceAccessTool{ID: "operation-" + "delete_document", Name: "delete_document", DisplayName: "削除"}},
	}
	save := func() {
		t.Helper()
		if err := f.s.McpResourceServers.Save(context.Background(), server); err != nil {
			t.Fatal(err)
		}
	}
	save()
	assignment := map[string]any{"resource_type": "idmagic_mcp_operation", "resource_id": server.ID + "/operation-read_document", "relation": "invoke", "subject_type": "agent", "subject_id": "A1"}
	change := func(key string, tuple map[string]any) {
		f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{key: []any{tuple}})
	}
	change("writes", assignment)
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	_ = f.connect(t, token)
	post := func(tool string, want int, wantEffects int32) {
		t.Helper()
		response, body := f.post(t, tool, gatewayBearer(token))
		if response.StatusCode != want || f.effects.Load() != wantEffects || f.deleted.Load() != 0 {
			t.Fatalf("tool=%s status=%d body=%s read=%d delete=%d", tool, response.StatusCode, body, f.effects.Load(), f.deleted.Load())
		}
	}
	post("read_document", http.StatusOK, 1)
	post("delete_document", http.StatusForbidden, 1)
	change("deletes", assignment)
	post("read_document", http.StatusForbidden, 1)
	change("writes", assignment)
	server.AccessRules[0].ResourceType = "document"
	server.AccessRules[0].Relation = "viewer"
	server.AccessRules[0].Tool.ResourceIDArgument = "document_id"
	save()
	change("deletes", d1ViewerTuple())
	post("read_document", http.StatusForbidden, 1)
	change("writes", d1ViewerTuple())
	post("read_document", http.StatusOK, 2)
}
