package db_postgres

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authorization/domain"
	"github.com/ambi/idmagic/backend/authorization/testing_contract"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var realmSeq atomic.Uint64

func seedTenant(t *testing.T, db sharedpg.DB) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tenant := &tenancydomain.Tenant{
		ID: id, Realm: fmt.Sprintf("authz-tenant-%d", realmSeq.Add(1)), DisplayName: "Authorization Test Tenant",
		Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return id
}

// newFixture は契約が使う記号名ごとに実在するテナントを 1 つ用意する。
// PostgreSQL 側は外部キーを持つので、記号名をそのままテナント id には使えない。
func newFixture(t *testing.T, tenantNames ...string) testing_contract.Fixture {
	t.Helper()
	pool := pgtest.Require(t)
	tenants := make(map[string]string, len(tenantNames))
	for _, name := range tenantNames {
		tenants[name] = seedTenant(t, pool)
	}
	return testing_contract.Fixture{
		Tuples:  &RelationTupleRepository{Pool: pool},
		Models:  &AuthorizationModelRepository{Pool: pool},
		Tenants: tenants,
	}
}

func TestRelationTupleRepositoryContract(t *testing.T) {
	testing_contract.RunRelationTupleRepositoryContract(t, newFixture)
}

func TestAuthorizationModelRepositoryContract(t *testing.T) {
	testing_contract.RunAuthorizationModelRepositoryContract(t, newFixture)
}

// REQ-AUTHORIZATION-001: 版の採番は INSERT の中で行うので、同時に published しても
// 同じ版を 2 つ作らない。採番を読み書き 2 回に分ければここで衝突する。
func TestConcurrentPublishNeverReusesAVersion(t *testing.T) {
	f := newFixture(t, "tenant-a")
	tenantA := f.Tenant("tenant-a")
	minimal := []domain.ResourceTypeDefinition{
		{Name: "user"},
		{Name: "document", Relations: []domain.RelationDefinition{
			{Name: "viewer", Rewrites: []domain.RelationRewrite{
				{Kind: domain.RewriteDirect, DirectSubjectTypes: []string{"user"}},
			}},
		}},
	}

	const concurrency = 8
	var wg sync.WaitGroup
	versions := make([]int, concurrency)
	errs := make([]error, concurrency)
	for i := range concurrency {
		wg.Go(func() {
			id, err := spec.NewUUIDv4()
			if err != nil {
				errs[i] = err
				return
			}
			stored, _, err := f.Models.Publish(context.Background(), &domain.AuthorizationModel{
				ID: id, TenantID: tenantA, ResourceTypes: minimal,
			})
			if err != nil {
				errs[i] = err
				return
			}
			versions[i] = stored.Version
		})
	}
	wg.Wait()

	seen := map[int]bool{}
	published := 0
	for i, err := range errs {
		if err != nil {
			// 一意制約による衝突は「版を採り損ねた」だけで、重複した版は残らない。
			continue
		}
		if seen[versions[i]] {
			t.Fatalf("version %d was assigned twice", versions[i])
		}
		seen[versions[i]] = true
		published++
	}
	if published == 0 {
		t.Fatal("no concurrent publish succeeded")
	}
	latest, err := f.Models.Latest(context.Background(), tenantA)
	if err != nil || latest == nil {
		t.Fatalf("Latest = (%v, %v)", latest, err)
	}
	if latest.Version != published {
		t.Fatalf("Latest version = %d, want %d (one version per successful publish)", latest.Version, published)
	}
}
