package usecases_test

import (
	"context"
	"testing"
	"time"

	passwordmemory "github.com/ambi/idmagic/backend/authentication/internal/password/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/internal/password/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

func TestChangePasswordAutoClearsUpdatePasswordAction(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	idm := idmanagement.NewMemoryModule()
	lifecycle := idmanagement.LifecycleDeps{
		PasswordHasher: testing_passwords.NewHasher(), PasswordHistoryRepo: passwordmemory.NewPasswordHistoryRepository(),
	}
	events := &[]spec.DomainEvent{}
	emit := func(event spec.DomainEvent) error { *events = append(*events, event); return nil }
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	email := "carol@example.com"
	user, err := idm.AdminUsers(lifecycle, emit).CreateUser(ctx, idmanagement.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "carol", Password: "initial-password-9182",
		Email: &email, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 6, 20, 14, 0, 0, 0, time.UTC)
	if _, err := idm.UserLifecycleCommands(lifecycle, emit, "admin").SetUserRequiredAction(
		ctx, user.TenantID, user.ID, idmdomain.RequiredActionUpdatePassword, true, now,
	); err != nil {
		t.Fatal(err)
	}
	*events = (*events)[:0]

	updated, err := authusecases.ChangePassword(ctx, authusecases.ChangePasswordDeps{
		UserRepo:            idm.UserRepo,
		PasswordHasher:      lifecycle.PasswordHasher,
		PasswordHistoryRepo: lifecycle.PasswordHistoryRepo,
		Emit:                emit,
	}, authusecases.ChangePasswordInput{
		Sub:             user.ID,
		CurrentPassword: "initial-password-9182",
		NewPassword:     "fresh-pass-77182",
		Now:             now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if len(updated.Lifecycle.RequiredActions) != 0 {
		t.Fatalf("update_password should be auto-cleared, got %v", updated.Lifecycle.RequiredActions)
	}
	if updated.Lifecycle.PasswordChangedAt == nil {
		t.Fatal("password_changed_at was not set")
	}
	var sawCleared bool
	for _, e := range *events {
		if cleared, ok := e.(*idmdomain.UserRequiredActionCleared); ok {
			sawCleared = true
			if cleared.ActorUserID != user.ID {
				t.Fatalf("auto-clear actorUserID=%s, want self %s", cleared.ActorUserID, user.ID)
			}
		}
	}
	if !sawCleared {
		t.Fatal("UserRequiredActionCleared was not emitted on auto-clear")
	}
}
