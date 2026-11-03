package server_http_test

import (
	"context"
	"net/http"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-OAUTH2-083, REQ-AUTHORIZATION-017, REQ-MCPGATEWAY-010: 対象ごとの権限を持つツールは、モデルを作らずに組み込みの割当で指定した対象だけを許可し、範囲の追加と取消が次の評価に効く。
func TestTargetScopedToolsHonorAssignedTargetsWithoutAModel(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	ctx := context.Background()
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("server=%v err=%v", server, err)
	}
	server.AccessRules = []oauthdomain.ResourceAccessRule{{
		ResourceType: "idmagic_mcp_operation", Relation: "invoke", Scopes: []string{"tools:read"},
		Tool: &oauthdomain.ResourceAccessTool{ID: "operation-read", Name: "read_document", DisplayName: "閲覧", ResourceIDArgument: "document_id"},
	}}
	if err := f.s.McpResourceServers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	operation := server.ID + "/operation-read"
	tuple := func(resourceID, relation string) map[string]any {
		return map[string]any{"resource_type": "idmagic_mcp_operation", "resource_id": resourceID, "relation": relation, "subject_type": "agent", "subject_id": "A1"}
	}
	change := func(key string, tuples ...map[string]any) {
		t.Helper()
		items := make([]any, len(tuples))
		for index, item := range tuples {
			items[index] = item
		}
		f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{key: items})
	}
	var token string
	expect := func(documentID string, want int) {
		t.Helper()
		before := f.effects.Load()
		response, body := f.postDocument(t, "read_document", documentID, gatewayBearer(token))
		executed := f.effects.Load() > before
		if response.StatusCode != want || executed != (want == http.StatusOK) {
			t.Fatalf("document=%s status=%d executed=%v body=%s, want %d", documentID, response.StatusCode, executed, body, want)
		}
	}

	// 操作割当だけで範囲がなければ、従来の document の関係があってもどの対象も許可しない。
	change("writes", tuple(operation, "invoke"))
	token = f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	_ = f.connect(t, token)
	expect("d1", http.StatusForbidden)

	// 指定した対象だけを許可する。
	change("writes", tuple(operation+"/d1", "invoke"))
	expect("d1", http.StatusOK)
	expect("d2", http.StatusForbidden)

	// すべての対象へ広げ、取り消すと指定した対象だけに戻る。
	change("writes", tuple(operation, "invoke_any_target"))
	expect("d2", http.StatusOK)
	change("deletes", tuple(operation, "invoke_any_target"))
	expect("d2", http.StatusForbidden)
	expect("d1", http.StatusOK)

	// 対象の割当だけでは、操作割当の取消を補わない。
	change("deletes", tuple(operation, "invoke"))
	expect("d1", http.StatusForbidden)
}
