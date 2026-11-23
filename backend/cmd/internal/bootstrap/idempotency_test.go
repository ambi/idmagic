package bootstrap

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/idempotency"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SYSTEM-036: PostgreSQL の正式な組み立てを作り直しても暗号化した応答を再生でき、登録した鍵の移行処理が残る応答を検出する。
func TestPostgresIdempotencySurvivesReassembly(t *testing.T) {
	pool := pgtest.Require(t)
	// 開発用のマスター鍵はプロセスごとに変わる。本番と同じ外部の鍵保管を
	// HTTP のフェイクで用意し、二つの構成が同じマスター鍵を使う。
	masterKey, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	transit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct{ Plaintext, Ciphertext string }
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"data":{}}`))
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var output map[string]string
		if name, ok := strings.CutPrefix(r.URL.Path, "/v1/transit/encrypt/"); ok {
			plain, err := base64.StdEncoding.DecodeString(input.Plaintext)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			wrapped, _, err := masterKey.WrapDataKey(r.Context(), name, plain)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			output = map[string]string{"ciphertext": base64.StdEncoding.EncodeToString(wrapped)}
		} else if name, ok := strings.CutPrefix(r.URL.Path, "/v1/transit/decrypt/"); ok {
			wrapped, err := base64.StdEncoding.DecodeString(input.Ciphertext)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			plain, err := masterKey.UnwrapDataKey(r.Context(), name, wrapped, "cleartext-dev")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			output = map[string]string{"plaintext": base64.StdEncoding.EncodeToString(plain)}
		} else {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"data": output}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(transit.Close)
	env := map[string]string{
		"PERSISTENCE": "postgres", "DATABASE_URL": pool.Config().ConnString(), "KEY_PROVIDER": "local",
		"DATA_KEY_PROVIDER": "openbao", "OPENBAO_ADDR": transit.URL, "OPENBAO_TOKEN": "test-token",
	}
	loader := NewConfigLoader(func(key string) string { return env[key] })
	config := LoadSharedConfig(loader)
	if err := loader.Err(); err != nil {
		t.Fatal(err)
	}
	first, err := Assemble(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(first.Close)
	tenantID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	now := pgtest.Now()
	if err := first.Tenancy.TenantRepo.Save(t.Context(), &tenancydomain.Tenant{
		ID: tenantID, Realm: "idem-" + tenantID, DisplayName: "Idempotency", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if first.Idempotency == nil {
		t.Fatal("PostgreSQL idempotency ledger is not wired")
	}
	record, _, err := first.Idempotency.Claim(t.Context(), tenantID, "admin", "key", "fingerprint", now)
	if err != nil {
		t.Fatal(err)
	}
	want := idempotency.Response{Status: 201, Body: []byte("persisted-secret")}
	if err := first.Idempotency.Complete(t.Context(), record, want); err != nil {
		t.Fatal(err)
	}
	second, err := Assemble(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(second.Close)
	if second.Idempotency == nil {
		t.Fatal("reassembled idempotency ledger is not wired")
	}
	restored, acquired, err := second.Idempotency.Claim(t.Context(), tenantID, "admin", "key", "fingerprint", now.Add(time.Second))
	if err != nil || acquired || restored.ID != record.ID || bytes.Contains(restored.Ciphertext, want.Body) {
		t.Fatalf("restored=%+v acquired=%v error=%v", restored, acquired, err)
	}
	got, err := second.Idempotency.Read(t.Context(), restored)
	if err != nil || got.Status != want.Status || !bytes.Equal(got.Body, want.Body) {
		t.Fatalf("replayed=%+v error=%v", got, err)
	}
	if _, err := second.DataKeys.RotateTenantDataKey(t.Context(), tenantID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	migrator, ok := second.DataKeys.Migrators.Lookup("http_idempotency_response")
	if !ok {
		t.Fatal("idempotency response migrator is not wired")
	}
	if pending, err := migrator.PendingCount(t.Context(), tenantID, 2); err != nil || pending != 1 {
		t.Fatalf("pending=%d error=%v", pending, err)
	}
}
