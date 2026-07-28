package envelope_crypto

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type fakeMasterKeyProvider struct {
	sealKey []byte
	healthy bool
}

func newFakeMasterKeyProvider() *fakeMasterKeyProvider {
	return &fakeMasterKeyProvider{sealKey: []byte("fake-master-key-material-32byte"), healthy: true}
}

func (f *fakeMasterKeyProvider) WrapDataKey(_ context.Context, tenantID string, plaintextDEK []byte) ([]byte, string, error) {
	wrapped := make([]byte, len(plaintextDEK))
	for i, b := range plaintextDEK {
		wrapped[i] = b ^ f.sealKey[i%len(f.sealKey)]
	}
	return wrapped, "fake-master-key:" + tenantID, nil
}

func (f *fakeMasterKeyProvider) UnwrapDataKey(_ context.Context, tenantID string, wrapped []byte, masterKeyID string) ([]byte, error) {
	if masterKeyID != "fake-master-key:"+tenantID {
		return nil, errors.New("fake master key: tenant/key id mismatch")
	}
	plaintext := make([]byte, len(wrapped))
	for i, b := range wrapped {
		plaintext[i] = b ^ f.sealKey[i%len(f.sealKey)]
	}
	return plaintext, nil
}

func (f *fakeMasterKeyProvider) Healthy(_ context.Context) bool {
	return f.healthy
}

// TestGenerateWrapUnwrapRoundTrip は scenario
// "テナント初回利用時にDEKがbootstrapされる" (spec/contexts/data-keys.yaml) の
// wrap/unwrap 部分を GenerateDataKey→Wrap→Unwrap の往復で検証する。
func TestGenerateWrapUnwrapRoundTrip(t *testing.T) {
	ctx := context.Background()
	crypto := NewTinkEnvelopeCrypto(newFakeMasterKeyProvider())

	dek, err := crypto.GenerateDataKey(ctx)
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	if len(dek) == 0 {
		t.Fatal("GenerateDataKey returned empty key material")
	}

	wrapped, masterKeyID, err := crypto.Wrap(ctx, "tenant-a", dek)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}
	if masterKeyID == "" {
		t.Fatal("Wrap returned empty master key id")
	}

	unwrapped, err := crypto.Unwrap(ctx, "tenant-a", wrapped, masterKeyID)
	if err != nil {
		t.Fatalf("Unwrap failed: %v", err)
	}
	if !bytes.Equal(unwrapped, dek) {
		t.Fatal("Unwrap did not return the original plaintext DEK")
	}
}

// TestUnwrapRejectsWrongTenant は AAD/鍵境界を跨いだ wrap 材料の付け替えを
// fail-closed で拒否することを検証する (ADR-148)。
func TestUnwrapRejectsWrongTenant(t *testing.T) {
	ctx := context.Background()
	crypto := NewTinkEnvelopeCrypto(newFakeMasterKeyProvider())

	dek, err := crypto.GenerateDataKey(ctx)
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	wrapped, masterKeyID, err := crypto.Wrap(ctx, "tenant-a", dek)
	if err != nil {
		t.Fatalf("Wrap failed: %v", err)
	}

	if _, err := crypto.Unwrap(ctx, "tenant-b", wrapped, masterKeyID); err == nil {
		t.Fatal("expected Unwrap to fail-closed for the wrong tenant")
	}
}

// TestEncryptDecryptKnownAnswer verifies Tink AEAD round-trips a secret using
// a generated DEK (scenario 全参照の再暗号化後にDEKをdestroyできる 前提の
// encrypt/decrypt path, spec/contexts/data-keys.yaml).
func TestEncryptDecryptKnownAnswer(t *testing.T) {
	ctx := context.Background()
	crypto := NewTinkEnvelopeCrypto(newFakeMasterKeyProvider())

	dek, err := crypto.GenerateDataKey(ctx)
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	aad := AAD{TenantID: "tenant-a", Context: "Authentication", Table: "mfa_factors", RecordID: "user-1", Field: "secret"}
	plaintext := []byte("JBSWY3DPEHPK3PXP")

	ciphertext, err := crypto.Encrypt(ctx, dek, aad, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext must not equal plaintext")
	}

	decrypted, err := crypto.Decrypt(ctx, dek, aad, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted plaintext mismatch: got %q want %q", decrypted, plaintext)
	}
}

// TestDecryptFailsClosedOnTamperedCiphertext ensures a bit-flipped ciphertext
// is rejected rather than returning corrupted plaintext (ADR-148 fail-closed).
func TestDecryptFailsClosedOnTamperedCiphertext(t *testing.T) {
	ctx := context.Background()
	crypto := NewTinkEnvelopeCrypto(newFakeMasterKeyProvider())

	dek, err := crypto.GenerateDataKey(ctx)
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	aad := AAD{TenantID: "tenant-a", Context: "Authentication", Table: "mfa_factors", RecordID: "user-1", Field: "secret"}
	ciphertext, err := crypto.Encrypt(ctx, dek, aad, []byte("JBSWY3DPEHPK3PXP"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	tampered := append([]byte(nil), ciphertext...)
	tampered[len(tampered)-1] ^= 0xFF

	if _, err := crypto.Decrypt(ctx, dek, aad, tampered); err == nil {
		t.Fatal("expected Decrypt to fail-closed on tampered ciphertext")
	} else if !errors.Is(err, ErrDecryptionFailed) {
		t.Fatalf("expected ErrDecryptionFailed, got %v", err)
	}
}

// TestDecryptFailsClosedOnAADMismatch は ciphertext をテナント/テーブル/フィールドを
// またいで付け替えても復号できないことを検証する (ADR-148 の AAD 束縛)。
func TestDecryptFailsClosedOnAADMismatch(t *testing.T) {
	ctx := context.Background()
	crypto := NewTinkEnvelopeCrypto(newFakeMasterKeyProvider())

	dek, err := crypto.GenerateDataKey(ctx)
	if err != nil {
		t.Fatalf("GenerateDataKey failed: %v", err)
	}
	aad := AAD{TenantID: "tenant-a", Context: "Authentication", Table: "mfa_factors", RecordID: "user-1", Field: "secret"}
	ciphertext, err := crypto.Encrypt(ctx, dek, aad, []byte("JBSWY3DPEHPK3PXP"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	wrongTenantAAD := aad
	wrongTenantAAD.TenantID = "tenant-b"
	if _, err := crypto.Decrypt(ctx, dek, wrongTenantAAD, ciphertext); err == nil {
		t.Fatal("expected Decrypt to fail-closed when tenant in AAD differs")
	} else if !errors.Is(err, ErrDecryptionFailed) {
		t.Fatalf("expected ErrDecryptionFailed, got %v", err)
	}

	wrongFieldAAD := aad
	wrongFieldAAD.Field = "other_field"
	if _, err := crypto.Decrypt(ctx, dek, wrongFieldAAD, ciphertext); err == nil {
		t.Fatal("expected Decrypt to fail-closed when field in AAD differs")
	}
}

// TestHealthyDelegatesToMasterKeyProvider ensures a failing master key
// provider is surfaced as unhealthy (fail-closed signal, mirrors SigningKeys'
// KeyStore.Healthy pattern).
func TestHealthyDelegatesToMasterKeyProvider(t *testing.T) {
	ctx := context.Background()
	provider := newFakeMasterKeyProvider()
	crypto := NewTinkEnvelopeCrypto(provider)

	if !crypto.Healthy(ctx) {
		t.Fatal("expected Healthy to be true when provider is healthy")
	}
	provider.healthy = false
	if crypto.Healthy(ctx) {
		t.Fatal("expected Healthy to be false when provider is unhealthy")
	}
}
