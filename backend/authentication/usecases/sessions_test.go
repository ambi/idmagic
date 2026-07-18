package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func seedSession(t *testing.T, store *memory.SessionStore, id, sub string, authTime time.Time) {
	t.Helper()
	if err := store.Save(context.Background(), &domain.LoginSession{
		ID: id, TenantID: tenancydomain.DefaultTenantID, UserID: sub, AuthTime: authTime.Unix(),
		AMR: []string{"pwd"}, ACR: "urn:mace:incommon:iap:silver",
		ExpiresAt: authTime.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestListSessionsMarksCurrentAndSortsDesc(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	base := time.Now().UTC().Truncate(time.Second)
	seedSession(t, store, "s1", "alice", base)
	seedSession(t, store, "s2", "alice", base.Add(time.Minute))
	seedSession(t, store, "s3", "bob", base.Add(2*time.Minute))

	views, err := usecases.ListSessions(ctx, store, "alice", "s2")
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("len(views)=%d, want 2", len(views))
	}
	// 新しい順: s2 が先頭。
	if views[0].ID != "s2" || !views[0].Current {
		t.Fatalf("first view=%#v, want s2 current", views[0])
	}
	if views[1].ID != "s1" || views[1].Current {
		t.Fatalf("second view=%#v, want s1 not current", views[1])
	}
}

func TestRevokeOwnSessionRejectsOthersSession(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	base := time.Now().UTC().Truncate(time.Second)
	seedSession(t, store, "s1", "alice", base)
	seedSession(t, store, "s2", "bob", base)

	// alice が bob のセッションを失効しようとしても拒否される。
	if err := usecases.RevokeOwnSession(ctx, usecases.SessionDeps{Store: store},
		"alice", "s2", base); !errors.Is(err, usecases.ErrSessionNotFound) {
		t.Fatalf("error=%v, want ErrSessionNotFound", err)
	}
	if sess, _ := store.Find(ctx, "s2"); sess == nil {
		t.Fatal("bob's session was deleted")
	}

	// 自分のセッションは失効でき、SessionEnded が発火する。
	var events []spec.DomainEvent
	if err := usecases.RevokeOwnSession(ctx, usecases.SessionDeps{
		Store: store, Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, "alice", "s1", base); err != nil {
		t.Fatal(err)
	}
	if sess, _ := store.Find(ctx, "s1"); sess != nil {
		t.Fatal("alice's session was not deleted")
	}
	if len(events) != 1 || events[0].EventType() != "SessionEnded" {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestRevokeOtherSessionsKeepsCurrent(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	base := time.Now().UTC().Truncate(time.Second)
	seedSession(t, store, "s1", "alice", base)
	seedSession(t, store, "s2", "alice", base.Add(time.Minute))
	seedSession(t, store, "s3", "alice", base.Add(2*time.Minute))

	var events []spec.DomainEvent
	revokedIDs, err := usecases.RevokeOtherSessions(ctx, usecases.SessionDeps{
		Store: store, Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, "alice", "s2", base)
	if err != nil {
		t.Fatal(err)
	}
	remaining, _ := usecases.ListSessions(ctx, store, "alice", "s2")
	if len(remaining) != 1 || remaining[0].ID != "s2" {
		t.Fatalf("remaining=%#v, want only s2", remaining)
	}
	if len(events) != 2 {
		t.Fatalf("len(events)=%d, want 2", len(events))
	}
	// oauth2 側の RevokeTokensBySid 呼び出し元 (account_sessions_handler) が失効対象の
	// sid 一覧をここから受け取る (ADR-127)。
	if len(revokedIDs) != 2 || !slices.Contains(revokedIDs, "s1") || !slices.Contains(revokedIDs, "s3") {
		t.Fatalf("revokedIDs=%#v, want [s1 s3]", revokedIDs)
	}
}
