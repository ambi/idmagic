package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	authnmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"

	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-AUTHENTICATION-010, EX-AUTHENTICATION-010-01: 正しい現在のパスワードでの変更がハッシュと password_changed_at を更新し、PasswordChanged を発行することを固定する。
func TestChangePasswordUpdatesHashAndEmitsEvent(t *testing.T) {
	t.Parallel()

	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash("demo-password-1234")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	user := &userdomain.User{
		ID: "user-1", PreferredUsername: "alice", PasswordHash: hash,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}
	if err := userRepo.Save(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent

	updated, err := ChangePassword(context.Background(), ChangePasswordDeps{
		UserRepo:            userRepo,
		PasswordHasher:      hasher,
		PasswordHistoryRepo: historyRepo,
		Emit: func(event spec.DomainEvent) error {
			events = append(events, event)
			return nil
		},
	}, ChangePasswordInput{
		Sub:             user.ID,
		CurrentPassword: "demo-password-1234",
		NewPassword:     "fresh-pass-9182",
		Now:             now,
	})
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if updated.UpdatedAt != now {
		t.Fatalf("updated_at=%s, want %s", updated.UpdatedAt, now)
	}
	// 有効期限の判定はこの時刻だけを読む (REQ-AUTHENTICATION-024)。更新し忘れると、
	// 変更したはずの利用者が次のログインで再び変更を強制される。
	if updated.Lifecycle.PasswordChangedAt == nil || !updated.Lifecycle.PasswordChangedAt.Equal(now) {
		t.Fatalf("password_changed_at=%v, want %s", updated.Lifecycle.PasswordChangedAt, now)
	}
	ok, err := hasher.Verify("fresh-pass-9182", updated.PasswordHash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("updated hash does not verify new password")
	}
	recent, err := historyRepo.Recent(context.Background(), user.ID, PasswordPolicyHistoryDepth)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 1 {
		t.Fatalf("history entries=%d, want 1", len(recent))
	}
	ok, err = hasher.Verify("fresh-pass-9182", recent[0].Encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("history hash does not verify new password")
	}
	if len(events) != 1 {
		t.Fatalf("events=%d, want 1", len(events))
	}
	if _, ok := events[0].(*authdomain.PasswordChanged); !ok {
		t.Fatalf("event type=%T, want *authdomain.PasswordChanged", events[0])
	}
}

func TestChangePasswordRejectsCurrentPasswordMismatch(t *testing.T) {
	t.Parallel()

	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash("demo-password-1234")
	if err != nil {
		t.Fatal(err)
	}
	if err := userRepo.Save(context.Background(), &userdomain.User{
		ID: "user-1", PreferredUsername: "alice", PasswordHash: hash,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	_, err = ChangePassword(context.Background(), ChangePasswordDeps{
		UserRepo:            userRepo,
		PasswordHasher:      hasher,
		PasswordHistoryRepo: historyRepo,
	}, ChangePasswordInput{
		Sub:             "user-1",
		CurrentPassword: "wrong-password",
		NewPassword:     "fresh-pass-9182",
	})
	if !errors.Is(err, ErrCurrentPasswordMismatch) {
		t.Fatalf("err=%v, want current password mismatch", err)
	}
}

func TestChangePasswordHonorsTenantOverridePolicy(t *testing.T) {
	t.Parallel()

	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash("demo-password-1234")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	if err := userRepo.Save(context.Background(), &userdomain.User{
		ID: "user-1", PreferredUsername: "alice", PasswordHash: hash,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	strict := PasswordPolicySnapshot{MinLength: 24, MaxLength: 128, HistoryDepth: 5}
	_, err = ChangePassword(context.Background(), ChangePasswordDeps{
		UserRepo:            userRepo,
		PasswordHasher:      hasher,
		PasswordHistoryRepo: historyRepo,
		Policy:              strict,
	}, ChangePasswordInput{
		Sub:             "user-1",
		CurrentPassword: "demo-password-1234",
		NewPassword:     "fresh-pass-9182",
		Now:             now,
	})
	var policyErr *PasswordPolicyError
	if !errors.As(err, &policyErr) {
		t.Fatalf("err=%v, want PasswordPolicyError under tenant strict policy", err)
	}
	if len(policyErr.Violations) == 0 || policyErr.Violations[0] != ViolationTooShort {
		t.Fatalf("violations=%v, want first too_short", policyErr.Violations)
	}
}

// reset-password.
//
//spec:covers REQ-AUTHENTICATION-010: change-password runs the same validation stages as
func TestChangePasswordRejectsBreachedPassword(t *testing.T) {
	t.Parallel()

	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	hash, err := hasher.Hash("demo-password-1234")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	if err := userRepo.Save(context.Background(), &userdomain.User{
		ID: "user-1", PreferredUsername: "alice", PasswordHash: hash,
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	_, err = ChangePassword(context.Background(), ChangePasswordDeps{
		UserRepo:                userRepo,
		PasswordHasher:          hasher,
		PasswordHistoryRepo:     historyRepo,
		BreachedPasswordChecker: breachedChecker{},
	}, ChangePasswordInput{
		Sub:             "user-1",
		CurrentPassword: "demo-password-1234",
		NewPassword:     "fresh-pass-9182",
		Now:             now,
	})
	var policyErr *PasswordPolicyError
	if !errors.As(err, &policyErr) {
		t.Fatalf("err=%v, want PasswordPolicyError for a breached password", err)
	}
	if len(policyErr.Violations) != 1 || policyErr.Violations[0] != ViolationBreached {
		t.Fatalf("violations=%v, want [breached]", policyErr.Violations)
	}
}

type breachedChecker struct{}

func (breachedChecker) IsBreached(context.Context, string) bool { return true }

//spec:covers REQ-AUTHENTICATION-010, EX-AUTHENTICATION-010-03: 直近 5 件の履歴に一致する新しいパスワードが拒否され、保存されているハッシュが変わらないことを固定する。
func TestChangePasswordRejectsPasswordReuse(t *testing.T) {
	t.Parallel()

	userRepo := usermemory.NewUserRepository()
	historyRepo := authnmemory.NewPasswordHistoryRepository()
	hasher := testing_passwords.NewHasher()
	initialHash, err := hasher.Hash("demo-password-1234")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := userRepo.Save(context.Background(), &userdomain.User{
		ID: "user-1", PreferredUsername: "alice", PasswordHash: initialHash,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	reusedHash, err := hasher.Hash("pw-history-aaaa-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := historyRepo.Add(context.Background(), "user-1", reusedHash, now); err != nil {
		t.Fatal(err)
	}

	_, err = ChangePassword(context.Background(), ChangePasswordDeps{
		UserRepo:            userRepo,
		PasswordHasher:      hasher,
		PasswordHistoryRepo: historyRepo,
	}, ChangePasswordInput{
		Sub:             "user-1",
		CurrentPassword: "demo-password-1234",
		NewPassword:     "pw-history-aaaa-1",
	})
	if !errors.Is(err, ErrPasswordReused) {
		t.Fatalf("err=%v, want password reused", err)
	}
	// 拒否の戻り値だけでは、履歴と照合する前に保存する実装を通してしまう。
	stored, err := userRepo.FindBySub(context.Background(), "user-1")
	if err != nil || stored == nil {
		t.Fatalf("user=%v err=%v", stored, err)
	}
	if stored.PasswordHash != initialHash {
		t.Fatal("再利用を拒否したのに保存されたハッシュが変わった")
	}
}
