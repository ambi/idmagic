package usecases_test

// 主要ユースケース追跡: REQ-SHAREDSIGNALS-001。

import (
	"context"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentmodel "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	dbmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssusecases "github.com/ambi/idmagic/backend/sharedsignals/usecases"
)

// TestAdvanceRevocationEpoch_EmitsForEachAgent — scenario
// `kill-switchは既発行トークンをintrospectionで即時無効化する` の前提: AdvanceRevocationEpoch
// が epoch を前進させ RevocationEpochAdvanced / AgentAccessRevoked を emit する。
func TestAdvanceRevocationEpoch_EmitsForEachAgent(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewAgentRevocationEpochRepository()
	var emitted []spec.DomainEvent
	deps := ssusecases.RevocationDeps{
		EpochRepo: repo,
		Emit: func(e spec.DomainEvent) error {
			emitted = append(emitted, e)
			return nil
		},
	}
	now := time.Now().UTC()

	if err := ssusecases.AdvanceRevocationEpoch(ctx, deps, "tenant-a", []string{"agent_1", "agent_2"}, ssdomain.RevocationReasonAgentKilled, nil, now); err != nil {
		t.Fatalf("AdvanceRevocationEpoch: %v", err)
	}

	if len(emitted) != 4 {
		t.Fatalf("expected 4 events (Advanced+Revoked per agent), got %d: %+v", len(emitted), emitted)
	}
	for _, agentID := range []string{"agent_1", "agent_2"} {
		got, err := repo.FindByAgent(ctx, "tenant-a", agentID)
		if err != nil {
			t.Fatalf("FindByAgent(%s): %v", agentID, err)
		}
		if got == nil || !got.Epoch.Equal(now) {
			t.Fatalf("agent %s epoch not advanced: %+v", agentID, got)
		}
	}
}

// TestAdvanceRevocationEpoch_AlreadyRevokedIsIdempotent — RED: 既に後の epoch を持つ
// Agent への再前進要求は ErrEpochNotAdvancing を吸収し、event を再 emit しない (冪等)。
func TestAdvanceRevocationEpoch_AlreadyRevokedIsIdempotent(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewAgentRevocationEpochRepository()
	now := time.Now().UTC()
	if err := repo.Advance(ctx, ssdomain.AgentRevocationEpoch{
		AgentID: "agent_1", TenantID: "tenant-a", Epoch: now, Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}); err != nil {
		t.Fatalf("seed Advance: %v", err)
	}

	var emitted []spec.DomainEvent
	deps := ssusecases.RevocationDeps{EpochRepo: repo, Emit: func(e spec.DomainEvent) error { emitted = append(emitted, e); return nil }}

	earlier := now.Add(-time.Hour)
	if err := ssusecases.AdvanceRevocationEpoch(ctx, deps, "tenant-a", []string{"agent_1"}, ssdomain.RevocationReasonAgentDisabled, nil, earlier); err != nil {
		t.Fatalf("AdvanceRevocationEpoch (already ahead): %v", err)
	}
	if len(emitted) != 0 {
		t.Fatalf("expected no events for an already-later epoch, got %+v", emitted)
	}
	got, err := repo.FindByAgent(ctx, "tenant-a", "agent_1")
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if !got.Epoch.Equal(now) || got.Reason != ssdomain.RevocationReasonAgentKilled {
		t.Fatalf("epoch must not regress: %+v", got)
	}
}

func TestCheckRevocationEpoch_UnknownAgentReturnsNil(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewAgentRevocationEpochRepository()
	deps := ssusecases.RevocationDeps{EpochRepo: repo}

	got, err := ssusecases.CheckRevocationEpoch(ctx, deps, "tenant-a", "no-such-agent")
	if err != nil {
		t.Fatalf("CheckRevocationEpoch: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for a never-revoked agent, got %+v", got)
	}
}

func seedAgent(ctx context.Context, t *testing.T, repo *agentmemory.AgentRepository, id, tenantID, ownerUserID string) {
	t.Helper()
	agent := &agentmodel.Agent{
		ID: id, TenantID: tenantID, Name: id, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: ownerUserID, Status: idmdomain.AgentStatusActive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatalf("seed agent %s: %v", id, err)
	}
}

// TestAgentRevocationReactor_ReactsToAgentEvents — RED: React が
// AgentKilled/AgentDisabled/AgentCredentialUnbound のそれぞれで対象 Agent の epoch
// のみ前進させる (wi-58)。KillAgent 等はこのイベントを既に emit しており、
// reactor 側の追加呼び出しは不要 (トリガーは既存の Emit)。
func TestAgentRevocationReactor_ReactsToAgentEvents(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	cases := []struct {
		name   string
		event  spec.DomainEvent
		reason ssdomain.RevocationReason
	}{
		{"AgentKilled", &idmdomain.AgentKilled{At: now, TenantID: "tenant-a", AgentID: "agent_1"}, ssdomain.RevocationReasonAgentKilled},
		{"AgentDisabled", &idmdomain.AgentDisabled{At: now, TenantID: "tenant-a", AgentID: "agent_1"}, ssdomain.RevocationReasonAgentDisabled},
		{"AgentCredentialUnbound", &idmdomain.AgentCredentialUnbound{At: now, TenantID: "tenant-a", AgentID: "agent_1"}, ssdomain.RevocationReasonAgentCredentialUnbound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			epochRepo := dbmemory.NewAgentRevocationEpochRepository()
			reactor := &ssusecases.AgentRevocationReactor{EpochRepo: epochRepo, AgentRepo: agentmemory.NewAgentRepository()}
			if err := reactor.React(ctx, tc.event); err != nil {
				t.Fatalf("React: %v", err)
			}
			got, err := epochRepo.FindByAgent(ctx, "tenant-a", "agent_1")
			if err != nil {
				t.Fatalf("FindByAgent: %v", err)
			}
			if got == nil || got.Reason != tc.reason {
				t.Fatalf("agent_1 not revoked as %s: %+v", tc.reason, got)
			}
		})
	}
}

// TestAgentRevocationReactor_ReactsToOwnerEvents — RED: React が
// UserDisabled/UserSoftDeleted/UserDeleted のそれぞれで owner_user_id の一致する
// Agent 群だけを一括前進させ、他所有者の Agent には触れない。
func TestAgentRevocationReactor_ReactsToOwnerEvents(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	cases := []struct {
		name   string
		event  spec.DomainEvent
		reason ssdomain.RevocationReason
	}{
		{"UserDisabled", &idmdomain.UserDisabled{At: now, TenantID: "tenant-a", TargetUserID: "user_1"}, ssdomain.RevocationReasonOwnerDisabled},
		{"UserSoftDeleted", &idmdomain.UserSoftDeleted{At: now, TenantID: "tenant-a", TargetUserID: "user_1"}, ssdomain.RevocationReasonOwnerDisabled},
		{"UserDeleted", &idmdomain.UserDeleted{At: now, TenantID: "tenant-a", TargetUserID: "user_1"}, ssdomain.RevocationReasonOwnerDeleted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			epochRepo := dbmemory.NewAgentRevocationEpochRepository()
			agentRepo := agentmemory.NewAgentRepository()
			seedAgent(ctx, t, agentRepo, "agent_1", "tenant-a", "user_1")
			seedAgent(ctx, t, agentRepo, "agent_2", "tenant-a", "user_1")
			seedAgent(ctx, t, agentRepo, "agent_3", "tenant-a", "user_2")
			reactor := &ssusecases.AgentRevocationReactor{EpochRepo: epochRepo, AgentRepo: agentRepo}

			if err := reactor.React(ctx, tc.event); err != nil {
				t.Fatalf("React: %v", err)
			}
			for _, id := range []string{"agent_1", "agent_2"} {
				got, _ := epochRepo.FindByAgent(ctx, "tenant-a", id)
				if got == nil || got.Reason != tc.reason {
					t.Fatalf("%s must be revoked via owner offboard: %+v", id, got)
				}
			}
			got3, _ := epochRepo.FindByAgent(ctx, "tenant-a", "agent_3")
			if got3 != nil {
				t.Fatalf("agent_3 (different owner) must be untouched, got %+v", got3)
			}
		})
	}
}

// TestAgentRevocationReactor_OwnerWithNoAgentsIsNoop — owner に Agent が 1 件も
// 無い場合は repo を触らず nil を返す (empty set no-op)。
func TestAgentRevocationReactor_OwnerWithNoAgentsIsNoop(t *testing.T) {
	ctx := context.Background()
	reactor := &ssusecases.AgentRevocationReactor{
		EpochRepo: dbmemory.NewAgentRevocationEpochRepository(), AgentRepo: agentmemory.NewAgentRepository(),
	}
	if err := reactor.React(ctx, &idmdomain.UserDeleted{At: time.Now().UTC(), TenantID: "tenant-a", TargetUserID: "user_without_agents"}); err != nil {
		t.Fatalf("React (empty owner set): %v", err)
	}
}

// TestAgentRevocationReactor_IgnoresUnrelatedEvents — React はハンドリング対象外の
// DomainEvent を no-op で無視する (未知の event で panic/error にならない)。
func TestAgentRevocationReactor_IgnoresUnrelatedEvents(t *testing.T) {
	ctx := context.Background()
	reactor := &ssusecases.AgentRevocationReactor{
		EpochRepo: dbmemory.NewAgentRevocationEpochRepository(), AgentRepo: agentmemory.NewAgentRepository(),
	}
	if err := reactor.React(ctx, &idmdomain.AgentRegistered{At: time.Now().UTC(), TenantID: "tenant-a", AgentID: "agent_1"}); err != nil {
		t.Fatalf("React (unrelated event): %v", err)
	}
}

// TestAgentRevocationReactor_NilEpochRepoIsNoop — RED: EpochRepo が nil の
// lightweight wiring (SharedSignals.Module 未配線のテスト等) では、対象イベントを
// 受けても panic せず no-op で返す。composition root は常に non-nil な reactor を
// 渡すため (deps_http.Deps.ReactiveEmit)、reactor 自身がこの nil-skip を保証する
// 必要がある (wi-58)。
func TestAgentRevocationReactor_NilEpochRepoIsNoop(t *testing.T) {
	ctx := context.Background()
	reactor := &ssusecases.AgentRevocationReactor{AgentRepo: agentmemory.NewAgentRepository()}
	if err := reactor.React(ctx, &idmdomain.AgentKilled{At: time.Now().UTC(), TenantID: "tenant-a", AgentID: "agent_1"}); err != nil {
		t.Fatalf("React (nil EpochRepo): %v", err)
	}
}
