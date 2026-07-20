package usecases_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/session/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/authentication/session/usecases"
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

// ADR-127: RP-Initiated Logout (/end_session) は所有者 (sub) を検証済みでない sid
// (id_token_hint または browser cookie 由来) から直接失効する。既に失効済み/未知の
// sid は Find が有効セッションのみ返すため自然に no-op (idempotent) になる。
func TestEndSessionRevokesBySidAndEmitsEvent(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	base := time.Now().UTC().Truncate(time.Second)
	seedSession(t, store, "s1", "alice", base)

	var events []spec.DomainEvent
	if err := usecases.EndSession(ctx, usecases.SessionDeps{
		Store: store, Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, "s1", base); err != nil {
		t.Fatal(err)
	}
	if sess, _ := store.Find(ctx, "s1"); sess != nil {
		t.Fatal("s1 was not revoked")
	}
	if len(events) != 1 || events[0].EventType() != "SessionEnded" {
		t.Fatalf("unexpected events: %#v", events)
	}

	// idempotent: 2 回目 (Find は有効セッションのみ返すため) の呼び出しはイベントを
	// 再発行しない。
	events = nil
	if err := usecases.EndSession(ctx, usecases.SessionDeps{
		Store: store, Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, "s1", base); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no re-emission, got %#v", events)
	}
}

// wi-28 T007 (ADR-127 決定9): admin 向け session 管理は self-service と対で、
// 既存の ListUserSignInActivity と同じアクセス制御パターン (TenantAdministrator,
// resource=User/input.user_id) を踏襲する。current マーカーは持たない。
func TestAdminListSessionsHasNoCurrentMarker(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	base := time.Now().UTC().Truncate(time.Second)
	seedSession(t, store, "s1", "alice", base)
	seedSession(t, store, "s2", "alice", base.Add(time.Minute))
	seedSession(t, store, "s3", "bob", base)

	views, err := usecases.AdminListSessions(ctx, store, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 {
		t.Fatalf("len(views)=%d, want 2", len(views))
	}
	// 新しい順: s2 が先頭。
	if views[0].ID != "s2" || views[1].ID != "s1" {
		t.Fatalf("unexpected order: %#v", views)
	}
	if views[0].UserID != "alice" {
		t.Fatalf("UserID=%q, want alice", views[0].UserID)
	}
}

func TestAdminRevokeSessionRejectsSessionOfOtherUser(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	base := time.Now().UTC().Truncate(time.Second)
	seedSession(t, store, "s1", "alice", base)

	// admin が bob 向けのURLで alice のセッションを失効しようとしても対象不一致で拒否。
	if err := usecases.AdminRevokeSession(ctx, usecases.SessionDeps{Store: store},
		"admin-1", "bob", "s1", base); !errors.Is(err, usecases.ErrSessionNotFound) {
		t.Fatalf("error=%v, want ErrSessionNotFound", err)
	}
	if sess, _ := store.Find(ctx, "s1"); sess == nil {
		t.Fatal("alice's session was revoked despite user_id mismatch")
	}

	var events []spec.DomainEvent
	if err := usecases.AdminRevokeSession(ctx, usecases.SessionDeps{
		Store: store, Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, "admin-1", "alice", "s1", base); err != nil {
		t.Fatal(err)
	}
	if sess, _ := store.Find(ctx, "s1"); sess != nil {
		t.Fatal("alice's session was not revoked")
	}
	if len(events) != 1 || events[0].EventType() != "SessionEnded" {
		t.Fatalf("unexpected events: %#v", events)
	}
	ended, ok := events[0].(*authdomain.SessionEnded)
	if !ok || ended.ActorUserID != "admin-1" {
		t.Fatalf("ActorUserID should be the admin, got %#v", events[0])
	}
}

func TestAdminRevokeUserSessionsRevokesAllWithNoExclusion(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	base := time.Now().UTC().Truncate(time.Second)
	seedSession(t, store, "s1", "alice", base)
	seedSession(t, store, "s2", "alice", base.Add(time.Minute))
	seedSession(t, store, "s3", "bob", base)

	revokedIDs, err := usecases.AdminRevokeUserSessions(ctx, usecases.SessionDeps{Store: store}, "admin-1", "alice", base)
	if err != nil {
		t.Fatal(err)
	}
	if len(revokedIDs) != 2 || !slices.Contains(revokedIDs, "s1") || !slices.Contains(revokedIDs, "s2") {
		t.Fatalf("revokedIDs=%#v, want [s1 s2]", revokedIDs)
	}
	remaining, _ := usecases.AdminListSessions(ctx, store, "alice")
	if len(remaining) != 0 {
		t.Fatalf("remaining=%#v, want none", remaining)
	}
	if sess, _ := store.Find(ctx, "s3"); sess == nil {
		t.Fatal("bob's session must not be revoked")
	}
}

func TestEndSessionUnknownSidIsNoop(t *testing.T) {
	ctx := context.Background()
	store := memory.NewSessionStore()
	var events []spec.DomainEvent
	if err := usecases.EndSession(ctx, usecases.SessionDeps{
		Store: store, Emit: func(e spec.DomainEvent) { events = append(events, e) },
	}, "unknown-sid", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("unexpected events: %#v", events)
	}
}
