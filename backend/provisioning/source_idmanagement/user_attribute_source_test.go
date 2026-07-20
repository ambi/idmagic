package source_idmanagement_test

import (
	"context"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	identitysource "github.com/ambi/idmagic/backend/provisioning/source_idmanagement"
)

func TestUserAttributeSource_ResolveAttributes_MapsCoreFields(t *testing.T) {
	userRepo := usermemory.NewUserRepository()
	ctx := context.Background()
	name := "Alice Example"
	given := "Alice"
	family := "Example"
	email := "alice@example.com"
	user := &userdomain.User{
		ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash",
		Name: &name, GivenName: &given, FamilyName: &family, Email: &email,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
	}
	if err := userRepo.Save(ctx, user); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	source := &identitysource.UserAttributeSource{UserRepo: userRepo}
	attrs, exists, err := source.ResolveAttributes(ctx, "tenant-a", domain.SourceTypeUser, "user-1")
	if err != nil {
		t.Fatalf("ResolveAttributes() error = %v", err)
	}
	if !exists {
		t.Fatal("ResolveAttributes() exists = false, want true")
	}
	want := map[string]any{
		"id": "user-1", "preferred_username": "alice", "display_name": "Alice Example",
		"given_name": "Alice", "family_name": "Example", "email": "alice@example.com", "active": true,
	}
	for k, v := range want {
		if attrs[k] != v {
			t.Errorf("attrs[%q] = %v, want %v", k, attrs[k], v)
		}
	}
}

func TestUserAttributeSource_ResolveAttributes_NotFoundOrOtherTenant(t *testing.T) {
	userRepo := usermemory.NewUserRepository()
	ctx := context.Background()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}}
	_ = userRepo.Save(ctx, user)

	source := &identitysource.UserAttributeSource{UserRepo: userRepo}
	_, exists, err := source.ResolveAttributes(ctx, "tenant-a", domain.SourceTypeUser, "missing")
	if err != nil || exists {
		t.Errorf("ResolveAttributes() for missing user = (exists=%v, err=%v), want (false, nil)", exists, err)
	}
	_, exists, err = source.ResolveAttributes(ctx, "tenant-b", domain.SourceTypeUser, "user-1")
	if err != nil || exists {
		t.Errorf("ResolveAttributes() across tenants = (exists=%v, err=%v), want (false, nil)", exists, err)
	}
}

func TestUserAttributeSource_ResolveAttributes_DisabledUserIsInactive(t *testing.T) {
	userRepo := usermemory.NewUserRepository()
	ctx := context.Background()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled}}
	_ = userRepo.Save(ctx, user)

	source := &identitysource.UserAttributeSource{UserRepo: userRepo}
	attrs, exists, err := source.ResolveAttributes(ctx, "tenant-a", domain.SourceTypeUser, "user-1")
	if err != nil || !exists {
		t.Fatalf("ResolveAttributes() = (exists=%v, err=%v)", exists, err)
	}
	if attrs["active"] != false {
		t.Errorf("attrs[active] = %v, want false for a disabled user", attrs["active"])
	}
}

// TestUserAttributeSource_ResolveAttributes_DeletedUserStillResolvesAsInactive
// pins wi-45 T008's E2E finding: a deactivate/delete delivery created for
// TriggerUserDeleted must still resolve attributes for the now-tombstoned
// user, or DeprovisionPolicy.OnDelete=deactivate (the default) would silently
// never reach the downstream (FindBySub excludes deleted users; the fix uses
// FindBySubIncludingDeleted).
func TestUserAttributeSource_ResolveAttributes_DeletedUserStillResolvesAsInactive(t *testing.T) {
	userRepo := usermemory.NewUserRepository()
	ctx := context.Background()
	user := &userdomain.User{
		ID: "user-1", TenantID: "tenant-a", PreferredUsername: "deleted:user-1", PasswordHash: "hash",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDeleted},
	}
	if err := userRepo.Save(ctx, user); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	source := &identitysource.UserAttributeSource{UserRepo: userRepo}
	attrs, exists, err := source.ResolveAttributes(ctx, "tenant-a", domain.SourceTypeUser, "user-1")
	if err != nil {
		t.Fatalf("ResolveAttributes() error = %v", err)
	}
	if !exists {
		t.Fatal("ResolveAttributes() exists = false for a tombstoned user, want true (needed to deliver deactivate)")
	}
	if attrs["active"] != false {
		t.Errorf("attrs[active] = %v, want false for a deleted user", attrs["active"])
	}
}
