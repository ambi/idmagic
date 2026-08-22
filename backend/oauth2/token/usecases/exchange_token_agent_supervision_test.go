package usecases

// REQ-OAUTH2-050 をトークン交換の経路で確かめる (wi-376 T003)。交換は人間の承認を
// 記録しないため、交換に関与する Agent のいずれかが Supervised なら発行しない。
// 判定そのものは backend/oauth2/usecases の TestAgentRequiresHumanApproval が持ち、
// ここは ExchangeToken が関与する Agent をすべて洗い出していることを見る。

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/shared/spec"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

func seedSupervisionAgent(t *testing.T, repo *agentmemory.AgentRepository, id, clientID string, kind idmdomain.AgentKind) {
	t.Helper()
	now := time.Now().UTC()
	if err := repo.Save(context.Background(), &agentdomain.Agent{
		ID: id, TenantID: kernel.DefaultTenantID, Name: id, Kind: kind, OwnerUserID: "owner_1",
		Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if clientID != "" {
		if _, err := repo.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
			AgentID: id, ClientID: clientID, CreatedAt: now,
		}); err != nil {
			t.Fatalf("seed binding: %v", err)
		}
	}
}

func assertApprovalRequired(t *testing.T, issuer *recordingIssuer, events []spec.DomainEvent, err error, agentID string) {
	t.Helper()
	var oauthErr *OAuthError
	if !errors.As(err, &oauthErr) || oauthErr.Code != "unauthorized_client" {
		t.Fatalf("expected unauthorized_client, got %v", err)
	}
	if issuer.calls != 0 {
		t.Fatal("a token was issued for a supervised agent")
	}
	for _, e := range events {
		if required, is := e.(*domain.AgentApprovalRequired); is && required.AgentID == agentID {
			return
		}
	}
	t.Fatalf("expected AgentApprovalRequired for %q, got %v", agentID, events)
}

// TestExchangeTokenRejectsSupervisedWorkloadAgent — ワークロード ID 連携の attestation
// が Supervised な Agent の client へ写る交換は成立しない。
func TestExchangeTokenRejectsSupervisedWorkloadAgent(t *testing.T) {
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, nil)
	deps.ClientRepo.(*oauth2memory.OAuth2ClientRepository).Seed(&domain.OAuth2Client{
		ClientID: "agent-bound-client", ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{spec.GrantClientCredentials}, Scope: "read",
		CreatedAt: time.Now().UTC(),
	})
	agentRepo := agentmemory.NewAgentRepository()
	seedSupervisionAgent(t, agentRepo, "agent_1", "agent-bound-client", idmdomain.AgentKindSupervised)
	deps.AgentRepo = agentRepo
	deps.WorkloadVerifier = fakeWorkloadVerifier{grant: &workloaddomain.WorkloadIdentityGrant{
		AgentID: "agent_1", ClientID: "agent-bound-client",
		TrustBundleID: "bundle_1", BindingID: "binding_1",
	}}
	var events []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }

	_, err := ExchangeToken(context.Background(), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "external-svid",
		SubjectTokenType: tokenTypeJWTURN, Resource: []string{"https://api.example"},
	}, time.Now().UTC())
	assertApprovalRequired(t, issuer, events, err, "agent_1")
}

// TestExchangeTokenRejectsSupervisedSubjectAgent — 承認を経て発行済みのトークンで
// あっても、そこからの派生は拒否する。一つの承認は一つのトークンに対応する。
func TestExchangeTokenRejectsSupervisedSubjectAgent(t *testing.T) {
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, map[string]*ports.IntrospectionResult{
		"approved": {Active: true, Sub: "agent-bound-client", Scope: "read write", AgentID: "agent_1"},
	})
	agentRepo := agentmemory.NewAgentRepository()
	seedSupervisionAgent(t, agentRepo, "agent_1", "agent-bound-client", idmdomain.AgentKindSupervised)
	deps.AgentRepo = agentRepo
	var events []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }

	_, err := ExchangeToken(context.Background(), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "approved", Resource: []string{"https://api.example"},
	}, time.Now().UTC())
	assertApprovalRequired(t, issuer, events, err, "agent_1")
}

// TestExchangeTokenRejectsSupervisedActingClient — 交換を行うクライアント自身が
// Supervised な Agent に束縛されているなら、利用者のトークンを代行する交換も拒否する。
func TestExchangeTokenRejectsSupervisedActingClient(t *testing.T) {
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, map[string]*ports.IntrospectionResult{
		"subj": {Active: true, Sub: "user-1", Scope: "read write"},
	})
	agentRepo := agentmemory.NewAgentRepository()
	seedSupervisionAgent(t, agentRepo, "agent_2", "client", idmdomain.AgentKindSupervised)
	deps.AgentRepo = agentRepo
	var events []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }

	_, err := ExchangeToken(context.Background(), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"},
	}, time.Now().UTC())
	assertApprovalRequired(t, issuer, events, err, "agent_2")
}

// TestExchangeTokenAllowsAutonomousAgents — Autonomous だけが関与する交換は退行しない。
func TestExchangeTokenAllowsAutonomousAgents(t *testing.T) {
	issuer := &recordingIssuer{}
	deps := newExchangeTokenDeps(t, issuer, map[string]*ports.IntrospectionResult{
		"subj": {Active: true, Sub: "agent-bound-client", Scope: "read write", AgentID: "agent_1"},
	})
	agentRepo := agentmemory.NewAgentRepository()
	seedSupervisionAgent(t, agentRepo, "agent_1", "agent-bound-client", idmdomain.AgentKindAutonomous)
	seedSupervisionAgent(t, agentRepo, "agent_2", "client", idmdomain.AgentKindAutonomous)
	deps.AgentRepo = agentRepo

	if _, err := ExchangeToken(context.Background(), deps, ExchangeTokenInput{
		ClientID: "client", SubjectToken: "subj", Resource: []string{"https://api.example"},
	}, time.Now().UTC()); err != nil {
		t.Fatalf("expected autonomous agents to be unaffected: %v", err)
	}
	if issuer.calls != 1 {
		t.Fatalf("expected one issuance, got %d", issuer.calls)
	}
}
