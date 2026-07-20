package db_postgres

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/session/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func withTenant(ctx context.Context, tenant *tenancydomain.Tenant) context.Context {
	return tenancy.WithTenant(ctx, tenant, "", "")
}

// TestSessionRepositoryRoundTrip: scenario `ユーザーは自分の有効なセッションを一覧して
// 失効できる` (wi-253) — save/find/list/idempotent revoke/tenant isolation/batch cleanup
// を PostgreSQL の実際の index/制約に対して検証する。
func TestSessionRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	otherTenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &SessionRepository{Pool: db}
	ctx := withTenant(context.Background(), tenant)
	// Find / ListBySub は expires_at をリクエスト時点の実時計 (time.Now()) と比較するため
	// (port の Find(ctx, id) に now 引数が無く、production は常に実時計を使う設計、wi-253)、
	// ここも実時計を基準にした相対時刻を使う。pgfixtures.TestClock の固定日時は使わない。
	now := time.Now().UTC()

	sess := &domain.LoginSession{
		ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, UserID: user.ID,
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: "urn:idmagic:acr:pwd",
		ExpiresAt: now.Add(time.Hour),
	}
	if err := repo.Save(ctx, sess); err != nil {
		t.Fatalf("save: %v", err)
	}

	t.Run("Find resolves an active session", func(t *testing.T) {
		got, err := repo.Find(ctx, sess.ID)
		if err != nil || got == nil {
			t.Fatalf("find: %v %+v", err, got)
		}
		if got.UserID != user.ID || len(got.AMR) != 1 || got.AMR[0] != "pwd" {
			t.Fatalf("unexpected row: %+v", got)
		}
	})

	t.Run("Find enforces tenant isolation (fail-closed)", func(t *testing.T) {
		wrongTenantCtx := withTenant(context.Background(), otherTenant)
		got, err := repo.Find(wrongTenantCtx, sess.ID)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil across tenant boundary, got %+v", got)
		}
	})

	t.Run("Save upserts mutable fields without disturbing last_seen_at", func(t *testing.T) {
		if err := repo.Touch(ctx, sess.ID, now.Add(10*time.Minute)); err != nil {
			t.Fatalf("touch: %v", err)
		}
		sess.AMR = []string{"pwd", "otp"}
		sess.AuthenticationPending = false
		sess.StepUpAt = now.Unix()
		if err := repo.Save(ctx, sess); err != nil {
			t.Fatalf("save (update): %v", err)
		}
		got, err := repo.Find(ctx, sess.ID)
		if err != nil || got == nil {
			t.Fatalf("find: %v %+v", err, got)
		}
		if len(got.AMR) != 2 || got.StepUpAt != now.Unix() {
			t.Fatalf("upsert did not apply mutable fields: %+v", got)
		}
		if !got.LastSeenAt.Equal(now.Add(10 * time.Minute)) {
			t.Fatalf("Save must not reset last_seen_at set by Touch: got %v", got.LastSeenAt)
		}
	})

	t.Run("Touch is coarse-grained", func(t *testing.T) {
		t1 := now.Add(time.Hour)
		if err := repo.Touch(ctx, sess.ID, t1); err != nil {
			t.Fatalf("touch: %v", err)
		}
		got, _ := repo.Find(ctx, sess.ID)
		if !got.LastSeenAt.Equal(t1) {
			t.Fatalf("LastSeenAt = %v, want %v", got.LastSeenAt, t1)
		}

		t2 := t1.Add(time.Minute) // < LoginSessionTouchInterval (5m)
		if err := repo.Touch(ctx, sess.ID, t2); err != nil {
			t.Fatalf("touch: %v", err)
		}
		got, _ = repo.Find(ctx, sess.ID)
		if !got.LastSeenAt.Equal(t1) {
			t.Fatalf("touch within interval must not persist: got %v, want %v", got.LastSeenAt, t1)
		}

		t3 := t1.Add(domain.LoginSessionTouchInterval + time.Second)
		if err := repo.Touch(ctx, sess.ID, t3); err != nil {
			t.Fatalf("touch: %v", err)
		}
		got, _ = repo.Find(ctx, sess.ID)
		if !got.LastSeenAt.Equal(t3) {
			t.Fatalf("touch past interval must persist: got %v, want %v", got.LastSeenAt, t3)
		}
	})

	t.Run("ListBySub returns newest first and excludes pending", func(t *testing.T) {
		older := &domain.LoginSession{
			ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, UserID: user.ID,
			AuthTime: now.Add(-time.Hour).Unix(), AMR: []string{"pwd"}, ACR: "urn:idmagic:acr:pwd",
			ExpiresAt: now.Add(time.Hour),
		}
		pending := &domain.LoginSession{
			ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, UserID: user.ID,
			AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: "urn:idmagic:acr:pwd",
			AuthenticationPending: true, ExpiresAt: now.Add(time.Hour),
		}
		if err := repo.Save(ctx, older); err != nil {
			t.Fatalf("save older: %v", err)
		}
		if err := repo.Save(ctx, pending); err != nil {
			t.Fatalf("save pending: %v", err)
		}

		list, err := repo.ListBySub(ctx, user.ID)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		ids := make(map[string]bool)
		for _, s := range list {
			ids[s.ID] = true
		}
		if !ids[sess.ID] || !ids[older.ID] {
			t.Fatalf("expected sess and older in list, got %v", ids)
		}
		if ids[pending.ID] {
			t.Fatalf("pending session must be excluded from list")
		}
		if list[0].AuthTime < list[len(list)-1].AuthTime {
			t.Fatalf("expected newest-first order: %+v", list)
		}
	})

	t.Run("FindOwned rejects wrong owner and Revoke is idempotent", func(t *testing.T) {
		otherUser := pgfixtures.SeedUser(t, db, tenant.ID) //nolint:contextcheck // fixture helper intentionally uses context.Background(); out of this WI's scope to thread ctx through pgfixtures
		if got, err := repo.FindOwned(ctx, sess.ID, otherUser.ID); err != nil || got != nil {
			t.Fatalf("expected nil for wrong owner: %v %+v", err, got)
		}

		revokeAt := now.Add(2 * time.Hour)
		if err := repo.Revoke(ctx, sess.ID, spec.SessionEndSelfRevoke, revokeAt); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		if got, _ := repo.Find(ctx, sess.ID); got != nil {
			t.Fatalf("revoked session must not resolve as active")
		}
		owned, err := repo.FindOwned(ctx, sess.ID, user.ID)
		if err != nil || owned == nil {
			t.Fatalf("find owned after revoke: %v %+v", err, owned)
		}
		if owned.RevokedAt == nil || !owned.RevokedAt.Equal(revokeAt) || owned.RevokeReason == nil || *owned.RevokeReason != spec.SessionEndSelfRevoke {
			t.Fatalf("unexpected tombstone: %+v", owned)
		}

		// 再失効は idempotent。最初の revoked_at / reason を保持する。
		later := revokeAt.Add(time.Minute)
		if err := repo.Revoke(ctx, sess.ID, spec.SessionEndAdminRevoke, later); err != nil {
			t.Fatalf("second revoke: %v", err)
		}
		owned, _ = repo.FindOwned(ctx, sess.ID, user.ID)
		if !owned.RevokedAt.Equal(revokeAt) || *owned.RevokeReason != spec.SessionEndSelfRevoke {
			t.Fatalf("second revoke overwrote tombstone: %+v", owned)
		}
	})

	t.Run("DeleteAllForSub physically erases rows", func(t *testing.T) {
		u := pgfixtures.SeedUser(t, db, tenant.ID) //nolint:contextcheck // fixture helper intentionally uses context.Background(); out of this WI's scope to thread ctx through pgfixtures
		s := &domain.LoginSession{
			ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, UserID: u.ID,
			AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: "urn:idmagic:acr:pwd",
			ExpiresAt: now.Add(time.Hour),
		}
		if err := repo.Save(ctx, s); err != nil {
			t.Fatalf("save: %v", err)
		}
		if err := repo.DeleteAllForSub(ctx, u.ID); err != nil {
			t.Fatalf("delete all: %v", err)
		}
		owned, err := repo.FindOwned(ctx, s.ID, u.ID)
		if err != nil || owned != nil {
			t.Fatalf("expected physical erasure: %v %+v", err, owned)
		}
	})

	t.Run("DeleteExpiredBatch deletes in small batches", func(t *testing.T) {
		u := pgfixtures.SeedUser(t, db, tenant.ID) //nolint:contextcheck // fixture helper intentionally uses context.Background(); out of this WI's scope to thread ctx through pgfixtures
		// cutoff は他 subtest が作った (now + 1h 前後の) 行を巻き込まない、この subtest 専用の
		// 狭い窓にする。DeleteExpiredBatch はテナント/ユーザーで絞らない全体 housekeeping。
		cutoff := now.Add(time.Minute)
		var ids []string
		for range 3 {
			s := &domain.LoginSession{
				ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, UserID: u.ID,
				AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: "urn:idmagic:acr:pwd",
				ExpiresAt: now.Add(-time.Minute),
			}
			if err := repo.Save(ctx, s); err != nil {
				t.Fatalf("save: %v", err)
			}
			ids = append(ids, s.ID)
		}

		deleted, err := repo.DeleteExpiredBatch(ctx, cutoff, 2)
		if err != nil {
			t.Fatalf("delete expired batch: %v", err)
		}
		if deleted != 2 {
			t.Fatalf("expected batch limit of 2, got %d", deleted)
		}
		deleted, err = repo.DeleteExpiredBatch(ctx, cutoff, 10)
		if err != nil {
			t.Fatalf("delete expired batch: %v", err)
		}
		if deleted != 1 {
			t.Fatalf("expected 1 remaining row, got %d", deleted)
		}
		for _, id := range ids {
			if owned, _ := repo.FindOwned(ctx, id, u.ID); owned != nil {
				t.Fatalf("expected %s to be purged by housekeeping", id)
			}
		}
	})
}

// TestSessionResolutionSurvivesProcessRestart: scenario `ユーザーは自分の有効なセッションを
// 一覧して失効できる` extension「process 再起動を挟んでセッション一覧を取得する」(wi-253)。
// SessionRepository / SessionManager は Pool 以外の状態を持たないため、"process 再起動" を
// 「同じ DB に対して新しいインスタンスを作る」ことでシミュレートする。旧 Valkey 実装ではプロセス
// 終了で session が失われたが、PostgreSQL を正本にした後は生き残ることを確認する。
func TestSessionResolutionSurvivesProcessRestart(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	ctx := withTenant(context.Background(), tenant)

	// "process A": ログインしてセッションを作る。
	managerA := authusecases.NewSessionManager(&SessionRepository{Pool: db})
	authCtx, err := managerA.Create(ctx, user.ID, []string{"pwd"}, time.Time{})
	if err != nil || authCtx == nil {
		t.Fatalf("create: %v %+v", err, authCtx)
	}
	cookie := http.Header{"Cookie": []string{authusecases.SessionCookie + "=" + authCtx.SessionID}}

	// "process B": 別インスタンスの SessionManager/SessionRepository が同じ Pool で解決する。
	managerB := authusecases.NewSessionManager(&SessionRepository{Pool: db})
	resolved, err := managerB.Resolve(ctx, cookie)
	if err != nil || resolved == nil {
		t.Fatalf("resolve after restart: %v %+v", err, resolved)
	}
	if resolved.UserID != user.ID || resolved.SessionID != authCtx.SessionID {
		t.Fatalf("unexpected resolved context: %+v", resolved)
	}

	// "process B" で失効させると、"process C" (さらに別インスタンス) でも未認証になる。
	if err := managerB.Revoke(ctx, cookie.Get("Cookie")); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	managerC := authusecases.NewSessionManager(&SessionRepository{Pool: db})
	resolvedAfterRevoke, err := managerC.Resolve(ctx, cookie)
	if err != nil {
		t.Fatalf("resolve after revoke: %v", err)
	}
	if resolvedAfterRevoke != nil {
		t.Fatalf("expected revoked session to require re-authentication, got %+v", resolvedAfterRevoke)
	}
}
