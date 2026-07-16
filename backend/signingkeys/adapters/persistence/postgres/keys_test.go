package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres"
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

	rotated, err := store.Rotate(ctx)
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
}
