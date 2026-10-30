package usecases

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/internal/db_memory"
	"github.com/ambi/idmagic/backend/datakeys/internal/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

func TestImplementsCacheInvalidatorPort(t *testing.T) {
	var _ ports.CacheInvalidator = (*DataKeyCache)(nil)
}

// TestGetActiveReturnsUnwrappedDEK verifies the cache resolves the tenant's
// active DEK on first call by unwrapping through the repository + crypto
// port, used by record-owning repositories (T005) to encrypt new ciphertext.
func TestGetActiveReturnsUnwrappedDEK(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	deps := Deps{Repository: repo, Crypto: crypto}

	key, err := BootstrapTenantDataKey(context.Background(), deps, "tenant-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}

	cache := NewDataKeyCache(repo, crypto)
	version, plaintextDEK, err := cache.GetActive(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}
	if version != key.Version {
		t.Fatalf("expected version %d, got %d", key.Version, version)
	}
	if len(plaintextDEK) == 0 {
		t.Fatal("expected non-empty plaintext DEK")
	}
}

func TestGetActiveFailsClosedWhenNoActiveKey(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	cache := NewDataKeyCache(repo, envelope_crypto.NewTinkEnvelopeCrypto(master))

	if _, _, err := cache.GetActive(context.Background(), "tenant-a"); err == nil {
		t.Fatal("expected GetActive to fail when tenant has no active key")
	}
}

// TestGetActiveResolvesRotatedVersion ensures GetActive re-resolves the new
// active version after a rotation rather than serving the pre-rotation DEK.
func TestGetActiveResolvesRotatedVersion(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	deps := Deps{Repository: repo, Crypto: crypto}
	ctx := context.Background()
	now := time.Now().UTC()

	first, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now)
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}

	cache := NewDataKeyCache(repo, crypto)
	firstVersion, _, err := cache.GetActive(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("GetActive (before rotate) failed: %v", err)
	}
	if firstVersion != first.Version {
		t.Fatalf("expected cached version %d, got %d", first.Version, firstVersion)
	}

	deps.Cache = cache
	next, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	secondVersion, _, err := cache.GetActive(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("GetActive (after rotate) failed: %v", err)
	}
	if secondVersion != next.Version {
		t.Fatalf("expected cache to resolve the rotated version %d, got %d (stale cache not invalidated)", next.Version, secondVersion)
	}
}

// countingMasterKeyProvider はアンラップの回数を数える。
type countingMasterKeyProvider struct {
	envelope_crypto.MasterKeyProvider
	unwraps atomic.Int32
}

func (p *countingMasterKeyProvider) UnwrapDataKey(ctx context.Context, tenantID string, wrapped []byte, masterKeyID string) ([]byte, error) {
	p.unwraps.Add(1)
	return p.MasterKeyProvider.UnwrapDataKey(ctx, tenantID, wrapped, masterKeyID)
}

// 状態は毎回読むが、バージョンごとのアンラップは 1 回だけである。
func TestRepeatedLookupsUnwrapOncePerVersion(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	provider := &countingMasterKeyProvider{MasterKeyProvider: master}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(provider)
	ctx := context.Background()
	key, err := BootstrapTenantDataKey(ctx, Deps{Repository: repo, Crypto: crypto}, "tenant-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}

	cache := NewDataKeyCache(repo, crypto)
	for range 3 {
		if _, _, err := cache.GetActive(ctx, "tenant-a"); err != nil {
			t.Fatalf("GetActive failed: %v", err)
		}
		if _, err := cache.GetByVersion(ctx, "tenant-a", key.Version); err != nil {
			t.Fatalf("GetByVersion failed: %v", err)
		}
	}
	if got := provider.unwraps.Load(); got != 1 {
		t.Fatalf("unwraps = %d, want 1", got)
	}

	cache.Invalidate("tenant-a")
	if _, _, err := cache.GetActive(ctx, "tenant-a"); err != nil {
		t.Fatalf("GetActive after Invalidate failed: %v", err)
	}
	if got := provider.unwraps.Load(); got != 2 {
		t.Fatalf("unwraps after Invalidate = %d, want 2", got)
	}
}

// TestGetByVersionDecryptsRetiringVersion covers decrypting a secret that
// was encrypted under a version which has since rotated out to retiring
// (scenario "DEKをrotationしても既存暗号文が復号できる", docs/modules/data-keys/).
func TestGetByVersionDecryptsRetiringVersion(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	deps := Deps{Repository: repo, Crypto: crypto}
	ctx := context.Background()
	now := time.Now().UTC()

	first, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now)
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if _, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	cache := NewDataKeyCache(repo, crypto)
	plaintextDEK, err := cache.GetByVersion(ctx, "tenant-a", first.Version)
	if err != nil {
		t.Fatalf("GetByVersion(retiring) failed: %v", err)
	}
	if len(plaintextDEK) == 0 {
		t.Fatal("expected non-empty plaintext DEK for the retiring version")
	}
}

// TestGetByVersionFailsClosedForDestroyedVersion ensures a destroyed
// version's ciphertext can never be decrypted, even if the cache still held
// a stale entry (fail-closed / crypto-shredding).
func TestGetByVersionFailsClosedForDestroyedVersion(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	deps := Deps{Repository: repo, Crypto: crypto}
	ctx := context.Background()
	now := time.Now().UTC()

	first, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now)
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if _, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}
	if err := DestroyTenantDataKey(ctx, deps, "tenant-a", first.Version, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("DestroyTenantDataKey failed: %v", err)
	}

	cache := NewDataKeyCache(repo, crypto)
	if _, err := cache.GetByVersion(ctx, "tenant-a", first.Version); err == nil {
		t.Fatal("expected GetByVersion to fail-closed for a destroyed version")
	} else if !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("expected ErrDataKeyUnavailable, got %v", err)
	}
}
