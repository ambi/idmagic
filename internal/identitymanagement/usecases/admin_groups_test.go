package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	idmusecases "github.com/ambi/idmagic/internal/identitymanagement/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func newGroupDeps(t *testing.T) (idmusecases.AdminGroupDeps, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := memory.NewUserRepository()
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&spec.User{
		ID: "user_alice", TenantID: spec.DefaultTenantID, PreferredUsername: "alice",
		PasswordHash: "x", Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	userRepo.Seed(&spec.User{
		ID: "user_other", TenantID: "acme", PreferredUsername: "other",
		PasswordHash: "x", Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	events := &[]spec.DomainEvent{}
	deps := idmusecases.AdminGroupDeps{
		GroupRepo: memory.NewGroupRepository(),
		UserRepo:  userRepo,
		Emit:      func(e spec.DomainEvent) { *events = append(*events, e) },
	}
	return deps, events
}

func eventTypes(events []spec.DomainEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.EventType()
	}
	return out
}

func TestGroupCreateAddMemberEffectiveRoles(t *testing.T) {
	ctx := context.Background()
	deps, events := newGroupDeps(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)

	group, err := idmusecases.CreateGroup(ctx, deps, idmusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Roles: []string{"catalog:read"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 名前一意性
	if _, err := idmusecases.CreateGroup(ctx, deps, idmusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "Engineering", Now: now,
	}); !errors.Is(err, idmusecases.ErrGroupNameConflict) {
		t.Fatalf("expected name conflict, got %v", err)
	}

	if err := idmusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}
	// 冪等: 再追加では event は増えない
	if err := idmusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}

	view, err := idmusecases.UserGroups(ctx, deps, "user_alice")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(view.EffectiveRoles, []string{"catalog:read"}) {
		t.Fatalf("effective roles = %v", view.EffectiveRoles)
	}
	if !slices.Equal(view.GroupRoles, []string{"catalog:read"}) || len(view.DirectRoles) != 0 {
		t.Fatalf("direct=%v group=%v", view.DirectRoles, view.GroupRoles)
	}

	got := eventTypes(*events)
	want := []string{"GroupCreated", "GroupMemberAdded"}
	if !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestListGetUpdateAndRemoveGroupMember(t *testing.T) {
	ctx := context.Background()
	deps, events := newGroupDeps(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	description := "  platform team  "
	group, err := idmusecases.CreateGroup(ctx, deps, idmusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Description: &description, Roles: []string{"catalog:read"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := idmusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}

	list, err := idmusecases.ListGroups(ctx, deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Group.ID != group.ID || list[0].MemberCount != 1 {
		t.Fatalf("list = %+v", list)
	}
	got, members, err := idmusecases.GetGroup(ctx, deps, group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Description == nil || *got.Description != "platform team" || len(members) != 1 {
		t.Fatalf("get group=%+v members=%+v", got, members)
	}

	*events = (*events)[:0]
	newName := "platform"
	emptyDescription := " "
	roles := []string{"catalog:write"}
	updated, err := idmusecases.UpdateGroup(ctx, deps, idmusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Name: &newName, Description: &emptyDescription, Roles: &roles, Now: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "platform" || updated.Description != nil || !slices.Equal(updated.Roles, roles) {
		t.Fatalf("updated = %+v", updated)
	}
	if !slices.Equal(eventTypes(*events), []string{"GroupUpdated"}) {
		t.Fatalf("events = %v", eventTypes(*events))
	}

	*events = (*events)[:0]
	if err := idmusecases.RemoveMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}
	if err := idmusecases.RemoveMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eventTypes(*events), []string{"GroupMemberRemoved"}) {
		t.Fatalf("events = %v", eventTypes(*events))
	}
}

func TestUpdateGroupValidationErrors(t *testing.T) {
	ctx := context.Background()
	deps, _ := newGroupDeps(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	group, err := idmusecases.CreateGroup(ctx, deps, idmusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idmusecases.CreateGroup(ctx, deps, idmusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "support", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	blank := " "
	if _, err := idmusecases.UpdateGroup(ctx, deps, idmusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Name: &blank,
	}); !errors.Is(err, idmusecases.ErrGroupNameEmpty) {
		t.Fatalf("expected ErrGroupNameEmpty, got %v", err)
	}
	dupe := "support"
	if _, err := idmusecases.UpdateGroup(ctx, deps, idmusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Name: &dupe,
	}); !errors.Is(err, idmusecases.ErrGroupNameConflict) {
		t.Fatalf("expected ErrGroupNameConflict, got %v", err)
	}
	roles := []string{" "}
	if _, err := idmusecases.UpdateGroup(ctx, deps, idmusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Roles: &roles,
	}); err == nil {
		t.Fatal("expected invalid role error")
	}
	if _, err := idmusecases.UpdateGroup(ctx, deps, idmusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: "ghost", Name: &dupe,
	}); !errors.Is(err, idmusecases.ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

func TestAddMemberRejectsCrossTenantUser(t *testing.T) {
	ctx := context.Background()
	deps, _ := newGroupDeps(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	group, err := idmusecases.CreateGroup(ctx, deps, idmusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := idmusecases.AddMember(ctx, deps, "operator", group.ID, "user_other", now); !errors.Is(err, idmusecases.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for cross-tenant user, got %v", err)
	}
}

func TestDeleteGroupCascadesMembership(t *testing.T) {
	ctx := context.Background()
	deps, events := newGroupDeps(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	group, err := idmusecases.CreateGroup(ctx, deps, idmusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := idmusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}
	*events = (*events)[:0]
	if err := idmusecases.DeleteGroup(ctx, deps, "operator", group.ID, now); err != nil {
		t.Fatal(err)
	}
	got := eventTypes(*events)
	want := []string{"GroupMemberRemoved", "GroupDeleted"}
	if !slices.Equal(got, want) {
		t.Fatalf("delete events = %v, want %v", got, want)
	}
	if _, _, err := idmusecases.GetGroup(ctx, deps, group.ID); !errors.Is(err, idmusecases.ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound after delete, got %v", err)
	}
}
