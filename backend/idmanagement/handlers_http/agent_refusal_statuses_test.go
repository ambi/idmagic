package handlers_http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// agentRefusalCase は Agent 管理 API の 1 本の要求と、TypeSpec が宣言する拒否の形である。
type agentRefusalCase struct {
	name   string
	method string
	path   string
	body   any
	status int
	code   string
}

// agentOperationsOn は agentID を対象にする Agent 管理 API の要求を、宣言する拒否の形とともに並べる。
func agentOperationsOn(agentID string, status int, code string) []agentRefusalCase {
	base := "/api/admin/v1/agents/" + agentID
	return []agentRefusalCase{
		{name: "UpdateAgent", method: http.MethodPatch, path: base, body: map[string]any{"description": "x"}, status: status, code: code},
		{name: "DisableAgent", method: http.MethodPost, path: base + "/disable", status: status, code: code},
		{name: "EnableAgent", method: http.MethodPost, path: base + "/enable", status: status, code: code},
		{name: "KillAgent", method: http.MethodPost, path: base + "/kill", status: status, code: code},
		{name: "DeleteAgent", method: http.MethodDelete, path: base, status: status, code: code},
		{name: "BindAgentCredential", method: http.MethodPost, path: base + "/credentials", body: map[string]any{"client_id": idmRefusalClient}, status: status, code: code},
	}
}

func (f *idmRefusalFixture) expectAgentRefusal(t *testing.T, tenantID, sessionID string, tc agentRefusalCase) {
	t.Helper()
	response := f.send(t, idmRefusalRequest{
		method: tc.method, tenantID: tenantID, path: tc.path, body: tc.body,
		sessionID: sessionID, csrf: idmRefusalCSRF,
	})
	if response.Code != tc.status || idmProblemCode(t, response) != tc.code {
		t.Fatalf("%s: status=%d body=%s, want %d %s", tc.name, response.Code, response.Body.String(), tc.status, tc.code)
	}
}

// Agent 管理 API の拒否を、TypeSpec が宣言する status と type で書くこと。
// `mise run check-status-drift` は writeAdminAgentError と changeAgentStatus の中を追跡しないため、
// ユースケースのエラーから status への写像はこの境界で固定する。
//
//spec:covers REQ-IDMANAGEMENT-009: Agent の管理操作の拒否が、TypeSpec の宣言する 401、404、409 と type で返ること。
func TestAgentRefusalsUseDeclaredStatuses(t *testing.T) {
	t.Run("存在しない Agent は 404 agent_not_found で、別テナントの Agent も同じ応答になる", func(t *testing.T) {
		fixture := newIdmRefusalServer(t)
		admin := fixture.seedSession(t, "sess-admin-missing-agent", tenancydomain.DefaultTenantID, idmRefusalAdmin)
		foreignAdmin := fixture.seedSession(t, "sess-admin-acme-missing-agent", idmRefusalOtherTenant, idmRefusalForeignAdmin)
		missing := append(agentOperationsOn("agent-that-exists-nowhere", http.StatusNotFound, "agent_not_found"),
			agentRefusalCase{name: "GetAgent", method: http.MethodGet, path: "/api/admin/v1/agents/agent-that-exists-nowhere", status: http.StatusNotFound, code: "agent_not_found"},
			agentRefusalCase{name: "UnbindAgentCredential", method: http.MethodDelete, path: "/api/admin/v1/agents/agent-that-exists-nowhere/credentials/" + idmRefusalClient, status: http.StatusNotFound, code: "agent_not_found"},
		)
		for _, tc := range missing {
			fixture.expectAgentRefusal(t, tenancydomain.DefaultTenantID, admin, tc)
		}
		// acme の管理者から default の Agent を指すと、存在しない Agent と区別できないこと。
		foreign := append(agentOperationsOn(idmRefusalAgent, http.StatusNotFound, "agent_not_found"),
			agentRefusalCase{name: "GetAgent", method: http.MethodGet, path: "/api/admin/v1/agents/" + idmRefusalAgent, status: http.StatusNotFound, code: "agent_not_found"},
		)
		for _, tc := range foreign {
			fixture.expectAgentRefusal(t, idmRefusalOtherTenant, foreignAdmin, tc)
		}
	})

	t.Run("停止済みの Agent への変更は 409 agent_killed", func(t *testing.T) {
		fixture := newIdmRefusalServer(t)
		admin := fixture.seedSession(t, "sess-admin-killed-agent", tenancydomain.DefaultTenantID, idmRefusalAdmin)
		now := time.Now().UTC()
		if err := fixture.agents.Save(context.Background(), &agentdomain.Agent{
			ID: "agent-killed", TenantID: tenancydomain.DefaultTenantID, Name: "killed-agent",
			Kind: idmdomain.AgentKindAutonomous, OwnerUserID: idmRefusalAdmin,
			Status: idmdomain.AgentStatusKilled, KilledAt: &now, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		for _, tc := range agentOperationsOn("agent-killed", http.StatusConflict, "agent_killed") {
			fixture.expectAgentRefusal(t, tenancydomain.DefaultTenantID, admin, tc)
		}
	})

	t.Run("別の Agent に束縛済みのクライアントのバインドは 409 agent_client_already_bound", func(t *testing.T) {
		fixture := newIdmRefusalServer(t)
		admin := fixture.seedSession(t, "sess-admin-bound-client", tenancydomain.DefaultTenantID, idmRefusalAdmin)
		now := time.Now().UTC()
		if err := fixture.agents.Save(context.Background(), &agentdomain.Agent{
			ID: "agent-second", TenantID: tenancydomain.DefaultTenantID, Name: "second-agent",
			Kind: idmdomain.AgentKindAutonomous, OwnerUserID: idmRefusalAdmin,
			Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
		if response := fixture.send(t, idmRefusalRequest{
			method: http.MethodPost, path: "/api/admin/v1/agents/" + idmRefusalAgent + "/credentials",
			body: map[string]any{"client_id": idmRefusalClient}, sessionID: admin, csrf: idmRefusalCSRF,
		}); response.Code != http.StatusNoContent {
			t.Fatalf("前提が壊れている: 最初のバインドが status=%d body=%s", response.Code, response.Body.String())
		}
		fixture.expectAgentRefusal(t, tenancydomain.DefaultTenantID, admin, agentRefusalCase{
			name: "BindAgentCredential", method: http.MethodPost,
			path:   "/api/admin/v1/agents/agent-second/credentials",
			body:   map[string]any{"client_id": idmRefusalClient},
			status: http.StatusConflict, code: "agent_client_already_bound",
		})
	})

	t.Run("セッションの無い状態変更は 401 authentication_required", func(t *testing.T) {
		fixture := newIdmRefusalServer(t)
		for _, tc := range agentOperationsOn(idmRefusalAgent, http.StatusUnauthorized, "authentication_required") {
			fixture.expectAgentRefusal(t, tenancydomain.DefaultTenantID, "", tc)
		}
	})
}
