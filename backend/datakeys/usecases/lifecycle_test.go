package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/db_memory"
	"github.com/ambi/idmagic/backend/datakeys/domain"
	jobsdbmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newTestDeps(t *testing.T) Deps {
	t.Helper()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	return Deps{
		Repository: db_memory.NewDataKeyRepository(),
		Crypto:     envelope_crypto.NewTinkEnvelopeCrypto(master),
	}
}

// TestBootstrapTenantDataKey covers scenario
// "テナント初回利用時にDEKがbootstrapされる" (spec/contexts/data-keys.yaml).
func TestBootstrapTenantDataKey(t *testing.T) {
	deps := newTestDeps(t)
	var emitted []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }

	key, err := BootstrapTenantDataKey(context.Background(), deps, "tenant-a", time.Now().UTC())
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if key.Version != 1 || key.Status != domain.DataKeyStatusActive {
		t.Fatalf("expected active version 1, got version=%d status=%s", key.Version, key.Status)
	}
	if len(key.WrappedDEK) == 0 {
		t.Fatal("expected wrapped_dek to be populated")
	}

	if len(emitted) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(emitted))
	}
	if emitted[0].EventType() != "DataEncryptionKeyBootstrapped" {
		t.Fatalf("unexpected event type: %s", emitted[0].EventType())
	}
}

// TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion covers scenario
// "DEKをrotationしても既存暗号文が復号できる" (spec/contexts/data-keys.yaml): a
// secret encrypted under the pre-rotation active DEK must still decrypt via
// its retiring version after rotation.
func TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	now := time.Now().UTC()

	first, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now)
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	plaintextDEKv1, err := deps.Crypto.Unwrap(ctx, "tenant-a", first.WrappedDEK, first.MasterKeyID)
	if err != nil {
		t.Fatalf("Unwrap v1 failed: %v", err)
	}
	aad := envelope_crypto.AAD{TenantID: "tenant-a", Context: "Authentication", Table: "mfa_factors", RecordID: "user-1", Field: "secret"}
	ciphertext, err := deps.Crypto.Encrypt(ctx, plaintextDEKv1, aad, []byte("JBSWY3DPEHPK3PXP"))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	var emitted []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }
	next, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}
	if next.Version != 2 {
		t.Fatalf("expected rotated version 2, got %d", next.Version)
	}
	if len(emitted) != 1 || emitted[0].EventType() != "DataEncryptionKeyRotated" {
		t.Fatalf("expected 1 DataEncryptionKeyRotated event, got %+v", emitted)
	}

	// version 1 is now retiring but its ciphertext must still decrypt.
	decrypted, err := deps.Crypto.Decrypt(ctx, plaintextDEKv1, aad, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt with retiring v1 DEK failed: %v", err)
	}
	if string(decrypted) != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("unexpected decrypted value: %s", decrypted)
	}
}

func TestRotateTenantDataKeyFailsWithoutBootstrap(t *testing.T) {
	deps := newTestDeps(t)
	if _, err := RotateTenantDataKey(context.Background(), deps, "tenant-a", time.Now().UTC()); !errors.Is(err, domain.ErrNoActiveDataKey) {
		t.Fatalf("expected ErrNoActiveDataKey, got %v", err)
	}
}

// TestDisableTenantDataKeyRejectsActiveVersion covers scenario
// "activeなDEKは直接disableできない" (spec/contexts/data-keys.yaml).
func TestDisableTenantDataKeyRejectsActiveVersion(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	now := time.Now().UTC()
	key, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now)
	if err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if err := DisableTenantDataKey(ctx, deps, "tenant-a", key.Version, now); !errors.Is(err, domain.ErrDataKeyIsActive) {
		t.Fatalf("expected ErrDataKeyIsActive, got %v", err)
	}
}

// TestDisableTenantDataKeyLocksOutRetiringVersion covers scenario
// "retiringのDEKを即時ロックアウトできる" (spec/contexts/data-keys.yaml).
func TestDisableTenantDataKeyLocksOutRetiringVersion(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if _, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	var emitted []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }
	if err := DisableTenantDataKey(ctx, deps, "tenant-a", 1, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("DisableTenantDataKey failed: %v", err)
	}
	if len(emitted) != 1 || emitted[0].EventType() != "DataEncryptionKeyDisabled" {
		t.Fatalf("expected 1 DataEncryptionKeyDisabled event, got %+v", emitted)
	}
}

// TestDestroyTenantDataKeyErasesWrappedDEK covers scenario
// "全参照の再暗号化後にDEKをdestroyできる" (spec/contexts/data-keys.yaml).
func TestDestroyTenantDataKeyErasesWrappedDEK(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if _, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	var emitted []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }
	if err := DestroyTenantDataKey(ctx, deps, "tenant-a", 1, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("DestroyTenantDataKey failed: %v", err)
	}
	if len(emitted) != 1 || emitted[0].EventType() != "DataEncryptionKeyDestroyed" {
		t.Fatalf("expected 1 DataEncryptionKeyDestroyed event, got %+v", emitted)
	}

	destroyed, err := deps.Repository.FindByVersion(ctx, "tenant-a", 1)
	if err != nil {
		t.Fatalf("FindByVersion failed: %v", err)
	}
	if destroyed.WrappedDEK != nil {
		t.Fatal("expected wrapped_dek to be erased after destroy")
	}
}

// TestRotateTenantDataKeyEnqueuesReencryptionJobForRegisteredMigrators covers
// the spec/persistence.md design: rotation must kick off the
// resumable re-encryption job for every registered FieldMigrator so old
// references eventually migrate onto the new active version.
func TestRotateTenantDataKeyEnqueuesReencryptionJobForRegisteredMigrators(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}

	migrators := NewMigratorRegistry()
	migrators.Register("mfa_totp_secret", &fakeReencryptMigrator{})
	jobRepo := jobsdbmemory.NewJobRepository()
	deps.Migrators = migrators
	deps.Jobs = jobRepo

	if _, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	jobs, err := jobRepo.ListByTenantAndKinds(ctx, "tenant-a", []jobsdomain.JobKind{jobsdomain.KindDataKeyReencryption}, 10)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 reencryption job enqueued after rotate, got %d", len(jobs))
	}
}

// TestDestroyTenantDataKeyRejectsWhenMigratorReportsPendingRecords covers the
// new DestroyTenantDataKey destroy gate (spec/contexts/data-keys.yaml
// DataKeyStillReferencedError, wi-97 T006): crypto-shredding a version while
// a registered FieldMigrator still has unmigrated rows would make them
// permanently undecryptable.
func TestDestroyTenantDataKeyRejectsWhenMigratorReportsPendingRecords(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if _, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	migrators := NewMigratorRegistry()
	migrators.Register("mfa_totp_secret", &fakeReencryptMigrator{pending: 3})
	deps.Migrators = migrators

	if err := DestroyTenantDataKey(ctx, deps, "tenant-a", 1, now.Add(2*time.Hour)); !errors.Is(err, domain.ErrDataKeyStillReferenced) {
		t.Fatalf("expected ErrDataKeyStillReferenced, got %v", err)
	}

	key, err := deps.Repository.FindByVersion(ctx, "tenant-a", 1)
	if err != nil {
		t.Fatalf("FindByVersion failed: %v", err)
	}
	if key.WrappedDEK == nil {
		t.Fatal("expected wrapped_dek to survive a rejected destroy")
	}
}

// TestDestroyTenantDataKeyAllowsWhenMigratorReportsNoPendingRecords is the
// green-path counterpart: once every registered FieldMigrator reports 0
// pending rows, destroy proceeds normally.
func TestDestroyTenantDataKeyAllowsWhenMigratorReportsNoPendingRecords(t *testing.T) {
	deps := newTestDeps(t)
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", now); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}
	if _, err := RotateTenantDataKey(ctx, deps, "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatalf("RotateTenantDataKey failed: %v", err)
	}

	migrators := NewMigratorRegistry()
	migrators.Register("mfa_totp_secret", &fakeReencryptMigrator{pending: 0})
	deps.Migrators = migrators

	if err := DestroyTenantDataKey(ctx, deps, "tenant-a", 1, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("DestroyTenantDataKey failed: %v", err)
	}
}
