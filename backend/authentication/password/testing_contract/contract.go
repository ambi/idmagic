// Package testing_contract defines the shared password persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

type Fixture struct {
	History passwordports.PasswordHistoryRepository
	Reset   passwordports.PasswordResetTokenStore
	User    *userdomain.User
	Now     time.Time
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
	updated := *f.User
	updated.PasswordHash = "hash-reset"
	if err := f.Reset.ConsumeAndApply(ctx, passwordports.PasswordResetCommit{
		Digest: envelope.Digest, Now: f.Now.Add(time.Minute), User: &updated, PasswordEncoded: "hash-reset",
	}); err != nil {
		t.Fatalf("Reset.ConsumeAndApply: %v", err)
	}
	if found, err := f.Reset.Find(ctx, envelope.Digest); err != nil || found != nil {
		t.Fatalf("Reset.Find after consume = (%+v, %v)", found, err)
	}
}
