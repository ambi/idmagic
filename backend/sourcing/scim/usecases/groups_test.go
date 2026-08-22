package usecases_test

import (
	"context"
	"errors"
	"testing"

	scimdomain "github.com/ambi/idmagic/backend/sourcing/scim/domain"
	"github.com/ambi/idmagic/backend/sourcing/scim/usecases"
)

func memberOp(scimIDs ...string) []any {
	out := make([]any, len(scimIDs))
	for i, id := range scimIDs {
		out[i] = map[string]any{"value": id}
	}
	return out
}

func groupMembers(t *testing.T, group map[string]any) []string {
	t.Helper()
	raw, ok := group["members"].([]map[string]any)
	if !ok {
		t.Fatalf("members is not []map[string]any: %#v", group["members"])
	}
	out := make([]string, len(raw))
	for i, m := range raw {
		out[i] = m["value"].(string)
	}
	return out
}

func TestCreateGroupWithMembers(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	alice, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "alice@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := u.CreateGroup(ctx, scimTenant, map[string]any{
		"displayName": "Engineers", "members": memberOp(alice["id"].(string)),
	})
	if err != nil {
		t.Fatal(err)
	}
	members := groupMembers(t, group)
	if len(members) != 1 || members[0] != alice["id"] {
		t.Fatalf("members=%v", members)
	}
}

func TestCreateGroupRejectsDuplicateDisplayName(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Engineers"}); err != nil {
		t.Fatal(err)
	}
	_, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Engineers"})
	if !errors.Is(err, usecases.ErrDuplicate) {
		t.Fatalf("err=%v, want ErrDuplicate", err)
	}
}

func TestCreateGroupRejectsUnresolvableMember(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	_, err := u.CreateGroup(ctx, scimTenant, map[string]any{
		"displayName": "Ghosts", "members": memberOp("no-such-user"),
	})
	if _, ok := errors.AsType[*scimdomain.MutationError](err); !ok {
		t.Fatalf("err=%v, want *scimdomain.MutationError", err)
	}
}

func TestGetGroupNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.GetGroup(ctx, scimTenant, "no-such-id"); !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestUpdateGroupRenameAndReplaceMembers(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	alice, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "alice@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "bob@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := u.CreateGroup(ctx, scimTenant, map[string]any{
		"displayName": "Team A", "members": memberOp(alice["id"].(string)),
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := u.UpdateGroup(ctx, scimTenant, group["id"].(string), map[string]any{
		"displayName": "Team B", "members": memberOp(bob["id"].(string)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated["displayName"] != "Team B" {
		t.Fatalf("displayName=%v", updated["displayName"])
	}
	members := groupMembers(t, updated)
	if len(members) != 1 || members[0] != bob["id"] {
		t.Fatalf("members=%v, want only bob", members)
	}
}

func TestUpdateGroupNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.UpdateGroup(ctx, scimTenant, "no-such-id", map[string]any{"displayName": "X"}); !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestUpdateGroupRejectsRenameToExistingDisplayName(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Taken"}); err != nil {
		t.Fatal(err)
	}
	group, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Movable"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = u.UpdateGroup(ctx, scimTenant, group["id"].(string), map[string]any{"displayName": "Taken"})
	if !errors.Is(err, usecases.ErrDuplicate) {
		t.Fatalf("err=%v, want ErrDuplicate", err)
	}
}

func TestPatchGroupDisplayNameAndMembers(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	alice, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "alice@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	bob, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "bob@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Patchable"})
	if err != nil {
		t.Fatal(err)
	}
	scimID := group["id"].(string)

	t.Run("replace displayName", func(t *testing.T) {
		out, err := u.PatchGroup(ctx, scimTenant, scimID, patchBody(patchOp("replace", "displayName", "Renamed")))
		if err != nil {
			t.Fatal(err)
		}
		if out["displayName"] != "Renamed" {
			t.Fatalf("displayName=%v", out["displayName"])
		}
	})

	t.Run("add members", func(t *testing.T) {
		out, err := u.PatchGroup(ctx, scimTenant, scimID, patchBody(patchOp("add", "members", memberOp(alice["id"].(string), bob["id"].(string)))))
		if err != nil {
			t.Fatal(err)
		}
		members := groupMembers(t, out)
		if len(members) != 2 {
			t.Fatalf("members=%v", members)
		}
	})

	t.Run("remove one member is lenient about already-absent members", func(t *testing.T) {
		out, err := u.PatchGroup(ctx, scimTenant, scimID, patchBody(patchOp("remove", "members", memberOp(alice["id"].(string), "already-gone"))))
		if err != nil {
			t.Fatal(err)
		}
		members := groupMembers(t, out)
		if len(members) != 1 || members[0] != bob["id"] {
			t.Fatalf("members=%v, want only bob", members)
		}
	})

	t.Run("replace members resets the set", func(t *testing.T) {
		out, err := u.PatchGroup(ctx, scimTenant, scimID, patchBody(patchOp("replace", "members", memberOp(alice["id"].(string)))))
		if err != nil {
			t.Fatal(err)
		}
		members := groupMembers(t, out)
		if len(members) != 1 || members[0] != alice["id"] {
			t.Fatalf("members=%v, want only alice", members)
		}
	})

	t.Run("displayName cannot be removed", func(t *testing.T) {
		_, err := u.PatchGroup(ctx, scimTenant, scimID, patchBody(patchOp("remove", "displayName", nil)))
		if _, ok := errors.AsType[*scimdomain.MutationError](err); !ok {
			t.Fatalf("err=%v, want *scimdomain.MutationError", err)
		}
	})

	t.Run("rejects renaming to an existing displayName", func(t *testing.T) {
		if _, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Collides"}); err != nil {
			t.Fatal(err)
		}
		_, err := u.PatchGroup(ctx, scimTenant, scimID, patchBody(patchOp("replace", "displayName", "Collides")))
		if !errors.Is(err, usecases.ErrDuplicate) {
			t.Fatalf("err=%v, want ErrDuplicate", err)
		}
	})
}

func TestPatchGroupNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	_, err := u.PatchGroup(ctx, scimTenant, "no-such-id", patchBody(patchOp("replace", "displayName", "X")))
	if !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestDeleteGroup(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	group, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Disposable"})
	if err != nil {
		t.Fatal(err)
	}
	if err := u.DeleteGroup(ctx, scimTenant, group["id"].(string)); err != nil {
		t.Fatal(err)
	}
	if _, err := u.GetGroup(ctx, scimTenant, group["id"].(string)); !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound after delete", err)
	}
}

func TestDeleteGroupNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if err := u.DeleteGroup(ctx, scimTenant, "no-such-id"); err == nil {
		t.Fatal("expected an error for a missing group")
	}
}

func TestListGroupsFiltersByDisplayName(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "FindMe"}); err != nil {
		t.Fatal(err)
	}
	if _, err := u.CreateGroup(ctx, scimTenant, map[string]any{"displayName": "Other"}); err != nil {
		t.Fatal(err)
	}
	result, err := u.ListGroups(ctx, scimTenant, usecases.ListQuery{Filter: `displayName eq "FindMe"`})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Items[0]["displayName"] != "FindMe" {
		t.Fatalf("result=%+v", result)
	}
}
