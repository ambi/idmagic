package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/shared/logging"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// queryer is the read-only subset of shared/storage/db_postgres.DB that
// checkRestoreConsistency needs. A *pgxpool.Pool and a pgx.Tx both satisfy
// it, so tests can run the check inside a rolled-back transaction for
// isolation.
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

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

// ephemeralTables are truncated on restore; any row surviving in
// one of them after restore means the truncate step was skipped or failed.
var ephemeralTables = []string{
	"oauth2_authorization_requests",
	"oauth2_authorization_codes",
	"oauth2_par_requests",
	"oauth2_device_codes",
	"oauth2_replay_jtis",
	"oauth2_access_token_denylist",
	"webauthn_sessions",
	"login_throttle_counters",
	"saml_authnrequest_replays",
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

// checkRestoreConsistency inspects a just-restored database directly (raw
// SQL, not the domain usecases) since its entire job is verifying the
// physical restore, not exercising application behavior.
func checkRestoreConsistency(ctx context.Context, db queryer) (restoreConsistencyReport, error) {
	var report restoreConsistencyReport

	if err := db.QueryRow(ctx, "SELECT count(*) FROM tenants").Scan(&report.TenantCount); err != nil {
		return report, fmt.Errorf("count tenants: %w", err)
	}
	if err := db.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&report.UserCount); err != nil {
		return report, fmt.Errorf("count users: %w", err)
	}
	if err := db.QueryRow(ctx, "SELECT count(*) FROM oauth2_clients").Scan(&report.ClientCount); err != nil {
		return report, fmt.Errorf("count oauth2 clients: %w", err)
	}

	tenantsMissingKey, err := db.Query(ctx, `
		SELECT t.id FROM tenants t
		WHERE NOT EXISTS (
			SELECT 1 FROM signing_keys sk
			WHERE sk.tenant_id = t.id AND sk.key_usage = 'Signing' AND sk.active
		)
		ORDER BY t.id
	`)
	if err != nil {
		return report, fmt.Errorf("find tenants missing an active signing key: %w", err)
	}
	defer tenantsMissingKey.Close()
	for tenantsMissingKey.Next() {
		var id string
		if err := tenantsMissingKey.Scan(&id); err != nil {
			return report, fmt.Errorf("scan tenant id: %w", err)
		}
		report.TenantsMissingActiveSigningKey = append(report.TenantsMissingActiveSigningKey, id)
	}
	if err := tenantsMissingKey.Err(); err != nil {
		return report, fmt.Errorf("iterate tenants missing an active signing key: %w", err)
	}

	// The jobs_tenant_dedup_key_active_idx unique index already enforces this
	// invariant during normal operation; this query re-verifies it directly
	// as defense in depth against a restore that lost the index or applied
	// rows out of band.
	duplicateDedupKeys, err := db.Query(ctx, `
		SELECT tenant_id || ':' || dedup_key
		FROM jobs
		WHERE dedup_key IS NOT NULL AND status IN ('queued', 'running')
		GROUP BY tenant_id, dedup_key
		HAVING count(*) > 1
		ORDER BY 1
	`)
	if err != nil {
		return report, fmt.Errorf("find duplicate active job dedup keys: %w", err)
	}
	defer duplicateDedupKeys.Close()
	for duplicateDedupKeys.Next() {
		var key string
		if err := duplicateDedupKeys.Scan(&key); err != nil {
			return report, fmt.Errorf("scan duplicate job dedup key: %w", err)
		}
		report.DuplicateJobDedupKeys = append(report.DuplicateJobDedupKeys, key)
	}
	if err := duplicateDedupKeys.Err(); err != nil {
		return report, fmt.Errorf("iterate duplicate job dedup keys: %w", err)
	}

	for _, table := range ephemeralTables {
		var n int64
		if err := db.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			return report, fmt.Errorf("count ephemeral table %s: %w", table, err)
		}
		if n > 0 {
			report.NonEmptyEphemeralTables = append(report.NonEmptyEphemeralTables, table)
		}
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
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("restore-consistency-check requires DATABASE_URL")
	}

	pool, err := sharedpg.Open(ctx, databaseURL, sharedpg.DBConfig{
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
