package usecases_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/http/server_http/sourcingdeps"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing"
	scimmemory "github.com/ambi/idmagic/backend/sourcing/internal/scim/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type ownerNotifications struct{ calls []string }

func (n *ownerNotifications) NotifyUserMutation(_ context.Context, _, _ string, trigger userports.ProvisioningTrigger, _ time.Time) error {
	n.calls = append(n.calls, string(trigger))
	return nil
}

func (n *ownerNotifications) NotifyGroupMutation(_ context.Context, _, _ string, trigger groupports.ProvisioningTrigger, _ time.Time) error {
	n.calls = append(n.calls, string(trigger))
	return nil
}

func scimOwnerWrites() (*usecases.Usecases, *[]string, *ownerNotifications, tenantports.QuotaRepository) {
	identity := idmanagement.NewMemoryModule()
	refs := scimmemory.NewScimRepository()
	notifications := &ownerNotifications{}
	identity.ProvisioningNotifier, identity.GroupProvisioningNotifier = notifications, notifications
	quota := tenancy.NewMemoryModule().QuotaRepo
	var events []string
	emit := func(event spec.DomainEvent) { events = append(events, event.EventType()) }
	writes := sourcingdeps.NewWrites(identity, idmanagement.LifecycleDeps{QuotaRepo: quota}, sourcing.Module{ScimRepo: refs}, emit)
	u := usecases.NewUsecases(refs, identity.UserRepo, identity.GroupRepo, scimUserLifecycle(identity.UserRepo), emit, writes)
	return u, &events, notifications, quota
}

//spec:covers REQ-SOURCING-002: SCIM の作成と属性変更が、所有者の UserCreated と UserUpdated を発行する。
func TestSCIMUserWritesUseIdentityCommands(t *testing.T) {
	u, events, notifications, quota := scimOwnerWrites()
	ctx := context.Background()
	created, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := u.PatchUser(ctx, scimTenant, created["id"].(string), patchBody(patchOp("replace", "name.formatted", "Alice"))); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(*events, []string{"UserCreated", "UserUpdated"}) {
		t.Fatalf("events=%v, want UserCreated and UserUpdated", *events)
	}
	if !slices.Equal(notifications.calls, []string{"user_created", "user_attributes_changed"}) {
		t.Fatalf("notifications=%v", notifications.calls)
	}
	usage, err := quota.GetUsage(ctx, scimTenant)
	if err != nil || usage.Users != 1 {
		t.Fatalf("usage=%+v error=%v", usage, err)
	}
	t.Run("initially disabled", func(t *testing.T) {
		u, events, notifications, _ := scimOwnerWrites()
		created, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "disabled", "active": false})
		if err != nil || created["active"] != false {
			t.Fatalf("created=%v error=%v", created, err)
		}
		if !slices.Equal(*events, []string{"UserCreated"}) || !slices.Equal(notifications.calls, []string{"user_created"}) {
			t.Fatalf("events=%v notifications=%v", *events, notifications.calls)
		}
	})
}

//spec:covers REQ-SOURCING-005: SCIM の Group の作成、変更、所属の追加と削除が所有者のイベントを発行する。
func TestSCIMGroupWritesUseIdentityCommands(t *testing.T) {
	u, events, notifications, quota := scimOwnerWrites()
	ctx := context.Background()
	user, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "member"})
	if err != nil {
		t.Fatal(err)
	}
	*events = nil
	notifications.calls = nil
	created, err := u.CreateGroup(ctx, scimTenant, map[string]any{
		"displayName": "engineering", "members": []any{map[string]any{"value": user["id"]}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := u.UpdateGroup(ctx, scimTenant, created["id"].(string), map[string]any{"displayName": "platform"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"GroupCreated", "GroupMemberAdded", "GroupUpdated", "GroupMemberRemoved"}
	if !slices.Equal(*events, want) {
		t.Fatalf("events=%v, want %v", *events, want)
	}
	if !slices.Equal(notifications.calls, []string{"group_created", "group_membership_changed", "group_attributes_changed", "group_membership_changed"}) {
		t.Fatalf("notifications=%v", notifications.calls)
	}
	usage, err := quota.GetUsage(ctx, scimTenant)
	if err != nil || usage.Groups != 1 {
		t.Fatalf("usage=%+v error=%v", usage, err)
	}
}

//spec:covers REQ-SOURCING-005: 複数の所属の削除は、指定されたすべての所属を消す。
func TestSCIMGroupRemovesAllRequestedMembers(t *testing.T) {
	u, events, notifications, _ := scimOwnerWrites()
	ctx := t.Context()
	var memberIDs []string
	for _, name := range []string{"alice", "bob"} {
		user, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": name})
		if err != nil {
			t.Fatal(err)
		}
		memberIDs = append(memberIDs, user["id"].(string))
	}
	group, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "team", "members": memberOp(memberIDs...)})
	if err != nil {
		t.Fatal(err)
	}
	*events, notifications.calls = nil, nil
	updated, err := u.PatchGroup(ctx, scimTenant, group["id"].(string), patchBody(patchOp("remove", "members", memberOp(memberIDs...))))
	if err != nil {
		t.Fatal(err)
	}
	if members := groupMembers(t, updated); len(members) != 0 {
		t.Fatalf("members=%v, want none", members)
	}
	if !slices.Equal(*events, []string{"GroupMemberRemoved", "GroupMemberRemoved"}) || !slices.Equal(notifications.calls, []string{"group_membership_changed", "group_membership_changed"}) {
		t.Fatalf("events=%v notifications=%v", *events, notifications.calls)
	}
}
