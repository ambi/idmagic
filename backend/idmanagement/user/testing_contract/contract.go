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
