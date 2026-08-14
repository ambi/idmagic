package handlers_http_test

// REQ-OAUTH2-046 を HTTP 経路で確かめる (wi-324 T002/T005)。所有者がオフボードされた
// Agent は client_credentials で新しいトークンを取得できず、所有者が戻れば再開する。
// usecase 層の TestResolveIssuableAgent_OwnerOffboarding が判定そのものを持ち、
// ここは /token がその判定を通していることだけを見る。

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

func TestTokenClientCredentials_offboardedOwner_rejected(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	f := newTokenServer(t)
	agentRepo, userRepo := f.agentRepo, f.userRepo

	if err := agentRepo.Save(ctx, &agentdomain.Agent{
		ID: "agent-1", TenantID: tenancydomain.DefaultTenantID, Name: "agent-1",
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner-1", Status: idmdomain.AgentStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if _, err := agentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
		AgentID: "agent-1", ClientID: "client-conf", CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}
	saveOwner := func(t *testing.T, status idmdomain.UserStatus) {
		t.Helper()
		if err := userRepo.Save(ctx, &userdomain.User{
			ID: "owner-1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "owner-1",
			Lifecycle: userdomain.UserLifecycle{Status: status, StatusChangedAt: &now},
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed owner: %v", err)
		}
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

	saveOwner(t, idmdomain.UserStatusActive)
	if code, body := requestToken(t); code != http.StatusOK || body["access_token"] == nil {
		t.Fatalf("active owner: status=%d body=%v", code, body)
	}

	saveOwner(t, idmdomain.UserStatusDisabled)
	code, body := requestToken(t)
	if code == http.StatusOK {
		t.Fatalf("expected an offboarded owner to stop issuance, got %d %v", code, body)
	}
	if body["error"] != "invalid_client" {
		t.Fatalf("error = %v, want invalid_client", body["error"])
	}
	agent, err := agentRepo.FindByID(ctx, tenancydomain.DefaultTenantID, "agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if agent.Status != idmdomain.AgentStatusActive {
		t.Fatalf("expected the agent to stay Active, got %q", agent.Status)
	}

	// 所有者が戻れば発行も自動的に再開する (Agent を re-enable する必要はない)。
	saveOwner(t, idmdomain.UserStatusActive)
	if code, body := requestToken(t); code != http.StatusOK || body["access_token"] == nil {
		t.Fatalf("reinstated owner: status=%d body=%v", code, body)
	}
}
