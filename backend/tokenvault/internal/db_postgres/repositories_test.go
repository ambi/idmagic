package db_postgres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/tokenvault/internal/db_postgres"
	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/testing_contract"
)

func newCipher(t *testing.T, tenantIDs ...string) *datakeys.FieldCipher {
	t.Helper()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	module := datakeys.NewMemoryModule(envelope_crypto.NewTinkEnvelopeCrypto(master))
	for _, tenantID := range tenantIDs {
		if _, err := module.BootstrapTenantDataKey(context.Background(), tenantID, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}
	return module.FieldCipher()
}

func seedUser(t *testing.T, pool sharedpg.DB, tenantID string) string {
	t.Helper()
	return pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(pool).UserRepo, tenantID).ID
}

//spec:covers REQ-TOKENVAULT-012: 版とリースの条件付き更新、REQ-TOKENVAULT-004: 連携の束縛と一回性、REQ-TOKENVAULT-002: 削除が接続を消すこと
func TestPersistenceContract(t *testing.T) {
	pool := pgtest.Require(t)
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		a, b := pgfixtures.SeedTenant(t, pool), pgfixtures.SeedTenant(t, pool)
		cipher := newCipher(t, a.ID, b.ID)
		return testing_contract.Fixture{
			Connectors:  &db_postgres.ConnectorRepository{Pool: pool, Cipher: cipher},
			Connections: &db_postgres.ConnectionRepository{Pool: pool, Cipher: cipher},
			Attempts:    &db_postgres.AttemptStore{Pool: pool, Cipher: cipher},
			TenantA:     a.ID, TenantB: b.ID, UserA: seedUser(t, pool, a.ID), NewID: uuid.NewString, Now: pgtest.Now(),
		}
	})
}

type connectionFixture struct {
	pool        sharedpg.DB
	cipher      *datakeys.FieldCipher
	connections *db_postgres.ConnectionRepository
	tenantID    string
	connectorID string
	now         time.Time
}

func newConnectionFixture(t *testing.T) connectionFixture {
	t.Helper()
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	cipher := newCipher(t, tenant.ID)
	now := pgtest.Now()
	connector := &domain.Connector{
		ID: uuid.NewString(), TenantID: tenant.ID, Name: "c", Status: domain.ConnectorActive,
		AuthorizationEndpoint: "https://idp.example.com/a", TokenEndpoint: "https://idp.example.com/t",
		ClientID: "c", ClientSecret: "client-secret", TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&db_postgres.ConnectorRepository{Pool: pool, Cipher: cipher}).Create(context.Background(), connector); err != nil {
		t.Fatal(err)
	}
	return connectionFixture{pool, cipher, &db_postgres.ConnectionRepository{Pool: pool, Cipher: cipher}, tenant.ID, connector.ID, now}
}

func (f connectionFixture) link(t *testing.T, access, refresh string) *domain.Connection {
	t.Helper()
	ctx := context.Background()
	userID := seedUser(t, f.pool, f.tenantID)
	if err := f.connections.SaveLinked(ctx, &domain.Connection{
		ID: uuid.NewString(), TenantID: f.tenantID, ConnectorID: f.connectorID, UserID: userID,
		Status: domain.ConnectionActive, GrantedScopes: []string{}, AccessToken: access, RefreshToken: refresh, UpdatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
	found, err := f.connections.Find(ctx, f.tenantID, f.connectorID, userID)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

//spec:covers REQ-TOKENVAULT-003: 平文を保存しないこと、別の行へ複製した暗号文を読めないこと
func TestConnectionCiphertextIsBoundToItsRecord(t *testing.T) {
	f := newConnectionFixture(t)
	ctx := context.Background()
	victim := f.link(t, "victim-access-token", "victim-refresh-token")
	attacker := f.link(t, "attacker-access-token", "")

	var access, refresh []byte
	if err := f.pool.QueryRow(ctx, `SELECT access_token_ciphertext, refresh_token_ciphertext FROM token_vault_connections WHERE id=$1`, victim.ID).Scan(&access, &refresh); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(access), "victim-access-token") || strings.Contains(string(refresh), "victim-refresh-token") {
		t.Fatal("tokens are stored in plaintext")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE token_vault_connections SET access_token_ciphertext=$1 WHERE id=$2`, access, attacker.ID); err != nil {
		t.Fatal(err)
	}
	if found, err := f.connections.Find(ctx, f.tenantID, f.connectorID, attacker.UserID); err == nil {
		t.Fatalf("a ciphertext copied to another row decrypted: %+v", found)
	}
}

//spec:covers REQ-TOKENVAULT-007: 復号できない行でも解除が成功すること
func TestDeleteSucceedsWhenTokensCannotBeDecrypted(t *testing.T) {
	f := newConnectionFixture(t)
	ctx := context.Background()
	row := f.link(t, "access", "refresh")
	if _, err := f.pool.Exec(ctx, `UPDATE token_vault_connections SET access_token_ciphertext='\x00'::bytea WHERE id=$1`, row.ID); err != nil {
		t.Fatal(err)
	}
	deleted, err := f.connections.Delete(ctx, f.tenantID, f.connectorID, row.UserID)
	if err != nil || deleted == nil || deleted.AccessToken != "" {
		t.Fatalf("delete: %+v, %v", deleted, err)
	}
}

//spec:covers REQ-TOKENVAULT-003: 再暗号化が値を保ち、読んだ後に更新された行を上書きしないこと
func TestReencryptionMovesTokensAndKeepsTheirValues(t *testing.T) {
	f := newConnectionFixture(t)
	ctx := context.Background()
	row := f.link(t, "access", "refresh")
	var version int32
	if err := f.pool.QueryRow(ctx, `SELECT token_key_version FROM token_vault_connections WHERE id=$1`, row.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	migrator := &db_postgres.ConnectionTokenReencryptor{Repo: f.connections}
	if pending, err := migrator.PendingCount(ctx, f.tenantID, int(version)); err != nil || pending != 0 {
		t.Fatalf("pending before: %d, %v", pending, err)
	}
	// 有効な版を別の値として渡すと、行は未移行に数えられる。暗号化は DEK の現在の版を使う。
	if pending, err := migrator.PendingCount(ctx, f.tenantID, int(version)+1); err != nil || pending != 1 {
		t.Fatalf("pending: %d, %v", pending, err)
	}
	if migrated, err := migrator.ReencryptBatch(ctx, f.tenantID, int(version)+1, 10); err != nil || migrated != 1 {
		t.Fatalf("migrated: %d, %v", migrated, err)
	}
	found, err := f.connections.Find(ctx, f.tenantID, f.connectorID, row.UserID)
	if err != nil || found.AccessToken != "access" || found.RefreshToken != "refresh" || found.TokenVersion != row.TokenVersion {
		t.Fatalf("found: %+v, %v", found, err)
	}
}
