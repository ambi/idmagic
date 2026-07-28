package datakeys

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/db_memory"
	"github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

func newTestFieldCipher(t *testing.T) (*FieldCipher, usecases.Deps) {
	t.Helper()
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	deps := usecases.Deps{Repository: repo, Crypto: crypto}
	cache := usecases.NewDataKeyCache(repo, crypto)
	deps.Cache = cache
	return &FieldCipher{Repository: repo, Cache: cache, Crypto: crypto}, deps
}

// TestFieldCipherEncryptDecryptRoundTrip covers the T005 repository
// migration path: a field (e.g. MFA TOTP seed) encrypted with the tenant's
// active DEK must decrypt back to the original plaintext.
func TestFieldCipherEncryptDecryptRoundTrip(t *testing.T) {
	fc, deps := newTestFieldCipher(t)
	ctx := context.Background()
	if _, err := usecases.BootstrapTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}

	version, ciphertext, err := fc.Encrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected key version 1, got %d", version)
	}

	plaintext, err := fc.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if plaintext != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("unexpected plaintext: %s", plaintext)
	}
}

// TestFieldCipherEncryptBootstrapsFirstDataKey mirrors SigningKeys' lazy
// per-tenant key creation (ARCHITECTURE.md Lifecycle): a tenant's first
// encrypt call must not require a separate provisioning step to have
// already run BootstrapTenantDataKey.
func TestFieldCipherEncryptBootstrapsFirstDataKey(t *testing.T) {
	fc, _ := newTestFieldCipher(t)
	ctx := context.Background()

	version, ciphertext, err := fc.Encrypt(ctx, "tenant-new", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatalf("Encrypt failed to lazily bootstrap a data key: %v", err)
	}
	if version != 1 {
		t.Fatalf("expected first bootstrapped key version 1, got %d", version)
	}

	plaintext, err := fc.Decrypt(ctx, "tenant-new", "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if plaintext != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("unexpected plaintext: %s", plaintext)
	}
}

// TestFieldCipherDecryptFailsClosedAcrossRecords ensures ciphertext cannot be
// decrypted under a different record id (AAD binding, ADR-148) — e.g. one
// user's encrypted secret cannot be replayed onto another user's row.
func TestFieldCipherDecryptFailsClosedAcrossRecords(t *testing.T) {
	fc, deps := newTestFieldCipher(t)
	ctx := context.Background()
	if _, err := usecases.BootstrapTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}

	version, ciphertext, err := fc.Encrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if _, err := fc.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-2:totp", "secret", version, ciphertext); err == nil {
		t.Fatal("expected Decrypt to fail-closed for a mismatched record id")
	} else if !errors.Is(err, envelope_crypto.ErrDecryptionFailed) {
		t.Fatalf("expected ErrDecryptionFailed, got %v", err)
	}
}
