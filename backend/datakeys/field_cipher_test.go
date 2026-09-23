package datakeys

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/db_memory"
	"github.com/ambi/idmagic/backend/datakeys/domain"
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

//spec:covers REQ-DATAKEYS-002: 実際の FieldCipher 配線でも、回転前の暗号文は retiring 鍵で復号できる。
func TestFieldCipherDecryptsExistingCiphertextAfterRotation(t *testing.T) {
	fieldCipher, deps := newTestFieldCipher(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	if _, err := usecases.BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatal(err)
	}
	version, ciphertext, err := fieldCipher.Encrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	plaintext, err := fieldCipher.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("plaintext=%q", plaintext)
	}
}

//spec:covers EX-DATAKEYS-003-01: retiring の版 1 を disable すると保存先で disabled になり、キャッシュ済みだった版 1 の暗号文の復号が ErrDataKeyUnavailable で拒否される。
func TestFieldCipherDecryptFailsClosedAfterDisablingTheRetiringVersion(t *testing.T) {
	fieldCipher, deps := newTestFieldCipher(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	if _, err := usecases.BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatal(err)
	}
	version, ciphertext, err := fieldCipher.Encrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	// disable の前に retiring の版 1 で復号が通ることを確かめ、版 1 の DEK をキャッシュに載せる。
	if _, err := fieldCipher.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext); err != nil {
		t.Fatalf("Decrypt under retiring v1 failed: %v", err)
	}

	if err := usecases.DisableTenantDataKey(ctx, deps, "tenant-a", version, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	disabled, err := deps.Repository.FindByVersion(ctx, "tenant-a", version)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != domain.DataKeyStatusDisabled {
		t.Fatalf("v1 status = %s, want disabled", disabled.Status)
	}
	plaintext, err := fieldCipher.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext)
	if !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("Decrypt under disabled v1 = (%q, %v), want ErrDataKeyUnavailable", plaintext, err)
	}
}

// migratedFieldMigrator は、再暗号化ジョブがすべての参照を active の版へ移し終えた状態を表す。
type migratedFieldMigrator struct{}

func (migratedFieldMigrator) ReencryptBatch(context.Context, string, int, int) (int, error) {
	return 0, nil
}

func (migratedFieldMigrator) PendingCount(context.Context, string, int) (int, error) {
	return 0, nil
}

//spec:covers EX-DATAKEYS-005-01: 参照がすべて移行済みなら版 1 は destroyed になって wrapped_dek が消え、キャッシュを作り直しても版 1 の暗号文を復号できない。
func TestFieldCipherCannotDecryptUnderADestroyedVersion(t *testing.T) {
	fieldCipher, deps := newTestFieldCipher(t)
	migrators := usecases.NewMigratorRegistry()
	migrators.Register("mfa_totp_secret", migratedFieldMigrator{})
	deps.Migrators = migrators
	ctx := context.Background()
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	if _, err := usecases.BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatal(err)
	}
	version, ciphertext, err := fieldCipher.Encrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	if err := usecases.DestroyTenantDataKey(ctx, deps, "tenant-a", version, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("DestroyTenantDataKey failed: %v", err)
	}
	destroyed, err := deps.Repository.FindByVersion(ctx, "tenant-a", version)
	if err != nil {
		t.Fatal(err)
	}
	if destroyed.Status != domain.DataKeyStatusDestroyed || destroyed.WrappedDEK != nil {
		t.Fatalf("v1 status=%s wrapped_dek=%d byte(s), want destroyed with no wrapped_dek", destroyed.Status, len(destroyed.WrappedDEK))
	}
	// 別のレプリカや再起動後のワーカーのように、版 1 を一度もキャッシュしていない FieldCipher でも同じである。
	freshCipher := &FieldCipher{Repository: deps.Repository, Cache: usecases.NewDataKeyCache(deps.Repository, deps.Crypto), Crypto: deps.Crypto}
	for name, cipher := range map[string]*FieldCipher{"same cache": fieldCipher, "fresh cache": freshCipher} {
		plaintext, err := cipher.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext)
		if !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
			t.Fatalf("%s: Decrypt under destroyed v1 = (%q, %v), want ErrDataKeyUnavailable", name, plaintext, err)
		}
	}
}

// TestFieldCipherEncryptBootstrapsFirstDataKey mirrors SigningKeys' lazy
// per-tenant key creation (docs/domain/data-keys/internals.md): a tenant's first
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
// decrypted under a different record id (AAD binding) — e.g. one
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
	} else if !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("expected ErrDataKeyUnavailable, got %v", err)
	}
}

// TestFieldCipherDecryptFailsClosedForWrongKeyVersion covers wi-97 T008: a
// ciphertext encrypted under one DEK version must not decrypt under a
// different (but validly wrapped) version's key material — the AEAD
// authentication tag only verifies against the exact key that produced it,
// so an attacker who rewrites a row's stored key_version (or a bug that
// mismatches ciphertext/version) cannot silently decrypt to a garbled or
// substituted plaintext; it must fail-closed instead.
func TestFieldCipherDecryptFailsClosedForWrongKeyVersion(t *testing.T) {
	fc, deps := newTestFieldCipher(t)
	ctx := context.Background()
	if _, err := usecases.BootstrapTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	version, ciphertext, err := fc.Encrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if _, err := usecases.RotateTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	if _, err := fc.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", version+1, ciphertext); err == nil {
		t.Fatal("expected Decrypt to fail-closed when the claimed key version does not match the one that encrypted the ciphertext")
	} else if !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("expected ErrDataKeyUnavailable, got %v", err)
	}
}

// unwrapUnavailableMasterKeyProvider wraps a real MasterKeyProvider but lets
// tests flip Unwrap into failure — simulating a master-key provider (e.g.
// OpenBao) going unreachable strictly between a successful Wrap (earlier,
// while healthy) and a later Decrypt (wi-97 T008 provider-outage scenario).
type unwrapUnavailableMasterKeyProvider struct {
	envelope_crypto.MasterKeyProvider
	unwrapUnavailable bool
}

func (p *unwrapUnavailableMasterKeyProvider) UnwrapDataKey(ctx context.Context, tenantID string, wrapped []byte, masterKeyID string) ([]byte, error) {
	if p.unwrapUnavailable {
		return nil, errors.New("fake: master key provider unreachable")
	}
	return p.MasterKeyProvider.UnwrapDataKey(ctx, tenantID, wrapped, masterKeyID)
}

func (p *unwrapUnavailableMasterKeyProvider) Healthy(ctx context.Context) bool {
	return !p.unwrapUnavailable && p.MasterKeyProvider.Healthy(ctx)
}

// TestFieldCipherDecryptFailsClosedWhenProviderUnavailable covers wi-97 T008:
// once a secret is already encrypted, a master-key provider outage (OpenBao
// down/restarting) must deny decryption rather than serve stale cached
// plaintext or fall back to any other path.
func TestFieldCipherDecryptFailsClosedWhenProviderUnavailable(t *testing.T) {
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	provider := &unwrapUnavailableMasterKeyProvider{MasterKeyProvider: master}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(provider)
	cache := usecases.NewDataKeyCache(repo, crypto)
	fc := &FieldCipher{Repository: repo, Cache: cache, Crypto: crypto}
	deps := usecases.Deps{Repository: repo, Crypto: crypto, Cache: cache}
	ctx := context.Background()
	if _, err := usecases.BootstrapTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	version, ciphertext, err := fc.Encrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", "JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// A fresh cache forces GetByVersion to re-unwrap rather than serve a
	// value cached from before the outage, matching a worker restart or a
	// second replica that never had it cached.
	cache = usecases.NewDataKeyCache(repo, crypto)
	fc = &FieldCipher{Repository: repo, Cache: cache, Crypto: crypto}
	provider.unwrapUnavailable = true

	if _, err := fc.Decrypt(ctx, "tenant-a", "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext); err == nil {
		t.Fatal("expected Decrypt to fail-closed while the master key provider is unavailable")
	}
}
