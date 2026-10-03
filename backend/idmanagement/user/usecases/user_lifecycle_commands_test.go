package usecases_test

import (
	"context"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var lifecycleNow = time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)

type lifecycleFixture struct {
	adapter userusecases.UserLifecycleCommands
	users   *usermemory.UserRepository
	agents  *agentmemory.AgentRepository
	events  *[]spec.DomainEvent
}

func newLifecycleFixture(t *testing.T, status idmdomain.UserStatus) *lifecycleFixture {
	t.Helper()
	f := &lifecycleFixture{users: usermemory.NewUserRepository(), agents: agentmemory.NewAgentRepository(), events: &[]spec.DomainEvent{}}
	f.users.Seed(&userdomain.User{
		ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{},
		Lifecycle: userdomain.UserLifecycle{Status: status}, CreatedAt: lifecycleNow, UpdatedAt: lifecycleNow,
	})
	if err := f.agents.Save(context.Background(), &agentdomain.Agent{
		ID: "deploy-bot", TenantID: "tenant-a", Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: "alice", Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: lifecycleNow, UpdatedAt: lifecycleNow,
	}); err != nil {
		t.Fatal(err)
	}
	f.adapter = userusecases.UserLifecycleCommands{
		Deps: userusecases.AdminUserDeps{
			UserRepo: f.users, AgentRepo: f.agents,
			Emit: func(event spec.DomainEvent) error { *f.events = append(*f.events, event); return nil },
		},
		Actor: "lifecycle-workflow",
	}
	return f
}

//spec:covers EX-IDMANAGEMENT-081-02: ワークフローの disable_user が IdManagement の無効化を通り、User を Disabled にして UserDisabled を発行し、所有する Agent を Disabled にすること。
func TestWorkflowDisableStopsTheUserAndTheAgentsTheyOwn(t *testing.T) {
	f := newLifecycleFixture(t, idmdomain.UserStatusActive)
	changed, err := f.adapter.SetUserDisabled(context.Background(), "tenant-a", "alice", true, lifecycleNow)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v, want changed", changed, err)
	}
	user, _ := f.users.FindBySub(context.Background(), "alice")
	if user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("User の状態=%s, want disabled", user.Lifecycle.Status)
	}
	agent, _ := f.agents.FindByID(context.Background(), "tenant-a", "deploy-bot")
	if agent.Status != idmdomain.AgentStatusDisabled {
		t.Fatalf("Agent の状態=%s, want disabled", agent.Status)
	}
	var types []string
	for _, event := range *f.events {
		types = append(types, event.EventType())
	}
	if len(types) != 2 || types[0] != "UserDisabled" || types[1] != "AgentDisabled" {
		t.Fatalf("events=%v, want [UserDisabled AgentDisabled]", types)
	}
}

// 削除予約中の User は IdManagement が無効化を拒否する。ワークフローの手順としては変更なしである。
func TestWorkflowDisableLeavesAPendingDeletionUserUnchanged(t *testing.T) {
	f := newLifecycleFixture(t, idmdomain.UserStatusPendingDeletion)
	changed, err := f.adapter.SetUserDisabled(context.Background(), "tenant-a", "alice", true, lifecycleNow)
	if err != nil || changed {
		t.Fatalf("changed=%v err=%v, want unchanged", changed, err)
	}
	user, _ := f.users.FindBySub(context.Background(), "alice")
	if user.Lifecycle.Status != idmdomain.UserStatusPendingDeletion || len(*f.events) != 0 {
		t.Fatalf("status=%s events=%d, want pending deletion and none", user.Lifecycle.Status, len(*f.events))
	}
}
