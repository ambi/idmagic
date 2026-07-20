package usecases_test

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authnmemory "github.com/ambi/idmagic/backend/authentication/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/authentication/usecases"
)

func daysAgo(now time.Time, d int) time.Time {
	return now.Add(-time.Duration(d) * 24 * time.Hour)
}

func seedAudit(t *testing.T, store *auditmemory.AuditEventStore, id, eventType string, at time.Time) {
	t.Helper()
	if err := store.Append(context.Background(), &auditports.AuditEventRecord{
		ID: id, TenantID: tenancydomain.DefaultTenantID, Type: eventType, OccurredAt: at,
		Payload: map[string]any{"tenantId": tenancydomain.DefaultTenantID},
	}); err != nil {
		t.Fatal(err)
	}
}

func remainingAuditIDs(t *testing.T, store *auditmemory.AuditEventStore) map[string]bool {
	t.Helper()
	recs, err := store.List(context.Background(), auditports.AuditEventQuery{Limit: 1000})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, r := range recs {
		out[r.ID] = true
	}
	return out
}

func TestRetentionSweepDeletesByTypeBoundaries(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := auditmemory.NewAuditEventStore(0)

	// 失敗詳細 30 日: 29 日前は残り 31 日前は消える。
	seedAudit(t, store, "fail-29", (&authdomain.AuthenticationFailed{}).EventType(), daysAgo(now, 29))
	seedAudit(t, store, "fail-31", (&authdomain.AuthenticationFailed{}).EventType(), daysAgo(now, 31))
	// 成功 365 日: 364 日前は残り 366 日前は消える。
	seedAudit(t, store, "ok-364", (&authdomain.UserAuthenticated{}).EventType(), daysAgo(now, 364))
	seedAudit(t, store, "ok-366", (&authdomain.UserAuthenticated{}).EventType(), daysAgo(now, 366))
	// セッション / bucket 90 日: 89 日前は残り 91 日前は消える。
	seedAudit(t, store, "sess-89", (&authdomain.SessionStarted{}).EventType(), daysAgo(now, 89))
	seedAudit(t, store, "sess-91", (&authdomain.SessionStarted{}).EventType(), daysAgo(now, 91))
	seedAudit(t, store, "agg-91", (&authdomain.AuthenticationEventAggregated{}).EventType(), daysAgo(now, 91))
	// impersonation: cap 未設定なら無期限保持 (400 日前でも残る)。
	seedAudit(t, store, "imp-400", (&authdomain.SessionImpersonationStarted{}).EventType(), daysAgo(now, 400))

	res, err := usecases.RunRetentionSweep(ctx, store, nil, nil, usecases.DefaultRetentionPolicy(), now)
	if err != nil {
		t.Fatal(err)
	}
	if res.AuditEvents != 4 {
		t.Fatalf("deleted audit=%d, want 4", res.AuditEvents)
	}
	got := remainingAuditIDs(t, store)
	wantKept := []string{"fail-29", "ok-364", "sess-89", "imp-400"}
	wantGone := []string{"fail-31", "ok-366", "sess-91", "agg-91"}
	for _, id := range wantKept {
		if !got[id] {
			t.Errorf("%s should be kept", id)
		}
	}
	for _, id := range wantGone {
		if got[id] {
			t.Errorf("%s should be deleted", id)
		}
	}
}

func TestRetentionSweepKeepsFailureUsernamePlaintext(t *testing.T) {
	// ADR-104 (ADR-046 の username 条項を撤回): AuthenticationFailed.username は redact されず、
	// 他の failure イベントと同じ保持期間 (FailDays) でそのまま保持される。
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := auditmemory.NewAuditEventStore(0)
	oldFailure := &auditports.AuditEventRecord{
		ID: "fail-8", TenantID: tenancydomain.DefaultTenantID,
		Type: (&authdomain.AuthenticationFailed{}).EventType(), OccurredAt: daysAgo(now, 8),
		Payload: map[string]any{"tenantId": tenancydomain.DefaultTenantID, "username": "alice"},
	}
	if err := store.Append(ctx, oldFailure); err != nil {
		t.Fatal(err)
	}

	if _, err := usecases.RunRetentionSweep(ctx, store, nil, nil, usecases.DefaultRetentionPolicy(), now); err != nil {
		t.Fatal(err)
	}
	oldGot, _ := store.FindByID(ctx, "fail-8")
	if oldGot == nil {
		t.Fatal("fail-8 should survive the 30-day FailDays cutoff at day 8")
	}
	if oldGot.Payload["username"] != "alice" {
		t.Fatalf("username should remain plaintext, got %#v", oldGot.Payload["username"])
	}
}

func TestRetentionSweepGlobalCapShortensAndDeletesImpersonation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := auditmemory.NewAuditEventStore(0)
	// global cap 30 日: 成功も impersonation も 31 日前は消える。
	seedAudit(t, store, "ok-31", (&authdomain.UserAuthenticated{}).EventType(), daysAgo(now, 31))
	seedAudit(t, store, "ok-29", (&authdomain.UserAuthenticated{}).EventType(), daysAgo(now, 29))
	seedAudit(t, store, "imp-31", (&authdomain.SessionImpersonationStarted{}).EventType(), daysAgo(now, 31))

	policy := usecases.DefaultRetentionPolicy()
	policy.MaxDays = 30
	if _, err := usecases.RunRetentionSweep(ctx, store, nil, nil, policy, now); err != nil {
		t.Fatal(err)
	}
	got := remainingAuditIDs(t, store)
	if !got["ok-29"] {
		t.Error("ok-29 should be kept under cap=30")
	}
	if got["ok-31"] {
		t.Error("ok-31 should be deleted under cap=30")
	}
	if got["imp-31"] {
		t.Error("imp-31 should be deleted: global cap shortens impersonation")
	}
}

func TestRetentionSweepDeletesOldBuckets(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := authnmemory.NewAuthEventBucketStore()
	// 91 日前の窓と直近の窓を作る。
	if _, err := store.Record(ctx, "failed_login", tenancydomain.DefaultTenantID, "old-key", daysAgo(now, 91)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(ctx, "failed_login", tenancydomain.DefaultTenantID, "fresh-key", now); err != nil {
		t.Fatal(err)
	}
	res, err := usecases.RunRetentionSweep(ctx, nil, store, nil, usecases.DefaultRetentionPolicy(), now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Buckets != 1 {
		t.Fatalf("deleted buckets=%d, want 1", res.Buckets)
	}
	buckets, err := store.List(ctx, tenancydomain.DefaultTenantID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 1 || buckets[0].KeyHash != "fresh-key" {
		t.Fatalf("remaining buckets=%#v, want only fresh-key", buckets)
	}
}

// TestRetentionSweepDeletesExpiredSessions: wi-253 Plan §7 の housekeeping cleanup を
// retention sweep (ADR-045) に統合する。SessionDays (既定 90 日) を LoginSession の
// tombstone/期限切れ行の物理削除 cutoff に転用する。
func TestRetentionSweepDeletesExpiredSessions(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	store := sessionmemory.NewSessionStore()

	mustSave := func(id string, expiresAt time.Time) {
		t.Helper()
		if err := store.Save(ctx, &sessiondomain.LoginSession{
			ID: id, UserID: "user-1", AMR: []string{"pwd"}, ACR: "urn:idmagic:acr:pwd",
			ExpiresAt: expiresAt,
		}); err != nil {
			t.Fatal(err)
		}
	}
	// 90 日 cutoff: 89 日前に失効した行は残り、91 日前に失効した行は消える。
	mustSave("expired-89", daysAgo(now, 89))
	mustSave("expired-91", daysAgo(now, 91))
	mustSave("active", now.Add(time.Hour))

	res, err := usecases.RunRetentionSweep(ctx, nil, nil, store, usecases.DefaultRetentionPolicy(), now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Sessions != 1 {
		t.Fatalf("deleted sessions=%d, want 1", res.Sessions)
	}
	if owned, _ := store.FindOwned(ctx, "expired-91", "user-1"); owned != nil {
		t.Error("expired-91 should be purged past the 90-day cutoff")
	}
	if owned, _ := store.FindOwned(ctx, "expired-89", "user-1"); owned == nil {
		t.Error("expired-89 should survive within the 90-day cutoff")
	}
	if owned, _ := store.FindOwned(ctx, "active", "user-1"); owned == nil {
		t.Error("active session should survive")
	}
}
