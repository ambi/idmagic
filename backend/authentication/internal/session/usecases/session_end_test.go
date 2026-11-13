package usecases

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	memory "github.com/ambi/idmagic/backend/authentication/internal/session/db_memory"
	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

type sessionEndFixture struct {
	ctx     context.Context
	store   *memory.SessionStore
	quota   tenantports.QuotaRepository
	manager *SessionManager
	events  []*authdomain.SessionEnded
}

func newSessionEndFixture(t *testing.T) *sessionEndFixture {
	t.Helper()
	f := &sessionEndFixture{
		ctx:   testing_tenant.Default(context.Background()),
		store: memory.NewSessionStore(),
		quota: tenancy.NewMemoryModule().QuotaRepo,
	}
	f.manager = NewSessionManager(f.store)
	f.manager.QuotaRepo = f.quota
	f.manager.Emit = func(event spec.DomainEvent) {
		if ended, ok := event.(*authdomain.SessionEnded); ok {
			f.events = append(f.events, ended)
		}
	}
	return f
}

func (f *sessionEndFixture) login(t *testing.T, sub string) string {
	t.Helper()
	created, err := f.manager.Create(f.ctx, sub, []string{"pwd"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return created.SessionID
}

func (f *sessionEndFixture) activeSessions(t *testing.T) int {
	t.Helper()
	usage, err := f.quota.GetUsage(f.ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	return usage.ActiveSessions
}

//spec:covers REQ-AUTHENTICATION-045: SAML のシングルログアウトと WS-Federation のサインアウトが Cookie で指したセッションを `logout` で失効させ、使用量を一つ減らして `SessionEnded` を一度だけ発行し、二度目のサインアウトは使用量とイベントを変えない。
func TestSignOutThroughTheCookieEndsTheSessionOnce(t *testing.T) {
	f := newSessionEndFixture(t)
	sid := f.login(t, "user-1")
	f.login(t, "user-2")
	cookie := "__Host-" + SessionCookie + "=" + sid

	if err := f.manager.Revoke(f.ctx, cookie); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := f.manager.Revoke(f.ctx, cookie); err != nil {
		t.Fatalf("second Revoke: %v", err)
	}

	if got := f.activeSessions(t); got != 1 {
		t.Fatalf("active_sessions = %d, want 1", got)
	}
	if len(f.events) != 1 {
		t.Fatalf("SessionEnded = %d events, want 1", len(f.events))
	}
	if e := f.events[0]; e.SessionID != sid || e.UserID != "user-1" || e.ActorUserID != "user-1" || e.Reason != spec.SessionEndLogout || e.AllSessions {
		t.Fatalf("SessionEnded = %+v", e)
	}
	if found, err := f.store.Find(f.ctx, sid); err != nil || found != nil {
		t.Fatalf("Find after sign-out = %+v, %v; want no active session", found, err)
	}
}

//spec:covers REQ-AUTHENTICATION-045: 主体が有効でないと分かったセッションを `other` で失効させ、使用量を減らして `SessionEnded` を発行する。
func TestEndingTheSessionOfAnInactivePrincipalReleasesTheQuota(t *testing.T) {
	f := newSessionEndFixture(t)
	sid := f.login(t, "user-1")

	if err := f.manager.End(f.ctx, sid, spec.SessionEndOther, time.Now().UTC()); err != nil {
		t.Fatalf("End: %v", err)
	}

	if got := f.activeSessions(t); got != 0 {
		t.Fatalf("active_sessions = %d, want 0", got)
	}
	if len(f.events) != 1 || f.events[0].Reason != spec.SessionEndOther || f.events[0].SessionID != sid {
		t.Fatalf("SessionEnded = %+v, want one event with reason other", f.events)
	}
}

// revokedWhileListing は、一覧した直後に別の経路が同じセッションを失効させた保存先である。
type revokedWhileListing struct {
	*memory.SessionStore
}

func (s revokedWhileListing) ListBySub(ctx context.Context, sub string) ([]*domain.LoginSession, error) {
	sessions, err := s.SessionStore.ListBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	listed := make([]*domain.LoginSession, 0, len(sessions))
	for _, sess := range sessions {
		snapshot := *sess
		listed = append(listed, &snapshot)
		if _, err := s.Revoke(ctx, sess.ID, spec.SessionEndLogout, time.Now().UTC()); err != nil {
			return nil, err
		}
	}
	return listed, nil
}

//spec:covers REQ-AUTHENTICATION-045: 一覧の後に別の経路が失効させたセッションは、全失効が使用量を減らさず `SessionEnded` も発行しない。
func TestRevokingAllSessionsSkipsTheOnesAnotherPathEnded(t *testing.T) {
	f := newSessionEndFixture(t)
	f.login(t, "user-1")
	if err := f.quota.CheckAndIncrement(f.ctx, tenancydomain.DefaultTenantID, tenancydomain.ResourceActiveSessions, 1); err != nil {
		t.Fatal(err)
	}
	deps := SessionDeps{Store: revokedWhileListing{f.store}, Emit: f.manager.Emit, QuotaRepo: f.quota}

	revoked, err := AdminRevokeUserSessions(f.ctx, deps, "admin-1", "user-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("AdminRevokeUserSessions: %v", err)
	}

	if len(revoked) != 1 {
		t.Fatalf("revoked = %v, want the listed session", revoked)
	}
	if got := f.activeSessions(t); got != 2 {
		t.Fatalf("active_sessions = %d, want 2 (the other path owns the release)", got)
	}
	if len(f.events) != 0 {
		t.Fatalf("SessionEnded = %+v, want none", f.events)
	}
}

//spec:covers REQ-AUTHENTICATION-045: 期限切れのセッションを失効させても、使用量を占めていないので減らさない。
func TestRevokingAnExpiredSessionKeepsTheUsage(t *testing.T) {
	f := newSessionEndFixture(t)
	created := time.Now().UTC().Add(-2 * time.Hour)
	expired, err := f.manager.Create(f.ctx, "user-1", []string{"pwd"}, created)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := RevokeOwnSession(f.ctx, SessionDeps{Store: f.store, Emit: f.manager.Emit, QuotaRepo: f.quota}, "user-1", expired.SessionID, time.Now().UTC()); err != nil {
		t.Fatalf("RevokeOwnSession: %v", err)
	}

	if got := f.activeSessions(t); got != 1 {
		t.Fatalf("active_sessions = %d, want 1 (left for the recount)", got)
	}
	if len(f.events) != 1 {
		t.Fatalf("SessionEnded = %d events, want 1", len(f.events))
	}
}
