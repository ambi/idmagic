package db_memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

func changeEnvelope(digest actiontoken.Digest, newEmail string, now time.Time) actiontoken.Envelope {
	return actiontoken.Envelope{
		ID: string(digest) + "-id", Purpose: actiontoken.PurposeEmailChange, Subject: "user-1",
		Payload:  actiontoken.Payload{userports.PayloadKeyNewEmail: newEmail},
		IssuedAt: now, ExpiresAt: now.Add(time.Hour), Digest: digest,
	}
}

func newEmailChangeFixture(t *testing.T) (*EmailChangeTokenStore, *UserRepository, time.Time) {
	t.Helper()
	users := NewUserRepository()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	current := "old@example.com"
	users.Seed(&userdomain.User{
		ID: "user-1", PreferredUsername: "alice", PasswordHash: "unused",
		Email: &current, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
	})
	return NewEmailChangeTokenStore(users), users, now
}

func changeCommit(digest actiontoken.Digest, now time.Time) userports.EmailChangeCommit {
	newEmail := "new@example.com"
	return userports.EmailChangeCommit{
		Digest: digest, Now: now,
		User: &userdomain.User{
			ID: "user-1", PreferredUsername: "alice", PasswordHash: "unused",
			Email: &newEmail, EmailVerified: true, CreatedAt: now, UpdatedAt: now,
		},
	}
}

// 用途の束縛は保存の入口から効く。
func TestEmailChangeTokenStoreRefusesAnotherPurpose(t *testing.T) {
	store, _, now := newEmailChangeFixture(t)
	foreign := changeEnvelope("foreign", "new@example.com", now)
	foreign.Purpose = actiontoken.PurposePasswordReset
	if err := store.Save(context.Background(), foreign); !errors.Is(err, actiontoken.ErrPurposeMismatch) {
		t.Fatalf("Save error = %v, want ErrPurposeMismatch", err)
	}
	found, err := store.Find(context.Background(), "foreign")
	if err != nil || found != nil {
		t.Fatalf("the refused envelope was stored: %#v %v", found, err)
	}
}

func TestEmailChangeTokenStoreKeepsTheLatestTokenForSubject(t *testing.T) {
	store, _, now := newEmailChangeFixture(t)
	if err := store.Save(context.Background(), changeEnvelope("old", "first@example.com", now)); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), changeEnvelope("new", "second@example.com", now)); err != nil {
		t.Fatal(err)
	}
	if found, _ := store.Find(context.Background(), "old"); found != nil {
		t.Error("the superseded token is still usable")
	}
	found, _ := store.Find(context.Background(), "new")
	if found == nil || found.Payload[userports.PayloadKeyNewEmail] != "second@example.com" {
		t.Fatalf("latest token = %#v", found)
	}
}

// Find は読むだけである。確認リンクの先読みが状態を変えないことの土台になる。
func TestEmailChangeTokenStoreFindDoesNotConsume(t *testing.T) {
	store, users, now := newEmailChangeFixture(t)
	if err := store.Save(context.Background(), changeEnvelope("token", "new@example.com", now)); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if found, err := store.Find(context.Background(), "token"); err != nil || found == nil {
			t.Fatalf("Find = %#v, %v", found, err)
		}
	}
	stored, _ := users.FindBySub(context.Background(), "user-1")
	if stored.Email == nil || *stored.Email != "old@example.com" {
		t.Fatalf("email changed by a read: %v", stored.Email)
	}
	if err := store.ConsumeAndApply(context.Background(), changeCommit("token", now)); err != nil {
		t.Fatalf("ConsumeAndApply after reads: %v", err)
	}
}

func TestEmailChangeTokenStoreAppliesOnceUnderConcurrency(t *testing.T) {
	store, users, now := newEmailChangeFixture(t)
	if err := store.Save(context.Background(), changeEnvelope("token", "new@example.com", now)); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			results <- store.ConsumeAndApply(context.Background(), changeCommit("token", now))
		})
	}
	wg.Wait()
	close(results)

	successes, refusals := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, actiontoken.ErrAlreadyConsumed):
			refusals++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 || refusals != 1 {
		t.Fatalf("successes=%d refusals=%d, want 1 and 1", successes, refusals)
	}
	stored, _ := users.FindBySub(context.Background(), "user-1")
	if stored.Email == nil || *stored.Email != "new@example.com" {
		t.Fatalf("email = %v, want the single applied commit", stored.Email)
	}
}

// 作用が失敗したら、トークンは未使用のまま残る。PostgreSQL 側の ROLLBACK と同じ意味である。
func TestEmailChangeTokenStoreLeavesTheTokenUnusedWhenTheEffectFails(t *testing.T) {
	_, users, now := newEmailChangeFixture(t)
	store := NewEmailChangeTokenStore(failingUserSave{UserRepository: users})
	if err := store.Save(context.Background(), changeEnvelope("token", "new@example.com", now)); err != nil {
		t.Fatal(err)
	}

	if err := store.ConsumeAndApply(
		context.Background(), changeCommit("token", now),
	); !errors.Is(err, errUserSaveUnavailable) {
		t.Fatalf("error = %v, want the save failure", err)
	}
	stored, _ := users.FindBySub(context.Background(), "user-1")
	if stored.Email == nil || *stored.Email != "old@example.com" {
		t.Fatalf("email = %v, want it unchanged", stored.Email)
	}
	found, err := store.Find(context.Background(), "token")
	if err != nil || found == nil {
		t.Fatalf("token was consumed by a failed effect: %#v %v", found, err)
	}
}

var errUserSaveUnavailable = errors.New("the user repository is unavailable")

// failingUserSave は保存だけが失敗する UserRepository である。
type failingUserSave struct {
	*UserRepository
}

func (failingUserSave) Save(context.Context, *userdomain.User) error {
	return errUserSaveUnavailable
}

func TestEmailChangeTokenStoreRefusesAnUnknownDigest(t *testing.T) {
	store, _, now := newEmailChangeFixture(t)
	if err := store.ConsumeAndApply(
		context.Background(), changeCommit("never-issued", now),
	); !errors.Is(err, actiontoken.ErrAlreadyConsumed) {
		t.Fatalf("error = %v, want ErrAlreadyConsumed", err)
	}
}
