package handlers_http_test

// REQ-OAUTH2-050 を HTTP 経路で確かめる (wi-376 T003)。client_credentials は人間の
// 承認を記録しないため、Supervised な Agent へはトークンを発行しない。判定そのものは
// usecase 層の TestAgentRequiresHumanApproval が持ち、ここは /token がその判定を
// 通していることだけを見る。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestTokenClientCredentials_supervisedAgent_rejected(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	f := newTokenServer(t)
	if err := f.userRepo.Save(ctx, &userdomain.User{
		ID: "owner-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "owner-1",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive, StatusChangedAt: &now},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	saveAgent := func(t *testing.T, kind idmdomain.AgentKind) {
		t.Helper()
		if err := f.agentRepo.Save(ctx, &agentdomain.Agent{
			ID: "agent-1", TenantID: tenancydomain.DefaultTenantID, Name: "agent-1",
			Kind: kind, OwnerUserID: "owner-1", Status: idmdomain.AgentStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed agent: %v", err)
		}
	}
	saveAgent(t, idmdomain.AgentKindAutonomous)
	if _, err := f.agentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
		AgentID: "agent-1", ClientID: "client-conf", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	requestToken := func(t *testing.T) (int, map[string]any) {
		t.Helper()
		form := url.Values{
			"client_id": {"client-conf"}, "client_secret": {"secret-conf"},
			"grant_type": {"client_credentials"}, "scope": {"openid"},
		}
		req := httptest.NewRequest(http.MethodPost, "/realms/default/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		f.e.ServeHTTP(rec, req)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body
	}

	if code, body := requestToken(t); code != http.StatusOK || body["access_token"] == nil {
		t.Fatalf("autonomous agent: status=%d body=%v", code, body)
	}

	saveAgent(t, idmdomain.AgentKindSupervised)
	code, body := requestToken(t)
	if code == http.StatusOK {
		t.Fatalf("expected a supervised agent to need human approval, got %d %v", code, body)
	}
	if body["error"] != "unauthorized_client" {
		t.Fatalf("error = %v, want unauthorized_client", body["error"])
	}

	// 区分が既知の値でなければ承認が必要な側へ倒す。
	saveAgent(t, idmdomain.AgentKind("mystery"))
	if code, body := requestToken(t); code == http.StatusOK || body["error"] != "unauthorized_client" {
		t.Fatalf("expected an unknown kind to fail closed, got %d %v", code, body)
	}
}
