package usecases_test

import (
	"context"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_memory"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-IDMANAGEMENT-015, REQ-IDMANAGEMENT-063: モジュール向けの所属操作は実際の変更だけを返し、同じ操作でイベントを増やさず、別テナントの参照を拒否する。
func TestGroupMembershipCommandsDesiredState(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{ID: "alice", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	groups := groupmemory.NewGroupRepository()
	if err := groups.Save(ctx, &groupdomain.Group{ID: "engineering", TenantID: "tenant-a", Name: "Engineering", MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	events := []spec.DomainEvent{}
	commands := groupusecases.GroupMembershipCommands{Actor: "workflow", Deps: groupusecases.AdminGroupDeps{UserRepo: users, GroupRepo: groups, Emit: func(event spec.DomainEvent) error { events = append(events, event); return nil }}}
	for i, present := range []bool{true, true, false, false} {
		changed, err := commands.SetGroupMembership(ctx, "tenant-a", "engineering", "alice", present, now)
		if err != nil || changed != (i%2 == 0) {
			t.Fatalf("operation %d = %v, %v", i, changed, err)
		}
	}
	if len(events) != 2 || events[0].EventType() != "GroupMemberAdded" || events[1].EventType() != "GroupMemberRemoved" {
		t.Fatalf("events = %+v", events)
	}
	if changed, err := commands.SetGroupMembership(ctx, "tenant-b", "engineering", "alice", true, now); changed || err == nil {
		t.Fatalf("foreign = %v, %v", changed, err)
	}
	if len(events) != 2 {
		t.Fatalf("foreign event count = %d", len(events))
	}
}
