package server_http_test

// ガードレールを管理 API で保存し、正式な /token を通して発行の結果を確かめる。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	ratememory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

func (f *agentRevocationFixture) putGuardrail(t *testing.T, agentID string, body map[string]any) (int, map[string]any) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, stack.Issuer+"/realms/default/api/admin/v1/agents/"+agentID+"/guardrail-policy", bytes.NewReader(raw))
	request.Header.Set("Authorization", "Bearer "+f.adminToken)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	response := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	return recorder.Code, response
}

func newGuardrailE2EFixture(t *testing.T) *agentRevocationFixture {
	t.Helper()
	limiter := ratememory.NewRateLimiter(rlports.RateLimitConfigs{"token": {MaxRequests: 1000, WindowSeconds: 60}})
	return newAgentRevocationFixtureWith(t, []stack.Option{stack.WithRateLimiter(limiter)}, "guarded-agent")
}

//spec:covers REQ-OAUTH2-073, REQ-OAUTH2-074: 保存したポリシーが正式な /token の発行を止める
func TestAgentGuardrailLimitsIssuanceThroughToken(t *testing.T) {
	f := newGuardrailE2EFixture(t)
	clientID := agentClientID("guarded-agent")
	if status, body := f.putGuardrail(t, "guarded-agent", map[string]any{
		"expected_version":      "",
		"issuance_rate":         map[string]any{"max_issuances": 1, "window_seconds": 3600},
		"max_token_ttl_seconds": 60,
	}); status != http.StatusOK {
		t.Fatalf("save status=%d body=%v", status, body)
	}
	form := url.Values{"grant_type": {"client_credentials"}, "scope": {"tools:read"}}
	status, body := f.post(t, "token", clientID, form)
	if status != http.StatusOK || body["expires_in"] != float64(60) {
		t.Fatalf("first status=%d body=%v", status, body)
	}
	status, body = f.post(t, "token", clientID, form)
	if status != http.StatusBadRequest || body["error"] != "access_denied" || body["access_token"] != nil {
		t.Fatalf("second status=%d body=%v", status, body)
	}
}

//
//spec:covers REQ-OAUTH2-077, EX-OAUTH2-077-01, EX-OAUTH2-077-02: 古いバージョンと別のテナントの Agent の保存を拒否する
//verify:operation GetAdminAgentGuardrailPolicy: 二層の取得と 404
//verify:operation PutAdminAgentGuardrailPolicy: 保存の応答とバージョン、400、404、409
//verify:operation DeleteAdminAgentGuardrailPolicy: 削除の 204 と取得への反映
func TestAdminAgentGuardrailPolicy(t *testing.T) {
	f := newGuardrailE2EFixture(t)
	status, saved := f.putGuardrail(t, "guarded-agent", map[string]any{"expected_version": "", "require_approval": true})
	if status != http.StatusOK || saved["version"] == "" {
		t.Fatalf("create status=%d body=%v", status, saved)
	}
	if status, body := f.putGuardrail(t, "guarded-agent", map[string]any{"expected_version": "", "require_approval": false}); status != http.StatusConflict {
		t.Fatalf("stale status=%d body=%v", status, body)
	}
	if status, body := f.putGuardrail(t, "guarded-agent", map[string]any{"expected_version": saved["version"], "max_token_ttl_seconds": 0}); status != http.StatusBadRequest {
		t.Fatalf("invalid status=%d body=%v", status, body)
	}
	if status, body := f.putGuardrail(t, "guarded-agent", map[string]any{"expected_version": saved["version"]}); status != http.StatusOK {
		t.Fatalf("update status=%d body=%v", status, body)
	}
	if status, body := f.getGuardrail(t, "/api/admin/v1/agents/guarded-agent/guardrail-policy"); status != http.StatusOK || body["agent"] == nil || body["tenant_default"] != nil {
		t.Fatalf("get status=%d body=%v", status, body)
	}
	f.admin(t, http.MethodDelete, "/api/admin/v1/agents/guarded-agent/guardrail-policy", nil)
	if status, body := f.getGuardrail(t, "/api/admin/v1/agents/guarded-agent/guardrail-policy"); status != http.StatusOK || body["agent"] != nil {
		t.Fatalf("get after delete status=%d body=%v", status, body)
	}
	now := time.Now().UTC()
	if err := f.s.Agents.Save(context.Background(), &agentdomain.Agent{ID: "other-agent", TenantID: stack.OtherRealm, Name: "other", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: stack.UserID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if status, body := f.putGuardrail(t, "other-agent", map[string]any{"expected_version": ""}); status != http.StatusNotFound {
		t.Fatalf("other tenant status=%d body=%v", status, body)
	}
	if status, body := f.getGuardrail(t, "/api/admin/v1/agents/other-agent/guardrail-policy"); status != http.StatusNotFound {
		t.Fatalf("other tenant get status=%d body=%v", status, body)
	}
}

//
//spec:covers REQ-OAUTH2-077: agents:read では保存できず、テナントのデフォルトを保存と削除で切り替えられる
//verify:operation GetAdminAgentGuardrailDefault: 既定の取得
//verify:operation PutAdminAgentGuardrailDefault: 既定の保存と 403
//verify:operation DeleteAdminAgentGuardrailDefault: 既定の削除の 204
func TestAdminAgentGuardrailScopesAndTenantDefault(t *testing.T) {
	f := newGuardrailE2EFixture(t)
	admin := f.adminToken
	f.adminToken, _ = f.s.IssueApiToken(t, "default", apitokendomain.ScopeAgentsRead)
	if status, body := f.putGuardrail(t, "guarded-agent", map[string]any{"expected_version": ""}); status != http.StatusForbidden {
		t.Fatalf("read-only scope status=%d body=%v", status, body)
	}
	f.adminToken = admin
	f.admin(t, http.MethodPut, "/api/admin/v1/agent-guardrail-default", map[string]any{"expected_version": "", "max_token_ttl_seconds": 30})
	if status, body := f.getGuardrail(t, "/api/admin/v1/agent-guardrail-default"); status != http.StatusOK || body["tenant_default"] == nil {
		t.Fatalf("get default status=%d body=%v", status, body)
	}
	status, body := f.post(t, "token", agentClientID("guarded-agent"), url.Values{"grant_type": {"client_credentials"}, "scope": {"tools:read"}})
	if status != http.StatusOK || body["expires_in"] != float64(30) {
		t.Fatalf("tenant default status=%d body=%v", status, body)
	}
	f.admin(t, http.MethodDelete, "/api/admin/v1/agent-guardrail-default", nil)
	status, body = f.post(t, "token", agentClientID("guarded-agent"), url.Values{"grant_type": {"client_credentials"}, "scope": {"tools:read"}})
	if status != http.StatusOK || body["expires_in"] == float64(30) {
		t.Fatalf("after delete status=%d body=%v", status, body)
	}
}

func (f *agentRevocationFixture) getGuardrail(t *testing.T, path string) (int, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, stack.Issuer+"/realms/default"+path, http.NoBody)
	request.Header.Set("Authorization", "Bearer "+f.adminToken)
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	response := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &response)
	return recorder.Code, response
}
