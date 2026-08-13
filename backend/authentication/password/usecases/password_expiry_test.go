package usecases

import (
	"context"
	"slices"
	"testing"
	"time"

	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// REQ-AUTHENTICATION-024
func TestEnforcePasswordExpiry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	changedAt := now.AddDate(0, 0, -100)
	policyUpdatedAt := now.AddDate(0, 0, -200)
	expiring := passworddomain.PasswordPolicySnapshot{MaxAgeDays: 90, PolicyUpdatedAt: &policyUpdatedAt}

	t.Run("expired password gains the update_password required action", func(t *testing.T) {
		user := &userdomain.User{
			ID: "user-1", PasswordHash: "$argon2id$hash",
			Lifecycle: userdomain.UserLifecycle{PasswordChangedAt: &changedAt},
		}
		if !EnforcePasswordExpiry(user, expiring, now) {
			t.Fatal("expected the expired password to be enforced")
		}
		if !slices.Contains(user.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
			t.Fatalf("required actions=%v, want update_password", user.Lifecycle.RequiredActions)
		}
	})

	t.Run("passwordless user is never forced", func(t *testing.T) {
		user := &userdomain.User{
			ID:        "user-2",
			Lifecycle: userdomain.UserLifecycle{PasswordChangedAt: &changedAt},
		}
		if EnforcePasswordExpiry(user, expiring, now) {
			t.Fatal("a user without a password credential must not be forced to change it")
		}
		if len(user.Lifecycle.RequiredActions) != 0 {
			t.Fatalf("required actions=%v, want none", user.Lifecycle.RequiredActions)
		}
	})

	t.Run("already required action is not duplicated", func(t *testing.T) {
		user := &userdomain.User{
			ID: "user-3", PasswordHash: "$argon2id$hash",
			Lifecycle: userdomain.UserLifecycle{
				PasswordChangedAt: &changedAt,
				RequiredActions:   []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword},
			},
		}
		if EnforcePasswordExpiry(user, expiring, now) {
			t.Fatal("an already-required action must not report a change")
		}
		if len(user.Lifecycle.RequiredActions) != 1 {
			t.Fatalf("required actions=%v, want exactly one", user.Lifecycle.RequiredActions)
		}
	})

	t.Run("policy without expiry leaves the user alone", func(t *testing.T) {
		user := &userdomain.User{
			ID: "user-4", PasswordHash: "$argon2id$hash",
			Lifecycle: userdomain.UserLifecycle{PasswordChangedAt: &changedAt},
		}
		snap := passworddomain.PasswordPolicySnapshot{MinLength: 12, MaxLength: 128, HistoryDepth: 5}
		if EnforcePasswordExpiry(user, snap, now) {
			t.Fatal("expiry is opt-in; a policy without max age must not force a change")
		}
	})

	t.Run("nil user is a no-op", func(t *testing.T) {
		if EnforcePasswordExpiry(nil, expiring, now) {
			t.Fatal("nil user must not report a change")
		}
	})
}

// REQ-TENANCY-019: every password-setting path reads the same tenant-resolved policy.
func TestResolveTenantPolicy(t *testing.T) {
	t.Parallel()

	minLength := 20
	maxAge := 45
	updatedAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	tenant := &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Acme",
		Status:                  tenancydomain.TenantStatusActive,
		PasswordPolicyOverride:  &tenancydomain.PasswordPolicyOverride{MinLength: &minLength, MaxAgeDays: &maxAge},
		PasswordPolicyUpdatedAt: &updatedAt,
		CreatedAt:               updatedAt, UpdatedAt: updatedAt,
	}
	repo := tenancymemory.NewTenantRepository()
	if err := repo.Save(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}

	assertOverride := func(t *testing.T, snap passworddomain.PasswordPolicySnapshot) {
		t.Helper()
		if snap.MinLength != 20 || snap.MaxAgeDays != 45 {
			t.Fatalf("snapshot=%+v, want tenant override applied", snap)
		}
		if snap.MaxLength != PasswordPolicyMaxLength || snap.HistoryDepth != PasswordPolicyHistoryDepth {
			t.Fatalf("snapshot=%+v, want global defaults for unset fields", snap)
		}
		if snap.PolicyUpdatedAt == nil || !snap.PolicyUpdatedAt.Equal(updatedAt) {
			t.Fatalf("PolicyUpdatedAt=%v, want %v", snap.PolicyUpdatedAt, updatedAt)
		}
	}

	t.Run("tenant resolved by middleware is used as is", func(t *testing.T) {
		ctx := tenancy.WithTenant(context.Background(), tenant, "https://idmagic.test", "")
		assertOverride(t, ResolveTenantPolicy(ctx, nil))
	})

	t.Run("without a resolved tenant the repository is consulted", func(t *testing.T) {
		assertOverride(t, ResolveTenantPolicy(context.Background(), repo))
	})

	t.Run("no tenant and no repository falls back to the global default", func(t *testing.T) {
		snap := ResolveTenantPolicy(context.Background(), nil)
		if snap.MinLength != PasswordPolicyMinLength || snap.MaxAgeDays != 0 {
			t.Fatalf("snapshot=%+v, want global defaults", snap)
		}
	})
}
