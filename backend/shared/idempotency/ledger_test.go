package idempotency_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys"
	"github.com/ambi/idmagic/backend/shared/idempotency"
	contract "github.com/ambi/idmagic/backend/shared/idempotency/testing_contract"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

//spec:covers REQ-SYSTEM-036: メモリでも同時受付、テナントと主体、期限、旧実行と再暗号化の競合を保存先で検査する。
func TestMemoryStoreContract(t *testing.T) {
	contract.Run(t, func() idempotency.Store { return idempotency.NewMemoryStore() }, "tenant-a", "tenant-b")
}

//spec:covers REQ-SYSTEM-036: 応答全体を実際の DEK で暗号化し、別レコードへのコピーを拒否し、鍵の移行後も同じ応答を復元できる。
func TestLedgerEncryptsResponsesAndMigratesDataKeys(t *testing.T) {
	provider, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	keys := datakeys.NewMemoryModule(envelope_crypto.NewTinkEnvelopeCrypto(provider))
	store := idempotency.NewMemoryStore()
	ledger := idempotency.New(store, keys.FieldCipher())
	keys.Migrators.Register("http_idempotency_response", ledger)
	now := time.Now().UTC().Truncate(time.Microsecond)
	record, acquired, err := ledger.Claim(t.Context(), "tenant-a", "admin", "opaque-key", "fingerprint", now)
	if err != nil || !acquired {
		t.Fatalf("claim=%v error=%v", acquired, err)
	}
	want := idempotency.Response{Status: 201, Headers: map[string][]string{"Location": {"/secret-resource"}}, Body: []byte(`{"secret":"unique-secret-value"}`)}
	if err := ledger.Complete(t.Context(), record, want); err != nil {
		t.Fatal(err)
	}
	record, _, err = ledger.Claim(t.Context(), "tenant-a", "admin", "opaque-key", "fingerprint", now)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(record.Ciphertext, []byte("unique-secret-value")) || bytes.Contains(record.Ciphertext, []byte("secret-resource")) || record.KeyHash == "opaque-key" {
		t.Fatal("plaintext was stored")
	}
	got, err := ledger.Read(t.Context(), record)
	if err != nil || !bytes.Equal(got.Body, want.Body) || got.Headers["Location"][0] != "/secret-resource" {
		t.Fatalf("response=%+v error=%v", got, err)
	}
	tampered := record
	tampered.ID = "another-record"
	if _, err := ledger.Read(t.Context(), tampered); err == nil {
		t.Fatal("ciphertext copied to a different record was decrypted")
	}
	if _, err := keys.RotateTenantDataKey(t.Context(), "tenant-a", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if count, err := ledger.PendingCount(t.Context(), "tenant-a", 2); err != nil || count != 1 {
		t.Fatalf("pending=%d error=%v", count, err)
	}
	if count, err := ledger.ReencryptBatch(t.Context(), "tenant-a", 2, 10, []int32{1}); err != nil || count != 0 {
		t.Fatalf("excluded migrated=%d error=%v", count, err)
	}
	if count, err := ledger.ReencryptBatch(t.Context(), "tenant-a", 2, 10, nil); err != nil || count != 1 {
		t.Fatalf("migrated=%d error=%v", count, err)
	}
	record, _, err = ledger.Claim(t.Context(), "tenant-a", "admin", "opaque-key", "fingerprint", now)
	if err != nil || record.KeyVersion != 2 {
		t.Fatalf("version=%d error=%v", record.KeyVersion, err)
	}
	got, err = ledger.Read(t.Context(), record)
	if err != nil || !bytes.Equal(got.Body, want.Body) {
		t.Fatalf("migrated response=%+v error=%v", got, err)
	}
	if count, err := ledger.PendingCount(t.Context(), "tenant-a", 2); err != nil || count != 0 {
		t.Fatalf("pending after migration=%d error=%v", count, err)
	}
}
