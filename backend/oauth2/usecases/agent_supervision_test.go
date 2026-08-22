package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// TestAgentRequiresHumanApproval — RED: REQ-OAUTH2-050
// (spec/contexts/oauth2/scenarios.md)。承認を記録しない発行経路を通ってよいのは
// autonomous と確認できた Agent だけで、判定は区分の否定形で行う。
func TestAgentRequiresHumanApproval(t *testing.T) {
	cases := []struct {
		name string
		kind idmdomain.AgentKind
		want bool
	}{
		{"Autonomous", idmdomain.AgentKindAutonomous, false},
		{"Supervised", idmdomain.AgentKindSupervised, true},
		{"UnknownKindFailsClosed", idmdomain.AgentKind("mystery"), true},
		{"EmptyKindFailsClosed", idmdomain.AgentKind(""), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &agentdomain.Agent{ID: "agent_1", Kind: tc.kind}
			if got := sharedusecases.AgentRequiresHumanApproval(agent); got != tc.want {
				t.Fatalf("kind %q: expected %v, got %v", tc.kind, tc.want, got)
			}
		})
	}

	t.Run("NoAgentIsUnaffected", func(t *testing.T) {
		if sharedusecases.AgentRequiresHumanApproval(nil) {
			t.Fatal("expected a client with no bound agent to need no approval")
		}
	})
}

// TestResolveIssuableAgentWithoutApproval — RED: REQ-OAUTH2-050。承認を記録しない
// 発行経路は Supervised な Agent を unauthorized_client で拒否し、判断の根拠とした
// 区分を AgentApprovalRequired へ残す。
func TestResolveIssuableAgentWithoutApproval(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	now := time.Now().UTC()

	seed := func(t *testing.T, kind idmdomain.AgentKind) (sharedusecases.AgentIssuanceDeps, *[]spec.DomainEvent) {
		t.Helper()
		agentRepo := agentmemory.NewAgentRepository()
		if err := agentRepo.Save(context.Background(), &agentdomain.Agent{
			ID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Name: "agent_1",
			Kind: kind, OwnerUserID: "owner_1", Status: idmdomain.AgentStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed agent: %v", err)
		}
		if _, err := agentRepo.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
			AgentID: "agent_1", ClientID: "agent_client", CreatedAt: now,
		}); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
		userRepo := usermemory.NewUserRepository()
		if err := userRepo.Save(context.Background(), &userdomain.User{
			ID: "owner_1", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "owner_1",
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive, StatusChangedAt: &now},
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("seed owner: %v", err)
		}
		events := &[]spec.DomainEvent{}
		return sharedusecases.AgentIssuanceDeps{
			AgentRepo: agentRepo, UserRepo: userRepo,
			Emit: func(e spec.DomainEvent) { *events = append(*events, e) },
		}, events
	}

	t.Run("SupervisedRejected", func(t *testing.T) {
		deps, events := seed(t, idmdomain.AgentKindSupervised)
		_, err := sharedusecases.ResolveIssuableAgentWithoutApproval(ctx, deps, "agent_client", "client_credentials", now)
		var oauthErr *sharedusecases.OAuthError
		if !errors.As(err, &oauthErr) || oauthErr.Code != "unauthorized_client" {
			t.Fatalf("expected unauthorized_client, got %v", err)
		}
		if len(*events) != 1 {
			t.Fatalf("expected one audit event, got %d", len(*events))
		}
		required, ok := (*events)[0].(*domain.AgentApprovalRequired)
		if !ok {
			t.Fatalf("expected AgentApprovalRequired, got %T", (*events)[0])
		}
		if required.AgentID != "agent_1" || required.ClientID != "agent_client" ||
			required.Kind != string(idmdomain.AgentKindSupervised) || required.GrantType != "client_credentials" {
			t.Fatalf("audit event lost the grounds of the decision: %+v", required)
		}
	})

	t.Run("UnknownKindRecordsTheValueAsRead", func(t *testing.T) {
		deps, events := seed(t, idmdomain.AgentKind("mystery"))
		if _, err := sharedusecases.ResolveIssuableAgentWithoutApproval(ctx, deps, "agent_client", "client_credentials", now); err == nil {
			t.Fatal("expected an unknown kind to fail closed")
		}
		required, ok := (*events)[0].(*domain.AgentApprovalRequired)
		if !ok || required.Kind != "mystery" {
			t.Fatalf("expected the kind to be recorded as read, got %+v", (*events)[0])
		}
	})

	t.Run("AutonomousIssues", func(t *testing.T) {
		deps, events := seed(t, idmdomain.AgentKindAutonomous)
		agent, err := sharedusecases.ResolveIssuableAgentWithoutApproval(ctx, deps, "agent_client", "client_credentials", now)
		if err != nil {
			t.Fatalf("expected an autonomous agent to be unaffected: %v", err)
		}
		if agent == nil || agent.ID != "agent_1" {
			t.Fatalf("expected the bound agent, got %v", agent)
		}
		if len(*events) != 0 {
			t.Fatalf("expected no refusal event, got %v", *events)
		}
	})

	t.Run("UnboundClientIsUnaffected", func(t *testing.T) {
		deps, _ := seed(t, idmdomain.AgentKindSupervised)
		agent, err := sharedusecases.ResolveIssuableAgentWithoutApproval(ctx, deps, "plain_client", "client_credentials", now)
		if err != nil || agent != nil {
			t.Fatalf("expected a client with no bound agent to be unaffected, got %v %v", agent, err)
		}
	})
}
