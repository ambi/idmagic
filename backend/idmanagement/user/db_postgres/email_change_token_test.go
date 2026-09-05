package db_postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func seedEmailChangeToken(
	t *testing.T, db sharedpg.DB, subject, newEmail string,
) (*EmailChangeTokenStore, actiontoken.Envelope) {
	t.Helper()
	store := &EmailChangeTokenStore{Pool: db}
	envelope := actiontoken.Envelope{
		ID:      newUUID(t),
		Purpose: actiontoken.PurposeEmailChange,
		Subject: subject,
		Payload: actiontoken.Payload{userports.PayloadKeyNewEmail: newEmail},
		// テストは行の内容だけを見るので、ダイジェストは一意な文字列で足りる。
		Digest:    actiontoken.Digest(uniqueID("digest")),
		IssuedAt:  testClock(),
		ExpiresAt: testClock().Add(time.Hour),
	}
	if err := store.Save(context.Background(), envelope); err != nil {
		t.Fatalf("save: %v", err)
	}
	return store, envelope
}

func changedUser(user *userdomain.User, newEmail string) *userdomain.User {
	updated := *user
	address := newEmail
	updated.Email = &address
	updated.EmailVerified = true
	updated.UpdatedAt = testClock().Add(time.Minute)
	return &updated
}

func TestEmailChangeTokenStoreRefusesAnotherPurpose(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	store := &EmailChangeTokenStore{Pool: db}

	foreign := actiontoken.Envelope{
		ID: newUUID(t), Purpose: actiontoken.PurposePasswordReset, Subject: user.ID,
		Digest: actiontoken.Digest(uniqueID("digest")), IssuedAt: testClock(),
		ExpiresAt: testClock().Add(time.Hour),
	}
	if err := store.Save(context.Background(), foreign); !errors.Is(err, actiontoken.ErrPurposeMismatch) {
		t.Fatalf("save error = %v, want ErrPurposeMismatch", err)
	}
	found, err := store.Find(context.Background(), foreign.Digest)
	if err != nil || found != nil {
		t.Fatalf("the refused envelope was stored: %+v %v", found, err)
	}
}

func TestEmailChangeTokenStoreFindReadsWithoutConsuming(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	store, envelope := seedEmailChangeToken(t, db, user.ID, "new@example.com")
	ctx := context.Background()

	for range 3 {
		found, err := store.Find(ctx, envelope.Digest)
		if err != nil || found == nil {
			t.Fatalf("find: %v %+v", err, found)
		}
		if found.Purpose != actiontoken.PurposeEmailChange ||
			found.Payload[userports.PayloadKeyNewEmail] != "new@example.com" ||
			found.ID != envelope.ID {
			t.Fatalf("round trip lost data: %+v", found)
		}
	}
	if err := store.ConsumeAndApply(ctx, userports.EmailChangeCommit{
		Digest: envelope.Digest, Now: testClock().Add(time.Minute),
		User: changedUser(user, "new@example.com"),
	}); err != nil {
		t.Fatalf("consume after reads: %v", err)
	}
	// 消費済みの行はもう読めない。
	if found, err := store.Find(ctx, envelope.Digest); err != nil || found != nil {
		t.Fatalf("consumed token is still readable: %+v %v", found, err)
	}
}

// 並行する二つの確定要求のうち、作用が起きるのは一方だけである。
func TestEmailChangeTokenStoreAppliesOnceUnderConcurrency(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	store, envelope := seedEmailChangeToken(t, db, user.ID, "new@example.com")

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			results <- store.ConsumeAndApply(context.Background(), userports.EmailChangeCommit{
				Digest: envelope.Digest, Now: testClock().Add(time.Minute),
				User: changedUser(user, "new@example.com"),
			})
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
}

// 作用が失敗したらトランザクションごと巻き戻る。トークンは未使用のまま残り、
// 対象の primary email も変わらない。
func TestEmailChangeTokenStoreRollsBackWhenTheEffectFails(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	rival := seedUser(t, db, tenant.ID)
	store, envelope := seedEmailChangeToken(t, db, user.ID, "new@example.com")
	ctx := context.Background()

	// 同じテナントで使われている preferred_username へ変えようとして、
	// 一意制約に当てる。トークンを使用済みにした後で作用が失敗する経路である。
	broken := changedUser(user, "new@example.com")
	broken.PreferredUsername = rival.PreferredUsername
	if err := store.ConsumeAndApply(ctx, userports.EmailChangeCommit{
		Digest: envelope.Digest, Now: testClock().Add(time.Minute), User: broken,
	}); err == nil {
		t.Fatal("the conflicting effect was accepted")
	}

	found, err := store.Find(ctx, envelope.Digest)
	if err != nil || found == nil {
		t.Fatalf("the failed effect consumed the token: %+v %v", found, err)
	}
	stored, err := (&UserRepository{Pool: db}).FindBySub(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Email != nil {
		t.Fatalf("the failed effect changed the email: %v", stored.Email)
	}
}

func TestEmailChangeTokenStoreKeepsOnlyTheLatestTokenPerUser(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	_, first := seedEmailChangeToken(t, db, user.ID, "first@example.com")
	store, second := seedEmailChangeToken(t, db, user.ID, "second@example.com")

	if found, err := store.Find(context.Background(), first.Digest); err != nil || found != nil {
		t.Fatalf("the superseded token is still usable: %+v %v", found, err)
	}
	if found, err := store.Find(context.Background(), second.Digest); err != nil || found == nil {
		t.Fatalf("the latest token is missing: %+v %v", found, err)
	}
}
