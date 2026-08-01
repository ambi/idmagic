package main

import (
	"context"
	"os"
	"testing"
	"time"

	fixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

// resetTables truncates every table reachable from tenants (CASCADE), giving
// each test a clean slate on the shared embedded-postgres instance.
func resetTables(tb testing.TB) {
	tb.Helper()
	db := pgtest.Require(tb)
	if _, err := db.Exec(context.Background(), "TRUNCATE tenants CASCADE"); err != nil {
		tb.Fatalf("truncate tenants: %v", err)
	}
}

func seedActiveSigningKey(tb testing.TB, tenantID string) {
	tb.Helper()
	db := pgtest.Require(tb)
	_, err := db.Exec(context.Background(), `
		INSERT INTO signing_keys (kid, tenant_id, alg, provider, key_usage, scope_id, public_jwk, private_jwk, active, created_at, updated_at)
		VALUES ($1, $2, $3, 'Postgres', 'Signing', 'default', '{}', '{}', true, $4, $4)
	`, fixtures.UniqueID("kid"), tenantID, signingdomain.SigAlgPS256, pgtest.Now())
	if err != nil {
		tb.Fatalf("seed signing key: %v", err)
	}
}

func TestCheckRestoreConsistency_healthyDatabaseHasNoErrors(t *testing.T) {
	resetTables(t)
	db := pgtest.Require(t)
	tenant := fixtures.SeedTenant(t, db)
	fixtures.SeedUser(t, db, tenant.ID)
	fixtures.SeedClient(t, db, tenant.ID)
	seedActiveSigningKey(t, tenant.ID)

	report, err := checkRestoreConsistency(context.Background(), db)
	if err != nil {
		t.Fatalf("checkRestoreConsistency: %v", err)
	}
	if errs := report.Errors(); len(errs) != 0 {
		t.Fatalf("expected a healthy restore to report no errors, got %v", errs)
	}
}

func TestCheckRestoreConsistency_emptyDatabaseReportsMissingBaseline(t *testing.T) {
	resetTables(t)
	db := pgtest.Require(t)

	report, err := checkRestoreConsistency(context.Background(), db)
	if err != nil {
		t.Fatalf("checkRestoreConsistency: %v", err)
	}
	errs := report.Errors()
	if len(errs) == 0 {
		t.Fatal("expected an empty database to report errors, got none")
	}
	if report.TenantCount != 0 || report.UserCount != 0 || report.ClientCount != 0 {
		t.Fatalf("expected zero counts on an empty database, got %+v", report)
	}
}

func TestCheckRestoreConsistency_tenantMissingActiveSigningKeyIsReported(t *testing.T) {
	resetTables(t)
	db := pgtest.Require(t)
	tenant := fixtures.SeedTenant(t, db)
	fixtures.SeedUser(t, db, tenant.ID)
	fixtures.SeedClient(t, db, tenant.ID)
	// deliberately no signing key seeded for this tenant

	report, err := checkRestoreConsistency(context.Background(), db)
	if err != nil {
		t.Fatalf("checkRestoreConsistency: %v", err)
	}
	if len(report.TenantsMissingActiveSigningKey) != 1 || report.TenantsMissingActiveSigningKey[0] != tenant.ID {
		t.Fatalf("expected tenant %s to be reported as missing an active signing key, got %v", tenant.ID, report.TenantsMissingActiveSigningKey)
	}
}

func TestCheckRestoreConsistency_nonEmptyEphemeralTableIsReported(t *testing.T) {
	resetTables(t)
	db := pgtest.Require(t)
	tenant := fixtures.SeedTenant(t, db)
	fixtures.SeedUser(t, db, tenant.ID)
	fixtures.SeedClient(t, db, tenant.ID)
	seedActiveSigningKey(t, tenant.ID)

	expiresAt := pgtest.Now().Add(1 * time.Minute)
	_, err := db.Exec(context.Background(), `
		INSERT INTO oauth2_authorization_requests (id, tenant_id, expires_at, payload)
		VALUES ($1, $2, $3, '{}')
	`, fixtures.NewUUID(t), tenant.ID, expiresAt)
	if err != nil {
		t.Fatalf("seed stray ephemeral row: %v", err)
	}

	report, err := checkRestoreConsistency(context.Background(), db)
	if err != nil {
		t.Fatalf("checkRestoreConsistency: %v", err)
	}
	found := false
	for _, table := range report.NonEmptyEphemeralTables {
		if table == "oauth2_authorization_requests" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected oauth2_authorization_requests to be reported as non-empty, got %v", report.NonEmptyEphemeralTables)
	}
}
