package usecases_test

// 主要ユースケース追跡: REQ-SOURCING-002。

import (
	"context"
	"errors"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	scimmemory "github.com/ambi/idmagic/backend/sourcing/scim/db_memory"
	scimdomain "github.com/ambi/idmagic/backend/sourcing/scim/domain"
	"github.com/ambi/idmagic/backend/sourcing/scim/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func newScimUsecases() (*usecases.Usecases, *usermemory.UserRepository) {
	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	scimRepo := scimmemory.NewScimRepository()
	return usecases.NewUsecases(scimRepo, userRepo, groupRepo, func(spec.DomainEvent) {}), userRepo
}

const scimTenant = tenancydomain.DefaultTenantID

func TestCreateUserRejectsDuplicateUsername(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "dup@example.com"}); err != nil {
		t.Fatal(err)
	}
	_, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "dup@example.com"})
	if !errors.Is(err, usecases.ErrDuplicate) {
		t.Fatalf("err=%v, want ErrDuplicate", err)
	}
}

func TestGetUserNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.GetUser(ctx, scimTenant, "no-such-id"); !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestCreateUserWithEnterpriseExtensionAndManager(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	manager, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "manager@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	managerScimID := manager["id"].(string)

	created, err := u.CreateUser(ctx, scimTenant, map[string]any{
		"userName": "report@example.com",
		scimdomain.EnterpriseUserSchemaURN: map[string]any{
			"employeeNumber": "E-1", "department": "R&D",
			"manager": map[string]any{"value": managerScimID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ext, ok := created[scimdomain.EnterpriseUserSchemaURN].(map[string]any)
	if !ok {
		t.Fatalf("expected enterprise extension in response, got %#v", created)
	}
	if ext["employeeNumber"] != "E-1" || ext["department"] != "R&D" {
		t.Fatalf("ext=%#v", ext)
	}
	managerRef, ok := ext["manager"].(map[string]any)
	if !ok || managerRef["value"] != managerScimID {
		t.Fatalf("manager=%#v, want value=%s", ext["manager"], managerScimID)
	}
}

func TestCreateUserRejectsUnresolvableManager(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	_, err := u.CreateUser(ctx, scimTenant, map[string]any{
		"userName": "report@example.com",
		scimdomain.EnterpriseUserSchemaURN: map[string]any{
			"manager": map[string]any{"value": "no-such-manager"},
		},
	})
	if _, ok := errors.AsType[*scimdomain.MutationError](err); !ok {
		t.Fatalf("err=%v, want *scimdomain.MutationError", err)
	}
}

func TestUpdateUserFullReplace(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	created, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "alice@example.com", "active": true})
	if err != nil {
		t.Fatal(err)
	}
	scimID := created["id"].(string)

	updated, err := u.UpdateUser(ctx, scimTenant, scimID, map[string]any{
		"userName": "alice2@example.com", "active": false,
		"name": map[string]any{"givenName": "Alice", "familyName": "Two", "formatted": "Alice Two"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated["userName"] != "alice2@example.com" || updated["active"] != false {
		t.Fatalf("updated=%#v", updated)
	}
}

func TestUpdateUserNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.UpdateUser(ctx, scimTenant, "no-such-id", map[string]any{"userName": "x@example.com"}); !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestUpdateUserRejectsRenameToExistingUsername(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "taken@example.com"}); err != nil {
		t.Fatal(err)
	}
	created, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "movable@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = u.UpdateUser(ctx, scimTenant, created["id"].(string), map[string]any{"userName": "taken@example.com"})
	if !errors.Is(err, usecases.ErrDuplicate) {
		t.Fatalf("err=%v, want ErrDuplicate", err)
	}
}

func patchOp(op, path string, value any) map[string]any {
	m := map[string]any{"op": op, "path": path}
	if value != nil {
		m["value"] = value
	}
	return m
}

func patchBody(ops ...map[string]any) map[string]any {
	raw := make([]any, len(ops))
	for i, op := range ops {
		raw[i] = op
	}
	return map[string]any{"Operations": raw}
}

func TestPatchUserOperations(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	created, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "patchme@example.com", "active": true})
	if err != nil {
		t.Fatal(err)
	}
	scimID := created["id"].(string)

	t.Run("replace userName", func(t *testing.T) {
		out, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "userName", "patched@example.com")))
		if err != nil {
			t.Fatal(err)
		}
		if out["userName"] != "patched@example.com" {
			t.Fatalf("userName=%v", out["userName"])
		}
	})

	t.Run("replace and remove givenName", func(t *testing.T) {
		if _, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "name.givenName", "Pat"))); err != nil {
			t.Fatal(err)
		}
		out, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("remove", "name.givenName", nil)))
		if err != nil {
			t.Fatal(err)
		}
		name := out["name"].(map[string]any)
		if name["givenName"] != "" {
			t.Fatalf("givenName=%v, want cleared", name["givenName"])
		}
	})

	t.Run("remove name clears the whole name object", func(t *testing.T) {
		if _, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "name", map[string]any{"givenName": "G", "familyName": "F", "formatted": "G F"}))); err != nil {
			t.Fatal(err)
		}
		out, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("remove", "name", nil)))
		if err != nil {
			t.Fatal(err)
		}
		name := out["name"].(map[string]any)
		if name["givenName"] != "" || name["familyName"] != "" || name["formatted"] != "" {
			t.Fatalf("name=%#v, want fully cleared", name)
		}
	})

	t.Run("active toggles and remove reactivates", func(t *testing.T) {
		if _, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "active", false))); err != nil {
			t.Fatal(err)
		}
		out, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("remove", "active", nil)))
		if err != nil {
			t.Fatal(err)
		}
		if out["active"] != true {
			t.Fatalf("active=%v, want true after remove", out["active"])
		}
	})

	t.Run("emails replace and remove", func(t *testing.T) {
		out, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "emails", []any{
			map[string]any{"value": "new@example.com", "primary": true},
		})))
		if err != nil {
			t.Fatal(err)
		}
		emails := out["emails"].([]map[string]any)
		if len(emails) != 1 || emails[0]["value"] != "new@example.com" {
			t.Fatalf("emails=%#v", emails)
		}
		out, err = u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("remove", "emails", nil)))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := out["emails"]; ok {
			t.Fatalf("expected emails omitted after remove, got %#v", out["emails"])
		}
	})

	t.Run("employeeNumber and department set and remove", func(t *testing.T) {
		out, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(
			patchOp("replace", "employeeNumber", "E-9"),
			patchOp("replace", "department", "Ops"),
		))
		if err != nil {
			t.Fatal(err)
		}
		ext := out[scimdomain.EnterpriseUserSchemaURN].(map[string]any)
		if ext["employeeNumber"] != "E-9" || ext["department"] != "Ops" {
			t.Fatalf("ext=%#v", ext)
		}
		out, err = u.PatchUser(ctx, scimTenant, scimID, patchBody(
			patchOp("remove", "employeeNumber", nil), patchOp("remove", "department", nil),
		))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := out[scimdomain.EnterpriseUserSchemaURN]; ok {
			t.Fatalf("expected extension omitted once empty, got %#v", out)
		}
	})

	t.Run("manager set and remove", func(t *testing.T) {
		manager, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "boss@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		out, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "manager", map[string]any{"value": manager["id"]})))
		if err != nil {
			t.Fatal(err)
		}
		ext := out[scimdomain.EnterpriseUserSchemaURN].(map[string]any)
		if ext["manager"].(map[string]any)["value"] != manager["id"] {
			t.Fatalf("manager=%#v", ext["manager"])
		}
		out, err = u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("remove", "manager", nil)))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := out[scimdomain.EnterpriseUserSchemaURN]; ok {
			t.Fatalf("expected extension omitted once manager cleared, got %#v", out)
		}
	})

	t.Run("unresolvable manager is rejected before any mutation", func(t *testing.T) {
		_, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "manager", map[string]any{"value": "ghost"})))
		if _, ok := errors.AsType[*scimdomain.MutationError](err); !ok {
			t.Fatalf("err=%v, want *scimdomain.MutationError", err)
		}
	})

	t.Run("rejects a rename to an existing userName", func(t *testing.T) {
		if _, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "taken2@example.com"}); err != nil {
			t.Fatal(err)
		}
		_, err := u.PatchUser(ctx, scimTenant, scimID, patchBody(patchOp("replace", "userName", "taken2@example.com")))
		if !errors.Is(err, usecases.ErrDuplicate) {
			t.Fatalf("err=%v, want ErrDuplicate", err)
		}
	})
}

func TestPatchUserNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	_, err := u.PatchUser(ctx, scimTenant, "no-such-id", patchBody(patchOp("replace", "active", true)))
	if !errors.Is(err, usecases.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}

func TestDeleteUserSoftDeletes(t *testing.T) {
	ctx := context.Background()
	u, userRepo := newScimUsecases()
	created, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "gone@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := u.DeleteUser(ctx, scimTenant, created["id"].(string)); err != nil {
		t.Fatal(err)
	}
	users, err := userRepo.FindAll(ctx, scimTenant)
	if err != nil {
		t.Fatal(err)
	}
	var found *userdomain.User
	for _, usr := range users {
		if usr.PreferredUsername == "gone@example.com" {
			found = usr
		}
	}
	if found == nil || found.Lifecycle.Status != idmdomain.UserStatusPendingDeletion {
		t.Fatalf("user=%+v, want status PendingDeletion", found)
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if err := u.DeleteUser(ctx, scimTenant, "no-such-id"); err == nil {
		t.Fatal("expected an error for a missing user")
	}
}

func TestListUsersFiltersByUsername(t *testing.T) {
	ctx := context.Background()
	u, _ := newScimUsecases()
	if _, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "findme@example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := u.CreateUser(ctx, scimTenant, map[string]any{"userName": "other@example.com"}); err != nil {
		t.Fatal(err)
	}
	result, err := u.ListUsers(ctx, scimTenant, usecases.ListQuery{Filter: `userName eq "findme@example.com"`})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 || result.Items[0]["userName"] != "findme@example.com" {
		t.Fatalf("result=%+v", result)
	}
}
