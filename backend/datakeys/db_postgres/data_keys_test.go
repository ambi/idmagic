package db_postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestImplementsDataKeyRepositoryPort(t *testing.T) {
	var _ ports.DataKeyRepository = (*DataKeyRepository)(nil)
}

func seedTenant(t *testing.T, db *DataKeyRepository) *tenancydomain.Tenant {
	t.Helper()
	now := pgtest.Now()
	tenant := &tenancydomain.Tenant{
		ID: mustUUID(t), Realm: "datakeys-" + mustUUID(t)[:8], DisplayName: "DataKeys Test",
		Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&tenancypostgres.TenantRepository{Pool: db.Pool}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

func mustUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("generate uuid: %v", err)
	}
	return id
}

// TestBootstrapRotateDisableDestroyLifecycle covers the full
// DataEncryptionKeyLifecycle against a real PostgreSQL database (scenarios in
// spec/contexts/data-keys.yaml): bootstrap creates active v1, rotate demotes
// it to retiring and activates v2, disable locks out the retiring version,
// destroy erases wrapped_dek.
func TestBootstrapRotateDisableDestroyLifecycle(t *testing.T) {
	pool := pgtest.Require(t)
	repo, err := NewDataKeyRepository(context.Background(), pool)
	if err != nil {
		t.Fatalf("NewDataKeyRepository failed: %v", err)
	}
	tenant := seedTenant(t, repo)
	ctx := context.Background()
	now := pgtest.Now()

	v1, err := repo.Bootstrap(ctx, tenant.ID, []byte("wrapped-1"), "master-1", now)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	if v1.Version != 1 || v1.Status != domain.DataKeyStatusActive {
		t.Fatalf("expected active v1, got version=%d status=%s", v1.Version, v1.Status)
	}

	if _, err := repo.Bootstrap(ctx, tenant.ID, []byte("wrapped-again"), "master-1", now); !errors.Is(err, domain.ErrDataKeyAlreadyBootstrapped) {
		t.Fatalf("expected ErrDataKeyAlreadyBootstrapped, got %v", err)
	}

	next, previous, err := repo.Rotate(ctx, tenant.ID, []byte("wrapped-2"), "master-1", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Rotate failed: %v", err)
	}
	if next.Version != 2 || next.Status != domain.DataKeyStatusActive {
		t.Fatalf("expected active v2, got version=%d status=%s", next.Version, next.Status)
	}
	if previous.Version != 1 || previous.Status != domain.DataKeyStatusRetiring {
		t.Fatalf("expected retiring v1, got version=%d status=%s", previous.Version, previous.Status)
	}

	active, err := repo.FindActive(ctx, tenant.ID)
	if err != nil || active.Version != 2 {
		t.Fatalf("FindActive failed: %v %+v", err, active)
	}

	if _, err := repo.Disable(ctx, tenant.ID, 2, now.Add(2*time.Hour)); !errors.Is(err, domain.ErrDataKeyIsActive) {
		t.Fatalf("expected ErrDataKeyIsActive disabling active version, got %v", err)
	}

	disabled, err := repo.Disable(ctx, tenant.ID, 1, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("Disable v1 failed: %v", err)
	}
	if disabled.Status != domain.DataKeyStatusDisabled {
		t.Fatalf("expected disabled status, got %s", disabled.Status)
	}

	destroyed, err := repo.Destroy(ctx, tenant.ID, 1, now.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("Destroy v1 failed: %v", err)
	}
	if destroyed.Status != domain.DataKeyStatusDestroyed || destroyed.WrappedDEK != nil {
		t.Fatalf("expected destroyed status with erased wrapped_dek, got status=%s wrapped_dek=%v", destroyed.Status, destroyed.WrappedDEK)
	}

	all, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(all) != 2 || all[0].Version != 2 || all[1].Version != 1 {
		t.Fatalf("ListByTenant unexpected result: %v %+v", err, all)
	}
}

func TestFindActiveReturnsErrNoActiveDataKeyForUnknownTenant(t *testing.T) {
	pool := pgtest.Require(t)
	repo, err := NewDataKeyRepository(context.Background(), pool)
	if err != nil {
		t.Fatalf("NewDataKeyRepository failed: %v", err)
	}
	tenant := seedTenant(t, repo)

	if _, err := repo.FindActive(context.Background(), tenant.ID); !errors.Is(err, domain.ErrNoActiveDataKey) {
		t.Fatalf("expected ErrNoActiveDataKey, got %v", err)
	}
}
