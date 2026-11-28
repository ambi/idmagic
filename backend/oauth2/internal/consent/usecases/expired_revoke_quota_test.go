package usecases

import (
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// 期限切れは保存した状態ではなく判定なので、保存先が Granted のまま返す期限切れの同意を、
// 撤回の時刻で Granted ではないと判定する。
func seededConsentQuota(t *testing.T) (ConsentDeps, *oauth2memory.ConsentRepository, tenantports.QuotaRepository) {
	t.Helper()
	ctx := tenantContext(quotaTenantID)
	quotaRepo := tenancy.NewMemoryModule().QuotaRepo
	if err := quotaRepo.CheckAndIncrement(ctx, quotaTenantID, tenancydomain.ResourceConsents, 1); err != nil {
		t.Fatalf("seed usage: %v", err)
	}
	consentRepo := oauth2memory.NewConsentRepository()
	return ConsentDeps{ConsentRepo: consentRepo, QuotaRepo: quotaRepo}, consentRepo, quotaRepo
}

//spec:covers REQ-OAUTH2-032: 期限を過ぎた同意は Granted ではないので、撤回しても consents の使用量を変えない
func TestRevokeExpiredConsentKeepsConsentQuota(t *testing.T) {
	ctx := tenantContext(quotaTenantID)
	deps, consentRepo, quotaRepo := seededConsentQuota(t)
	grantedAt := time.Date(2025, 7, 5, 12, 0, 0, 0, time.UTC)
	expiresAt := grantedAt.Add(365 * 24 * time.Hour)
	if err := consentRepo.Save(ctx, quotaTenantID, &oauthdomain.Consent{
		UserID: "user-1", ClientID: "client-1", Scopes: []string{"openid"},
		State: oauthdomain.ConsentGranted, GrantedAt: grantedAt, ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("seed consent: %v", err)
	}

	if err := RevokeConsent(ctx, deps, "user-1", "user-1", "client-1", expiresAt); err != nil {
		t.Fatal(err)
	}
	if got := consentsUsage(t, quotaRepo); got != 1 {
		t.Fatalf("consents usage after revoking an expired consent = %d, want 1", got)
	}
}

//spec:covers REQ-OAUTH2-058: 期限を過ぎた付与は Granted ではないので、撤回しても consents の使用量を変えない
func TestRevokeExpiredAgentDelegationKeepsConsentQuota(t *testing.T) {
	ctx := tenantContext(quotaTenantID)
	deps, consentRepo, quotaRepo := seededConsentQuota(t)
	grantedAt := time.Date(2025, 7, 5, 12, 0, 0, 0, time.UTC)
	expiresAt := grantedAt.Add(365 * 24 * time.Hour)
	if err := consentRepo.SaveAgentDelegation(ctx, quotaTenantID, &oauthdomain.Consent{
		UserID: "user-1", AgentID: "agent-1", Scopes: []string{"documents:read"},
		State: oauthdomain.ConsentGranted, GrantedAt: grantedAt, ExpiresAt: expiresAt,
	}); err != nil {
		t.Fatalf("seed delegation: %v", err)
	}

	if err := RevokeAgentDelegation(ctx, deps, "user-1", "agent-1", expiresAt); err != nil {
		t.Fatal(err)
	}
	if got := consentsUsage(t, quotaRepo); got != 1 {
		t.Fatalf("consents usage after revoking an expired delegation = %d, want 1", got)
	}
}
