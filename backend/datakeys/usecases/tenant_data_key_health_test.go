package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/db_memory"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type fakeTenantRepo struct {
	tenants []*tenancydomain.Tenant
}

func (f *fakeTenantRepo) FindByID(ctx context.Context, id string) (*tenancydomain.Tenant, error) {
	for _, t := range f.tenants {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, nil //nolint:nilnil // test fake, not exercised by these tests
}

func (f *fakeTenantRepo) FindByRealm(ctx context.Context, realm string) (*tenancydomain.Tenant, error) {
	return nil, nil //nolint:nilnil // test fake, not exercised by these tests
}

func (f *fakeTenantRepo) FindAll(ctx context.Context) ([]*tenancydomain.Tenant, error) {
	return f.tenants, nil
}

func (f *fakeTenantRepo) Save(ctx context.Context, tenant *tenancydomain.Tenant) error {
	return nil
}

// TestListTenantDataKeyHealth_ReportsBootstrappedTenantsAndOmitsUnbootstrapped
// covers scenario "systemAdminがテナント横断でDEK健全性を一覧する"
// (spec/contexts/data-keys.yaml): each tenant's active_version/status/
// provider/provider_reachable is returned without key material, and a
// tenant with no DataEncryptionKey yet is simply absent rather than erroring.
func TestListTenantDataKeyHealth_ReportsBootstrappedTenantsAndOmitsUnbootstrapped(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	now := time.Now().UTC()
	if _, err := BootstrapTenantDataKey(context.Background(), Deps{Repository: repo, Crypto: crypto}, "tenant-a", now); err != nil {
		t.Fatalf("bootstrap tenant-a: %v", err)
	}

	tenantRepo := &fakeTenantRepo{tenants: []*tenancydomain.Tenant{
		{ID: "tenant-a"}, {ID: "tenant-b"},
	}}

	health, err := ListTenantDataKeyHealth(context.Background(), ListTenantDataKeyHealthDeps{
		TenantRepo: tenantRepo, Repository: repo, Crypto: crypto,
	})
	if err != nil {
		t.Fatalf("ListTenantDataKeyHealth failed: %v", err)
	}
	if len(health) != 1 {
		t.Fatalf("expected 1 tenant reported (tenant-b never bootstrapped), got %d: %+v", len(health), health)
	}
	if health[0].TenantID != "tenant-a" || health[0].ActiveVersion != 1 || health[0].Provider != "tink_cleartext" || !health[0].ProviderReachable {
		t.Fatalf("unexpected health entry: %+v", health[0])
	}
}

func TestListTenantDataKeyHealth_ReflectsUnreachableProvider(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	if _, err := BootstrapTenantDataKey(context.Background(), Deps{Repository: repo, Crypto: crypto}, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("bootstrap tenant-a: %v", err)
	}
	tenantRepo := &fakeTenantRepo{tenants: []*tenancydomain.Tenant{{ID: "tenant-a"}}}

	health, err := ListTenantDataKeyHealth(context.Background(), ListTenantDataKeyHealthDeps{
		TenantRepo: tenantRepo, Repository: repo, Crypto: &unhealthyCrypto{EnvelopeCrypto: crypto},
	})
	if err != nil {
		t.Fatalf("ListTenantDataKeyHealth failed: %v", err)
	}
	if len(health) != 1 || health[0].ProviderReachable {
		t.Fatalf("expected provider_reachable=false, got %+v", health)
	}
}

type unhealthyCrypto struct {
	envelope_crypto.EnvelopeCrypto
}

func (unhealthyCrypto) Healthy(ctx context.Context) bool { return false }
