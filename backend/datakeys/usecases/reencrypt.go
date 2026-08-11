package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// ReencryptBatchSize is how many rows ReencryptTenantField asks a
// FieldMigrator to migrate per ReencryptBatch call.
const ReencryptBatchSize = 200

// ReencryptMaxBatchesPerRun caps how many batches a single
// data_key_reencryption Job execution drives before yielding its worker slot
// back (lane fairness): remaining work is picked up by a re-enqueued
// continuation Job (see ReencryptionHandler) rather than one Job
// monopolizing a bulk worker forever.
const ReencryptMaxBatchesPerRun = 25

// ErrFieldMigratorNotRegistered is returned when a data_key_reencryption Job
// names a migrator that was never registered — a worker running a stale
// binary that predates the consumer context's registration.
var ErrFieldMigratorNotRegistered = errors.New("datakeys: no field migrator registered for name")

// ReencryptDeps are the dependencies for ReencryptTenantField and
// ReencryptionHandler (wi-97 T006).
type ReencryptDeps struct {
	Repository ports.DataKeyRepository
	Migrators  *MigratorRegistry
	// Jobs enqueues the continuation Job when a run hits
	// ReencryptMaxBatchesPerRun with rows still pending. nil skips
	// re-enqueueing (wiring gaps in tests/tools); production bootstrap
	// always sets it.
	Jobs jobsports.JobRepository
	// Now returns the current time for the continuation Job's enqueue;
	// defaults to time.Now().UTC() when nil.
	Now func() time.Time
}

func (d ReencryptDeps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now().UTC()
}

// ReencryptTenantField drives migratorName through up to
// ReencryptMaxBatchesPerRun batches of tenantID's pending rows, re-encrypting
// each onto the tenant's current active DataEncryptionKey version. It
// returns how many rows it migrated this call and how many remain pending
// (0 means fully migrated: DestroyTenantDataKey's gate can pass for
// migratorName).
func ReencryptTenantField(ctx context.Context, deps ReencryptDeps, tenantID, migratorName string) (migrated, remaining int, err error) {
	migrator, ok := deps.Migrators.Lookup(migratorName)
	if !ok {
		return 0, 0, fmt.Errorf("%w: %q", ErrFieldMigratorNotRegistered, migratorName)
	}
	active, err := deps.Repository.FindActive(ctx, tenantID)
	if err != nil {
		return 0, 0, err
	}
	for range ReencryptMaxBatchesPerRun {
		n, batchErr := migrator.ReencryptBatch(ctx, tenantID, active.Version, ReencryptBatchSize)
		if batchErr != nil {
			return migrated, 0, batchErr
		}
		migrated += n
		if n < ReencryptBatchSize {
			break
		}
	}
	remaining, err = migrator.PendingCount(ctx, tenantID, active.Version)
	if err != nil {
		return migrated, 0, err
	}
	return migrated, remaining, nil
}

// ReencryptionDedupKey is the JobHandlerIdempotency dedup key shared by both
// the Rotate-triggered enqueue and the Handler's continuation re-enqueue, so
// a (tenant, migrator) pair never has more than one outstanding
// data_key_reencryption Job at a time.
func ReencryptionDedupKey(tenantID, migratorName string) string {
	return "data_key_reencryption:" + tenantID + ":" + migratorName
}

// ReencryptParams is the data_key_reencryption Job's params payload.
type ReencryptParams struct {
	TenantID string `json:"tenant_id"`
	Migrator string `json:"migrator"`
}

// ReencryptResult is the data_key_reencryption Job's result payload.
type ReencryptResult struct {
	Migrated  int `json:"migrated"`
	Remaining int `json:"remaining"`
}

// EnqueueReencryptionJob enqueues the data_key_reencryption Job for
// tenantID/migratorName, or — via JobHandlerIdempotency dedup — reuses the
// already-outstanding one.
func EnqueueReencryptionJob(ctx context.Context, repo jobsports.JobRepository, tenantID, migratorName string, now time.Time) error {
	params, err := json.Marshal(ReencryptParams{TenantID: tenantID, Migrator: migratorName})
	if err != nil {
		return err
	}
	dedupKey := ReencryptionDedupKey(tenantID, migratorName)
	_, err = jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: repo}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindDataKeyReencryption, Params: params, DedupKey: &dedupKey,
	}, now)
	return err
}

// ReencryptionHandler is the jobs.Handler for
// jobsdomain.KindDataKeyReencryption (wi-97 T006). It is idempotent
// (JobHandlerIdempotency): a row already on the active version is simply not
// reselected by the underlying FieldMigrator. When rows remain after
// ReencryptMaxBatchesPerRun batches, it re-enqueues a dedup'd continuation
// Job instead of looping forever inside one worker slot.
func ReencryptionHandler(deps ReencryptDeps) func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		var p ReencryptParams
		if err := json.Unmarshal(job.Params, &p); err != nil {
			return nil, err
		}
		migrated, remaining, err := ReencryptTenantField(ctx, deps, p.TenantID, p.Migrator)
		if err != nil {
			return nil, err
		}
		if remaining > 0 && deps.Jobs != nil {
			if enqErr := EnqueueReencryptionJob(ctx, deps.Jobs, p.TenantID, p.Migrator, deps.now()); enqErr != nil {
				logging.Warn(ctx, "datakeys: failed to re-enqueue continuation reencryption job", "error", enqErr, "tenant_id", p.TenantID, "migrator", p.Migrator)
			}
		}
		return json.Marshal(ReencryptResult{Migrated: migrated, Remaining: remaining})
	}
}
