package envelope_cleartext

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

// TestWrapUnwrapRoundTrip covers the dev/local path of scenario
// "テナント初回利用時にDEKがbootstrapされる" (spec/contexts/data-keys.yaml):
// no OpenBao required, wrap/unwrap must still round-trip.
func TestWrapUnwrapRoundTrip(t *testing.T) {
	ctx := context.Background()
	provider, err := NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	plaintextDEK := []byte("0123456789abcdef0123456789abcdef")

	wrapped, masterKeyID, err := provider.WrapDataKey(ctx, "tenant-a", plaintextDEK)
	if err != nil {
		t.Fatalf("WrapDataKey failed: %v", err)
	}
	if bytes.Equal(wrapped, plaintextDEK) {
		t.Fatal("wrapped DEK must not equal plaintext")
	}

	unwrapped, err := provider.UnwrapDataKey(ctx, "tenant-a", wrapped, masterKeyID)
	if err != nil {
		t.Fatalf("UnwrapDataKey failed: %v", err)
	}
	if !bytes.Equal(unwrapped, plaintextDEK) {
		t.Fatal("UnwrapDataKey did not return the original plaintext DEK")
	}
}

// TestUnwrapRejectsWrongTenant ensures a wrapped DEK cannot be unwrapped
// under a different tenant's AAD binding (fail-closed).
func TestUnwrapRejectsWrongTenant(t *testing.T) {
	ctx := context.Background()
	provider, err := NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	wrapped, masterKeyID, err := provider.WrapDataKey(ctx, "tenant-a", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("WrapDataKey failed: %v", err)
	}

	if _, err := provider.UnwrapDataKey(ctx, "tenant-b", wrapped, masterKeyID); err == nil {
		t.Fatal("expected UnwrapDataKey to fail-closed for the wrong tenant")
	} else if !errors.Is(err, envelope_crypto.ErrDecryptionFailed) {
		t.Fatalf("expected ErrDecryptionFailed, got %v", err)
	}
}

// TestHealthyIsAlwaysTrue: the cleartext provider has no external dependency,
// so it is always healthy (dev/local only, never selected in production).
func TestHealthyIsAlwaysTrue(t *testing.T) {
	provider, err := NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	if !provider.Healthy(context.Background()) {
		t.Fatal("expected cleartext provider to always report healthy")
	}
}

// TestImplementsMasterKeyProviderPort is a compile-time-ish check that the
// concrete type satisfies the shared port.
func TestImplementsMasterKeyProviderPort(t *testing.T) {
	var _ envelope_crypto.MasterKeyProvider = (*CleartextMasterKeyProvider)(nil)
}
