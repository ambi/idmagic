package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/db_memory"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

func TestImplementsCacheInvalidatorPort(t *testing.T) {
	var _ ports.CacheInvalidator = (*DataKeyCache)(nil)
}

// TestGetActiveReturnsUnwrappedDEK verifies the cache resolves the tenant's
// active DEK on first call by unwrapping through the repository + crypto
// port, used by record-owning repositories (T005) to encrypt/decrypt fields.
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
	keyID, plaintextDEK, err := cache.GetActive(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("GetActive failed: %v", err)
	}
	if keyID != key.ID {
		t.Fatalf("expected keyID %s, got %s", key.ID, keyID)
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

// TestInvalidateForcesReUnwrapAfterRotate ensures a cached DEK is dropped on
// Invalidate, so a subsequent GetActive re-resolves the new active version
// rather than serving the pre-rotation DEK from a stale cache entry
// (ADR-148: no worker replica keeps encrypting with a superseded DEK).
func TestInvalidateForcesReUnwrapAfterRotate(t *testing.T) {
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
	firstKeyID, _, err := cache.GetActive(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("GetActive (before rotate) failed: %v", err)
	}
	if firstKeyID != first.ID {
		t.Fatalf("expected cached keyID %s, got %s", first.ID, firstKeyID)
	}

	deps.Cache = cache
	next, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	secondKeyID, _, err := cache.GetActive(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("GetActive (after rotate) failed: %v", err)
	}
	if secondKeyID != next.ID {
		t.Fatalf("expected cache to resolve the rotated keyID %s, got %s (stale cache not invalidated)", next.ID, secondKeyID)
	}
}
