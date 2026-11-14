package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

// 保持期間の期待値は仕様の表から書き写し、製品の定数から計算しない。種別名は発行元の
// EventType() から取り、Audit の分類が発行元の種別名から外れたことも検出する。
func daysAgo(now time.Time, d int) time.Time {
	return now.Add(-time.Duration(d) * 24 * time.Hour)
}

func seedAudit(t *testing.T, m audit.Module, id, eventType string, at time.Time, payload map[string]any) {
	t.Helper()
	if payload == nil {
		payload = map[string]any{}
	}
	payload["tenantId"] = tenancydomain.DefaultTenantID
	if err := m.AuditEventRepo.Append(testing_tenant.Default(context.Background()), &auditports.AuditEventRecord{
		ID: id, TenantID: tenancydomain.DefaultTenantID, Type: eventType, OccurredAt: at, Payload: payload,
	}); err != nil {
		t.Fatal(err)
	}
}

func remainingAuditIDs(t *testing.T, m audit.Module) map[string]bool {
	t.Helper()
	recs, err := m.AuditEventRepo.List(testing_tenant.Default(context.Background()), auditports.AuditEventQuery{Limit: 1000})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, r := range recs {
		out[r.ID] = true
	}
	return out
}

func assertRetention(t *testing.T, got map[string]bool, kept, gone []string) {
	t.Helper()
	for _, id := range kept {
		if !got[id] {
			t.Errorf("%s は保持期間の内側なので残っていなければならない", id)
		}
	}
	for _, id := range gone {
		if got[id] {
			t.Errorf("%s は保持期間の外側なので消えていなければならない", id)
		}
	}
}

//spec:covers REQ-AUDIT-013: 種類ごとの保持期間の直前の記録が残り、直後の記録が消える。なりすましもデフォルトの期間で消える。
func TestRetentionSweepAppliesThePeriodOfEachEventType(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	m := audit.NewMemoryModule(0)

	periods := []struct {
		name string
		typ  string
		days int
	}{
		{"fail", (&authdomain.AuthenticationFailed{}).EventType(), 30},
		{"agg", (&authdomain.AuthenticationEventAggregated{}).EventType(), 90},
		{"mfa", (&authdomain.MfaChallengeIssued{}).EventType(), 90},
		{"backup", (&authdomain.BackupCodeConsumed{}).EventType(), 90},
		{"bypass", (&authdomain.MfaEnrollmentBypassExpired{}).EventType(), 90},
		{"sess", (&authdomain.SessionEnded{}).EventType(), 90},
		{"ok", (&authdomain.UserAuthenticated{}).EventType(), 365},
		{"consent", (&consentdomain.ConsentRevokedEvent{}).EventType(), 365},
		// なりすましは短い期間へ入れず、デフォルトの期間で消す。無期限には残さない。
		{"imp-start", (&authdomain.SessionImpersonationStarted{}).EventType(), 365},
		{"imp-end", (&authdomain.SessionImpersonationEnded{}).EventType(), 365},
	}
	var kept, gone []string
	for _, p := range periods {
		inside, outside := p.name+"-inside", p.name+"-outside"
		seedAudit(t, m, inside, p.typ, daysAgo(now, p.days).Add(time.Minute), nil)
		seedAudit(t, m, outside, p.typ, daysAgo(now, p.days).Add(-time.Minute), nil)
		kept = append(kept, inside)
		gone = append(gone, outside)
	}
	deleted, err := m.PurgeExpiredEvents(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != int64(len(gone)) {
		t.Errorf("deleted=%d, want %d", deleted, len(gone))
	}
	assertRetention(t, remainingAuditIDs(t, m), kept, gone)
}

func TestRetentionSweepKeepsFailureUsernamePlaintext(t *testing.T) {
	// AuthenticationFailed.username は redact されず、失敗の保持期間の間そのまま残る。
	ctx := testing_tenant.Default(context.Background())
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	m := audit.NewMemoryModule(0)
	seedAudit(t, m, "fail-8", (&authdomain.AuthenticationFailed{}).EventType(), daysAgo(now, 8), map[string]any{"username": "alice"})

	if _, err := m.PurgeExpiredEvents(ctx, now); err != nil {
		t.Fatal(err)
	}
	got, _ := m.AuditEventRepo.FindByID(ctx, "fail-8")
	if got == nil {
		t.Fatal("fail-8 should survive the 30-day failure period at day 8")
	}
	if got.Payload["username"] != "alice" {
		t.Fatalf("username should remain plaintext, got %#v", got.Payload["username"])
	}
}

// 保持期間の内側では残り、外側では消える。
//
// 片側だけを見るテストは行を区別できない。内側だけなら「何も消さない」実装が通り、
// 外側だけなら「全部消す」実装が通る。種類も 2 つ要る。行が「セキュリティおよび認可」と
// 2 つ挙げているのに、期間の型が違う (認可はデフォルトの 365 日、失敗は種類別の 30 日) ため、
// 片方の種類だけでは他方の期間が守られていることを言えない。
//
//spec:covers GDPR-PROCESSING-RECORDS: セキュリティイベントと認可イベントの監査記録は、定義済みの期間の内側で残り外側で消える。
func TestRetentionKeepsSecurityAndAuthorizationRecordsWithinTheDefinedPeriod(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	m := audit.NewMemoryModule(0)

	securityType := (&authdomain.AuthenticationFailed{}).EventType()
	authorizationType := (&consentdomain.ConsentRevokedEvent{}).EventType()
	seedAudit(t, m, "security-inside", securityType, daysAgo(now, 29), nil)
	seedAudit(t, m, "security-outside", securityType, daysAgo(now, 31), nil)
	seedAudit(t, m, "authorization-inside", authorizationType, daysAgo(now, 364), nil)
	seedAudit(t, m, "authorization-outside", authorizationType, daysAgo(now, 366), nil)

	if _, err := m.PurgeExpiredEvents(ctx, now); err != nil {
		t.Fatal(err)
	}
	assertRetention(t, remainingAuditIDs(t, m),
		[]string{"security-inside", "authorization-inside"},
		[]string{"security-outside", "authorization-outside"})
}
