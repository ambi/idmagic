package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newGroupDeps(t *testing.T) (groupusecases.AdminGroupDeps, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "user_alice", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice",
		PasswordHash: "x", Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	userRepo.Seed(&userdomain.User{
		ID: "user_other", TenantID: "acme", PreferredUsername: "other",
		PasswordHash: "x", Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	})
	events := &[]spec.DomainEvent{}
	deps := groupusecases.AdminGroupDeps{
		GroupRepo: groupmemory.NewGroupRepository(),
		UserRepo:  userRepo,
		Emit:      func(e spec.DomainEvent) error { *events = append(*events, e); return nil },
		QuotaRepo: tenancymemory.NewQuotaRepository(),
	}
	return deps, events
}

// newGroupDepsWithQuota is like newGroupDeps but pins the tenant's groups
// Hard Quota to limit, for wi-160 enforcement tests (ADR-134).
func newGroupDepsWithQuota(t *testing.T, tenantID string, limit int) groupusecases.AdminGroupDeps {
	t.Helper()
	deps, _ := newGroupDeps(t)
	if err := deps.QuotaRepo.SetQuota(context.Background(), tenantID, &tenancydomain.TenantQuota{Groups: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	return deps
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

	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Roles: []string{"catalog:read"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	// 名前一意性
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "Engineering", Now: now,
	}); !errors.Is(err, groupusecases.ErrGroupNameConflict) {
		t.Fatalf("expected name conflict, got %v", err)
	}

	if err := groupusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}
	// 冪等: 再追加では event は増えない
	if err := groupusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}

	view, err := groupusecases.UserGroups(ctx, deps, "user_alice")
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
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Description: &description, Roles: []string{"catalog:read"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := groupusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}

	list, err := groupusecases.ListGroups(ctx, deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Group.ID != group.ID || list[0].MemberCount != 1 {
		t.Fatalf("list = %+v", list)
	}
	got, members, err := groupusecases.GetGroup(ctx, deps, group.ID)
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
	updated, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
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
	if err := groupusecases.RemoveMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}
	if err := groupusecases.RemoveMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
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
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "support", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	blank := " "
	if _, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Name: &blank,
	}); !errors.Is(err, groupusecases.ErrGroupNameEmpty) {
		t.Fatalf("expected ErrGroupNameEmpty, got %v", err)
	}
	dupe := "support"
	if _, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Name: &dupe,
	}); !errors.Is(err, groupusecases.ErrGroupNameConflict) {
		t.Fatalf("expected ErrGroupNameConflict, got %v", err)
	}
	roles := []string{" "}
	if _, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: group.ID, Roles: &roles,
	}); err == nil {
		t.Fatal("expected invalid role error")
	}
	if _, err := groupusecases.UpdateGroup(ctx, deps, groupusecases.UpdateGroupInput{
		ActorUserID: "operator", ID: "ghost", Name: &dupe,
	}); !errors.Is(err, groupusecases.ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}
}

// TestCreateGroup_rejectsWhenHardQuotaExceeded is a wi-160 T004.2 RED test for
// the SCL scenario "Hard Quota を超過したリソース作成は拒否される"
// (spec/contexts/tenancy.yaml).
func TestCreateGroup_rejectsWhenHardQuotaExceeded(t *testing.T) {
	ctx := context.Background()
	deps := newGroupDepsWithQuota(t, tenancydomain.DefaultTenantID, 1)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	}); err != nil {
		t.Fatalf("first CreateGroup: %v", err)
	}
	_, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "support", Now: now,
	})
	var qErr *tenancydomain.QuotaExceededError
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *domain.QuotaExceededError, got %v", err)
	}
	if qErr.Resource != tenancydomain.ResourceGroups {
		t.Fatalf("unexpected resource: %s", qErr.Resource)
	}
	list, err := groupusecases.ListGroups(ctx, deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected rejected create to not persist a second group, got %d groups", len(list))
	}
}

// TestDeleteGroup_decrementsQuotaUsage is a wi-160 T004.2 RED test: deleting a
// group must free its quota slot so a subsequent create at the same limit
// succeeds.
func TestDeleteGroup_decrementsQuotaUsage(t *testing.T) {
	ctx := context.Background()
	deps := newGroupDepsWithQuota(t, tenancydomain.DefaultTenantID, 1)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := groupusecases.DeleteGroup(ctx, deps, "operator", group.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "support", Now: now,
	}); err != nil {
		t.Fatalf("expected create to succeed after delete freed quota, got %v", err)
	}
}

func TestAddMemberRejectsCrossTenantUser(t *testing.T) {
	ctx := context.Background()
	deps, _ := newGroupDeps(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := groupusecases.AddMember(ctx, deps, "operator", group.ID, "user_other", now); !errors.Is(err, idmusecases.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound for cross-tenant user, got %v", err)
	}
}

func TestDeleteGroupCascadesMembership(t *testing.T) {
	ctx := context.Background()
	deps, events := newGroupDeps(t)
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	group, err := groupusecases.CreateGroup(ctx, deps, groupusecases.CreateGroupInput{
		ActorUserID: "operator", Name: "engineering", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := groupusecases.AddMember(ctx, deps, "operator", group.ID, "user_alice", now); err != nil {
		t.Fatal(err)
	}
	*events = (*events)[:0]
	if err := groupusecases.DeleteGroup(ctx, deps, "operator", group.ID, now); err != nil {
		t.Fatal(err)
	}
	got := eventTypes(*events)
	want := []string{"GroupMemberRemoved", "GroupDeleted"}
	if !slices.Equal(got, want) {
		t.Fatalf("delete events = %v, want %v", got, want)
	}
	if _, _, err := groupusecases.GetGroup(ctx, deps, group.ID); !errors.Is(err, groupusecases.ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound after delete, got %v", err)
	}
}
