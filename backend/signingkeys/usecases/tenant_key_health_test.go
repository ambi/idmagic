package usecases

import (
	"context"
	"errors"
	"testing"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

type fakeKeyStore struct {
	keys       []*signingdomain.SigningKey
	getKeysErr error
}

func (f *fakeKeyStore) GetActiveKey(ctx context.Context) (*signingdomain.SigningKey, error) {
	return nil, errors.New("unimplemented")
}

func (f *fakeKeyStore) GetAllKeys(ctx context.Context) ([]*signingdomain.SigningKey, error) {
	return f.keys, f.getKeysErr
}

func (f *fakeKeyStore) FindByKID(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	return nil, errors.New("unimplemented")
}

func (f *fakeKeyStore) Rotate(ctx context.Context) (*signingdomain.SigningKey, error) {
	return nil, errors.New("unimplemented")
}

func (f *fakeKeyStore) Disable(ctx context.Context, kid string) (*signingdomain.SigningKey, error) {
	return nil, errors.New("unimplemented")
}
func (f *fakeKeyStore) Provider() signingdomain.KeyProvider { return signingdomain.KeyProviderPostgres }
func (f *fakeKeyStore) Healthy(ctx context.Context) bool    { return true }

func TestListTenantKeyHealth(t *testing.T) {
	ctx := context.Background()
	tenantRepo := tenancymemory.NewTenantRepository()
	keyStore := &fakeKeyStore{}

	deps := TenantKeyHealthDeps{
		TenantRepo: tenantRepo,
		KeyStore:   keyStore,
	}

	// テナントを 2 つ作成
	_ = tenantRepo.Save(ctx, &tenancydomain.Tenant{ID: "tenant-a"})
	_ = tenantRepo.Save(ctx, &tenancydomain.Tenant{ID: "tenant-b"})

	t.Run("Succeeds", func(t *testing.T) {
		keyStore.keys = []*signingdomain.SigningKey{
			{Kid: "key-1", Active: false},
			{Kid: "key-2", Active: true},
		}
		keyStore.getKeysErr = nil

		results, err := ListTenantKeyHealth(ctx, deps)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 health records, got %d", len(results))
		}

		// tenant-a と tenant-b それぞれについてアクティブキーIDが "key-2" であること
		for _, r := range results {
			if r.ActiveKid != "key-2" {
				t.Errorf("expected ActiveKid to be 'key-2', got %q", r.ActiveKid)
			}
			if r.JWKSKeyCount != 2 {
				t.Errorf("expected key count 2, got %d", r.JWKSKeyCount)
			}
			if !r.Healthy {
				t.Error("expected healthy status to be true")
			}
		}
	})

	t.Run("KeyStoreError", func(t *testing.T) {
		keyStore.getKeysErr = errors.New("key store error")
		_, err := ListTenantKeyHealth(ctx, deps)
		if err == nil {
			t.Error("expected error from GetAllKeys, got nil")
		}
	})
}
