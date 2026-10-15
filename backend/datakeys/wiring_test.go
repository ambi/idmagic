package datakeys_test

import (
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

//spec:covers REQ-DATAKEYS-002: PostgreSQL 構築から暗号化と回転を行い、別の Module も retiring 鍵で復号できる。
func TestPostgresModuleDecryptsPersistedCiphertextAfterRotation(t *testing.T) {
	pool := pgtest.Require(t)
	ctx := t.Context()
	now := pgtest.Now()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	if err := tenancypostgres.NewPostgresModule(pool).TenantRepo.Save(ctx, &tenancydomain.Tenant{ID: id, Realm: "module-" + id[:8], DisplayName: "Module", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	master, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(master)
	module, err := datakeys.NewPostgresModule(ctx, pool, crypto)
	if err != nil {
		t.Fatal(err)
	}
	version, ciphertext, err := module.FieldCipher().Encrypt(ctx, id, "Authentication", "mfa_factors", "user-1:totp", "secret", "persisted-secret")
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("version = %d; want 1", version)
	}
	if _, err := module.RotateTenantDataKey(ctx, id, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	fresh, err := datakeys.NewPostgresModule(ctx, pool, crypto)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := fresh.FieldCipher().Decrypt(ctx, id, "Authentication", "mfa_factors", "user-1:totp", "secret", version, ciphertext)
	if err != nil || plaintext != "persisted-secret" {
		t.Fatalf("Decrypt = %q, %v; want persisted-secret", plaintext, err)
	}
}
