package db_memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
)

func envelope(digest actiontoken.Digest, now time.Time) actiontoken.Envelope {
	return actiontoken.Envelope{
		ID: string(digest) + "-id", Purpose: actiontoken.PurposePasswordReset, Subject: "user",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute), Digest: digest,
	}
}

func newFixture(t *testing.T) (*PasswordResetTokenStore, *usermemory.UserRepository, time.Time) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	users.Seed(&userdomain.User{
		ID: "user", PreferredUsername: "alice", PasswordHash: "before",
		CreatedAt: now, UpdatedAt: now,
	})
	return NewPasswordResetTokenStore(users, NewPasswordHistoryRepository()), users, now
}

func commitFor(digest actiontoken.Digest, now time.Time, hash string) passwordports.PasswordResetCommit {
	return passwordports.PasswordResetCommit{
		Digest: digest, Now: now, PasswordEncoded: hash,
		User: &userdomain.User{
			ID: "user", PreferredUsername: "alice", PasswordHash: hash,
			CreatedAt: now, UpdatedAt: now,
		},
	}
}

// 用途の束縛は保存の入口から効く。ほかの用途のエンベロープはこの store に入らない。
func TestPasswordResetTokenStoreRefusesAnotherPurpose(t *testing.T) {
	store, _, now := newFixture(t)
	foreign := envelope("foreign", now)
	foreign.Purpose = actiontoken.PurposeEmailChange
	if err := store.Save(context.Background(), foreign); !errors.Is(err, actiontoken.ErrPurposeMismatch) {
		t.Fatalf("Save error = %v, want ErrPurposeMismatch", err)
	}
	found, err := store.Find(context.Background(), "foreign")
	if err != nil || found != nil {
		t.Fatalf("the refused envelope was stored: %#v %v", found, err)
	}
}

func TestPasswordResetTokenStoreInvalidatesPreviousTokenForSubject(t *testing.T) {
	store, _, now := newFixture(t)
	for _, digest := range []actiontoken.Digest{"old", "new"} {
		if err := store.Save(context.Background(), envelope(digest, now)); err != nil {
			t.Fatal(err)
		}
	}
	old, _ := store.Find(context.Background(), "old")
	next, _ := store.Find(context.Background(), "new")
	if old != nil || next == nil {
		t.Fatalf("old=%#v new=%#v", old, next)
	}
}

// Find は読むだけである。リンクの先読みが状態を変えないことの土台になる。
func TestPasswordResetTokenStoreFindDoesNotConsume(t *testing.T) {
	store, users, now := newFixture(t)
	if err := store.Save(context.Background(), envelope("token", now)); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		found, err := store.Find(context.Background(), "token")
		if err != nil || found == nil {
			t.Fatalf("Find = %#v, %v", found, err)
		}
	}
	stored, _ := users.FindBySub(context.Background(), "user")
	if stored.PasswordHash != "before" {
		t.Fatalf("password changed by a read: %q", stored.PasswordHash)
	}
	if err := store.ConsumeAndApply(context.Background(), commitFor("token", now, "after")); err != nil {
		t.Fatalf("ConsumeAndApply after reads: %v", err)
	}
}

func TestPasswordResetTokenStoreAppliesOnceUnderConcurrency(t *testing.T) {
	store, users, now := newFixture(t)
	if err := store.Save(context.Background(), envelope("token", now)); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := range 2 {
		wg.Go(func() {
			results <- store.ConsumeAndApply(context.Background(), commitFor("token", now, "after"+string(rune('a'+i))))
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
	// 拒否された側の作用は起きていない。保存されたパスワードは一つだけである。
	stored, _ := users.FindBySub(context.Background(), "user")
	if stored.PasswordHash != "aftera" && stored.PasswordHash != "afterb" {
		t.Fatalf("password = %q, want exactly one of the two commits", stored.PasswordHash)
	}
}

func TestPasswordResetTokenStoreRefusesAnUnknownDigest(t *testing.T) {
	store, _, now := newFixture(t)
	if err := store.ConsumeAndApply(
		context.Background(), commitFor("never-issued", now, "after"),
	); !errors.Is(err, actiontoken.ErrAlreadyConsumed) {
		t.Fatalf("error = %v, want ErrAlreadyConsumed", err)
	}
}

// 作用が失敗したら、トークンは未使用のまま残り、対象も元の姿に戻る。
func TestPasswordResetTokenStoreLeavesTheTokenUnusedWhenTheEffectFails(t *testing.T) {
	users := usermemory.NewUserRepository()
	now := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)
	users.Seed(&userdomain.User{
		ID: "user", PreferredUsername: "alice", PasswordHash: "before",
		CreatedAt: now, UpdatedAt: now,
	})
	store := NewPasswordResetTokenStore(users, failingHistory{})
	if err := store.Save(context.Background(), envelope("token", now)); err != nil {
		t.Fatal(err)
	}

	if err := store.ConsumeAndApply(
		context.Background(), commitFor("token", now, "after"),
	); !errors.Is(err, errHistoryUnavailable) {
		t.Fatalf("error = %v, want the history failure", err)
	}
	stored, _ := users.FindBySub(context.Background(), "user")
	if stored.PasswordHash != "before" {
		t.Fatalf("password = %q, want the effect rolled back", stored.PasswordHash)
	}
	found, err := store.Find(context.Background(), "token")
	if err != nil || found == nil {
		t.Fatalf("token was consumed by a failed effect: %#v %v", found, err)
	}
}

var errHistoryUnavailable = errors.New("password history is unavailable")

type failingHistory struct{}

var _ passwordports.PasswordHistoryRepository = failingHistory{}

func (failingHistory) Recent(context.Context, string, int) ([]passwordports.PasswordHistoryEntry, error) {
	return nil, nil
}

func (failingHistory) Add(context.Context, string, string, time.Time) error {
	return errHistoryUnavailable
}

func (failingHistory) DeleteAllForSub(context.Context, string) error { return nil }
