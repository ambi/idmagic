package server_http_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-IDMANAGEMENT-106: 公開 API で作った束縛と付与の変更、正式な発行、別テナントと不在の拒否を観測する。
//verify:operation GetAgentAccessSummary: 200 の本文、401、403、404 の Problem Details を照合する。
func TestAgentAccessSummaryReflectsChangesAndTenantBoundary(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithAudit(), stack.WithApiTokens())
	client, key := f.keyClient(t, "read write", "client_credentials")
	agent := f.agent(t, stack.UserID, client, "autonomous")
	path := "/api/admin/v1/agents/" + agent + "/access-summary"
	read := func() map[string]any {
		return f.request(t, http.MethodGet, path, nil, f.admin, http.StatusOK)
	}
	initial := read()
	if initial["last_token_issued_at"] != nil || len(initial["delegations"].([]any)) != 0 {
		t.Fatalf("unissued agent=%v", initial)
	}
	clients := initial["clients"].([]any)
	if len(clients) != 1 || clients[0].(map[string]any)["client_id"] != client || len(clients[0].(map[string]any)["scopes"].([]any)) != 2 {
		t.Fatalf("bound scopes=%v", clients)
	}
	f.request(t, http.MethodPost, "/api/account/v1/agent-delegations", map[string]any{
		"agent_id": agent, "scopes": []string{"read"},
	}, f.admin, http.StatusCreated)
	f.signedForm(t, key, client, "/token", url.Values{"grant_type": {"client_credentials"}, "scope": {"read"}}, http.StatusOK)
	current := read()
	grants := current["delegations"].([]any)
	if len(grants) != 1 || grants[0].(map[string]any)["user_id"] != stack.AdminUserID || grants[0].(map[string]any)["active"] != true || current["last_token_issued_at"] == nil || current["as_of"] == nil {
		t.Fatalf("issued and delegated agent=%v", current)
	}
	f.request(t, http.MethodPost, "/api/account/v1/agent-delegations/"+agent+"/revoke", nil, f.admin, http.StatusNoContent)
	f.request(t, http.MethodPatch, "/api/admin/v1/clients/"+client, map[string]any{"scope": "write"}, f.admin, http.StatusOK)
	current = read()
	grant := current["delegations"].([]any)[0].(map[string]any)
	if grant["active"] != false || grant["state"] != "revoked" || grant["revoked_at"] == nil {
		t.Fatalf("revoked delegation=%v", grant)
	}
	if scopes := current["clients"].([]any)[0].(map[string]any)["scopes"].([]any); len(scopes) != 1 || scopes[0] != "write" {
		t.Fatalf("updated client scopes=%v", scopes)
	}
	other, err := f.s.Agents.FindByID(context.Background(), tenancydomain.DefaultTenantID, agent)
	if err != nil || other == nil {
		t.Fatalf("agent=%v error=%v", other, err)
	}
	foreign := *other
	foreign.ID, foreign.TenantID = "foreign-agent", stack.OtherRealm
	if err := f.s.Agents.Save(context.Background(), &foreign); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"foreign-agent", "absent-agent"} {
		body := f.request(t, http.MethodGet, "/api/admin/v1/agents/"+id+"/access-summary", nil, f.admin, http.StatusNotFound)
		if body["type"] != "urn:idmagic:error:agent_not_found" {
			t.Fatalf("missing agent problem=%v", body)
		}
	}
	if body := f.request(t, http.MethodGet, path, nil, nil, http.StatusUnauthorized); body["type"] != "urn:idmagic:error:authentication_required" {
		t.Fatalf("unauthenticated problem=%v", body)
	}
	f.user(t, "access-viewer")
	browserClient, _ := f.keyClient(t, "openid", "authorization_code")
	_, ordinary := f.login(t, "access-viewer", browserClient, "openid")
	if body := f.request(t, http.MethodGet, path, nil, ordinary, http.StatusForbidden); body["type"] != "urn:idmagic:error:access_denied" {
		t.Fatalf("forbidden problem=%v", body)
	}
	reader, _ := f.s.IssueApiToken(t, "default", apitokendomain.ScopeAgentsRead)
	if body := f.request(t, http.MethodGet, path, nil, nil, http.StatusOK, reader); len(body["clients"].([]any)) != 1 {
		t.Fatalf("scoped API token materials=%v", body)
	}
	usersOnly, _ := f.s.IssueApiToken(t, "default", apitokendomain.ScopeUsersRead)
	if body := f.request(t, http.MethodGet, path, nil, nil, http.StatusForbidden, usersOnly); body["type"] != "urn:idmagic:error:insufficient_scope" {
		t.Fatalf("insufficient scope problem=%v", body)
	}
}
