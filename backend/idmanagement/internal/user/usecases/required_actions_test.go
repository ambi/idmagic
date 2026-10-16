package usecases_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newRequiredActionFixture(t *testing.T) (context.Context, userusecases.AdminUserDeps, *[]spec.DomainEvent, *userdomain.User) {
	t.Helper()
	ctx := testing_tenant.Default(context.Background())
	userRepo := usermemory.NewUserRepository()
	historyRepo := authentication.NewMemoryModule(nil).PasswordHistoryRepo
	hasher := testing_passwords.NewHasher()
	events := &[]spec.DomainEvent{}
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, PasswordHasher: hasher, PasswordHistoryRepo: historyRepo,
		Emit: func(event spec.DomainEvent) error { *events = append(*events, event); return nil },
	}
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	email := "carol@example.com"
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "carol", Password: "initial-password-9182",
		Email: &email, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	*events = (*events)[:0]
	return ctx, deps, events, user
}

//spec:covers REQ-IDMANAGEMENT-047: 必須操作の付与と解除は状態と対応イベントを変更し、同じ付与を繰り返しても重複しないことを固定する。
func TestSetAndClearUserRequiredAction(t *testing.T) {
	ctx, deps, events, user := newRequiredActionFixture(t)
	now := time.Date(2026, 6, 20, 13, 0, 0, 0, time.UTC)

	updated, err := userusecases.SetUserRequiredAction(
		ctx, deps, "admin", user.ID, idmdomain.RequiredActionUpdatePassword, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(updated.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
		t.Fatalf("required actions=%v, want update_password", updated.Lifecycle.RequiredActions)
	}
	if got := (*events)[len(*events)-1]; got.EventType() != "UserRequiredActionSet" {
		t.Fatalf("event=%s, want UserRequiredActionSet", got.EventType())
	}

	// 冪等: 二重付与してもイベントを増やさず単一のまま。
	before := len(*events)
	updated, err = userusecases.SetUserRequiredAction(
		ctx, deps, "admin", user.ID, idmdomain.RequiredActionUpdatePassword, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Lifecycle.RequiredActions) != 1 {
		t.Fatalf("required actions=%v, want single", updated.Lifecycle.RequiredActions)
	}
	if len(*events) != before {
		t.Fatalf("idempotent set emitted extra events: %d", len(*events)-before)
	}

	updated, err = userusecases.ClearUserRequiredAction(
		ctx, deps, "admin", user.ID, idmdomain.RequiredActionUpdatePassword, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Lifecycle.RequiredActions) != 0 {
		t.Fatalf("required actions=%v, want empty", updated.Lifecycle.RequiredActions)
	}
	if got := (*events)[len(*events)-1]; got.EventType() != "UserRequiredActionCleared" {
		t.Fatalf("event=%s, want UserRequiredActionCleared", got.EventType())
	}
}

//spec:covers REQ-IDMANAGEMENT-047: モジュール向け操作は実際の付与と解除だけ changed を返し、重複した操作と別テナントの参照ではイベントを増やさない。
func TestUserRequiredActionCommandsDesiredState(t *testing.T) {
	ctx, deps, events, user := newRequiredActionFixture(t)
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	commands := userusecases.UserLifecycleCommands{Deps: deps, Actor: "workflow"}
	for i, required := range []bool{true, true, false, false} {
		changed, err := commands.SetUserRequiredAction(ctx, user.TenantID, user.ID, idmdomain.RequiredActionUpdatePassword, required, now)
		if err != nil || changed != (i%2 == 0) {
			t.Fatalf("operation %d = %v, %v", i, changed, err)
		}
	}
	if len(*events) != 2 {
		t.Fatalf("event count = %d", len(*events))
	}
	if changed, err := commands.SetUserRequiredAction(ctx, "foreign-tenant", user.ID, idmdomain.RequiredActionUpdatePassword, true, now); changed || err == nil {
		t.Fatalf("foreign = %v, %v", changed, err)
	}
	if len(*events) != 2 {
		t.Fatalf("foreign event count = %d", len(*events))
	}
}

func TestSetUserRequiredActionRejectsUnknownAction(t *testing.T) {
	ctx, deps, _, user := newRequiredActionFixture(t)
	_, err := userusecases.SetUserRequiredAction(
		ctx, deps, "admin", user.ID, idmdomain.RequiredAction("teleport"), time.Now().UTC(),
	)
	if err == nil {
		t.Fatal("expected error for unknown required action")
	}
}
