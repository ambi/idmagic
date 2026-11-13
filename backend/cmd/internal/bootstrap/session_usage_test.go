package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-TENANCY-051: 保持期限バッチが、期限切れ、失効、削除で終わったセッションの分を含む使用量を、テナントごとに有効なセッション（確認待ちを含む）の件数へ置き換え、ほかのリソースの使用量を変えない。
func TestRetentionSweepRecountsActiveSessionUsage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	idm := idmanagement.NewMemoryModule()
	deps := &Dependencies{
		Tenancy: tenancy.NewMemoryModule(), Audit: audit.NewMemoryModule(100), IdManagement: idm,
		Authentication: authentication.NewMemoryModule(idm.UserRepo), OAuth2: oauth2.Module{EventSink: &recordingEventSink{}},
	}
	tenants := map[string]*tenancydomain.Tenant{}
	for _, id := range []string{"tenant-a", "tenant-b"} {
		tenant := &tenancydomain.Tenant{ID: id, Realm: id, Status: tenancydomain.TenantStatusActive}
		if err := deps.Tenancy.TenantRepo.Save(ctx, tenant); err != nil {
			t.Fatal(err)
		}
		tenants[id] = tenant
		if err := deps.Tenancy.QuotaRepo.CheckAndIncrement(ctx, id, tenancydomain.ResourceGroups, 2); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	saveSession := func(tenantID string, expiresAt time.Time, pending bool) string {
		t.Helper()
		n++
		id := tenantID + "-session-" + string(rune('0'+n))
		sess := &domain.LoginSession{
			ID: id, TenantID: tenantID, UserID: "user", AuthTime: now.Unix(), AMR: []string{"pwd"},
			ACR: "urn:idmagic:acr:pwd", ExpiresAt: expiresAt, AuthenticationPending: pending,
		}
		if err := deps.Authentication.SessionStore.Save(ctx, sess); err != nil {
			t.Fatal(err)
		}
		return id
	}
	// tenant-a：有効が一件と確認待ちが一件。期限切れ二件と失効済み一件は使用量を占めない。
	// 加算は五件分あり、期限切れの二件と、減算されずに消えた一件の分だけ多い。
	saveSession("tenant-a", now.Add(time.Minute), false)
	saveSession("tenant-a", now.Add(time.Hour), true)
	saveSession("tenant-a", now.Add(-time.Minute), false)
	saveSession("tenant-a", now, false)
	revoked := saveSession("tenant-a", now.Add(time.Hour), false)
	tenantCtx := tenantports.WithTenant(ctx, tenants["tenant-a"], "", "")
	if _, err := deps.Authentication.SessionStore.Revoke(tenantCtx, revoked, spec.SessionEndLogout, now); err != nil {
		t.Fatal(err)
	}
	if err := deps.Tenancy.QuotaRepo.CheckAndIncrement(ctx, "tenant-a", tenancydomain.ResourceActiveSessions, 5); err != nil {
		t.Fatal(err)
	}
	// tenant-b：有効が一件あるのに、使用量は 0 まで減りすぎている。
	saveSession("tenant-b", now.Add(time.Hour), false)

	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}

	for tenantID, want := range map[string]int{"tenant-a": 2, "tenant-b": 1} {
		usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, tenantID)
		if err != nil {
			t.Fatal(err)
		}
		if usage.ActiveSessions != want || usage.Groups != 2 {
			t.Fatalf("%s: usage active_sessions=%d groups=%d, want active_sessions=%d groups=2", tenantID, usage.ActiveSessions, usage.Groups, want)
		}
	}
}
