package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/db_memory"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	jobsdbmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

// fakeReencryptMigrator lets tests script ReencryptBatch's return sequence
// and PendingCount's answer, and observe how many batches actually ran.
type fakeReencryptMigrator struct {
	batchReturns []int
	calls        int
	pending      int
}

func (f *fakeReencryptMigrator) ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (int, error) {
	idx := f.calls
	f.calls++
	if idx < len(f.batchReturns) {
		return f.batchReturns[idx], nil
	}
	return 0, nil
}

func (f *fakeReencryptMigrator) PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error) {
	return f.pending, nil
}

// newReencryptTestRepo bootstraps a DataKeyRepository with "tenant-a"'s
// first DataEncryptionKey active, the tenant every test in this file uses.
func newReencryptTestRepo(t *testing.T) ports.DataKeyRepository {
	t.Helper()
	repo := db_memory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	if _, err := BootstrapTenantDataKey(context.Background(), Deps{Repository: repo, Crypto: crypto}, "tenant-a", time.Now().UTC()); err != nil {
		t.Fatalf("bootstrap tenant data key: %v", err)
	}
	return repo
}

func TestReencryptTenantField_MigratesAllInOneRunWhenUnderCap(t *testing.T) {
	repo := newReencryptTestRepo(t)
	migrators := NewMigratorRegistry()
	migrator := &fakeReencryptMigrator{batchReturns: []int{3}, pending: 0}
	migrators.Register("mfa_totp_secret", migrator)

	migrated, remaining, err := ReencryptTenantField(context.Background(), ReencryptDeps{Repository: repo, Migrators: migrators}, "tenant-a", "mfa_totp_secret")
	if err != nil {
		t.Fatalf("ReencryptTenantField failed: %v", err)
	}
	if migrated != 3 || remaining != 0 {
		t.Fatalf("migrated=%d remaining=%d, want 3 0", migrated, remaining)
	}
	if migrator.calls != 1 {
		t.Fatalf("expected exactly 1 ReencryptBatch call (fewer than a full batch), got %d", migrator.calls)
	}
}

// TestReencryptTenantField_StopsAtMaxBatchesPerRunAndReportsRemaining covers
// the lane-fairness cap: a tenant with more pending rows than
// ReencryptMaxBatchesPerRun*ReencryptBatchSize is not migrated to completion
// in one call; ReencryptionHandler is responsible for the continuation.
func TestReencryptTenantField_StopsAtMaxBatchesPerRunAndReportsRemaining(t *testing.T) {
	repo := newReencryptTestRepo(t)
	migrators := NewMigratorRegistry()
	full := make([]int, ReencryptMaxBatchesPerRun+5)
	for i := range full {
		full[i] = ReencryptBatchSize
	}
	migrator := &fakeReencryptMigrator{batchReturns: full, pending: 999}
	migrators.Register("mfa_totp_secret", migrator)

	migrated, remaining, err := ReencryptTenantField(context.Background(), ReencryptDeps{Repository: repo, Migrators: migrators}, "tenant-a", "mfa_totp_secret")
	if err != nil {
		t.Fatalf("ReencryptTenantField failed: %v", err)
	}
	if migrator.calls != ReencryptMaxBatchesPerRun {
		t.Fatalf("expected exactly %d ReencryptBatch calls, got %d", ReencryptMaxBatchesPerRun, migrator.calls)
	}
	if want := ReencryptMaxBatchesPerRun * ReencryptBatchSize; migrated != want {
		t.Fatalf("migrated = %d, want %d", migrated, want)
	}
	if remaining != 999 {
		t.Fatalf("remaining = %d, want 999", remaining)
	}
}

func TestReencryptTenantField_UnregisteredMigratorReturnsError(t *testing.T) {
	repo := newReencryptTestRepo(t)
	_, _, err := ReencryptTenantField(context.Background(), ReencryptDeps{Repository: repo, Migrators: NewMigratorRegistry()}, "tenant-a", "unregistered")
	if !errors.Is(err, ErrFieldMigratorNotRegistered) {
		t.Fatalf("expected ErrFieldMigratorNotRegistered, got %v", err)
	}
}

func TestReencryptionHandler_ReturnsResultJSONWithoutEnqueueingWhenDone(t *testing.T) {
	repo := newReencryptTestRepo(t)
	migrators := NewMigratorRegistry()
	migrators.Register("mfa_totp_secret", &fakeReencryptMigrator{batchReturns: []int{2}, pending: 0})
	jobRepo := jobsdbmemory.NewJobRepository()

	handler := ReencryptionHandler(ReencryptDeps{Repository: repo, Migrators: migrators, Jobs: jobRepo})
	params, err := json.Marshal(ReencryptParams{TenantID: "tenant-a", Migrator: "mfa_totp_secret"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	result, err := handler(context.Background(), &jobsdomain.Job{TenantID: "tenant-a", Kind: jobsdomain.KindDataKeyReencryption, Params: params})
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}
	var got ReencryptResult
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got.Migrated != 2 || got.Remaining != 0 {
		t.Fatalf("result = %+v, want {2 0}", got)
	}

	jobs, err := jobRepo.ListByTenantAndKinds(context.Background(), "tenant-a", []jobsdomain.JobKind{jobsdomain.KindDataKeyReencryption}, 10)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected no continuation job enqueued when fully migrated, got %d", len(jobs))
	}
}

func TestReencryptionHandler_ReenqueuesContinuationWhenRemaining(t *testing.T) {
	repo := newReencryptTestRepo(t)
	migrators := NewMigratorRegistry()
	migrators.Register("mfa_totp_secret", &fakeReencryptMigrator{batchReturns: []int{2}, pending: 500})
	jobRepo := jobsdbmemory.NewJobRepository()

	handler := ReencryptionHandler(ReencryptDeps{Repository: repo, Migrators: migrators, Jobs: jobRepo})
	params, err := json.Marshal(ReencryptParams{TenantID: "tenant-a", Migrator: "mfa_totp_secret"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	if _, err := handler(context.Background(), &jobsdomain.Job{TenantID: "tenant-a", Kind: jobsdomain.KindDataKeyReencryption, Params: params}); err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	jobs, err := jobRepo.ListByTenantAndKinds(context.Background(), "tenant-a", []jobsdomain.JobKind{jobsdomain.KindDataKeyReencryption}, 10)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 continuation job enqueued, got %d", len(jobs))
	}
	if jobs[0].DedupKey == nil || *jobs[0].DedupKey != ReencryptionDedupKey("tenant-a", "mfa_totp_secret") {
		t.Fatalf("continuation job DedupKey = %v, want %q", jobs[0].DedupKey, ReencryptionDedupKey("tenant-a", "mfa_totp_secret"))
	}
}

func TestEnqueueReencryptionJob_DedupsRepeatedCalls(t *testing.T) {
	jobRepo := jobsdbmemory.NewJobRepository()
	now := time.Now().UTC()
	if err := EnqueueReencryptionJob(context.Background(), jobRepo, "tenant-a", "mfa_totp_secret", now); err != nil {
		t.Fatalf("first EnqueueReencryptionJob failed: %v", err)
	}
	if err := EnqueueReencryptionJob(context.Background(), jobRepo, "tenant-a", "mfa_totp_secret", now); err != nil {
		t.Fatalf("second EnqueueReencryptionJob failed: %v", err)
	}

	jobs, err := jobRepo.ListByTenantAndKinds(context.Background(), "tenant-a", []jobsdomain.JobKind{jobsdomain.KindDataKeyReencryption}, 10)
	if err != nil {
		t.Fatalf("ListByTenantAndKinds: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected dedup to collapse repeated enqueues into 1 job, got %d", len(jobs))
	}
}
