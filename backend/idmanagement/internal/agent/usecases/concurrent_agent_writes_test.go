package usecases_test

// Agent を変える書き込みが、読んだ後に別の書き手が行った変更を書き戻さないことを観測する。
// 競合の窓は、保存先を包んで、次の書き込みの直前に別の書き手の変更を差し込むことで決定的に作る。

import (
	"context"
	"errors"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	agentusecases "github.com/ambi/idmagic/backend/idmanagement/internal/agent/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

// changeBeforeNextWrite は、meanwhile が設定されている間の最初の書き込みの直前に、
// 別の書き手として Agent を変える。変更は包む前の保存先へ直接書く。
type changeBeforeNextWrite struct {
	agentports.AgentRepository
	meanwhile func(*agentdomain.Agent)
}

func (r *changeBeforeNextWrite) changeFirst(ctx context.Context, tenantID, id string) {
	meanwhile := r.meanwhile
	if meanwhile == nil {
		return
	}
	r.meanwhile = nil
	agent, err := r.FindByID(ctx, tenantID, id)
	if err != nil || agent == nil {
		panic("concurrent change: agent not found")
	}
	meanwhile(agent)
	if err := r.AgentRepository.Save(ctx, agent); err != nil {
		panic(err)
	}
}

func (r *changeBeforeNextWrite) Save(ctx context.Context, agent *agentdomain.Agent) error {
	r.changeFirst(ctx, agent.TenantID, agent.ID)
	return r.AgentRepository.Save(ctx, agent)
}

func (r *changeBeforeNextWrite) Update(
	ctx context.Context, tenantID, id string, change agentports.AgentChange,
) (*agentdomain.Agent, error) {
	r.changeFirst(ctx, tenantID, id)
	return r.AgentRepository.Update(ctx, tenantID, id, change)
}

func offboarded(agent *agentdomain.Agent) {
	agent.Status = idmdomain.AgentStatusDisabled
}

func killed(agent *agentdomain.Agent) {
	agent.Status = idmdomain.AgentStatusKilled
	now := time.Date(2026, 6, 22, 12, 30, 0, 0, time.UTC)
	agent.KilledAt = &now
}

func describedAs(description string) func(*agentdomain.Agent) {
	return func(agent *agentdomain.Agent) { agent.Description = &description }
}

// 状態の遷移はどれも、読んだ後の変更を書き戻さない。停止は一方向の終端なので、停止の後に
// 届いた更新、無効化と再有効化は agent_killed で拒否し、停止を取り消さない。
//
//spec:covers REQ-IDMANAGEMENT-081: 管理者の更新の間に所有者の停止で Agent が無効化されても、更新の後も Agent は Disabled のまま残り、所有者の停止に伴う無効化も間の更新を書き戻さないこと。
//spec:covers REQ-IDMANAGEMENT-077: 管理者の更新、無効化と再有効化の間に Agent が停止されると、その操作は agent_killed で拒否され、Agent は Killed のまま残ること。
func TestAgentWritesKeepAStateChangeMadeMeanwhile(t *testing.T) {
	now := time.Date(2026, 6, 22, 13, 0, 0, 0, time.UTC)
	edited := "edited meanwhile"
	updatedDescription := "updated by the administrator"
	for _, tc := range []struct {
		name       string
		meanwhile  func(*agentdomain.Agent)
		write      func(context.Context, agentusecases.AdminAgentDeps, string) error
		wantErr    error
		wantStatus idmdomain.AgentStatus
		wantDesc   *string
	}{
		{
			name: "update keeps an owner offboarding disable", meanwhile: offboarded,
			write: func(ctx context.Context, deps agentusecases.AdminAgentDeps, id string) error {
				_, err := agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{
					ActorUserID: "operator", ID: id, Description: &updatedDescription, Now: now,
				})
				return err
			},
			wantStatus: idmdomain.AgentStatusDisabled, wantDesc: &updatedDescription,
		},
		{
			name: "update is refused after a kill", meanwhile: killed,
			write: func(ctx context.Context, deps agentusecases.AdminAgentDeps, id string) error {
				_, err := agentusecases.UpdateAgent(ctx, deps, agentusecases.UpdateAgentInput{
					ActorUserID: "operator", ID: id, Description: &updatedDescription, Now: now,
				})
				return err
			},
			wantErr: agentusecases.ErrAgentKilled, wantStatus: idmdomain.AgentStatusKilled,
		},
		{
			name: "enable is refused after a kill", meanwhile: killed,
			write: func(ctx context.Context, deps agentusecases.AdminAgentDeps, id string) error {
				_, err := agentusecases.SetAgentDisabled(ctx, deps, "operator", id, false, now)
				return err
			},
			wantErr: agentusecases.ErrAgentKilled, wantStatus: idmdomain.AgentStatusKilled,
		},
		{
			name: "kill keeps an update", meanwhile: describedAs(edited),
			write: func(ctx context.Context, deps agentusecases.AdminAgentDeps, id string) error {
				_, err := agentusecases.KillAgent(ctx, deps, "operator", id, now)
				return err
			},
			wantStatus: idmdomain.AgentStatusKilled, wantDesc: &edited,
		},
		{
			name: "owner offboarding keeps an update", meanwhile: describedAs(edited),
			write: func(ctx context.Context, deps agentusecases.AdminAgentDeps, _ string) error {
				return agentusecases.DisableAgentsOwnedBy(ctx, agentusecases.AgentOwnerDeps{
					AgentRepo: deps.AgentRepo, Emit: deps.Emit,
				}, tenancydomain.DefaultTenantID, "operator", "operator", now)
			},
			wantStatus: idmdomain.AgentStatusDisabled, wantDesc: &edited,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := testing_tenant.Default(context.Background())
			deps, _ := newAgentDeps(t)
			agent, err := agentusecases.RegisterAgent(ctx, deps, agentusecases.RegisterAgentInput{
				Kind: idmdomain.AgentKindAutonomous, ActorUserID: "operator", Name: "deploy-bot", Now: now,
			})
			if err != nil {
				t.Fatal(err)
			}
			race := &changeBeforeNextWrite{AgentRepository: deps.AgentRepo, meanwhile: tc.meanwhile}
			deps.AgentRepo = race

			if err := tc.write(ctx, deps, agent.ID); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if race.meanwhile != nil {
				t.Fatal("前提が壊れている: 書き込みが Agent を書かなかった")
			}
			stored, err := race.FindByID(ctx, agent.TenantID, agent.ID)
			if err != nil || stored == nil {
				t.Fatalf("FindByID = (%+v, %v)", stored, err)
			}
			if stored.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q: the write reverted the change made meanwhile", stored.Status, tc.wantStatus)
			}
			if tc.wantDesc != nil && (stored.Description == nil || *stored.Description != *tc.wantDesc) {
				t.Fatalf("description = %v, want %q", stored.Description, *tc.wantDesc)
			}
		})
	}
}

// 所有者の停止は所有するすべての Active の Agent を無効化し、無効化した Agent ごとに
// AgentDisabled を発行する。一覧を読んだ後に停止された Agent は、無効化もイベントもしない。
//
//spec:covers REQ-IDMANAGEMENT-081: 所有者の停止が、一覧を読んだ後に停止された Agent を Killed のまま残してイベントを出さず、残りの Active の Agent を無効化して Agent ごとに AgentDisabled を発行すること。
func TestOwnerOffboardingSkipsAnAgentKilledMeanwhile(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	now := time.Date(2026, 6, 22, 13, 0, 0, 0, time.UTC)
	deps, events := newAgentDeps(t)
	var ids []string
	for _, name := range []string{"first-bot", "second-bot"} {
		agent, err := agentusecases.RegisterAgent(ctx, deps, agentusecases.RegisterAgentInput{
			Kind: idmdomain.AgentKindAutonomous, ActorUserID: "operator", Name: name, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, agent.ID)
	}
	race := &changeBeforeNextWrite{AgentRepository: deps.AgentRepo, meanwhile: killed}
	*events = nil

	if err := agentusecases.DisableAgentsOwnedBy(ctx, agentusecases.AgentOwnerDeps{AgentRepo: race, Emit: deps.Emit},
		tenancydomain.DefaultTenantID, "operator", "operator", now); err != nil {
		t.Fatal(err)
	}

	statuses := map[idmdomain.AgentStatus]int{}
	for _, id := range ids {
		stored, err := race.FindByID(ctx, tenancydomain.DefaultTenantID, id)
		if err != nil || stored == nil {
			t.Fatalf("FindByID = (%+v, %v)", stored, err)
		}
		statuses[stored.Status]++
	}
	if statuses[idmdomain.AgentStatusKilled] != 1 || statuses[idmdomain.AgentStatusDisabled] != 1 {
		t.Fatalf("statuses = %v, want one killed and one disabled", statuses)
	}
	if got := agentEventTypes(*events); len(got) != 1 || got[0] != "AgentDisabled" {
		t.Fatalf("events = %v, want exactly one AgentDisabled", got)
	}
}
