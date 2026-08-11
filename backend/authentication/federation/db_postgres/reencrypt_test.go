package db_postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/federation/db_postgres"
	"github.com/ambi/idmagic/backend/datakeys"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

type stubEnvResolver struct {
	values map[string]string
}

func (s stubEnvResolver) Resolve(_ context.Context, reference string) (string, error) {
	value, ok := s.values[reference]
	if !ok {
		return "", errors.New("referenced environment secret is unavailable")
	}
	return value, nil
}

func activeVersion(t *testing.T, cipher *datakeys.FieldCipher, tenantID string) int {
	t.Helper()
	version, _, err := cipher.Cache.GetActive(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	return version
}

func TestConnectionSecretReencryptorMigratesResolvableLegacyReference(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	cipher := newTestCipher(t, tenant.ID)
	connections := &db_postgres.ConnectionRepository{Pool: pool}
	connection := testConnection(t, tenant.ID, now)
	connection.ID = "resolvable"
	connection.SecretReference = "env:OIDC_SECRET"
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	cipheredConnections := &db_postgres.ConnectionRepository{Pool: pool, Cipher: cipher}
	migrator := &db_postgres.ConnectionSecretReencryptor{
		Repo:        cipheredConnections,
		EnvResolver: stubEnvResolver{values: map[string]string{"env:OIDC_SECRET": "resolved-secret-value"}},
	}

	version := activeVersion(t, cipher, tenant.ID)
	pendingBefore, err := migrator.PendingCount(ctx, tenant.ID, version)
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if pendingBefore != 1 {
		t.Fatalf("pendingBefore=%d, want 1", pendingBefore)
	}

	migrated, err := migrator.ReencryptBatch(ctx, tenant.ID, version, 10)
	if err != nil {
		t.Fatalf("ReencryptBatch: %v", err)
	}
	if migrated != 1 {
		t.Fatalf("migrated=%d, want 1", migrated)
	}

	pendingAfter, err := migrator.PendingCount(ctx, tenant.ID, version)
	if err != nil {
		t.Fatalf("PendingCount after: %v", err)
	}
	if pendingAfter != 0 {
		t.Fatalf("pendingAfter=%d, want 0", pendingAfter)
	}

	found, err := cipheredConnections.Find(ctx, tenant.ID, connection.ID)
	if err != nil || found == nil || found.SecretReference != "resolved-secret-value" {
		t.Fatalf("Find after migration=(%+v,%v), want resolved plaintext", found, err)
	}

	row, err := db_postgres.New(pool).FindIdentityProviderConnection(ctx, db_postgres.FindIdentityProviderConnectionParams{
		TenantID: tenant.ID, ID: connection.ID,
	})
	if err != nil {
		t.Fatalf("raw find: %v", err)
	}
	if row.SecretReference != "" {
		t.Fatal("expected legacy plaintext secret_reference column to be cleared after migration")
	}
	if len(row.SecretCiphertext) == 0 {
		t.Fatal("expected secret_ciphertext to be populated after migration")
	}
}

// RED: a legacy reference whose environment variable is gone cannot be resolved
// automatically; ReencryptBatch must skip it (not fail the batch) and leave it pending.
func TestConnectionSecretReencryptorSkipsUnresolvableLegacyReference(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	cipher := newTestCipher(t, tenant.ID)
	connections := &db_postgres.ConnectionRepository{Pool: pool}
	connection := testConnection(t, tenant.ID, now)
	connection.ID = "unresolvable"
	connection.SecretReference = "env:MISSING_SECRET"
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	migrator := &db_postgres.ConnectionSecretReencryptor{
		Repo:        &db_postgres.ConnectionRepository{Pool: pool, Cipher: cipher},
		EnvResolver: stubEnvResolver{values: map[string]string{}},
	}
	version := activeVersion(t, cipher, tenant.ID)

	migrated, err := migrator.ReencryptBatch(ctx, tenant.ID, version, 10)
	if err != nil {
		t.Fatalf("ReencryptBatch: %v", err)
	}
	if migrated != 0 {
		t.Fatalf("migrated=%d, want 0", migrated)
	}
	pending, err := migrator.PendingCount(ctx, tenant.ID, version)
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if pending != 1 {
		t.Fatalf("pending=%d, want 1 (left for a manual re-entry)", pending)
	}
}

func TestConnectionSecretReencryptorSkipsRowsWithNoSecretConfigured(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	cipher := newTestCipher(t, tenant.ID)
	connections := &db_postgres.ConnectionRepository{Pool: pool}
	connection := testConnection(t, tenant.ID, now)
	connection.ID = "no-secret"
	connection.SecretReference = ""
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("seed Save: %v", err)
	}

	migrator := &db_postgres.ConnectionSecretReencryptor{Repo: &db_postgres.ConnectionRepository{Pool: pool, Cipher: cipher}}
	version := activeVersion(t, cipher, tenant.ID)
	pending, err := migrator.PendingCount(ctx, tenant.ID, version)
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending=%d, want 0 (nothing to migrate)", pending)
	}
}
