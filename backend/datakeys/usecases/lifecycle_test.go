package usecases

// 主要ユースケース追跡: REQ-DATAKEYS-002。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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

// generatedKeyRecorder は、ユースケースの内側で生成された平文 DEK を記録する。
// 平文がどこにも残らないことは、生成された値そのものを探さなければ確かめられない。
type generatedKeyRecorder struct {
	envelope_crypto.EnvelopeCrypto
	generated [][]byte
}

func (r *generatedKeyRecorder) GenerateDataKey(ctx context.Context) ([]byte, error) {
	dek, err := r.EnvelopeCrypto.GenerateDataKey(ctx)
	if err == nil {
		r.generated = append(r.generated, append([]byte(nil), dek...))
	}
	return dek, err
}

//spec:covers EX-DATAKEYS-001-01: 保存先を読み直して版 1 が active であり、保存値は生成された平文 DEK を wrap した形だけで、保存値にもイベントにも平文 DEK が現れない。
func TestBootstrapTenantDataKeyPersistsOnlyTheWrappedVersionOne(t *testing.T) {
	deps := newTestDeps(t)
	recorder := &generatedKeyRecorder{EnvelopeCrypto: deps.Crypto}
	deps.Crypto = recorder
	var emitted []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }
	ctx := context.Background()

	if _, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("BootstrapTenantDataKey failed: %v", err)
	}

	stored, err := deps.Repository.FindActive(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("FindActive failed: %v", err)
	}
	if stored.Version != 1 || stored.Status != domain.DataKeyStatusActive {
		t.Fatalf("stored key version=%d status=%s, want active version 1", stored.Version, stored.Status)
	}
	if len(recorder.generated) != 1 {
		t.Fatalf("generated %d data keys, want 1", len(recorder.generated))
	}
	plaintextDEK := recorder.generated[0]
	unwrapped, err := deps.Crypto.Unwrap(ctx, "tenant-a", stored.WrappedDEK, stored.MasterKeyID)
	if err != nil {
		t.Fatalf("Unwrap stored wrapped_dek failed: %v", err)
	}
	if !bytes.Equal(unwrapped, plaintextDEK) {
		t.Fatal("stored wrapped_dek does not wrap the generated data key")
	}

	// 平文は生のバイト列としても、JSON が []byte に用いる base64 としても探す。
	plaintextForms := [][]byte{plaintextDEK, []byte(base64.StdEncoding.EncodeToString(plaintextDEK))}
	storedJSON, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if len(emitted) != 1 || emitted[0].EventType() != "DataEncryptionKeyBootstrapped" {
		t.Fatalf("expected 1 DataEncryptionKeyBootstrapped event, got %+v", emitted)
	}
	eventJSON, err := json.Marshal(emitted[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, form := range plaintextForms {
		if bytes.Contains(stored.WrappedDEK, form) || bytes.Contains(storedJSON, form) {
			t.Fatal("the stored data key carries the plaintext DEK")
		}
		if bytes.Contains(eventJSON, form) {
			t.Fatalf("the bootstrap event carries the plaintext DEK: %s", eventJSON)
		}
	}
}

// wrapUnavailableMasterKeyProvider は、MasterKey プロバイダーが wrap の時点で到達できない状態を作る。
type wrapUnavailableMasterKeyProvider struct {
	envelope_crypto.MasterKeyProvider
	unreachable bool
}

func (p *wrapUnavailableMasterKeyProvider) WrapDataKey(ctx context.Context, tenantID string, plaintextDEK []byte) ([]byte, string, error) {
	if p.unreachable {
		return nil, "", errors.New("fake: master key provider unreachable")
	}
	return p.MasterKeyProvider.WrapDataKey(ctx, tenantID, plaintextDEK)
}

//spec:covers EX-DATAKEYS-001-02: プロバイダーに到達できないと ErrDataKeyUnavailable で失敗し、保存先に DEK が無くイベントも出ない。到達できれば同じ配線で作成される。
func TestBootstrapTenantDataKeyFailsClosedWhenMasterKeyProviderIsUnreachable(t *testing.T) {
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	provider := &wrapUnavailableMasterKeyProvider{MasterKeyProvider: master, unreachable: true}
	var emitted []spec.DomainEvent
	deps := Deps{
		Repository: db_memory.NewDataKeyRepository(),
		Crypto:     envelope_crypto.NewTinkEnvelopeCrypto(provider),
		Emit:       func(e spec.DomainEvent) { emitted = append(emitted, e) },
	}
	ctx := context.Background()

	_, err = BootstrapTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC())
	if !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("BootstrapTenantDataKey error = %v, want ErrDataKeyUnavailable", err)
	}
	keys, err := deps.Repository.ListAll(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("a refused bootstrap stored %d data key(s)", len(keys))
	}
	if len(emitted) != 0 {
		t.Fatalf("a refused bootstrap emitted %+v", emitted)
	}

	provider.unreachable = false
	if _, err := BootstrapTenantDataKey(ctx, deps, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("BootstrapTenantDataKey with a reachable provider failed: %v", err)
	}
}

//spec:covers EX-DATAKEYS-002-01: 保存先で版 2 が active、版 1 が retiring になり、保存先から取り出した retiring の版 1 で回転前の暗号文を復号できる。
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
	active, err := deps.Repository.FindActive(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("FindActive failed: %v", err)
	}
	if active.Version != 2 {
		t.Fatalf("stored active version = %d, want 2", active.Version)
	}

	// version 1 は retiring として保存に残り、そこから取り出した DEK で既存の
	// 暗号文が復号できなければならない。手元に持っている平文 DEK で復号しても、
	// 保存側が旧版を残したかどうかは分からない。
	retiring, err := deps.Repository.FindByVersion(ctx, "tenant-a", first.Version)
	if err != nil {
		t.Fatalf("FindByVersion(v1) failed: %v", err)
	}
	if retiring.Status != domain.DataKeyStatusRetiring {
		t.Fatalf("v1 status = %v, want retiring", retiring.Status)
	}
	storedDEKv1, err := deps.Crypto.Unwrap(ctx, "tenant-a", retiring.WrappedDEK, retiring.MasterKeyID)
	if err != nil {
		t.Fatalf("Unwrap stored v1 failed: %v", err)
	}
	decrypted, err := deps.Crypto.Decrypt(ctx, storedDEKv1, aad, ciphertext)
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

// InvalidRequestError は、ドメインでは ErrDataKeyIsActive として現れる。
//
//spec:covers EX-DATAKEYS-004-01: active の版 2 の disable は ErrDataKeyIsActive で拒否され、保存先で版 2 は active のままで、disable のイベントも出ない。
func TestDisableTenantDataKeyRejectsActiveVersion(t *testing.T) {
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
	if err := DisableTenantDataKey(ctx, deps, "tenant-a", 2, now.Add(2*time.Hour)); !errors.Is(err, domain.ErrDataKeyIsActive) {
		t.Fatalf("expected ErrDataKeyIsActive, got %v", err)
	}
	key, err := deps.Repository.FindByVersion(ctx, "tenant-a", 2)
	if err != nil {
		t.Fatalf("FindByVersion(v2) failed: %v", err)
	}
	if key.Status != domain.DataKeyStatusActive || key.DisabledAt != nil {
		t.Fatalf("v2 status=%s disabled_at=%v after a refused disable, want active", key.Status, key.DisabledAt)
	}
	if len(emitted) != 0 {
		t.Fatalf("a refused disable emitted %+v", emitted)
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
// the docs/design/data/database.md design: rotation must kick off the
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

// 登録済みの FieldMigrator に未移行の行が残るうちに版を破棄すると、その行は恒久的に復号できなくなる。
//
//spec:covers EX-DATAKEYS-005-02: 未移行の参照が残ると ErrDataKeyStillReferenced で拒否され、保存先で版 1 は retiring のまま wrapped_dek も残り、destroy のイベントも出ない。
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
	var emitted []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }

	if err := DestroyTenantDataKey(ctx, deps, "tenant-a", 1, now.Add(2*time.Hour)); !errors.Is(err, domain.ErrDataKeyStillReferenced) {
		t.Fatalf("expected ErrDataKeyStillReferenced, got %v", err)
	}

	key, err := deps.Repository.FindByVersion(ctx, "tenant-a", 1)
	if err != nil {
		t.Fatalf("FindByVersion failed: %v", err)
	}
	if key.Status != domain.DataKeyStatusRetiring {
		t.Fatalf("v1 status = %s after a rejected destroy, want retiring", key.Status)
	}
	if key.WrappedDEK == nil {
		t.Fatal("expected wrapped_dek to survive a rejected destroy")
	}
	if len(emitted) != 0 {
		t.Fatalf("a rejected destroy emitted %+v", emitted)
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
