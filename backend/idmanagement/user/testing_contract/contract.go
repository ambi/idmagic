// Package testing_contract defines the persistence contract shared by the
// in-memory and PostgreSQL adapters in the User subdomain.
package testing_contract

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// Fixture provides one isolated set of persistence adapters and the
// identifiers for database rows prepared by the adapter-specific harness.
type Fixture struct {
	Users          userports.UserRepository
	EmailTokens    userports.EmailChangeTokenStore
	Schemas        tenantports.TenantUserAttributeSchemaRepository
	ImportRows     userports.UserImportRowCommitter
	TenantA        string
	TenantB        string
	EmailUser      *userdomain.User
	ActorUserID    string
	Now            time.Time
	AssertImported func(t *testing.T, mutation userports.UserImportRowMutation)
}

// NewFixture creates an isolated fixture for one contract subtest.
type NewFixture func(t *testing.T) Fixture

// Run executes every persistence contract owned by the User subdomain.
func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	t.Run("user repository", func(t *testing.T) { runUserRepository(t, newFixture) })
	t.Run("email change tokens", func(t *testing.T) { runEmailChangeTokens(t, newFixture) })
	t.Run("tenant attribute schema", func(t *testing.T) { runTenantAttributeSchema(t, newFixture) })
	t.Run("import row", func(t *testing.T) { runImportRow(t, newFixture) })
}

func newID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new UUID: %v", err)
	}
	return id
}

func user(t *testing.T, tenantID, username string, now time.Time) *userdomain.User {
	t.Helper()
	email := username + "@example.com"
	return &userdomain.User{
		ID:                newID(t),
		TenantID:          tenantID,
		PreferredUsername: username,
		PasswordHash:      "hash",
		Email:             &email,
		Roles:             []string{"reader"},
		Lifecycle:         userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func runUserRepository(t *testing.T, newFixture NewFixture) {
	t.Helper()

	t.Run("round trips lookups and missing values", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		stored := user(t, f.TenantA, "alice", f.Now)
		if err := f.Users.Save(ctx, stored); err != nil {
			t.Fatalf("Save: %v", err)
		}
		bySub, err := f.Users.FindBySub(ctx, stored.ID)
		if err != nil || bySub == nil {
			t.Fatalf("FindBySub = (%+v, %v), want stored user", bySub, err)
		}
		if bySub.Email == nil || *bySub.Email != *stored.Email || !bySub.CreatedAt.Equal(f.Now) {
			t.Fatalf("round trip lost fields: %+v", bySub)
		}
		byUsername, err := f.Users.FindByUsername(ctx, f.TenantA, "alice")
		if err != nil || byUsername == nil || byUsername.ID != stored.ID {
			t.Fatalf("FindByUsername = (%+v, %v), want %s", byUsername, err, stored.ID)
		}
		byEmail, err := f.Users.FindByEmail(ctx, f.TenantA, "ALICE@EXAMPLE.COM")
		if err != nil || byEmail == nil || byEmail.ID != stored.ID {
			t.Fatalf("FindByEmail = (%+v, %v), want case-insensitive match", byEmail, err)
		}
		missing, err := f.Users.FindBySub(ctx, newID(t))
		if err != nil || missing != nil {
			t.Fatalf("missing FindBySub = (%+v, %v), want (nil, nil)", missing, err)
		}
	})

	t.Run("tenant boundary deletion and ordering", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		for _, candidate := range []*userdomain.User{
			user(t, f.TenantA, "charlie", f.Now),
			user(t, f.TenantA, "alice", f.Now),
			user(t, f.TenantA, "bob", f.Now),
			user(t, f.TenantB, "other", f.Now),
		} {
			if err := f.Users.Save(ctx, candidate); err != nil {
				t.Fatalf("Save(%s): %v", candidate.PreferredUsername, err)
			}
		}
		page, err := f.Users.ListPage(ctx, f.TenantA, "", "", 2)
		if err != nil || len(page) != 2 || page[0].PreferredUsername != "alice" || page[1].PreferredUsername != "bob" {
			t.Fatalf("first page = (%+v, %v), want alice, bob", page, err)
		}
		next, err := f.Users.ListPage(ctx, f.TenantA, page[1].PreferredUsername, page[1].ID, 2)
		if err != nil || len(next) != 1 || next[0].PreferredUsername != "charlie" {
			t.Fatalf("next page = (%+v, %v), want charlie", next, err)
		}
		deleted := user(t, f.TenantA, "deleted", f.Now)
		deleted.Lifecycle.Status = idmdomain.UserStatusDeleted
		if err := f.Users.Save(ctx, deleted); err != nil {
			t.Fatalf("Save deleted user: %v", err)
		}
		if got, err := f.Users.FindBySub(ctx, deleted.ID); err != nil || got != nil {
			t.Fatalf("FindBySub deleted = (%+v, %v), want (nil, nil)", got, err)
		}
		if got, err := f.Users.FindBySubIncludingDeleted(ctx, deleted.ID); err != nil || got == nil {
			t.Fatalf("FindBySubIncludingDeleted = (%+v, %v), want tombstone", got, err)
		}
		count, err := f.Users.Count(ctx, f.TenantA)
		if err != nil || count != 3 {
			t.Fatalf("Count = (%d, %v), want 3", count, err)
		}
	})

	// 保持期限の削除は、猶予期間を判定する削除予約の User と、完全削除を終えていない Tombstone を
	// 候補として引く (REQ-IDMANAGEMENT-044)。
	t.Run("purge candidates are pending deletions and unfinished tombstones of the tenant", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		saveWith := func(tenantID, username string, lifecycle userdomain.UserLifecycle) *userdomain.User {
			t.Helper()
			stored := user(t, tenantID, username, f.Now)
			stored.Lifecycle = lifecycle
			if err := f.Users.Save(ctx, stored); err != nil {
				t.Fatalf("Save(%s): %v", username, err)
			}
			return stored
		}
		since := f.Now.Add(-time.Hour)
		pending := userdomain.UserLifecycle{Status: idmdomain.UserStatusPendingDeletion, StatusChangedAt: &since}
		unfinished := userdomain.UserLifecycle{
			Status: idmdomain.UserStatusDeleted, StatusChangedAt: &since,
			PendingPurge: &userdomain.PendingPurge{Step: userdomain.PurgeStepAnnounce, ActorUserID: "admin", Reason: "offboarding"},
		}
		wantPending := saveWith(f.TenantA, "pending", pending)
		wantUnfinished := saveWith(f.TenantA, "unfinished", unfinished)
		saveWith(f.TenantA, "active", userdomain.UserLifecycle{Status: idmdomain.UserStatusActive})
		saveWith(f.TenantA, "finished", userdomain.UserLifecycle{Status: idmdomain.UserStatusDeleted, StatusChangedAt: &since})
		saveWith(f.TenantB, "other-pending", pending)

		candidates, err := f.Users.ListPurgeCandidates(ctx, f.TenantA)
		if err != nil {
			t.Fatalf("ListPurgeCandidates: %v", err)
		}
		got := map[string]*userdomain.User{}
		for _, candidate := range candidates {
			got[candidate.ID] = candidate
		}
		if len(got) != 2 || got[wantPending.ID] == nil || got[wantUnfinished.ID] == nil {
			t.Fatalf("candidates=%+v, want only the pending deletion and the unfinished tombstone of tenant A", candidates)
		}
		if marker := got[wantUnfinished.ID].Lifecycle.PendingPurge; marker == nil || *marker != *unfinished.PendingPurge {
			t.Fatalf("pending purge=%+v, want %+v", marker, unfinished.PendingPurge)
		}
	})

	t.Run("same tenant rejects duplicate username", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		if err := f.Users.Save(ctx, user(t, f.TenantA, "duplicate", f.Now)); err != nil {
			t.Fatalf("first Save: %v", err)
		}
		if err := f.Users.Save(ctx, user(t, f.TenantA, "duplicate", f.Now)); err == nil {
			t.Fatal("second Save accepted a duplicate preferred_username")
		}
	})

	// ユーザー名とメールアドレスは比較キーで引き、ユーザー名の一意性も比較キーで守る
	// (REQ-IDMANAGEMENT-042)。表記は入力のまま保存する。
	t.Run("names and emails compare by their case-folded keys", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		stored := user(t, f.TenantA, "Straße", f.Now)
		mixedEmail := "Straße@Example.com"
		stored.Email = &mixedEmail
		if err := f.Users.Save(ctx, stored); err != nil {
			t.Fatalf("first Save: %v", err)
		}
		if got, err := f.Users.FindByUsername(ctx, f.TenantA, " STRASSE "); err != nil || got == nil || got.ID != stored.ID || got.PreferredUsername != "Straße" {
			t.Fatalf("FindByUsername(STRASSE) = (%+v, %v), want the stored user with its own spelling", got, err)
		}
		if got, err := f.Users.FindByEmail(ctx, f.TenantA, "strasse@example.COM"); err != nil || got == nil || got.ID != stored.ID {
			t.Fatalf("FindByEmail = (%+v, %v), want the stored user", got, err)
		}
		if got, err := f.Users.FindByEmail(ctx, f.TenantA, "  "); err != nil || got != nil {
			t.Fatalf("FindByEmail(blank) = (%+v, %v), want (nil, nil)", got, err)
		}
		if err := f.Users.Save(ctx, user(t, f.TenantA, "strasse", f.Now)); err == nil {
			t.Fatal("Save accepted a username that differs only by case folding")
		}
		renamed := *stored
		renamed.PreferredUsername = "STRASSE"
		if err := f.Users.Save(ctx, &renamed); err != nil {
			t.Fatalf("Save of the same user under a new spelling: %v", err)
		}
		if got, err := f.Users.FindByUsername(ctx, f.TenantA, "straße"); err != nil || got == nil || got.PreferredUsername != "STRASSE" {
			t.Fatalf("FindByUsername after rename = (%+v, %v), want STRASSE", got, err)
		}
	})

	// ユーザー名の一意性は削除済みの User を数えない。Tombstone を保存し直しても、
	// 同じユーザー名を再利用している有効な User を押しのけない。
	t.Run("deleted user saves again while an active user reuses its username", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		tombstone := user(t, f.TenantA, "reused", f.Now)
		tombstone.Lifecycle.Status = idmdomain.UserStatusDeleted
		if err := f.Users.Save(ctx, tombstone); err != nil {
			t.Fatalf("Save tombstone: %v", err)
		}
		reuser := user(t, f.TenantA, "reused", f.Now)
		if err := f.Users.Save(ctx, reuser); err != nil {
			t.Fatalf("Save active user reusing the username: %v", err)
		}
		tombstone.UpdatedAt = f.Now.Add(time.Minute)
		if err := f.Users.Save(ctx, tombstone); err != nil {
			t.Fatalf("Save tombstone again: %v", err)
		}
		if got, err := f.Users.FindByUsername(ctx, f.TenantA, "reused"); err != nil || got == nil || got.ID != reuser.ID {
			t.Fatalf("FindByUsername = (%+v, %v), want the active user %s", got, err, reuser.ID)
		}
	})

	// 保存内容を変えられるのは Save だけである。読み取りの返り値と Save の引数は、
	// 呼び出し側が後から書き換えても保存内容と共有しない。
	t.Run("returned and saved users do not alias stored state", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		stored := aliasProbeUser(t, f.TenantA, f.Now)
		if err := f.Users.Save(ctx, stored); err != nil {
			t.Fatalf("Save: %v", err)
		}
		mutateUser(stored)
		assertAliasProbeUnchanged(t, f.Users, stored.ID, "after mutating the Save argument")

		active := idmdomain.UserStatusActive
		readers := map[string]func() ([]*userdomain.User, error){
			"FindBySub":                 func() ([]*userdomain.User, error) { return one(f.Users.FindBySub(ctx, stored.ID)) },
			"FindBySubIncludingDeleted": func() ([]*userdomain.User, error) { return one(f.Users.FindBySubIncludingDeleted(ctx, stored.ID)) },
			"FindByUsername":            func() ([]*userdomain.User, error) { return one(f.Users.FindByUsername(ctx, f.TenantA, "probe")) },
			"FindByEmail": func() ([]*userdomain.User, error) {
				return one(f.Users.FindByEmail(ctx, f.TenantA, "probe@example.com"))
			},
			"FindAll":        func() ([]*userdomain.User, error) { return f.Users.FindAll(ctx, f.TenantA) },
			"ListPage":       func() ([]*userdomain.User, error) { return f.Users.ListPage(ctx, f.TenantA, "", "", 10) },
			"ListPageBefore": func() ([]*userdomain.User, error) { return f.Users.ListPageBefore(ctx, f.TenantA, "", "", 10) },
			"ListPageFiltered": func() ([]*userdomain.User, error) {
				return f.Users.ListPageFiltered(ctx, f.TenantA, "", &active, "", "", 10)
			},
			"ListPageBeforeFiltered": func() ([]*userdomain.User, error) {
				return f.Users.ListPageBeforeFiltered(ctx, f.TenantA, "", &active, "", "", 10)
			},
		}
		for name, read := range readers {
			got, err := read()
			if err != nil || len(got) != 1 {
				t.Fatalf("%s = (%+v, %v), want the stored user", name, got, err)
			}
			mutateUser(got[0])
			assertAliasProbeUnchanged(t, f.Users, stored.ID, "after mutating the result of "+name)
		}
	})

	// 既存の User を保存し直しても、作成時刻と所属テナントは最初に保存した値のまま残る。
	t.Run("resave keeps the first created_at and tenant", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		stored := user(t, f.TenantA, "resaved", f.Now)
		if err := f.Users.Save(ctx, stored); err != nil {
			t.Fatalf("first Save: %v", err)
		}
		later := f.Now.Add(time.Hour)
		resaved := *stored
		resaved.TenantID = f.TenantB
		resaved.CreatedAt = later
		resaved.UpdatedAt = later
		if err := f.Users.Save(ctx, &resaved); err != nil {
			t.Fatalf("second Save: %v", err)
		}
		got, err := f.Users.FindBySub(ctx, stored.ID)
		if err != nil || got == nil {
			t.Fatalf("FindBySub = (%+v, %v), want the stored user", got, err)
		}
		if got.TenantID != f.TenantA || !got.CreatedAt.Equal(f.Now) || !got.UpdatedAt.Equal(later) {
			t.Fatalf("resaved user tenant=%s created_at=%s updated_at=%s, want tenant=%s created_at=%s updated_at=%s",
				got.TenantID, got.CreatedAt, got.UpdatedAt, f.TenantA, f.Now, later)
		}
	})

	// 状態が未設定の User は active として扱う (REQ-IDMANAGEMENT-005 の status による絞り込み)。
	t.Run("unset status filters as active", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		unset := user(t, f.TenantA, "unset", f.Now)
		unset.Lifecycle.Status = ""
		if err := f.Users.Save(ctx, unset); err != nil {
			t.Fatalf("Save: %v", err)
		}
		active := idmdomain.UserStatusActive
		if page, err := f.Users.ListPageFiltered(ctx, f.TenantA, "", &active, "", "", 10); err != nil || len(page) != 1 || page[0].ID != unset.ID {
			t.Fatalf("ListPageFiltered(active) = (%+v, %v), want the user without a status", page, err)
		}
		if page, err := f.Users.ListPageBeforeFiltered(ctx, f.TenantA, "", &active, "", "", 10); err != nil || len(page) != 1 || page[0].ID != unset.ID {
			t.Fatalf("ListPageBeforeFiltered(active) = (%+v, %v), want the user without a status", page, err)
		}
		if count, err := f.Users.CountFiltered(ctx, f.TenantA, "", &active); err != nil || count != 1 {
			t.Fatalf("CountFiltered(active) = (%d, %v), want 1", count, err)
		}
	})

	// 保存した時刻は、PostgreSQL の TIMESTAMPTZ と同じマイクロ秒に切り捨てて読み戻る。
	t.Run("timestamps read back truncated to microseconds", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		precise := f.Now.Add(789 * time.Nanosecond)
		stored := user(t, f.TenantA, "precise", precise)
		if err := f.Users.Save(ctx, stored); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := f.Users.FindBySub(ctx, stored.ID)
		if err != nil || got == nil {
			t.Fatalf("FindBySub = (%+v, %v), want the stored user", got, err)
		}
		if !got.CreatedAt.Equal(f.Now) || !got.UpdatedAt.Equal(f.Now) {
			t.Fatalf("created_at=%s updated_at=%s, want both %s", got.CreatedAt, got.UpdatedAt, f.Now)
		}
	})
}

func one(u *userdomain.User, err error) ([]*userdomain.User, error) {
	if u == nil {
		return nil, err
	}
	return []*userdomain.User{u}, err
}

// aliasProbeUser は、書き換えると共有が観測できる参照型のフィールドをすべて埋めた User を作る。
func aliasProbeUser(t *testing.T, tenantID string, now time.Time) *userdomain.User {
	t.Helper()
	probe := user(t, tenantID, "probe", now)
	probe.Name = new("Probe")
	probe.Lifecycle.RequiredActions = []idmdomain.RequiredAction{idmdomain.RequiredActionVerifyEmail}
	probe.Lifecycle.LastLoginAt = new(now)
	probe.Attributes = map[string]userdomain.AttributeValue{
		"teams": {Type: idmdomain.AttributeTypeStringArray, StringArray: []string{"blue"}},
	}
	return probe
}

func mutateUser(u *userdomain.User) {
	u.PreferredUsername = "mutated"
	*u.Name = "mutated"
	*u.Email = "mutated@example.com"
	u.Roles[0] = "mutated"
	u.Lifecycle.RequiredActions[0] = idmdomain.RequiredActionUpdatePassword
	*u.Lifecycle.LastLoginAt = u.Lifecycle.LastLoginAt.Add(time.Hour)
	u.Attributes["teams"].StringArray[0] = "mutated"
	u.Attributes["added"] = userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: new("mutated")}
}

func assertAliasProbeUnchanged(t *testing.T, users userports.UserRepository, id, when string) {
	t.Helper()
	got, err := users.FindBySub(context.Background(), id)
	if err != nil || got == nil {
		t.Fatalf("FindBySub %s = (%+v, %v), want the stored user", when, got, err)
	}
	if got.PreferredUsername != "probe" || got.Name == nil || *got.Name != "Probe" ||
		got.Email == nil || *got.Email != "probe@example.com" || !slices.Equal(got.Roles, []string{"reader"}) ||
		!slices.Equal(got.Lifecycle.RequiredActions, []idmdomain.RequiredAction{idmdomain.RequiredActionVerifyEmail}) ||
		got.Lifecycle.LastLoginAt == nil || !got.Lifecycle.LastLoginAt.Equal(got.CreatedAt) ||
		len(got.Attributes) != 1 || !slices.Equal(got.Attributes["teams"].StringArray, []string{"blue"}) {
		t.Fatalf("stored user changed %s: %+v", when, got)
	}
}

func envelope(t *testing.T, subject, email string, now time.Time) actiontoken.Envelope {
	t.Helper()
	return actiontoken.Envelope{
		ID:        newID(t),
		Purpose:   actiontoken.PurposeEmailChange,
		Subject:   subject,
		Payload:   actiontoken.Payload{userports.PayloadKeyNewEmail: email},
		Digest:    actiontoken.Digest(newID(t)),
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}
}

func emailCommit(source *userdomain.User, digest actiontoken.Digest, email string, now time.Time) userports.EmailChangeCommit {
	updated := *source
	updated.Email = new(email)
	updated.EmailVerified = true
	updated.UpdatedAt = now.Add(time.Minute)
	return userports.EmailChangeCommit{Digest: digest, Now: now.Add(time.Minute), User: &updated}
}

func runEmailChangeTokens(t *testing.T, newFixture NewFixture) {
	t.Helper()

	t.Run("purpose is bound and latest token replaces previous", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		foreign := envelope(t, f.EmailUser.ID, "foreign@example.com", f.Now)
		foreign.Purpose = actiontoken.PurposePasswordReset
		if err := f.EmailTokens.Save(ctx, foreign); !errors.Is(err, actiontoken.ErrPurposeMismatch) {
			t.Fatalf("foreign purpose error = %v, want ErrPurposeMismatch", err)
		}
		first := envelope(t, f.EmailUser.ID, "first@example.com", f.Now)
		second := envelope(t, f.EmailUser.ID, "second@example.com", f.Now.Add(time.Minute))
		if err := f.EmailTokens.Save(ctx, first); err != nil {
			t.Fatalf("Save first: %v", err)
		}
		if err := f.EmailTokens.Save(ctx, second); err != nil {
			t.Fatalf("Save second: %v", err)
		}
		if found, err := f.EmailTokens.Find(ctx, first.Digest); err != nil || found != nil {
			t.Fatalf("superseded token = (%+v, %v), want (nil, nil)", found, err)
		}
		if found, err := f.EmailTokens.Find(ctx, second.Digest); err != nil || found == nil {
			t.Fatalf("latest token = (%+v, %v), want readable token", found, err)
		}
	})

	t.Run("find does not consume and commit applies once", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		issued := envelope(t, f.EmailUser.ID, "new@example.com", f.Now)
		if err := f.EmailTokens.Save(ctx, issued); err != nil {
			t.Fatalf("Save: %v", err)
		}
		for range 2 {
			if found, err := f.EmailTokens.Find(ctx, issued.Digest); err != nil || found == nil {
				t.Fatalf("Find before consume = (%+v, %v)", found, err)
			}
		}
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for range 2 {
			wg.Go(func() {
				results <- f.EmailTokens.ConsumeAndApply(ctx, emailCommit(f.EmailUser, issued.Digest, "new@example.com", f.Now))
			})
		}
		wg.Wait()
		close(results)
		successes, consumed := 0, 0
		for err := range results {
			switch {
			case err == nil:
				successes++
			case errors.Is(err, actiontoken.ErrAlreadyConsumed):
				consumed++
			default:
				t.Fatalf("ConsumeAndApply: %v", err)
			}
		}
		if successes != 1 || consumed != 1 {
			t.Fatalf("results = %d success, %d consumed; want 1 and 1", successes, consumed)
		}
		stored, err := f.Users.FindBySub(ctx, f.EmailUser.ID)
		if err != nil || stored == nil || stored.Email == nil || *stored.Email != "new@example.com" {
			t.Fatalf("committed user = (%+v, %v)", stored, err)
		}
	})

	// ペイロードの map は、保存した側と読んだ側のどちらが書き換えても保存内容と共有しない。
	t.Run("payload does not alias stored token", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		issued := envelope(t, f.EmailUser.ID, "new@example.com", f.Now)
		if err := f.EmailTokens.Save(ctx, issued); err != nil {
			t.Fatalf("Save: %v", err)
		}
		issued.Payload[userports.PayloadKeyNewEmail] = "saved-side@example.com"
		found, err := f.EmailTokens.Find(ctx, issued.Digest)
		if err != nil || found == nil || found.Payload[userports.PayloadKeyNewEmail] != "new@example.com" {
			t.Fatalf("Find after mutating the Save argument = (%+v, %v), want new@example.com", found, err)
		}
		found.Payload[userports.PayloadKeyNewEmail] = "read-side@example.com"
		again, err := f.EmailTokens.Find(ctx, issued.Digest)
		if err != nil || again == nil || again.Payload[userports.PayloadKeyNewEmail] != "new@example.com" {
			t.Fatalf("Find after mutating a result = (%+v, %v), want new@example.com", again, err)
		}
	})

	t.Run("timestamps read back truncated to microseconds", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		issued := envelope(t, f.EmailUser.ID, "new@example.com", f.Now.Add(789*time.Nanosecond))
		if err := f.EmailTokens.Save(ctx, issued); err != nil {
			t.Fatalf("Save: %v", err)
		}
		found, err := f.EmailTokens.Find(ctx, issued.Digest)
		if err != nil || found == nil {
			t.Fatalf("Find = (%+v, %v), want the stored token", found, err)
		}
		if !found.IssuedAt.Equal(f.Now) || !found.ExpiresAt.Equal(f.Now.Add(time.Hour)) {
			t.Fatalf("issued_at=%s expires_at=%s, want %s and %s", found.IssuedAt, found.ExpiresAt, f.Now, f.Now.Add(time.Hour))
		}
	})

	t.Run("unknown digest is already consumed", func(t *testing.T) {
		f := newFixture(t)
		err := f.EmailTokens.ConsumeAndApply(context.Background(), emailCommit(
			f.EmailUser, actiontoken.Digest(newID(t)), "new@example.com", f.Now,
		))
		if !errors.Is(err, actiontoken.ErrAlreadyConsumed) {
			t.Fatalf("ConsumeAndApply error = %v, want ErrAlreadyConsumed", err)
		}
	})
}

func runTenantAttributeSchema(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if got, err := f.Schemas.FindByTenant(ctx, f.TenantA); err != nil || got != nil {
		t.Fatalf("missing schema = (%+v, %v), want (nil, nil)", got, err)
	}
	claim := "region"
	want := &userdomain.TenantUserAttributeSchema{
		TenantID: f.TenantA,
		Attributes: []userdomain.UserAttributeDef{{
			Key: "region", Label: "Region", Type: idmdomain.AttributeTypeString,
			Required: true, Visibility: idmdomain.AttrVisibilityClaimExposed, ClaimName: &claim,
		}},
		CreatedAt: f.Now,
		UpdatedAt: f.Now,
	}
	if err := f.Schemas.Save(ctx, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Schemas.FindByTenant(ctx, f.TenantA)
	if err != nil || got == nil || len(got.Attributes) != 1 || got.Attributes[0].Key != "region" {
		t.Fatalf("FindByTenant = (%+v, %v), want round trip", got, err)
	}
	got.Attributes[0].Key = "mutated"
	again, err := f.Schemas.FindByTenant(ctx, f.TenantA)
	if err != nil || again == nil || again.Attributes[0].Key != "region" {
		t.Fatalf("returned schema aliases stored state: (%+v, %v)", again, err)
	}
	if err := f.Schemas.Delete(ctx, f.TenantA); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Schemas.FindByTenant(ctx, f.TenantA); err != nil || got != nil {
		t.Fatalf("schema after Delete = (%+v, %v), want (nil, nil)", got, err)
	}
}

func runImportRow(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	imported := user(t, f.TenantA, "imported", f.Now)
	mutation := userports.UserImportRowMutation{
		After: imported, Changed: []string{"password", "email"}, ActorUserID: f.ActorUserID,
		AuditEventType: "user.imported", PasswordHistoryHash: "history-hash", Now: f.Now,
	}
	if err := f.ImportRows.CommitUserImportRow(context.Background(), mutation); err != nil {
		t.Fatalf("CommitUserImportRow: %v", err)
	}
	f.AssertImported(t, mutation)
	if !slices.Equal(mutation.Changed, []string{"password", "email"}) {
		t.Fatalf("committer mutated input Changed: %v", mutation.Changed)
	}
}
