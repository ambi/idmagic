package usecases_test

import (
	"context"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	scimmemory "github.com/ambi/idmagic/backend/sourcing/scim/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/scim/usecases"
)

// scimUserLifecycle は、組み立ての地点と同じく IdManagement の操作を SCIM のポートへ渡す。
func scimUserLifecycle(users userports.UserRepository) userusecases.UserLifecycleCommands {
	return userusecases.UserLifecycleCommands{Deps: userusecases.AdminUserDeps{UserRepo: users}, Actor: "scim"}
}

type scimOwnerFixture struct {
	u       *usecases.Usecases
	users   *usermemory.UserRepository
	agents  *agentmemory.AgentRepository
	events  []string
	scimID  string
	ownerID string
}

func newScimOwnerFixture(t *testing.T) *scimOwnerFixture {
	t.Helper()
	ctx := context.Background()
	f := &scimOwnerFixture{users: usermemory.NewUserRepository(), agents: agentmemory.NewAgentRepository()}
	lifecycle := userusecases.UserLifecycleCommands{
		Deps: userusecases.AdminUserDeps{
			UserRepo: f.users, AgentRepo: f.agents,
			Emit: func(event spec.DomainEvent) error { f.events = append(f.events, event.EventType()); return nil },
		},
		Actor: "scim",
	}
	f.u = usecases.NewUsecases(scimmemory.NewScimRepository(), f.users, groupmemory.NewGroupRepository(), lifecycle, func(spec.DomainEvent) {})
	created, err := f.u.CreateUser(ctx, scimTenant, map[string]any{"userName": "alice", "active": true})
	if err != nil {
		t.Fatal(err)
	}
	f.scimID, _ = created["id"].(string)
	owner, err := f.users.FindByUsername(ctx, scimTenant, "alice")
	if err != nil || owner == nil {
		t.Fatalf("owner=%v err=%v", owner, err)
	}
	f.ownerID = owner.ID
	now := time.Now()
	if err := f.agents.Save(ctx, &agentdomain.Agent{
		ID: "deploy-bot", TenantID: scimTenant, Name: "deploy-bot", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: owner.ID, Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *scimOwnerFixture) agentStatus(t *testing.T) idmdomain.AgentStatus {
	t.Helper()
	agent, err := f.agents.FindByID(context.Background(), scimTenant, "deploy-bot")
	if err != nil || agent == nil {
		t.Fatalf("agent=%v err=%v", agent, err)
	}
	return agent.Status
}

// REQ-IDMANAGEMENT-081 の主要な使い方：SCIM の取り込みが所有者を止めると、所有する Agent も止まる。
//
//spec:covers EX-IDMANAGEMENT-081-03: SCIM の PATCH による active=false が IdManagement の無効化を通り、User を Disabled にして UserDisabled を発行し、所有する Agent を Disabled にすること。
func TestScimDeactivationStopsTheUserAndTheAgentsTheyOwn(t *testing.T) {
	f := newScimOwnerFixture(t)
	out, err := f.u.PatchUser(context.Background(), scimTenant, f.scimID, patchBody(patchOp("replace", "active", false)))
	if err != nil {
		t.Fatal(err)
	}
	if out["active"] != false {
		t.Fatalf("応答の active=%v, want false", out["active"])
	}
	owner, _ := f.users.FindBySub(context.Background(), f.ownerID)
	if owner.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("User の状態=%s, want disabled", owner.Lifecycle.Status)
	}
	if got := f.agentStatus(t); got != idmdomain.AgentStatusDisabled {
		t.Fatalf("Agent の状態=%s, want disabled", got)
	}
	if len(f.events) != 2 || f.events[0] != "UserDisabled" || f.events[1] != "AgentDisabled" {
		t.Fatalf("events=%v, want [UserDisabled AgentDisabled]", f.events)
	}
}

// active に触れない PATCH は、Active の User も Disabled の User も止めず、再開もしない。
func TestScimPatchWithoutActiveLeavesTheUserStatusAlone(t *testing.T) {
	for _, active := range []bool{true, false} {
		f := newScimOwnerFixture(t)
		ctx := context.Background()
		if !active {
			if _, err := f.u.PatchUser(ctx, scimTenant, f.scimID, patchBody(patchOp("replace", "active", false))); err != nil {
				t.Fatal(err)
			}
		}
		f.events = nil
		out, err := f.u.PatchUser(ctx, scimTenant, f.scimID, patchBody(patchOp("replace", "name.givenName", "Alice")))
		if err != nil {
			t.Fatal(err)
		}
		if out["active"] != active || len(f.events) != 0 {
			t.Fatalf("active=%v events=%v, want active=%v and no events", out["active"], f.events, active)
		}
	}
}

// PUT の全置換でも active=false は IdManagement の無効化を通る。再び active=true にしても Agent は止まったままである。
func TestScimReplaceDeactivatesThroughIdManagementAndReactivationKeepsAgentsStopped(t *testing.T) {
	f := newScimOwnerFixture(t)
	ctx := context.Background()
	if _, err := f.u.UpdateUser(ctx, scimTenant, f.scimID, map[string]any{"userName": "alice", "active": false}); err != nil {
		t.Fatal(err)
	}
	if got := f.agentStatus(t); got != idmdomain.AgentStatusDisabled {
		t.Fatalf("無効化の後の Agent の状態=%s, want disabled", got)
	}
	out, err := f.u.UpdateUser(ctx, scimTenant, f.scimID, map[string]any{"userName": "alice", "active": true})
	if err != nil {
		t.Fatal(err)
	}
	if out["active"] != true {
		t.Fatalf("応答の active=%v, want true", out["active"])
	}
	if got := f.agentStatus(t); got != idmdomain.AgentStatusDisabled {
		t.Fatalf("再有効化の後の Agent の状態=%s, want disabled", got)
	}
	if f.events[len(f.events)-1] != "UserEnabled" {
		t.Fatalf("events=%v, want UserEnabled last", f.events)
	}
}

// SCIM の削除は IdManagement の削除の予約を通り、User を PendingDeletion にして所有する Agent を止める。
func TestScimDeleteSchedulesDeletionThroughIdManagement(t *testing.T) {
	f := newScimOwnerFixture(t)
	if err := f.u.DeleteUser(context.Background(), scimTenant, f.scimID); err != nil {
		t.Fatal(err)
	}
	owner, _ := f.users.FindBySub(context.Background(), f.ownerID)
	if owner.Lifecycle.Status != idmdomain.UserStatusPendingDeletion {
		t.Fatalf("User の状態=%s, want pending deletion", owner.Lifecycle.Status)
	}
	if got := f.agentStatus(t); got != idmdomain.AgentStatusDisabled {
		t.Fatalf("Agent の状態=%s, want disabled", got)
	}
	if len(f.events) == 0 || f.events[0] != "UserSoftDeleted" {
		t.Fatalf("events=%v, want UserSoftDeleted first", f.events)
	}
}
