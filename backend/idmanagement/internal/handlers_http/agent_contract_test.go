package handlers_http_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func idmContractAgent(t *testing.T, body map[string]any, name string) string {
	t.Helper()
	id, ok := body["id"].(string)
	if !ok || id == "" || body["tenant_id"] != tenancydomain.DefaultTenantID || body["name"] != name || body["kind"] != string(idmdomain.AgentKindAutonomous) || body["owner_user_id"] != idmRefusalAdmin {
		t.Fatalf("agent=%v", body)
	}
	if !reflect.DeepEqual(body["roles"], []any{"support"}) {
		t.Fatalf("agent roles=%v", body["roles"])
	}
	if _, ok := body["client_ids"].([]any); !ok {
		t.Fatalf("agent client_ids=%v", body["client_ids"])
	}
	idmContractTime(t, body, "created_at")
	idmContractTime(t, body, "updated_at")
	return id
}

//verify:operation RegisterAgent: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation GetAgent: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation UpdateAgent: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation DeleteAgent: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation BindAgentCredential: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation UnbindAgentCredential: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation DisableAgent: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation EnableAgent: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
//verify:operation KillAgent: Agent と資格情報の管理操作の HTTP 応答、保存状態および停止
func TestAgentManagementHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	session := f.seedSession(t, "agent-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	send := func(method, path string, body any) *httptest.ResponseRecorder {
		return f.send(t, idmRefusalRequest{method: method, path: path, body: body, sessionID: session, csrf: idmRefusalCSRF})
	}
	created := idmContractJSON(t, send(http.MethodPost, "/api/admin/v1/agents", map[string]any{
		"name": "Contract Agent", "description": "Contract description", "kind": "autonomous", "roles": []string{"support"},
	}), http.StatusCreated)
	id := idmContractAgent(t, created, "Contract Agent")
	if created["status"] != string(idmdomain.AgentStatusActive) || created["description"] != "Contract description" || len(created["client_ids"].([]any)) != 0 {
		t.Fatalf("new agent=%v", created)
	}
	path := "/api/admin/v1/agents/" + id
	getAgent := func() map[string]any {
		return idmContractJSON(t, send(http.MethodGet, path, nil), http.StatusOK)
	}
	if read := getAgent(); !reflect.DeepEqual(read, created) {
		t.Fatalf("agent was not saved: created=%v read=%v", created, read)
	}
	updated := idmContractJSON(t, send(http.MethodPatch, path, map[string]any{"name": "Renamed Agent"}), http.StatusOK)
	if got := idmContractAgent(t, updated, "Renamed Agent"); got != id || updated["created_at"] != created["created_at"] {
		t.Fatalf("updated agent=%v", updated)
	}
	if read := getAgent(); !reflect.DeepEqual(read, updated) {
		t.Fatalf("updated agent was not saved: updated=%v read=%v", updated, read)
	}
	idmContractEmpty(t, send(http.MethodPost, path+"/credentials", map[string]any{"client_id": idmRefusalClient}))
	if clients := getAgent()["client_ids"]; !reflect.DeepEqual(clients, []any{idmRefusalClient}) {
		t.Fatalf("bound client_ids=%v", clients)
	}
	idmContractEmpty(t, send(http.MethodDelete, path+"/credentials/"+idmRefusalClient, nil))
	if clients := getAgent()["client_ids"]; !reflect.DeepEqual(clients, []any{}) {
		t.Fatalf("unbound client_ids=%v", clients)
	}
	idmContractEmpty(t, send(http.MethodPost, path+"/disable", nil))
	disabled := getAgent()
	if disabled["status"] != string(idmdomain.AgentStatusDisabled) {
		t.Fatalf("disabled agent=%v", disabled)
	}
	idmContractTime(t, disabled, "disabled_at")
	idmContractEmpty(t, send(http.MethodPost, path+"/enable", nil))
	if enabled := getAgent(); enabled["status"] != string(idmdomain.AgentStatusActive) || enabled["disabled_at"] != nil {
		t.Fatalf("enabled agent=%v", enabled)
	}
	idmContractEmpty(t, send(http.MethodDelete, path, nil))
	idmContractProblem(t, send(http.MethodGet, path, nil), http.StatusNotFound, "agent_not_found")
	second := idmContractJSON(t, send(http.MethodPost, "/api/admin/v1/agents", map[string]any{
		"name": "Stopped Agent", "kind": "autonomous", "roles": []string{"support"},
	}), http.StatusCreated)
	path = "/api/admin/v1/agents/" + idmContractAgent(t, second, "Stopped Agent")
	idmContractEmpty(t, send(http.MethodPost, path+"/kill", nil))
	killed := getAgent()
	if killed["status"] != string(idmdomain.AgentStatusKilled) {
		t.Fatalf("killed agent=%v", killed)
	}
	idmContractTime(t, killed, "killed_at")
	idmContractProblem(t, send(http.MethodPost, path+"/enable", nil), http.StatusConflict, "agent_killed")
	idmContractProblem(t, send(http.MethodDelete, path, nil), http.StatusConflict, "agent_killed")
}

//verify:operation RegisterAgent: 非管理者への 403 Problem Details
//verify:operation GetAgent: 非管理者への 403 Problem Details
//verify:operation UpdateAgent: 非管理者への 403 Problem Details
//verify:operation DeleteAgent: 非管理者への 403 Problem Details
//verify:operation BindAgentCredential: 非管理者への 403 Problem Details
//verify:operation UnbindAgentCredential: 非管理者への 403 Problem Details
//verify:operation DisableAgent: 非管理者への 403 Problem Details
//verify:operation EnableAgent: 非管理者への 403 Problem Details
//verify:operation KillAgent: 非管理者への 403 Problem Details
func TestAgentManagementHTTPContractsRefuseNonAdmin(t *testing.T) {
	f := newIdmRefusalServer(t)
	session := f.seedSession(t, "agent-contract-denied", tenancydomain.DefaultTenantID, idmRefusalBob)
	path := "/api/admin/v1/agents/" + idmRefusalAgent
	before := *f.agent(t, idmRefusalAgent)
	for _, test := range []struct{ method, path string }{
		{http.MethodPost, "/api/admin/v1/agents"},
		{http.MethodGet, path},
		{http.MethodPatch, path},
		{http.MethodDelete, path},
		{http.MethodPost, path + "/credentials"},
		{http.MethodDelete, path + "/credentials/" + idmRefusalClient},
		{http.MethodPost, path + "/disable"},
		{http.MethodPost, path + "/enable"},
		{http.MethodPost, path + "/kill"},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			response := f.send(t, idmRefusalRequest{method: test.method, path: test.path, sessionID: session, csrf: idmRefusalCSRF, body: map[string]any{}})
			idmContractProblem(t, response, http.StatusForbidden, "access_denied")
		})
	}
	after := *f.agent(t, idmRefusalAgent)
	if !reflect.DeepEqual(after, before) || len(*f.events) != 0 {
		t.Fatalf("refused requests changed the agent: before=%+v after=%+v", before, after)
	}
}
