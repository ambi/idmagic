package server_http_test

import (
	"net/http"
	"net/url"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

//spec:covers REQ-OAUTH2-083, REQ-OAUTH2-084, REQ-OAUTH2-085, REQ-AUTHORIZATION-016, REQ-AUTHORIZATION-017: 正式な登録からモデル発行なしで実行し、未割当の同一scope操作と取消後の旧トークンを拒否する。
func TestAgentOperationCatalogPublicSetup(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAccessControl())
	f.request(t, http.MethodGet, "/api/admin/v1/authorization/model", nil, f.admin, http.StatusNotFound)
	owner := f.user(t, "catalog-owner")
	client, key := f.keyClient(t, "openid read write", "client_credentials")
	agent := f.agent(t, owner, client, "autonomous")
	evaluator, evaluatorKey := f.keyClient(t, "read", "client_credentials")
	rule := func(name, scope string) map[string]any {
		return map[string]any{"resource_type": "idmagic_mcp_operation", "relation": "invoke", "scopes": []string{scope}, "tool": map[string]any{"name": name, "display_name": name}}
	}
	server := f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers", map[string]any{"name": "Documents", "resource": setupResource, "scopes": []string{"read", "write"}, "client_id": evaluator, "access_rules": []any{rule("read_document", "read"), rule("delete_document", "read"), rule("write_document", "write")}}, f.admin, http.StatusCreated)
	tool := server["access_rules"].([]any)[0].(map[string]any)["tool"].(map[string]any)
	assignment := map[string]any{"resource_type": "idmagic_mcp_operation", "resource_id": setupString(t, tool, "assignment_resource_id"), "relation": "invoke", "subject_type": "agent", "subject_id": agent}
	f.request(t, http.MethodPost, "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []any{assignment}}, f.admin, http.StatusOK)
	issue := func() map[string]any {
		return f.signedForm(t, key, client, "/token", url.Values{"grant_type": {"client_credentials"}, "resource": {setupResource}}, http.StatusOK)
	}
	issued := issue()
	if issued["scope"] != "read" {
		t.Fatalf("scope=%v", issued["scope"])
	}
	token := setupString(t, issued, "access_token")
	evaluate := func(tool string, want bool) {
		t.Helper()
		result := f.signedForm(t, evaluatorKey, evaluator, "/resource-access/evaluate", url.Values{"token": {token}, "resource_type": {"idmagic_mcp_operation"}, "relation": {"invoke"}, "resource_id": {tool}}, http.StatusOK)
		if result["decision"] != want {
			t.Fatalf("tool=%s result=%v want=%v", tool, result, want)
		}
	}
	evaluate("read_document", true)
	evaluate("delete_document", false)
	evaluate("unknown", false)
	denied := f.signedForm(t, key, client, "/token", url.Values{"grant_type": {"client_credentials"}, "resource": {setupResource}, "scope": {"write"}}, http.StatusBadRequest)
	if denied["error"] != "invalid_scope" {
		t.Fatalf("unassigned scope: %v", denied)
	}
	f.request(t, http.MethodPost, "/api/admin/v1/authorization/relation-tuples", map[string]any{"deletes": []any{assignment}}, f.admin, http.StatusOK)
	evaluate("read_document", false)
	f.signedForm(t, key, client, "/token", url.Values{"grant_type": {"client_credentials"}, "resource": {setupResource}}, http.StatusBadRequest)
	// 同じツール名を再登録しても、以前の利用割当は復活しない。
	f.request(t, http.MethodPost, "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []any{assignment}}, f.admin, http.StatusOK)
	evaluate("read_document", true)
	path := "/api/admin/v1/mcp-resource-servers/" + setupString(t, server, "id")
	f.request(t, http.MethodPatch, path, map[string]any{"access_rules": []any{rule("delete_document", "read")}}, f.admin, http.StatusOK)
	evaluate("read_document", false)
	recreatedRule := rule("read_document", "read")
	recreatedRule["tool"].(map[string]any)["id"] = tool["id"]
	recreated := f.request(t, http.MethodPatch, path, map[string]any{"access_rules": []any{recreatedRule, rule("delete_document", "read")}}, f.admin, http.StatusOK)
	newTool := recreated["access_rules"].([]any)[0].(map[string]any)["tool"].(map[string]any)
	if newTool["assignment_resource_id"] == tool["assignment_resource_id"] {
		t.Fatal("removed operation reference was reused")
	}
	evaluate("read_document", false)
	f.request(t, http.MethodGet, "/api/admin/v1/authorization/model", nil, f.admin, http.StatusNotFound)
}
