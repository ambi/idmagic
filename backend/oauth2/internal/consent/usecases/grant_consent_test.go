package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// failConsentSave は同意の保存だけを失敗させる。
type failConsentSave struct {
	consentports.ConsentRepository
}

var errConsentSave = errors.New("consent store unavailable")

func (failConsentSave) Save(context.Context, string, *domain.Consent) error { return errConsentSave }

// quotaTenantID は使用量を観測するテストのテナントである。
const quotaTenantID = "acme"

func consentsUsage(t *testing.T, repo tenantports.QuotaRepository) int {
	t.Helper()
	usage, err := repo.GetUsage(context.Background(), quotaTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.Consents
}

//spec:covers REQ-TENANCY-050: 保存に失敗した同意の付与は consents の使用量を残さない。
func TestGrantConsentReleasesQuotaWhenSaveFails(t *testing.T) {
	const tenantID = quotaTenantID
	ctx := tenantContext(tenantID)
	quotaRepo := tenancy.NewMemoryModule().QuotaRepo
	deps := ConsentDeps{
		ConsentRepo: failConsentSave{ConsentRepository: oauth2memory.NewConsentRepository()},
		QuotaRepo:   quotaRepo,
	}
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	if _, err := GrantConsent(ctx, deps, tenantID, ConsentGrant{
		UserID: "alice", ClientID: "app", Scopes: []string{"openid"},
	}, now); !errors.Is(err, errConsentSave) {
		t.Fatalf("GrantConsent with a failing save = %v, want %v", err, errConsentSave)
	}
	if got := consentsUsage(t, quotaRepo); got != 0 {
		t.Fatalf("consents usage after a failed save = %d, want 0", got)
	}
}

// 保存は (テナント, 利用者, クライアント) の upsert なので、付与済みの同意の再付与は使用量を増やさず、
// 撤回済みの同意の再付与は一つ使う。
func TestGrantConsentConsumesQuotaOnlyForNewOrRevokedConsents(t *testing.T) {
	const tenantID = quotaTenantID
	ctx := tenantContext(tenantID)
	quotaRepo := tenancy.NewMemoryModule().QuotaRepo
	deps := ConsentDeps{ConsentRepo: oauth2memory.NewConsentRepository(), QuotaRepo: quotaRepo}
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	grant := func(scopes ...string) {
		t.Helper()
		if _, err := GrantConsent(ctx, deps, tenantID, ConsentGrant{UserID: "alice", ClientID: "app", Scopes: scopes}, now); err != nil {
			t.Fatalf("GrantConsent: %v", err)
		}
	}

	grant("openid")
	grant("openid", "profile")
	if got := consentsUsage(t, quotaRepo); got != 1 {
		t.Fatalf("consents usage after re-consenting a granted consent = %d, want 1", got)
	}
	if err := RevokeConsent(ctx, deps, "operator", "alice", "app", now); err != nil {
		t.Fatalf("RevokeConsent: %v", err)
	}
	grant("openid")
	if got := consentsUsage(t, quotaRepo); got != 1 {
		t.Fatalf("consents usage after re-consenting a revoked consent = %d, want 1", got)
	}
}
