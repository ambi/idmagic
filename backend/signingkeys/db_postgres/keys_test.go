package db_postgres

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestKeyStoreRotateAndLookup(t *testing.T) {
	db := pgtest.Require(t)
	// signing_keys.tenant_id は tenants(id) を参照する。NewKeyStore は default テナントの
	// active 鍵を bootstrap するため、default テナント行を用意しておく。
	now := time.Now().UTC()
	defaultTenant := &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := (&tenancypostgres.TenantRepository{Pool: db}).Save(context.Background(), defaultTenant); err != nil {
		t.Fatalf("seed default tenant: %v", err)
	}
	ctx := tenancy.WithTenant(context.Background(), defaultTenant, "", "")

	store, err := NewKeyStore(ctx, db)
	if err != nil {
		t.Fatalf("new key store: %v", err)
	}

	active, err := store.GetActiveKey(ctx)
	if err != nil || active == nil || !active.Active {
		t.Fatalf("get active key: %v %+v", err, active)
	}

	found, err := store.FindByKID(ctx, active.Kid)
	if err != nil || found == nil || found.Kid != active.Kid {
		t.Fatalf("find by kid: %v %+v", err, found)
	}

	rotated, err := store.Rotate(ctx, time.Now().UTC(), 7*24*time.Hour)
	if err != nil || rotated == nil || rotated.Kid == active.Kid {
		t.Fatalf("rotate: %v %+v (prev %s)", err, rotated, active.Kid)
	}

	newActive, err := store.GetActiveKey(ctx)
	if err != nil || newActive == nil || newActive.Kid != rotated.Kid {
		t.Fatalf("active after rotate: %v %+v", err, newActive)
	}

	all, err := store.GetAllKeys(ctx)
	if err != nil || len(all) < 2 {
		t.Fatalf("get all keys: %v len=%d", err, len(all))
	}

	// PostgreSQL advisory lock 内の cadence 再判定により、同時 batch でも
	// due rotation は tenant ごとに一度だけ実行される。
	old := time.Now().UTC().Add(-100 * 24 * time.Hour)
	if _, err := db.Exec(ctx, "UPDATE signing_keys SET created_at=$2 WHERE tenant_id=$1 AND active", defaultTenant.ID, old); err != nil {
		t.Fatalf("age active key: %v", err)
	}
	now = time.Now().UTC()
	results := make(chan *signingdomain.SigningKey, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			key, err := store.RotateIfDue(ctx, now, 90*24*time.Hour, 7*24*time.Hour)
			results <- key
			errs <- err
		})
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent RotateIfDue: %v", err)
		}
	}
	rotations := 0
	for key := range results {
		if key != nil {
			rotations++
		}
	}
	if rotations != 1 {
		t.Fatalf("concurrent due rotations = %d, want 1", rotations)
	}
}
