package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/internal/oauth2/ports"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
)

type fakeKeyStore struct {
	keys       []*ports.SigningKey
	getKeysErr error
}

func (f *fakeKeyStore) GetActiveKey(ctx context.Context) (*ports.SigningKey, error) {
	return nil, errors.New("unimplemented")
}

func (f *fakeKeyStore) GetAllKeys(ctx context.Context) ([]*ports.SigningKey, error) {
	return f.keys, f.getKeysErr
}

func (f *fakeKeyStore) FindByKID(ctx context.Context, kid string) (*ports.SigningKey, error) {
	return nil, errors.New("unimplemented")
}

func (f *fakeKeyStore) Rotate(ctx context.Context) (*ports.SigningKey, error) {
	return nil, errors.New("unimplemented")
}

func (f *fakeKeyStore) Disable(ctx context.Context, kid string) (*ports.SigningKey, error) {
	return nil, errors.New("unimplemented")
}
func (f *fakeKeyStore) Provider() spec.KeyProvider       { return spec.KeyProviderPostgres }
func (f *fakeKeyStore) Healthy(ctx context.Context) bool { return true }

func TestListTenantKeyHealth(t *testing.T) {
	ctx := context.Background()
	tenantRepo := memory.NewTenantRepository()
	keyStore := &fakeKeyStore{}

	deps := TenantKeyHealthDeps{
		TenantRepo: tenantRepo,
		KeyStore:   keyStore,
	}

	// テナントを 2 つ作成
	_ = tenantRepo.Save(ctx, &spec.Tenant{ID: "tenant-a"})
	_ = tenantRepo.Save(ctx, &spec.Tenant{ID: "tenant-b"})

	t.Run("Succeeds", func(t *testing.T) {
		keyStore.keys = []*ports.SigningKey{
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
