package db_postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/datakeys"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5/pgtype"
)

//spec:covers REQ-DATAKEYS-010: バッチ上限を超える無効化した行を残し、後続の退役した seed だけを移し、除外した行を未移行に数える。
//spec:covers REQ-DATAKEYS-003: 無効化したバージョンの暗号文を復号しない。
func TestMfaFactorReencryptionSkipsDisabledKeys(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := tenantports.WithTenant(context.Background(), tenant, "", "")
	stack := newTestDataKeyStack(t, tenant.ID)
	repo := &MfaFactorRepository{Pool: db, Cipher: stack.cipher}
	var users []string
	for range 203 {
		users = append(users, pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID).ID)
	}
	slices.Sort(users)
	var disabledCiphertexts [][]byte
	for _, userID := range users[:201] {
		if err := repo.Save(ctx, &domain.MfaFactor{
			UserID: userID, Type: spec.MfaFactorTOTP, Secret: new("disabled-seed"), CreatedAt: pgfixtures.TestClock(),
		}); err != nil {
			t.Fatal(err)
		}
		row, err := New(db).GetMfaFactor(ctx, GetMfaFactorParams{UserID: userID, Type: string(spec.MfaFactorTOTP)})
		if err != nil {
			t.Fatal(err)
		}
		disabledCiphertexts = append(disabledCiphertexts, row.SecretCiphertext)
	}
	stack.rotate(t, tenant.ID)
	if _, err := stack.module.Repository.Disable(ctx, tenant.ID, 1, pgfixtures.TestClock()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, &domain.MfaFactor{
		UserID: users[201], Type: spec.MfaFactorTOTP, Secret: new("retiring-seed"), CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatal(err)
	}
	stack.rotate(t, tenant.ID)
	if err := repo.Save(ctx, &domain.MfaFactor{
		UserID: users[202], Type: spec.MfaFactorTOTP, Secret: new("active-seed"), CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatal(err)
	}
	stack.module.Migrators.Register(MfaFactorMigratorName, &MfaFactorReencryptor{Repo: repo})
	params, err := json.Marshal(map[string]string{"tenant_id": tenant.ID, "migrator": MfaFactorMigratorName})
	if err != nil {
		t.Fatal(err)
	}
	job := &jobsdomain.Job{TenantID: tenant.ID, Params: params}
	for _, wantMigrated := range []int{1, 0} {
		result, err := stack.module.ReencryptionHandler(nil)(ctx, job)
		if err != nil {
			t.Fatal(err)
		}
		var counts struct{ Migrated, Remaining int }
		if err := json.Unmarshal(result, &counts); err != nil {
			t.Fatal(err)
		}
		if counts.Migrated != wantMigrated || counts.Remaining != 201 {
			t.Fatalf("result=%s, want migrated=%d remaining=201", result, wantMigrated)
		}
	}
	for i, userID := range users[:201] {
		row, err := New(db).GetMfaFactor(ctx, GetMfaFactorParams{UserID: userID, Type: string(spec.MfaFactorTOTP)})
		if err != nil {
			t.Fatal(err)
		}
		if row.SecretKeyVersion.Int32 != 1 || !bytes.Equal(row.SecretCiphertext, disabledCiphertexts[i]) {
			t.Fatal("無効化した行が変更された")
		}
	}
	for i, want := range []string{"retiring-seed", "active-seed"} {
		factor, err := repo.Find(ctx, users[201+i], spec.MfaFactorTOTP)
		if err != nil || factor == nil || factor.Secret == nil || *factor.Secret != want {
			t.Fatalf("seed=%+v err=%v, want %q", factor, err, want)
		}
		row, err := New(db).GetMfaFactor(ctx, GetMfaFactorParams{UserID: users[201+i], Type: string(spec.MfaFactorTOTP)})
		if err != nil || row.SecretKeyVersion.Int32 != 3 {
			t.Fatalf("key version=%v err=%v, want 3", row, err)
		}
	}
}

//spec:covers REQ-DATAKEYS-010: 退役した鍵の暗号文が破損している場合は無視せずジョブを失敗させ、その行を未移行に残す。
func TestMfaFactorReencryptionRejectsCorruptedCiphertext(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	ctx := tenantports.WithTenant(context.Background(), tenant, "", "")
	stack := newTestDataKeyStack(t, tenant.ID)
	repo := &MfaFactorRepository{Pool: db, Cipher: stack.cipher}
	if err := repo.Save(ctx, &domain.MfaFactor{
		UserID: user.ID, Type: spec.MfaFactorTOTP, Secret: new("seed"), CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatal(err)
	}
	stack.rotate(t, tenant.ID)
	if _, err := db.Exec(ctx, `UPDATE mfa_factors SET secret_ciphertext='\x00'::bytea WHERE user_id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	migrator := &MfaFactorReencryptor{Repo: repo}
	stack.module.Migrators.Register(MfaFactorMigratorName, migrator)
	params, err := json.Marshal(map[string]string{"tenant_id": tenant.ID, "migrator": MfaFactorMigratorName})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stack.module.ReencryptionHandler(nil)(ctx, &jobsdomain.Job{TenantID: tenant.ID, Params: params}); err == nil {
		t.Fatal("破損した暗号文でジョブが成功した")
	}
	row, err := New(db).GetMfaFactor(ctx, GetMfaFactorParams{UserID: user.ID, Type: string(spec.MfaFactorTOTP)})
	if err != nil || row.SecretKeyVersion.Int32 != 1 || !bytes.Equal(row.SecretCiphertext, []byte{0}) {
		t.Fatalf("row=%+v err=%v, want unchanged version 1", row, err)
	}
	if pending, err := migrator.PendingCount(ctx, tenant.ID, 2); err != nil || pending != 1 {
		t.Fatalf("pending=%d err=%v, want 1", pending, err)
	}
}

// testDataKeyStack bootstraps a real DataKeys stack (memory repository + Tink
// cleartext master key) and exposes every piece a reencryption test needs to
// drive rotation directly, unlike mfa_test.go's newTestCipher which only
// exposes the resulting FieldCipher.
type testDataKeyStack struct {
	module datakeys.Module
	cipher *datakeys.FieldCipher
}

func newTestDataKeyStack(t *testing.T, tenantID string) *testDataKeyStack {
	t.Helper()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatalf("NewCleartextMasterKeyProvider failed: %v", err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	module := datakeys.NewMemoryModule(crypto)
	if _, err := module.BootstrapTenantDataKey(context.Background(), tenantID, pgfixtures.TestClock()); err != nil {
		t.Fatalf("bootstrap tenant data key: %v", err)
	}
	return &testDataKeyStack{
		module: module, cipher: module.FieldCipher(),
	}
}

func (s *testDataKeyStack) rotate(t *testing.T, tenantID string) {
	t.Helper()
	if _, err := s.module.RotateTenantDataKey(context.Background(), tenantID, pgfixtures.TestClock()); err != nil {
		t.Fatalf("rotate tenant data key: %v", err)
	}
}

func (s *testDataKeyStack) activeVersion(t *testing.T, tenantID string) int {
	t.Helper()
	key, err := s.module.Repository.FindActive(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("FindActive: %v", err)
	}
	return key.Version
}

func TestMfaFactorReencryptor_MigratesLegacyPlaintext(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
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
	migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, activeVersion, 10, nil)
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

	ctx := tenantports.WithTenant(context.Background(), tenant, "", "")
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
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	stack := newTestDataKeyStack(t, tenant.ID)
	repo := &MfaFactorRepository{Pool: db, Cipher: stack.cipher}
	ctx := tenantports.WithTenant(context.Background(), tenant, "", "")

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

	migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, activeVersion, 10, nil)
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
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
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

	migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, activeVersion, 10, nil)
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
	userA := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenantA.ID)
	userB := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenantB.ID)
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
	migrated, err := migratorA.ReencryptBatch(context.Background(), tenantA.ID, stackA.activeVersion(t, tenantA.ID), 10, nil)
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
			user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
			if err := New(db).UpsertMfaFactor(context.Background(), UpsertMfaFactorParams{
				UserID: user.ID, Type: string(spec.MfaFactorTOTP),
				Secret:    pgtype.Text{String: fmt.Sprintf("plaintext-%d-%d", i, j), Valid: true},
				CreatedAt: pgfixtures.TestClock(),
			}); err != nil {
				t.Fatalf("seed legacy plaintext row: %v", err)
			}
		}
		migrator := &MfaFactorReencryptor{Repo: &MfaFactorRepository{Pool: db, Cipher: stack.cipher}}
		migrated, err := migrator.ReencryptBatch(context.Background(), tenant.ID, stack.activeVersion(t, tenant.ID), 100, nil)
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

// racingSecretCipher は再暗号化の最中に一度だけ onEncrypt を呼ぶ。読んだ後の変更を、
// タイミングに依らず同じ順序で起こす。
type racingSecretCipher struct {
	*datakeys.FieldCipher
	onEncrypt func()
}

func (c *racingSecretCipher) Encrypt(ctx context.Context, tenantID, recordContext, table, recordID, field, plaintext string) (int, []byte, error) {
	if hook := c.onEncrypt; hook != nil {
		c.onEncrypt = nil
		hook()
	}
	return c.FieldCipher.Encrypt(ctx, tenantID, recordContext, table, recordID, field, plaintext)
}

//spec:covers REQ-DATAKEYS-002: 再暗号化が読んだ後に登録し直された TOTP のシードを、古いシードで書き戻さない。
func TestMfaFactorReencryptorKeepsSeedEnrolledDuringBatch(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	stack := newTestDataKeyStack(t, tenant.ID)
	ctx := context.Background()
	if err := New(db).UpsertMfaFactor(ctx, UpsertMfaFactorParams{
		UserID: user.ID, Type: string(spec.MfaFactorTOTP),
		Secret: pgtype.Text{String: "old-seed", Valid: true}, CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cipher := &racingSecretCipher{FieldCipher: stack.cipher}
	repo := &MfaFactorRepository{Pool: db, Cipher: cipher}
	cipher.onEncrypt = func() {
		if err := New(db).UpsertMfaFactor(ctx, UpsertMfaFactorParams{
			UserID: user.ID, Type: string(spec.MfaFactorTOTP),
			Secret: pgtype.Text{String: "new-seed", Valid: true}, CreatedAt: pgfixtures.TestClock(),
		}); err != nil {
			t.Errorf("re-enroll: %v", err)
		}
	}
	migrated, err := (&MfaFactorReencryptor{Repo: repo}).ReencryptBatch(ctx, tenant.ID, stack.activeVersion(t, tenant.ID), 10, nil)
	if err != nil {
		t.Fatalf("ReencryptBatch: %v", err)
	}
	if migrated != 0 {
		t.Errorf("migrated = %d, want 0 for the row that changed", migrated)
	}
	var secret pgtype.Text
	if err := db.QueryRow(ctx, `SELECT secret FROM mfa_factors WHERE user_id = $1`, user.ID).Scan(&secret); err != nil {
		t.Fatal(err)
	}
	if secret.String != "new-seed" {
		t.Fatalf("stored seed = %q, want the seed enrolled during the batch", secret.String)
	}
}
