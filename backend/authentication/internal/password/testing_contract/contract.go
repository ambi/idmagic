// Package testing_contract defines the shared password persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

type Fixture struct {
	History passwordports.PasswordHistoryRepository
	Reset   passwordports.PasswordResetTokenStore
	// Users は Reset が書く User の保存先である。
	Users userports.UserRepository
	User  *userdomain.User
	Now   time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if err := f.History.Add(ctx, f.User.ID, "hash-old", f.Now); err != nil {
		t.Fatalf("History.Add: %v", err)
	}
	if err := f.History.Add(ctx, f.User.ID, "hash-new", f.Now.Add(time.Minute)); err != nil {
		t.Fatalf("History.Add: %v", err)
	}
	recent, err := f.History.Recent(ctx, f.User.ID, 2)
	if err != nil || len(recent) != 2 || recent[0].Encoded != "hash-new" || recent[1].Encoded != "hash-old" {
		t.Fatalf("History.Recent = (%+v, %v)", recent, err)
	}
	if err := f.History.DeleteAllForSub(ctx, f.User.ID); err != nil {
		t.Fatalf("History.DeleteAllForSub: %v", err)
	}
	if recent, err := f.History.Recent(ctx, f.User.ID, 2); err != nil || len(recent) != 0 {
		t.Fatalf("History after delete = (%+v, %v)", recent, err)
	}

	envelope := actiontoken.Envelope{
		ID:        "00000000-0000-0000-0000-000000000001",
		Purpose:   actiontoken.PurposePasswordReset,
		Subject:   f.User.ID,
		IssuedAt:  f.Now,
		ExpiresAt: f.Now.Add(time.Hour),
		Digest:    "password-reset-digest",
	}
	if err := f.Reset.Save(ctx, envelope); err != nil {
		t.Fatalf("Reset.Save: %v", err)
	}
	found, err := f.Reset.Find(ctx, envelope.Digest)
	if err != nil || found == nil || found.ID != envelope.ID {
		t.Fatalf("Reset.Find = (%+v, %v)", found, err)
	}
	// 確定は、トークンを読んだ後に別の書き手が行った無効化を書き戻さず、最新の User に
	// 新しいパスワードだけを当てる。
	disabled, err := f.Users.FindBySub(ctx, f.User.ID)
	if err != nil || disabled == nil {
		t.Fatalf("Users.FindBySub = (%+v, %v)", disabled, err)
	}
	disabled.Lifecycle.Status = idmdomain.UserStatusDisabled
	if err := f.Users.Save(ctx, disabled); err != nil {
		t.Fatalf("Users.Save: %v", err)
	}
	if err := f.Reset.ConsumeAndApply(ctx, passwordports.PasswordResetCommit{
		Digest: envelope.Digest, Now: f.Now.Add(time.Minute), UserID: f.User.ID, PasswordEncoded: "hash-reset",
		Apply: func(latest *userdomain.User) (bool, error) {
			latest.PasswordHash = "hash-reset"
			return true, nil
		},
	}); err != nil {
		t.Fatalf("Reset.ConsumeAndApply: %v", err)
	}
	if found, err := f.Reset.Find(ctx, envelope.Digest); err != nil || found != nil {
		t.Fatalf("Reset.Find after consume = (%+v, %v)", found, err)
	}
	stored, err := f.Users.FindBySub(ctx, f.User.ID)
	if err != nil || stored == nil || stored.PasswordHash != "hash-reset" || stored.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("user after reset = (%+v, %v), want the new hash on the disabled user", stored, err)
	}
	if recent, err := f.History.Recent(ctx, f.User.ID, 1); err != nil || len(recent) != 1 || recent[0].Encoded != "hash-reset" {
		t.Fatalf("History after reset = (%+v, %v)", recent, err)
	}
}
