package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/adapters/persistence/memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/adapters/persistence/memory"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestCreateUpdateAndDisableUser(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := crypto.NewArgon2idPasswordHasher()
	var events []spec.DomainEvent
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, PasswordHasher: hasher, PasswordHistoryRepo: historyRepo,
		Emit: func(event spec.DomainEvent) error { events = append(events, event); return nil },
	}
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	email := "bob@example.com"
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "bob", Password: "initial-password-9182",
		Email: &email, Roles: []string{"support", "support"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(user.Roles) != 1 || user.Roles[0] != "support" {
		t.Fatalf("roles=%v", user.Roles)
	}
	if events[0].EventType() != "UserCreated" {
		t.Fatalf("event=%s", events[0].EventType())
	}
	updatedName := "Bob"
	roles := []string{"admin", "support"}
	user, err = userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, Name: &updatedName, Roles: &roles, Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Name == nil || *user.Name != "Bob" || len(user.Roles) != 2 {
		t.Fatalf("updated user=%+v", user)
	}
	user, err = userusecases.SetUserDisabled(
		ctx, deps, "admin", user.ID, true, now.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatal("status was not set to disabled")
	}
	if got := events[len(events)-1].EventType(); got != "UserDisabled" {
		t.Fatalf("last event=%s", got)
	}
	user, err = userusecases.SetUserDisabled(
		ctx, deps, "admin", user.ID, false, now.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if user.Lifecycle.Status != idmdomain.UserStatusActive {
		t.Fatal("status was not cleared to active")
	}
	if got := events[len(events)-1].EventType(); got != "UserEnabled" {
		t.Fatalf("last event=%s", got)
	}
}

func TestUpdateUserExtraFieldsAndNoop(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, PasswordHasher: crypto.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
	}
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "charlie", Password: "initial-password-9182", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "taken", Password: "initial-password-9182", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	givenName := "Charlie"
	familyName := "Example"
	email := "charlie@example.com"
	verified := true
	updated, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID,
		GivenName: &givenName, FamilyName: &familyName, Email: &email, EmailVerified: &verified,
		Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.GivenName == nil || *updated.GivenName != givenName || updated.FamilyName == nil || *updated.FamilyName != familyName ||
		updated.Email == nil || *updated.Email != email || !updated.EmailVerified {
		t.Fatalf("updated user=%+v", updated)
	}

	noChange, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if noChange.ID != user.ID {
		t.Fatalf("no-op returned %+v", noChange)
	}

	taken := "taken"
	if _, err := userusecases.UpdateUser(ctx, deps, userusecases.UpdateUserInput{
		ActorUserID: "admin", Sub: user.ID, PreferredUsername: &taken,
	}); !errors.Is(err, userusecases.ErrUsernameConflict) {
		t.Fatalf("expected ErrUsernameConflict, got %v", err)
	}
}

func TestCreateUserRejectsDuplicateUsername(t *testing.T) {
	repo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	repo.Seed(&userdomain.User{
		ID: "existing", PreferredUsername: "bob", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	_, err := userusecases.CreateUser(context.Background(), userusecases.AdminUserDeps{
		UserRepo: repo, PasswordHasher: crypto.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
	}, userusecases.CreateUserInput{
		PreferredUsername: "bob", Password: "initial-password-9182",
	})
	if !errors.Is(err, userusecases.ErrUsernameConflict) {
		t.Fatalf("error=%v, want ErrUsernameConflict", err)
	}
}

func TestDeleteUserAnonymizesAndCascades(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	consentRepo := oauth2memory.NewConsentRepository()
	refreshStore := oauth2memory.NewRefreshTokenStore()
	deviceStore := oauth2memory.NewDeviceCodeStore()
	sessionStore := sessionmemory.NewSessionStore()
	mfaRepo := totpmemory.NewMfaFactorRepository()
	hasher := crypto.NewArgon2idPasswordHasher()
	var events []spec.DomainEvent
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, ConsentRepo: consentRepo, RefreshStore: refreshStore,
		DeviceCodeStore: deviceStore, SessionStore: sessionStore, MfaFactorRepo: mfaRepo,
		PasswordHasher: hasher, PasswordHistoryRepo: historyRepo,
		Emit: func(event spec.DomainEvent) error { events = append(events, event); return nil },
	}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	user, err := userusecases.CreateUser(ctx, deps, userusecases.CreateUserInput{
		ActorUserID: "admin", PreferredUsername: "alice", Password: "initial-password-9182",
		Roles: []string{"support"}, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Seed cascade artifacts.
	_ = consentRepo.Save(ctx, tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: user.ID, ClientID: "client-a",
		Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.AddDate(1, 0, 0),
	})
	_ = refreshStore.Save(ctx, &oauthdomain.RefreshTokenRecord{
		ID: "rt-1", TenantID: tenancydomain.DefaultTenantID, Hash: "hash-1",
		FamilyID: "fam-1", ClientID: "client-a", UserID: user.ID,
		Scopes: []string{"openid"}, IssuedAt: now,
		ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.AddDate(0, 0, 30),
	})
	_ = sessionStore.Save(ctx, &sessiondomain.LoginSession{
		ID: "sess-1", TenantID: tenancydomain.DefaultTenantID, UserID: user.ID,
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: "urn:mace:incommon:iap:silver",
		ExpiresAt: now.Add(time.Hour),
	})
	totpSecret := "JBSWY3DPEHPK3PXP"
	_ = mfaRepo.Save(ctx, &totpdomain.MfaFactor{
		UserID: user.ID, Type: spec.MfaFactorTOTP, Secret: &totpSecret, CreatedAt: now,
	})

	if err := userusecases.DeleteUser(ctx, deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: user.ID, Reason: "leaving company", Now: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if last, ok := events[len(events)-1].(*idmdomain.UserDeleted); !ok || last.TargetUserID != user.ID || last.Reason != "leaving company" {
		t.Fatalf("expected UserDeleted event with target=%s reason set, got %+v", user.ID, events[len(events)-1])
	}
	tombstone, err := userRepo.FindBySubIncludingDeleted(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tombstone == nil || !tombstone.IsDeleted() {
		t.Fatalf("expected tombstone with status=deleted, got %+v", tombstone)
	}
	if tombstone.PreferredUsername != "deleted:"+user.ID {
		t.Fatalf("preferred_username not anonymized: %s", tombstone.PreferredUsername)
	}
	if tombstone.Email != nil || tombstone.Name != nil || len(tombstone.Roles) != 0 || tombstone.MfaEnrolled {
		t.Fatalf("PII not anonymized: %+v", tombstone)
	}
	if seen, _ := userRepo.FindBySub(ctx, user.ID); seen != nil {
		t.Fatalf("FindBySub returned deleted user")
	}
	// Cascade verification.
	if remaining, _ := consentRepo.FindAll(ctx, tenancydomain.DefaultTenantID); len(remaining) != 0 {
		t.Fatalf("consent cascade leaked: %+v", remaining)
	}
	if rec, _ := refreshStore.FindByHash(ctx, "hash-1"); rec != nil {
		t.Fatalf("refresh cascade leaked: %+v", rec)
	}
	if sess, _ := sessionStore.Find(ctx, "sess-1"); sess != nil {
		t.Fatalf("session cascade leaked: %+v", sess)
	}
	if factors, _ := mfaRepo.ListBySub(ctx, user.ID); len(factors) != 0 {
		t.Fatalf("mfa cascade leaked: %+v", factors)
	}
	// Re-delete is no-op (no new UserDeleted event).
	prev := len(events)
	if err := userusecases.DeleteUser(ctx, deps, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: user.ID, Now: now.Add(2 * time.Hour),
	}); err != nil {
		t.Fatalf("idempotent delete failed: %v", err)
	}
	if len(events) != prev {
		t.Fatalf("idempotent delete emitted extra events")
	}
}

func TestDeleteUserRejectsSelfDelete(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-1", PreferredUsername: "admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	err := userusecases.DeleteUser(ctx, userusecases.AdminUserDeps{UserRepo: userRepo},
		userusecases.DeleteUserInput{ActorUserID: "admin-1", Sub: "admin-1", Now: now})
	if !errors.Is(err, userusecases.ErrSelfDeleteForbidden) {
		t.Fatalf("error=%v, want ErrSelfDeleteForbidden", err)
	}
}

func TestSetUserDisabledRejectsSelfDisable(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-1", PreferredUsername: "admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	deps := userusecases.AdminUserDeps{UserRepo: userRepo}

	// admin が自身を無効化しようとすると自爆防止に弾かれる。
	_, err := userusecases.SetUserDisabled(ctx, deps, "admin-1", "admin-1", true, now)
	if !errors.Is(err, userusecases.ErrSelfDisableForbidden) {
		t.Fatalf("disable self error=%v, want ErrSelfDisableForbidden", err)
	}

	// enable 方向は自身に対しても許可する (アクセス回復のみで誤操作リスクが低い)。
	if _, err := userusecases.SetUserDisabled(ctx, deps, "admin-1", "admin-1", false, now); err != nil {
		t.Fatalf("enable self error=%v, want nil", err)
	}
}

func TestSetUserDisabledAllowsDisablingOtherAdmin(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-2", PreferredUsername: "other-admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	deps := userusecases.AdminUserDeps{UserRepo: userRepo}

	user, err := userusecases.SetUserDisabled(ctx, deps, "admin-1", "admin-2", true, now)
	if err != nil {
		t.Fatalf("disable other admin error=%v, want nil", err)
	}
	if user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("status=%v, want disabled", user.Lifecycle.Status)
	}
}

// softDeleteTestDeps は soft-delete 系テスト用に cascade 対象リポジトリを揃えた
// deps と consent リポジトリ (cascade 温存の確認用) を返す。
func softDeleteTestDeps(events *[]spec.DomainEvent) (userusecases.AdminUserDeps, *oauth2memory.ConsentRepository, *usermemory.UserRepository) {
	userRepo := usermemory.NewUserRepository()
	consentRepo := oauth2memory.NewConsentRepository()
	deps := userusecases.AdminUserDeps{
		UserRepo: userRepo, ConsentRepo: consentRepo,
		RefreshStore: oauth2memory.NewRefreshTokenStore(), SessionStore: sessionmemory.NewSessionStore(),
		MfaFactorRepo: totpmemory.NewMfaFactorRepository(), PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
		Emit: func(event spec.DomainEvent) error { *events = append(*events, event); return nil },
	}
	return deps, consentRepo, userRepo
}

func TestSoftDeleteUserSetsPendingDeletionWithoutCascade(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, consentRepo, userRepo := softDeleteTestDeps(&events)
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash",
		Roles: []string{"support"}, CreatedAt: now, UpdatedAt: now,
	})
	_ = consentRepo.Save(ctx, tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: "alice-1", ClientID: "client-a",
		Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.AddDate(1, 0, 0),
	})

	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Reason: "maybe leaving", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	last, ok := events[len(events)-1].(*idmdomain.UserSoftDeleted)
	if !ok || last.TargetUserID != "alice-1" || last.Reason != "maybe leaving" {
		t.Fatalf("expected UserSoftDeleted with target/reason, got %+v", events[len(events)-1])
	}
	// status は PendingDeletion で、FindBySub でまだ見える (tombstone と違い可視)。
	user, _ := userRepo.FindBySub(ctx, "alice-1")
	if user == nil || !user.IsSoftDeleted() || user.IsActive() || user.IsDeleted() {
		t.Fatalf("expected visible soft-deleted user, got %+v", user)
	}
	// PII / cascade artifact は温存される。
	if user.Email != nil && *user.Email == "deleted:alice-1" {
		t.Fatal("PII was anonymized on soft-delete")
	}
	if remaining, _ := consentRepo.FindAll(ctx, tenancydomain.DefaultTenantID); len(remaining) != 1 {
		t.Fatalf("consent must be preserved on soft-delete, got %+v", remaining)
	}
	// 冪等: 再 soft-delete は追加イベントを出さない。
	prev := len(events)
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != prev {
		t.Fatal("idempotent soft-delete emitted extra events")
	}
}

func TestRestoreUserReturnsToActive(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, _, userRepo := softDeleteTestDeps(&events)
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	email := "alice@example.com"
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash", Email: &email,
		Roles: []string{"support"}, CreatedAt: now, UpdatedAt: now,
	})
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	restored, err := userusecases.RestoreUser(ctx, deps, "admin", "alice-1", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !restored.IsActive() || restored.Email == nil || *restored.Email != email {
		t.Fatalf("expected active restored user with PII intact, got %+v", restored)
	}
	if got := events[len(events)-1].EventType(); got != "UserRestored" {
		t.Fatalf("last event=%s, want UserRestored", got)
	}
}

func TestRestoreUserRejectsNonPendingAndExpired(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, _, userRepo := softDeleteTestDeps(&events)
	deps.SoftDeleteGraceSeconds = 60
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	// Active user への restore は ErrUserNotPendingDeletion。
	if _, err := userusecases.RestoreUser(ctx, deps, "admin", "alice-1", now); !errors.Is(err, userusecases.ErrUserNotPendingDeletion) {
		t.Fatalf("error=%v, want ErrUserNotPendingDeletion", err)
	}
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	// 猶予期間 (60s) 経過後の restore は ErrRestoreGracePeriodExpired。
	if _, err := userusecases.RestoreUser(ctx, deps, "admin", "alice-1", now.Add(2*time.Minute)); !errors.Is(err, userusecases.ErrRestoreGracePeriodExpired) {
		t.Fatalf("error=%v, want ErrRestoreGracePeriodExpired", err)
	}
}

func TestSoftDeleteAndRestoreRejectSelf(t *testing.T) {
	ctx := context.Background()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "admin-1", PreferredUsername: "admin", PasswordHash: "hash",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	deps := userusecases.AdminUserDeps{UserRepo: userRepo}
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin-1", Sub: "admin-1", Now: now,
	}); !errors.Is(err, userusecases.ErrSelfDeleteForbidden) {
		t.Fatalf("soft-delete self error=%v, want ErrSelfDeleteForbidden", err)
	}
	if _, err := userusecases.RestoreUser(ctx, deps, "admin-1", "admin-1", now); !errors.Is(err, userusecases.ErrSelfDeleteForbidden) {
		t.Fatalf("restore self error=%v, want ErrSelfDeleteForbidden", err)
	}
}

func TestPurgeExpiredSoftDeletedAnonymizesAfterGrace(t *testing.T) {
	ctx := context.Background()
	var events []spec.DomainEvent
	deps, _, userRepo := softDeleteTestDeps(&events)
	deps.SoftDeleteGraceSeconds = 1
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	userRepo.Seed(&userdomain.User{
		ID: "alice-1", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	if err := userusecases.SoftDeleteUser(ctx, deps, userusecases.SoftDeleteUserInput{
		ActorUserID: "admin", Sub: "alice-1", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	// 猶予期間内 (grace=1s) の purge は no-op。
	if err := userusecases.PurgeExpiredSoftDeleted(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	if user, _ := userRepo.FindBySub(ctx, "alice-1"); user == nil || !user.IsSoftDeleted() {
		t.Fatal("user must remain pending within grace")
	}
	// 猶予期間経過後の purge は anonymize cascade を実行し UserDeleted(auto_purge) を emit。
	if err := userusecases.PurgeExpiredSoftDeleted(ctx, deps, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	tombstone, _ := userRepo.FindBySubIncludingDeleted(ctx, "alice-1")
	if tombstone == nil || !tombstone.IsDeleted() {
		t.Fatalf("expected tombstone after auto-purge, got %+v", tombstone)
	}
	last, ok := events[len(events)-1].(*idmdomain.UserDeleted)
	if !ok || last.Reason != "auto_purge" {
		t.Fatalf("expected UserDeleted(auto_purge), got %+v", events[len(events)-1])
	}
}
