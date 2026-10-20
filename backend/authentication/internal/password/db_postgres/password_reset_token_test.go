package db_postgres

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	authnports "github.com/ambi/idmagic/backend/authentication/password/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func seedResetToken(
	t *testing.T, db sharedpg.DB, subject string,
) (*PasswordResetTokenStore, actiontoken.Envelope) {
	t.Helper()
	store := NewPasswordResetTokenStore(db, idmanagement.UserWriterInTx)
	envelope := actiontoken.Envelope{
		ID:      pgfixtures.NewUUID(t),
		Purpose: actiontoken.PurposePasswordReset,
		Subject: subject,
		// テストは行の内容だけを見るので、ダイジェストは一意な文字列で足りる。
		Digest:    actiontoken.Digest(pgfixtures.UniqueID("digest")),
		IssuedAt:  pgfixtures.TestClock(),
		ExpiresAt: pgfixtures.TestClock().Add(time.Hour),
	}
	if err := store.Save(context.Background(), envelope); err != nil {
		t.Fatalf("save: %v", err)
	}
	return store, envelope
}

func resetCommit(
	digest actiontoken.Digest, user *userdomain.User, hash string,
) authnports.PasswordResetCommit {
	return authnports.PasswordResetCommit{
		Digest: digest, Now: pgfixtures.TestClock().Add(time.Minute),
		UserID: user.ID, PasswordEncoded: hash,
		Apply: func(latest *userdomain.User) (bool, error) {
			latest.PasswordHash = hash
			latest.UpdatedAt = pgfixtures.TestClock().Add(time.Minute)
			return true, nil
		},
	}
}

func TestPasswordResetTokenStoreRefusesAnotherPurpose(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	store := NewPasswordResetTokenStore(db, idmanagement.UserWriterInTx)

	foreign := actiontoken.Envelope{
		ID: pgfixtures.NewUUID(t), Purpose: actiontoken.PurposeEmailChange, Subject: user.ID,
		Digest:   actiontoken.Digest(pgfixtures.UniqueID("digest")),
		IssuedAt: pgfixtures.TestClock(), ExpiresAt: pgfixtures.TestClock().Add(time.Hour),
	}
	if err := store.Save(context.Background(), foreign); !errors.Is(err, actiontoken.ErrPurposeMismatch) {
		t.Fatalf("save error = %v, want ErrPurposeMismatch", err)
	}
	if found, err := store.Find(context.Background(), foreign.Digest); err != nil || found != nil {
		t.Fatalf("the refused envelope was stored: %+v %v", found, err)
	}
}

func TestPasswordResetTokenStoreFindReadsWithoutConsuming(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	store, envelope := seedResetToken(t, db, user.ID)
	ctx := context.Background()

	for range 3 {
		found, err := store.Find(ctx, envelope.Digest)
		if err != nil || found == nil {
			t.Fatalf("find: %v %+v", err, found)
		}
		if found.Purpose != actiontoken.PurposePasswordReset ||
			found.Subject != user.ID || found.ID != envelope.ID {
			t.Fatalf("round trip lost data: %+v", found)
		}
	}
	if err := store.ConsumeAndApply(ctx, resetCommit(envelope.Digest, user, "encoded-1")); err != nil {
		t.Fatalf("consume after reads: %v", err)
	}
	if found, err := store.Find(ctx, envelope.Digest); err != nil || found != nil {
		t.Fatalf("consumed token is still readable: %+v %v", found, err)
	}
	// 作用は履歴まで届いている。応答だけでは見えない副作用を読み戻して確かめる。
	recent, err := (&PasswordHistoryRepository{Pool: db}).Recent(ctx, user.ID, 5)
	if err != nil || len(recent) != 1 || recent[0].Encoded != "encoded-1" {
		t.Fatalf("password history = %+v, %v", recent, err)
	}
}

// 並行する二つの確定要求のうち、作用が起きるのは一方だけである。
func TestPasswordResetTokenStoreAppliesOnceUnderConcurrency(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	store, envelope := seedResetToken(t, db, user.ID)

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			results <- store.ConsumeAndApply(
				context.Background(), resetCommit(envelope.Digest, user, "encoded-1"),
			)
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
	// 作用も一度だけ起きた。履歴が二本になっていない。
	recent, err := (&PasswordHistoryRepository{Pool: db}).Recent(context.Background(), user.ID, 5)
	if err != nil || len(recent) != 1 {
		t.Fatalf("password history entries = %d, want 1 (%v)", len(recent), err)
	}
}

// 作用が失敗したらトランザクションごと巻き戻る。トークンは未使用のまま残り、
// パスワードも履歴も変わらない。
func TestPasswordResetTokenStoreRollsBackWhenTheEffectFails(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	rival := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	store, envelope := seedResetToken(t, db, user.ID)
	ctx := context.Background()

	// 同じテナントで使われている preferred_username へ変えようとして、一意制約に当てる。
	// トークンを使用済みにした後で作用が失敗する経路である。
	commit := resetCommit(envelope.Digest, user, "encoded-1")
	applyPassword := commit.Apply
	commit.Apply = func(latest *userdomain.User) (bool, error) {
		latest.PreferredUsername = rival.PreferredUsername
		return applyPassword(latest)
	}
	if err := store.ConsumeAndApply(ctx, commit); err == nil {
		t.Fatal("the conflicting effect was accepted")
	}

	found, err := store.Find(ctx, envelope.Digest)
	if err != nil || found == nil {
		t.Fatalf("the failed effect consumed the token: %+v %v", found, err)
	}
	recent, err := (&PasswordHistoryRepository{Pool: db}).Recent(ctx, user.ID, 5)
	if err != nil || len(recent) != 0 {
		t.Fatalf("the failed effect left history behind: %+v %v", recent, err)
	}
	stored, err := (&PasswordHistoryRepository{Pool: db}).Recent(ctx, rival.ID, 5)
	if err != nil || len(stored) != 0 {
		t.Fatalf("the failed effect touched the rival: %+v %v", stored, err)
	}
}

// historyFailingDB は、トランザクションの中のパスワード履歴の追加だけを失敗させる。
// User の保存が成功した後で確定が失敗する経路を作るために使う。
type historyFailingDB struct{ sharedpg.DB }

func (d historyFailingDB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := d.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return historyFailingTx{tx}, nil
}

type historyFailingTx struct{ pgx.Tx }

var errHistoryInsert = errors.New("password history insert failed")

func (t historyFailingTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "INSERT INTO password_history") {
		return pgconn.CommandTag{}, errHistoryInsert
	}
	return t.Tx.Exec(ctx, sql, args...)
}

// User を保存した後の書き込みが失敗しても、User の変更はトークンの使用済み化と一緒に巻き戻る。
// User の保存が呼び出し側のトランザクションの外で確定する誤りを検出する。
func TestPasswordResetTokenStoreRollsBackTheUserWhenALaterWriteFails(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	_, envelope := seedResetToken(t, db, user.ID)
	ctx := context.Background()

	store := NewPasswordResetTokenStore(historyFailingDB{db}, idmanagement.UserWriterInTx)
	if err := store.ConsumeAndApply(ctx, resetCommit(envelope.Digest, user, "encoded-rolled-back")); !errors.Is(err, errHistoryInsert) {
		t.Fatalf("consume error = %v, want the history failure", err)
	}

	stored, err := idmanagement.NewPostgresModule(db).UserRepo.FindBySub(ctx, user.ID)
	if err != nil || stored == nil || stored.PasswordHash != user.PasswordHash {
		t.Fatalf("user after the failed commit = %+v, %v; want the password unchanged", stored, err)
	}
	if found, err := store.Find(ctx, envelope.Digest); err != nil || found == nil {
		t.Fatalf("the failed commit consumed the token: %+v %v", found, err)
	}
}

func TestPasswordResetTokenStoreKeepsOnlyTheLatestTokenPerUser(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	_, first := seedResetToken(t, db, user.ID)
	store, second := seedResetToken(t, db, user.ID)

	if found, err := store.Find(context.Background(), first.Digest); err != nil || found != nil {
		t.Fatalf("the superseded token is still usable: %+v %v", found, err)
	}
	if found, err := store.Find(context.Background(), second.Digest); err != nil || found == nil {
		t.Fatalf("the latest token is missing: %+v %v", found, err)
	}
}
