package postgres

import (
	"context"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"
)

func TestKeyStoreRotateAndLookup(t *testing.T) {
	db := requireDB(t)
	// signing_keys.tenant_id は tenants(id) を参照する。NewKeyStore は default テナントの
	// active 鍵を bootstrap するため、default テナント行を用意しておく。
	now := testClock()
	defaultTenant := &tenancydomain.Tenant{
		ID:          tenancydomain.DefaultTenantID,
		Realm:       tenancydomain.DefaultRealm,
		DisplayName: "Default",
		Status:      tenancydomain.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	// TenantRepository は tenancy/adapters/persistence/postgres へ移設済み (wi-179) で、本
	// パッケージの内部テストから import すると import cycle になるため、seedTenant 同様
	// 生 SQL で直接 INSERT する。
	_, err := db.Exec(context.Background(), `
INSERT INTO tenants (id,realm,display_name,status,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6)`,
		defaultTenant.ID, defaultTenant.Realm, defaultTenant.DisplayName, string(defaultTenant.Status),
		defaultTenant.CreatedAt, defaultTenant.UpdatedAt)
	if err != nil {
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
