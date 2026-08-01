package db_postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/datakeys"
	datakeysmemory "github.com/ambi/idmagic/backend/datakeys/db_memory"
	datakeysports "github.com/ambi/idmagic/backend/datakeys/ports"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5/pgtype"
)

// testDataKeyStack bootstraps a real DataKeys stack (memory repository + Tink
// cleartext master key) and exposes every piece a reencryption test needs to
// drive rotation directly, unlike mfa_test.go's newTestCipher which only
// exposes the resulting FieldCipher.
type testDataKeyStack struct {
	repo   datakeysports.DataKeyRepository
	crypto envelope_crypto.EnvelopeCrypto
	cache  *datakeysusecases.DataKeyCache
	cipher *datakeys.FieldCipher
}

func newTestDataKeyStack(t *testing.T, tenantID string) *testDataKeyStack {
	t.Helper()
	repo := datakeysmemory.NewDataKeyRepository()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	cache := datakeysusecases.NewDataKeyCache(repo, crypto)
	deps := datakeysusecases.Deps{Repository: repo, Crypto: crypto, Cache: cache}
	if _, err := datakeysusecases.BootstrapTenantDataKey(context.Background(), deps, tenantID, pgfixtures.TestClock()); err != nil {
		t.Fatalf("bootstrap tenant data key: %v", err)
	}
	return &testDataKeyStack{
		repo: repo, crypto: crypto, cache: cache,
		cipher: &datakeys.FieldCipher{Repository: repo, Cache: cache, Crypto: crypto},
	}
}

func (s *testDataKeyStack) rotate(t *testing.T, tenantID string) {
	t.Helper()
	deps := datakeysusecases.Deps{Repository: s.repo, Crypto: s.crypto, Cache: s.cache}
	if _, err := datakeysusecases.RotateTenantDataKey(context.Background(), deps, tenantID, pgfixtures.TestClock()); err != nil {
		t.Fatalf("rotate tenant data key: %v", err)
	}
}

func (s *testDataKeyStack) activeVersion(t *testing.T, tenantID string) int {
	t.Helper()
	key, err := s.repo.FindActive(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("FindActive: %v", err)
	}
	return key.Version
}

func TestMfaFactorReencryptor_MigratesLegacyPlaintext(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	stack := newTestDataKeyStack(t, tenant.ID)
	repo := &MfaFactorRepository{Pool: db, Cipher: stack.cipher}
	migrator := &MfaFactorReencryptor{Repo: repo}

	if err := New(db).UpsertMfaFactor(context.Background(), UpsertMfaFactorParams{
		UserID: user.ID, Type: string(spec.MfaFactorTOTP),
		Secret: pgtype.Text{String: "legacy-plaintext", Valid: true}, CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatalf("seed legacy plaintext row: %v", err)
	}

	activeVersion := stack.activeVersion(t, tenant.ID)
	migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, activeVersion, 10)
	if err != nil {
		t.Fatalf("ReencryptBatch failed: %v", err)
	}
	if migrated != 1 {
		t.Fatalf("migrated = %d, want 1", migrated)
	}

	pending, err := migrator.PendingCount(context.Background(), tenant.ID, activeVersion)
	if err != nil {
		t.Fatalf("PendingCount failed: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending = %d, want 0 after migration", pending)
	}

	row, err := New(db).GetMfaFactor(context.Background(), GetMfaFactorParams{UserID: user.ID, Type: string(spec.MfaFactorTOTP)})
	if err != nil {
		t.Fatalf("raw get: %v", err)
	}
	if row.Secret.Valid {
		t.Fatal("expected legacy plaintext column to be cleared after migration")
	}
	if !row.SecretKeyVersion.Valid || int(row.SecretKeyVersion.Int32) != activeVersion || len(row.SecretCiphertext) == 0 {
		t.Fatalf("expected row to carry active-version ciphertext, got version=%v ciphertext_len=%d", row.SecretKeyVersion, len(row.SecretCiphertext))
	}

	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	got, err := repo.Find(ctx, user.ID, spec.MfaFactorTOTP)
	if err != nil {
		t.Fatalf("Find after migration failed: %v", err)
	}
	if got == nil || got.Secret == nil || *got.Secret != "legacy-plaintext" {
		t.Fatalf("expected migrated secret to round-trip to original plaintext, got %+v", got)
	}
}

func TestMfaFactorReencryptor_MigratesStaleVersionAfterRotationAndPreservesValue(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	stack := newTestDataKeyStack(t, tenant.ID)
	repo := &MfaFactorRepository{Pool: db, Cipher: stack.cipher}
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")

	if err := repo.Save(ctx, &domain.MfaFactor{
		UserID: user.ID, Type: spec.MfaFactorTOTP, Secret: new("JBSWY3DPEHPK3PXP"), CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	stack.rotate(t, tenant.ID)
	activeVersion := stack.activeVersion(t, tenant.ID)
	if activeVersion != 2 {
		t.Fatalf("activeVersion = %d, want 2", activeVersion)
	}

	migrator := &MfaFactorReencryptor{Repo: repo}
	pendingBefore, err := migrator.PendingCount(context.Background(), tenant.ID, activeVersion)
	if err != nil {
		t.Fatalf("PendingCount failed: %v", err)
	}
	if pendingBefore != 1 {
		t.Fatalf("pendingBefore = %d, want 1 (row still on version 1)", pendingBefore)
	}

	migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, activeVersion, 10)
	if err != nil {
		t.Fatalf("ReencryptBatch failed: %v", err)
	}
	if migrated != 1 {
		t.Fatalf("migrated = %d, want 1", migrated)
	}

	pendingAfter, err := migrator.PendingCount(context.Background(), tenant.ID, activeVersion)
	if err != nil {
		t.Fatalf("PendingCount failed: %v", err)
	}
	if pendingAfter != 0 {
		t.Fatalf("pendingAfter = %d, want 0", pendingAfter)
	}

	got, err := repo.Find(ctx, user.ID, spec.MfaFactorTOTP)
	if err != nil || got == nil || got.Secret == nil || *got.Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("expected re-encrypted secret to round-trip: err=%v got=%+v", err, got)
	}
}

func TestMfaFactorReencryptor_SkipsRowsWithNoSecretMaterial(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	stack := newTestDataKeyStack(t, tenant.ID)
	repo := &MfaFactorRepository{Pool: db, Cipher: stack.cipher}
	migrator := &MfaFactorReencryptor{Repo: repo}

	if err := New(db).UpsertMfaFactor(context.Background(), UpsertMfaFactorParams{
		UserID: user.ID, Type: string(spec.MfaFactorWebAuthn), CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatalf("seed no-secret row: %v", err)
	}

	activeVersion := stack.activeVersion(t, tenant.ID)
	pending, err := migrator.PendingCount(context.Background(), tenant.ID, activeVersion)
	if err != nil {
		t.Fatalf("PendingCount failed: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending = %d, want 0 (no secret material to migrate)", pending)
	}

	migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, activeVersion, 10)
	if err != nil {
		t.Fatalf("ReencryptBatch failed: %v", err)
	}
	if migrated != 0 {
		t.Fatalf("migrated = %d, want 0", migrated)
	}
}

func TestMfaFactorReencryptor_TenantIsolation(t *testing.T) {
	db := pgtest.Require(t)
	tenantA := pgfixtures.SeedTenant(t, db)
	tenantB := pgfixtures.SeedTenant(t, db)
	userA := pgfixtures.SeedUser(t, db, tenantA.ID)
	userB := pgfixtures.SeedUser(t, db, tenantB.ID)
	stackA := newTestDataKeyStack(t, tenantA.ID)
	stackB := newTestDataKeyStack(t, tenantB.ID)

	if err := New(db).UpsertMfaFactor(context.Background(), UpsertMfaFactorParams{
		UserID: userA.ID, Type: string(spec.MfaFactorTOTP), Secret: pgtype.Text{String: "a-secret", Valid: true}, CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatalf("seed tenant A row: %v", err)
	}
	if err := New(db).UpsertMfaFactor(context.Background(), UpsertMfaFactorParams{
		UserID: userB.ID, Type: string(spec.MfaFactorTOTP), Secret: pgtype.Text{String: "b-secret", Valid: true}, CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatalf("seed tenant B row: %v", err)
	}

	migratorA := &MfaFactorReencryptor{Repo: &MfaFactorRepository{Pool: db, Cipher: stackA.cipher}}
	migrated, err := migratorA.ReencryptBatch(context.Background(), tenantA.ID, stackA.activeVersion(t, tenantA.ID), 10)
	if err != nil {
		t.Fatalf("ReencryptBatch failed: %v", err)
	}
	if migrated != 1 {
		t.Fatalf("migrated = %d, want 1 (only tenant A's row)", migrated)
	}

	pendingB, err := migratorA.PendingCount(context.Background(), tenantB.ID, stackB.activeVersion(t, tenantB.ID))
	if err != nil {
		t.Fatalf("PendingCount failed: %v", err)
	}
	if pendingB != 1 {
		t.Fatalf("pendingB = %d, want 1 (tenant B's row must be untouched)", pendingB)
	}
}

// TestMfaFactorReencryptor_NoPlaintextSurvivesBackfillAcrossTenants is the
// wi-97 T008 "DB plaintext scan" check: after backfilling every tenant's
// legacy rows, a raw scan of the entire mfa_factors table — not just the
// rows a single test touched — must find zero populated legacy plaintext
// columns.
func TestMfaFactorReencryptor_NoPlaintextSurvivesBackfillAcrossTenants(t *testing.T) {
	db := pgtest.Require(t)
	// This assertion intentionally scans the whole table. The package shares one
	// embedded PostgreSQL instance across tests, so remove factor rows left by
	// earlier isolation tests before this test establishes its own whole-table
	// backfill fixture.
	if _, err := db.Exec(context.Background(), "DELETE FROM mfa_factors"); err != nil {
		t.Fatalf("isolate plaintext scan fixture: %v", err)
	}
	const tenantCount, usersPerTenant = 3, 2
	for i := range tenantCount {
		tenant := pgfixtures.SeedTenant(t, db)
		stack := newTestDataKeyStack(t, tenant.ID)
		for j := range usersPerTenant {
			user := pgfixtures.SeedUser(t, db, tenant.ID)
			if err := New(db).UpsertMfaFactor(context.Background(), UpsertMfaFactorParams{
				UserID: user.ID, Type: string(spec.MfaFactorTOTP),
				Secret:    pgtype.Text{String: fmt.Sprintf("plaintext-%d-%d", i, j), Valid: true},
				CreatedAt: pgfixtures.TestClock(),
			}); err != nil {
				t.Fatalf("seed legacy plaintext row: %v", err)
			}
		}
		migrator := &MfaFactorReencryptor{Repo: &MfaFactorRepository{Pool: db, Cipher: stack.cipher}}
		migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, stack.activeVersion(t, tenant.ID), 100)
		if err != nil {
			t.Fatalf("ReencryptBatch failed: %v", err)
		}
		if migrated != usersPerTenant {
			t.Fatalf("migrated = %d, want %d", migrated, usersPerTenant)
		}
	}

	var remainingPlaintext int64
	if err := db.QueryRow(context.Background(), "SELECT count(*) FROM mfa_factors WHERE secret IS NOT NULL").Scan(&remainingPlaintext); err != nil {
		t.Fatalf("plaintext scan query failed: %v", err)
	}
	if remainingPlaintext != 0 {
		t.Fatalf("expected 0 rows with legacy plaintext remaining, got %d", remainingPlaintext)
	}
}
