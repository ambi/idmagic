package db_postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/datakeys"
	datakeysmemory "github.com/ambi/idmagic/backend/datakeys/db_memory"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5/pgtype"
)

// newTestCipher bootstraps a real DataKeys stack (memory repository + Tink
// cleartext master key, no OpenBao needed) with an active DEK for tenantID,
// mirroring how bootstrap wires backend/datakeys in production.
func newTestCipher(t *testing.T, tenantID string) *datakeys.FieldCipher {
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
	return &datakeys.FieldCipher{Cache: cache, Crypto: crypto}
}

func TestMfaFactorRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	repo := &MfaFactorRepository{Pool: db, Cipher: newTestCipher(t, tenant.ID)}

	now := pgfixtures.TestClock()
	factor := &domain.MfaFactor{
		UserID:    user.ID,
		Type:      spec.MfaFactorTOTP,
		Secret:    new("secret"),
		Label:     new("Authenticator"),
		CreatedAt: now,
	}
	if err := repo.Save(ctx, factor); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Find(ctx, user.ID, spec.MfaFactorTOTP)
	if err != nil || got == nil || got.Secret == nil || *got.Secret != "secret" {
		t.Fatalf("find: %v %+v", err, got)
	}

	// Encryption at rest: the raw row must carry ciphertext, not plaintext,
	// and the legacy plaintext column must be left NULL for new writes.
	row, err := New(db).GetMfaFactor(ctx, GetMfaFactorParams{UserID: user.ID, Type: string(spec.MfaFactorTOTP)})
	if err != nil {
		t.Fatalf("raw get: %v", err)
	}
	if !row.SecretKeyVersion.Valid || len(row.SecretCiphertext) == 0 {
		t.Fatal("expected secret_key_version/secret_ciphertext to be populated")
	}
	if row.Secret.Valid {
		t.Fatal("expected legacy plaintext secret column to be NULL after an encrypted write")
	}
	if string(row.SecretCiphertext) == "secret" {
		t.Fatal("ciphertext at rest must not equal plaintext")
	}

	list, err := repo.ListBySub(ctx, user.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, user.ID, spec.MfaFactorTOTP); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.Find(ctx, user.ID, spec.MfaFactorTOTP)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}

// TestMfaFactorRepositoryDualReadsLegacyPlaintext covers a pre-existing row
// (secret populated, ciphertext columns NULL) written before this migration:
// it must still be readable during the staged rollout (wi-97 T005/T006).
func TestMfaFactorRepositoryDualReadsLegacyPlaintext(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	repo := &MfaFactorRepository{Pool: db, Cipher: newTestCipher(t, tenant.ID)}

	if err := New(db).UpsertMfaFactor(ctx, UpsertMfaFactorParams{
		UserID:    user.ID,
		Type:      string(spec.MfaFactorTOTP),
		Secret:    pgtype.Text{String: "legacy-plaintext", Valid: true},
		CreatedAt: pgfixtures.TestClock(),
	}); err != nil {
		t.Fatalf("seed legacy plaintext row: %v", err)
	}

	got, err := repo.Find(ctx, user.ID, spec.MfaFactorTOTP)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || got.Secret == nil || *got.Secret != "legacy-plaintext" {
		t.Fatalf("expected dual-read of legacy plaintext row, got %+v", got)
	}
}

// TestMfaFactorRepositoryDecryptFailsClosedForWrongTenant ensures a factor
// saved under one tenant cannot be decrypted from a different tenant's
// context: the DEK is tenant-scoped, so cross-tenant confusion fails closed
// rather than silently returning a wrong or garbled secret.
func TestMfaFactorRepositoryDecryptFailsClosedForWrongTenant(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	otherTenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	cipher := newTestCipher(t, tenant.ID)
	ctx := tenancy.WithTenant(context.Background(), tenant, "", "")
	repo := &MfaFactorRepository{Pool: db, Cipher: cipher}

	factor := &domain.MfaFactor{UserID: user.ID, Type: spec.MfaFactorTOTP, Secret: new("secret"), CreatedAt: pgfixtures.TestClock()}
	if err := repo.Save(ctx, factor); err != nil {
		t.Fatalf("save: %v", err)
	}

	wrongCtx := tenancy.WithTenant(context.Background(), otherTenant, "", "")
	if _, err := repo.Find(wrongCtx, user.ID, spec.MfaFactorTOTP); err == nil {
		t.Fatal("expected Find to fail-closed when decrypting under the wrong tenant context")
	}
}
