package db_postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/db_postgres"
	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
	"github.com/ambi/idmagic/backend/datakeys"
	datakeysmemory "github.com/ambi/idmagic/backend/datakeys/db_memory"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userpg "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

// newTestCipher bootstraps a real DataKeys stack (memory repository + Tink cleartext master
// key, no OpenBao needed) with an active DEK for tenantID, mirroring
// totp/db_postgres/mfa_test.go's identically named helper.
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

// RED (interface: UpdateIdentityProviderConnection, ADR-150): a real (non "env:") secret
// value is encrypted at rest, never written to the legacy plaintext column.
func TestConnectionRepositoryEncryptsRealSecretAtRest(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	connections := &db_postgres.ConnectionRepository{Pool: pool, Cipher: newTestCipher(t, tenant.ID)}
	connection := testConnection(t, tenant.ID, now)
	connection.SecretReference = "s3cr3t-client-secret"
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := connections.Find(ctx, tenant.ID, connection.ID)
	if err != nil || found == nil || found.SecretReference != "s3cr3t-client-secret" {
		t.Fatalf("Find=(%+v,%v), want round-tripped plaintext", found, err)
	}

	row, err := db_postgres.New(pool).FindIdentityProviderConnection(ctx, db_postgres.FindIdentityProviderConnectionParams{
		TenantID: tenant.ID, ID: connection.ID,
	})
	if err != nil {
		t.Fatalf("raw find: %v", err)
	}
	if row.SecretReference != "" {
		t.Fatal("expected legacy plaintext secret_reference column to be empty after an encrypted write")
	}
	if len(row.SecretCiphertext) == 0 {
		t.Fatal("expected secret_ciphertext to be populated")
	}
	if string(row.SecretCiphertext) == "s3cr3t-client-secret" {
		t.Fatal("ciphertext at rest must not equal plaintext")
	}
}

// RED: a legacy "env:" reference is left untouched (dual-read/dual-write, unchanged
// behavior) even when a Cipher is configured — only real secret values are encrypted.
func TestConnectionRepositoryDualReadsLegacyEnvReference(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	connections := &db_postgres.ConnectionRepository{Pool: pool, Cipher: newTestCipher(t, tenant.ID)}
	connection := testConnection(t, tenant.ID, now)
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := connections.Find(ctx, tenant.ID, connection.ID)
	if err != nil || found == nil || found.SecretReference != "env:OIDC_SECRET" {
		t.Fatalf("Find=(%+v,%v), want unchanged env: reference", found, err)
	}
	row, err := db_postgres.New(pool).FindIdentityProviderConnection(ctx, db_postgres.FindIdentityProviderConnectionParams{
		TenantID: tenant.ID, ID: connection.ID,
	})
	if err != nil {
		t.Fatalf("raw find: %v", err)
	}
	if len(row.SecretCiphertext) != 0 {
		t.Fatal("expected no ciphertext for a legacy env: reference")
	}
}

// RED: re-saving a connection whose secret is an unresolved legacy env: reference —
// without the caller touching the secret field — must not corrupt it into ciphertext.
// This guards the "keep existing secret unchanged" copy-forward pattern used by
// handlers_http.updateAdmin: Find returns the raw reference string, and re-Save must
// recognize it as still legacy, not encrypt the literal "env:..." text.
func TestConnectionRepositoryPreservesLegacyReferenceAcrossUnrelatedSave(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	connections := &db_postgres.ConnectionRepository{Pool: pool, Cipher: newTestCipher(t, tenant.ID)}
	connection := testConnection(t, tenant.ID, now)
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := connections.Find(ctx, tenant.ID, connection.ID)
	if err != nil || found == nil {
		t.Fatalf("Find=(%+v,%v)", found, err)
	}
	found.DisplayName = "Renamed"
	if err := connections.Save(ctx, found); err != nil {
		t.Fatalf("unrelated Save: %v", err)
	}

	again, err := connections.Find(ctx, tenant.ID, connection.ID)
	if err != nil || again == nil || again.SecretReference != "env:OIDC_SECRET" {
		t.Fatalf("Find after unrelated save=(%+v,%v), want unchanged env: reference", again, err)
	}
}

func TestConnectionAndIdentityRepositoriesRoundTrip(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	connections := &db_postgres.ConnectionRepository{Pool: pool}
	identities := &db_postgres.IdentityRepository{Pool: pool}
	connection := testConnection(t, tenant.ID, now)
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("Save connection: %v", err)
	}
	if found, err := connections.Find(ctx, tenant.ID, connection.ID); err != nil || found == nil || found.SecretReference != connection.SecretReference {
		t.Fatalf("Find connection=(%+v,%v)", found, err)
	}

	userID := pgfixtures.NewUUID(t)
	user := &userdomain.User{
		ID: userID, TenantID: tenant.ID, PreferredUsername: "federated-user",
		PasswordHash: "!", Roles: []string{},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := (&userpg.UserRepository{Pool: pool}).Save(ctx, user); err != nil {
		t.Fatalf("Save user: %v", err)
	}
	link := &domain.FederatedIdentity{
		TenantID: tenant.ID, ProviderID: connection.ID, ExternalSubject: "external",
		LocalUserID: userID, LinkedAt: now,
	}
	if err := identities.Create(ctx, link); err != nil {
		t.Fatalf("Create link: %v", err)
	}
	conflict := *link
	conflict.ExternalSubject = "other"
	if err := identities.Create(ctx, &conflict); !errors.Is(err, federationports.ErrLinkConflict) {
		t.Fatalf("conflict err=%v", err)
	}
	if found, err := identities.FindBySubject(ctx, tenant.ID, connection.ID, "external"); err != nil || found == nil || found.LocalUserID != userID {
		t.Fatalf("FindBySubject=(%+v,%v)", found, err)
	}
}

func TestAttemptStoreConsumesOnceAndScopesTenant(t *testing.T) {
	pool := pgtest.Require(t)
	tenantA, tenantB := pgfixtures.SeedTenant(t, pool), pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), time.Now().UTC()
	connections := &db_postgres.ConnectionRepository{Pool: pool}
	connection := testConnection(t, tenantA.ID, now)
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatal(err)
	}
	attempts := &db_postgres.AttemptStore{Pool: pool}
	attempt := &domain.FederatedLoginAttempt{
		State: "state", TenantID: tenantA.ID, ProviderID: connection.ID, Protocol: domain.ProtocolOIDC,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := attempts.Save(ctx, attempt); err != nil {
		t.Fatal(err)
	}
	if _, err := attempts.Consume(ctx, tenantB.ID, attempt.State, now); !errors.Is(err, federationports.ErrAttemptNotFound) {
		t.Fatalf("cross tenant err=%v", err)
	}
	if _, err := attempts.Consume(ctx, tenantA.ID, attempt.State, now); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if _, err := attempts.Consume(ctx, tenantA.ID, attempt.State, now); !errors.Is(err, federationports.ErrAttemptConsumed) {
		t.Fatalf("second consume err=%v", err)
	}
}

func testConnection(t *testing.T, tenantID string, now time.Time) *domain.IdentityProviderConnection {
	t.Helper()
	return &domain.IdentityProviderConnection{
		ID: pgfixtures.NewUUID(t), TenantID: tenantID, DisplayName: "OIDC",
		Protocol: domain.ProtocolOIDC, Status: domain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client", SecretReference: "env:OIDC_SECRET",
		AuthorizationEndpoint: "https://idp.example/auth", TokenEndpoint: "https://idp.example/token",
		JWKSURI:       "https://idp.example/jwks",
		ClaimMapping:  domain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: domain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}
}
