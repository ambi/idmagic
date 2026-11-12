package db_postgres

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/accesscontrol/internal/domain"
	"github.com/ambi/idmagic/backend/accesscontrol/internal/testing_contract"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
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
	if err := tenancy.NewPostgresModule(db).TenantRepo.Save(context.Background(), tenant); err != nil {
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

// 比較と採番の原子性はトランザクションと一意制約に依存するので、PostgreSQL で契約を実行する。
//
//spec:covers REQ-AUTHORIZATION-018: 最新でないバージョンを基にした登録を拒否し、同じバージョンを基にした同時の登録は一つだけを受理する。
func TestAuthorizationModelRepositoryContract(t *testing.T) {
	testing_contract.RunAuthorizationModelRepositoryContract(t, newFixture)
}

// 同じバージョンを基にした登録が同時に比較を通った場合、後から挿入する側は一意制約の違反で失敗する。
// 契約の並行テストでは挿入が重ならないことが多く、この経路を確実には通らないので、先の挿入を
// コミットせずに保ち、後の Publish がその行のロックを待つのを確かめてからコミットする。
//
//spec:covers REQ-AUTHORIZATION-018: 同時の登録に先を越された登録を競合として拒否し、バージョンを作らない。
func TestPublishThatLosesTheInsertRaceIsAVersionConflict(t *testing.T) {
	f := newFixture(t, "tenant-a")
	tenantA := f.Tenant("tenant-a")
	pool := pgtest.Require(t)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	firstID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(pool).WithTx(tx).InsertAuthorizationModel(ctx, InsertAuthorizationModelParams{
		ID: firstID, TenantID: tenantA, ExpectedVersion: 0, Definition: []byte(`[]`), CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("hold the first insert: %v", err)
	}

	secondID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, _, err := f.Models.Publish(ctx, &domain.AuthorizationModel{
			ID: secondID, TenantID: tenantA, ResourceTypes: []domain.ResourceTypeDefinition{{Name: "user"}},
		}, 0)
		result <- err
	}()

	// 後の挿入が一意索引の行ロックを待つまで待つ。待つ前にコミットすると、比較で行を返さない経路を通る。
	deadline := time.Now().Add(10 * time.Second)
	for {
		var waiting int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE wait_event_type = 'Lock' AND query LIKE '%INSERT INTO authorization_models%'`).Scan(&waiting); err != nil {
			t.Fatalf("observe lock waits: %v", err)
		}
		if waiting > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the second publish never waited for the first insert")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit the first insert: %v", err)
	}

	if err := <-result; !errors.Is(err, domain.ErrModelVersionConflict) {
		t.Fatalf("Publish that lost the race = %v, want ErrModelVersionConflict", err)
	}
	latest, err := f.Models.Latest(ctx, tenantA)
	if err != nil || latest == nil || latest.ID != firstID || latest.Version != 1 {
		t.Fatalf("Latest = (%+v, %v), want the first insert as version 1", latest, err)
	}
}
