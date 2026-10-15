package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// quotaResources は CheckAndIncrement と Decrement が受け付ける資源の全体である。
var quotaResources = []string{
	domain.ResourceUsers, domain.ResourceGroups, domain.ResourceAgents, domain.ResourceApplications,
	domain.ResourceOAuth2Clients, domain.ResourceActiveSessions, domain.ResourceConsents, domain.ResourceActiveJobs,
	domain.ResourceSsfStreams, "audit_events_retained", "export_artifacts_bytes",
}

// seedQuotaTenant は tenant_usages の外部キーが指すテナントを作る。fixtures_postgres は
// このパッケージを import するので、ここでは使えない。
func seedQuotaTenant(ctx context.Context, t *testing.T, db sharedpg.DB) *domain.Tenant {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tenant := &domain.Tenant{ID: id, Realm: "quota-" + id, DisplayName: "Quota", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := (&TenantRepository{Pool: db}).Save(ctx, tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

// usageByResource は利用量を JSON のタグ（資源名と同じ文字列）で引ける形にする。
// テスト側に資源と列の対応表を持たないので、実装の対応表の誤りを写し取らない。
func usageByResource(ctx context.Context, t *testing.T, repo *QuotaRepository, tenantID string) map[string]int {
	t.Helper()
	usage, err := repo.GetUsage(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	encoded, err := json.Marshal(usage)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	if err := json.Unmarshal(encoded, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func quotaLimiting(t *testing.T, resource string, limit int) *domain.TenantQuota {
	t.Helper()
	encoded, err := json.Marshal(map[string]int{resource: limit})
	if err != nil {
		t.Fatal(err)
	}
	var quota domain.TenantQuota
	if err := json.Unmarshal(encoded, &quota); err != nil {
		t.Fatal(err)
	}
	return &quota
}

func TestQuotaRepositoryCountsOnlyTheNamedResourceUpToItsLimit(t *testing.T) {
	db := pgtest.Require(t)
	repo := NewQuotaRepository(db)
	ctx := context.Background()
	for _, resource := range quotaResources {
		t.Run(resource, func(t *testing.T) {
			tenant := seedQuotaTenant(ctx, t, db)
			if err := repo.SetQuota(ctx, tenant.ID, quotaLimiting(t, resource, 2)); err != nil {
				t.Fatalf("SetQuota: %v", err)
			}
			for range 2 {
				if err := repo.CheckAndIncrement(ctx, tenant.ID, resource, 1); err != nil {
					t.Fatalf("CheckAndIncrement within the limit: %v", err)
				}
			}
			var exceeded *domain.QuotaExceededError
			if err := repo.CheckAndIncrement(ctx, tenant.ID, resource, 1); !errors.As(err, &exceeded) || exceeded.Resource != resource {
				t.Fatalf("CheckAndIncrement past the limit err=%v, want QuotaExceededError for %s", err, resource)
			}
			usage := usageByResource(ctx, t, repo, tenant.ID)
			for name, value := range usage {
				want := 0
				if name == resource {
					want = 2
				}
				if value != want {
					t.Fatalf("usage[%s]=%d, want %d (all usage: %v)", name, value, want, usage)
				}
			}
			if err := repo.Decrement(ctx, tenant.ID, resource, 5); err != nil {
				t.Fatalf("Decrement: %v", err)
			}
			if got := usageByResource(ctx, t, repo, tenant.ID)[resource]; got != 0 {
				t.Fatalf("usage after decrementing past zero = %d, want 0", got)
			}
		})
	}
}

func TestQuotaRepositoryAppliesTheDefaultLimitWithoutAnOverride(t *testing.T) {
	db := pgtest.Require(t)
	repo := NewQuotaRepository(db)
	ctx := context.Background()
	tenant := seedQuotaTenant(ctx, t, db)
	limit := domain.DefaultTenantQuota[domain.ResourceActiveJobs]
	if err := repo.CheckAndIncrement(ctx, tenant.ID, domain.ResourceActiveJobs, limit); err != nil {
		t.Fatalf("CheckAndIncrement up to the default limit: %v", err)
	}
	var exceeded *domain.QuotaExceededError
	if err := repo.CheckAndIncrement(ctx, tenant.ID, domain.ResourceActiveJobs, 1); !errors.As(err, &exceeded) {
		t.Fatalf("CheckAndIncrement past the default limit err=%v, want QuotaExceededError", err)
	}
}

func TestQuotaRepositoryRejectsAnUnknownResource(t *testing.T) {
	db := pgtest.Require(t)
	repo := NewQuotaRepository(db)
	ctx := context.Background()
	tenant := seedQuotaTenant(ctx, t, db)
	if err := repo.CheckAndIncrement(ctx, tenant.ID, "widgets", 1); err == nil {
		t.Fatal("CheckAndIncrement accepted an unknown resource")
	}
	if err := repo.Decrement(ctx, tenant.ID, "widgets", 1); err == nil {
		t.Fatal("Decrement accepted an unknown resource")
	}
}
