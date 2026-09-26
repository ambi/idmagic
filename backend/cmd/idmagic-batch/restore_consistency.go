package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	restoredb "github.com/ambi/idmagic/backend/cmd/idmagic-batch/internal/restorecheck/db_postgres"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/shared/logging"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// restoreConsistencyReport is the result of a post-restore consistency check
// (wi-101): it verifies that a PostgreSQL restore did not leave the
// tenant/user/client baseline empty, signing keys resolvable, the jobs queue
// free of dedup_key violations, and ephemeral tables truncated as intended.
type restoreConsistencyReport struct {
	TenantCount                    int64
	UserCount                      int64
	ClientCount                    int64
	TenantsMissingActiveSigningKey []string
	DuplicateJobDedupKeys          []string
	NonEmptyEphemeralTables        []string
}

func (r restoreConsistencyReport) Errors() []string {
	var errs []string
	if r.TenantCount == 0 {
		errs = append(errs, "no tenants found")
	}
	if r.UserCount == 0 {
		errs = append(errs, "no users found")
	}
	if r.ClientCount == 0 {
		errs = append(errs, "no oauth2 clients found")
	}
	for _, id := range r.TenantsMissingActiveSigningKey {
		errs = append(errs, fmt.Sprintf("tenant %s has no active signing key", id))
	}
	for _, key := range r.DuplicateJobDedupKeys {
		errs = append(errs, fmt.Sprintf("duplicate active job dedup_key %s", key))
	}
	for _, table := range r.NonEmptyEphemeralTables {
		errs = append(errs, fmt.Sprintf("ephemeral table %s is not empty after restore", table))
	}
	return errs
}

// checkRestoreConsistency は復元直後のデータベースをユースケースを通さずに直接調べる。
// 確かめたいのは物理的な復元の結果であり、アプリケーションの振る舞いではないためである。
func checkRestoreConsistency(ctx context.Context, db restoredb.DBTX) (restoreConsistencyReport, error) {
	var report restoreConsistencyReport
	queries := restoredb.New(db)

	baseline, err := queries.CountRestoreBaseline(ctx)
	if err != nil {
		return report, fmt.Errorf("count baseline rows: %w", err)
	}
	report.TenantCount, report.UserCount, report.ClientCount = baseline.TenantCount, baseline.UserCount, baseline.ClientCount

	if report.TenantsMissingActiveSigningKey, err = queries.ListTenantsMissingActiveSigningKey(ctx); err != nil {
		return report, fmt.Errorf("find tenants missing an active signing key: %w", err)
	}
	if report.DuplicateJobDedupKeys, err = queries.ListDuplicateActiveJobDedupKeys(ctx); err != nil {
		return report, fmt.Errorf("find duplicate active job dedup keys: %w", err)
	}
	if report.NonEmptyEphemeralTables, err = queries.ListNonEmptyEphemeralTables(ctx); err != nil {
		return report, fmt.Errorf("find non-empty ephemeral tables: %w", err)
	}
	return report, nil
}

// runRestoreConsistencyCheck is the operator-facing entry point
// (`idmagic-batch restore-consistency-check`, wi-101). It opens its
// own short-lived connection pool rather than the full bootstrap.Dependencies
// graph: a post-restore check only needs to read PostgreSQL directly and
// must not depend on unrelated env config (WEBAUTHN_RP_ID etc.) that a
// restore drill environment may not set.
func runRestoreConsistencyCheck(ctx context.Context) error {
	loader := bootstrap.NewConfigLoader(os.Getenv)
	databaseURL := loader.RequiredSecret("DATABASE_URL")
	if err := loader.Err(); err != nil {
		return fmt.Errorf("load startup configuration: %w", err)
	}

	pool, err := sharedpg.Open(ctx, databaseURL.Value(), sharedpg.DBConfig{
		MaxConns:        4,
		MinConns:        1,
		MaxConnIdleTime: 30 * time.Second,
		MaxConnLifetime: 1 * time.Hour,
		ConnectTimeout:  5 * time.Second,
		QueryTimeout:    10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	report, err := checkRestoreConsistency(ctx, pool)
	if err != nil {
		return fmt.Errorf("check restore consistency: %w", err)
	}

	logging.Info(ctx, "restore consistency check",
		"tenant_count", report.TenantCount,
		"user_count", report.UserCount,
		"client_count", report.ClientCount,
	)

	if errs := report.Errors(); len(errs) > 0 {
		return fmt.Errorf("restore consistency check failed:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}
