package db_postgres_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/idmanagement"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/tokenvault/internal/db_postgres"
	tokenvaultdomain "github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/testing_contract"
)

//spec:covers REQ-DATAKEYS-010: コネクタと接続の両方で無効化した行を残し、退役した鍵の値を active に移しても未移行の参照を数え続ける。
func TestTokenVaultReencryptionSkipsDisabledKeys(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	module := datakeys.NewMemoryModule(envelope_crypto.NewTinkEnvelopeCrypto(master))
	connectors := &db_postgres.ConnectorRepository{Pool: pool, Cipher: module.FieldCipher()}
	connections := &db_postgres.ConnectionRepository{Pool: pool, Cipher: module.FieldCipher()}
	var saved []*tokenvaultdomain.Connection
	for i, id := range []string{"00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002"} {
		connector := &tokenvaultdomain.Connector{
			ID: id, TenantID: tenant.ID, Name: id, Status: tokenvaultdomain.ConnectorActive,
			AuthorizationEndpoint: "https://idp.example/a", TokenEndpoint: "https://idp.example/t",
			ClientID: id, ClientSecret: "client-secret", TokenEndpointAuthMethod: tokenvaultdomain.AuthMethodClientSecretBasic,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		}
		if err := connectors.Create(ctx, connector); err != nil {
			t.Fatal(err)
		}
		connection := &tokenvaultdomain.Connection{
			ID: id, TenantID: tenant.ID, ConnectorID: id, UserID: seedUser(t, pool, tenant.ID),
			Status: tokenvaultdomain.ConnectionActive, AccessToken: "access", RefreshToken: "refresh", UpdatedAt: now,
		}
		if err := connections.SaveLinked(ctx, connection); err != nil {
			t.Fatal(err)
		}
		saved = append(saved, connection)
		if _, err := module.RotateTenantDataKey(ctx, tenant.ID, now); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if _, err := module.Repository.Disable(ctx, tenant.ID, 1, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	queries := db_postgres.New(pool)
	disabledConnector, err := queries.FindConnector(ctx, db_postgres.FindConnectorParams{TenantID: tenant.ID, ID: saved[0].ConnectorID})
	if err != nil {
		t.Fatal(err)
	}
	disabledConnection, err := queries.FindConnection(ctx, db_postgres.FindConnectionParams{TenantID: tenant.ID, ConnectorID: saved[0].ConnectorID, UserID: saved[0].UserID})
	if err != nil {
		t.Fatal(err)
	}
	module.Migrators.Register(db_postgres.ConnectorSecretMigratorName, &db_postgres.ConnectorSecretReencryptor{Repo: connectors})
	module.Migrators.Register(db_postgres.ConnectionTokenMigratorName, &db_postgres.ConnectionTokenReencryptor{Repo: connections})
	for _, name := range []string{db_postgres.ConnectorSecretMigratorName, db_postgres.ConnectionTokenMigratorName} {
		params, err := json.Marshal(map[string]string{"tenant_id": tenant.ID, "migrator": name})
		if err != nil {
			t.Fatal(err)
		}
		result, err := module.ReencryptionHandler(nil)(ctx, &jobsdomain.Job{TenantID: tenant.ID, Params: params})
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		var counts struct{ Migrated, Remaining int }
		if err := json.Unmarshal(result, &counts); err != nil {
			t.Fatal(err)
		}
		if counts.Migrated != 1 || counts.Remaining != 1 {
			t.Fatalf("%s result=%s, want migrated=1 remaining=1", name, result)
		}
	}
	afterConnector, err := queries.FindConnector(ctx, db_postgres.FindConnectorParams{TenantID: tenant.ID, ID: saved[0].ConnectorID})
	if err != nil || afterConnector.SecretKeyVersion != 1 || !bytes.Equal(afterConnector.SecretCiphertext, disabledConnector.SecretCiphertext) {
		t.Fatalf("disabled connector=%+v err=%v", afterConnector, err)
	}
	afterConnection, err := queries.FindConnection(ctx, db_postgres.FindConnectionParams{TenantID: tenant.ID, ConnectorID: saved[0].ConnectorID, UserID: saved[0].UserID})
	if err != nil || afterConnection.TokenKeyVersion.Int32 != 1 || !bytes.Equal(afterConnection.AccessTokenCiphertext, disabledConnection.AccessTokenCiphertext) || !bytes.Equal(afterConnection.RefreshTokenCiphertext, disabledConnection.RefreshTokenCiphertext) {
		t.Fatalf("disabled connection=%+v err=%v", afterConnection, err)
	}
	connector, err := connectors.Find(ctx, tenant.ID, saved[1].ConnectorID)
	if err != nil || connector == nil || connector.ClientSecret != "client-secret" {
		t.Fatalf("migrated connector=%+v err=%v", connector, err)
	}
	connection, err := connections.Find(ctx, tenant.ID, saved[1].ConnectorID, saved[1].UserID)
	if err != nil || connection == nil || connection.AccessToken != "access" || connection.RefreshToken != "refresh" {
		t.Fatalf("migrated connection=%+v err=%v", connection, err)
	}
	for _, query := range []string{
		`SELECT secret_key_version FROM token_vault_connectors WHERE id=$1 AND tenant_id=$2`,
		`SELECT token_key_version FROM token_vault_connections WHERE id=$1 AND tenant_id=$2`,
	} {
		var version int
		if err := pool.QueryRow(ctx, query, saved[1].ID, tenant.ID).Scan(&version); err != nil || version != 3 {
			t.Fatalf("migrated key version=%d err=%v, want 3", version, err)
		}
	}
}

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
			TenantA:     a.ID, TenantB: b.ID, UserA: seedUser(t, pool, a.ID), NewID: func() string { return uuid.NewV4().String() }, Now: pgtest.Now(),
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
	connector := &tokenvaultdomain.Connector{
		ID: uuid.NewV4().String(), TenantID: tenant.ID, Name: "c", Status: tokenvaultdomain.ConnectorActive,
		AuthorizationEndpoint: "https://idp.example.com/a", TokenEndpoint: "https://idp.example.com/t",
		ClientID: "c", ClientSecret: "client-secret", TokenEndpointAuthMethod: tokenvaultdomain.AuthMethodClientSecretBasic,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&db_postgres.ConnectorRepository{Pool: pool, Cipher: cipher}).Create(context.Background(), connector); err != nil {
		t.Fatal(err)
	}
	return connectionFixture{pool, cipher, &db_postgres.ConnectionRepository{Pool: pool, Cipher: cipher}, tenant.ID, connector.ID, now}
}

func (f connectionFixture) link(t *testing.T, access, refresh string) *tokenvaultdomain.Connection {
	t.Helper()
	ctx := context.Background()
	userID := seedUser(t, f.pool, f.tenantID)
	if err := f.connections.SaveLinked(ctx, &tokenvaultdomain.Connection{
		ID: uuid.NewV4().String(), TenantID: f.tenantID, ConnectorID: f.connectorID, UserID: userID,
		Status: tokenvaultdomain.ConnectionActive, GrantedScopes: []string{}, AccessToken: access, RefreshToken: refresh, UpdatedAt: f.now,
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
	if migrated, err := migrator.ReencryptBatch(ctx, f.tenantID, int(version)+1, 10, nil); err != nil || migrated != 1 {
		t.Fatalf("migrated: %d, %v", migrated, err)
	}
	found, err := f.connections.Find(ctx, f.tenantID, f.connectorID, row.UserID)
	if err != nil || found.AccessToken != "access" || found.RefreshToken != "refresh" || found.TokenVersion != row.TokenVersion {
		t.Fatalf("found: %+v, %v", found, err)
	}
}
