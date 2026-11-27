package envelope_openbao

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
)

// recordedToken は、記録が X-Vault-Token に置いた値である。記録に使った root トークンはこの値へ置き換えてある。
const recordedToken = "recorded-root-token"

// openBaoRecording は、mise run record-transit -- openbao が OpenBao の開発用サーバーから取った記録を読む。
func openBaoRecording(t *testing.T) testing_httprecord.Recording {
	t.Helper()
	data, err := os.ReadFile("testdata/openbao-transit.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatalf("decode OpenBao recording: %v", err)
	}
	return rec
}

// recordedPlaintext は、記録した暗号化の要求が送った平文の DEK を返す。
func recordedPlaintext(t *testing.T, ex testing_httprecord.Exchange) []byte {
	t.Helper()
	var body struct {
		Plaintext string `json:"plaintext"`
	}
	if err := json.Unmarshal(ex.Request.Body, &body); err != nil {
		t.Fatal(err)
	}
	plaintext, err := base64.StdEncoding.DecodeString(body.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	return plaintext
}

// 既定の接頭辞のマスター鍵で、鍵の作成、DEK のラップ、戻しを OpenBao から取った記録に対して行う。
// 要求は方式、パス、Content-Type、X-Vault-Token、JSON の本文の一致まで記録と照合する。
// 記録のパスは OpenBao が受け付けた鍵名なので、製品の鍵名が OpenBao の名前の規則に合うことも確かめる。
//
//spec:covers REQ-DATAKEYS-001: テナントの初回の暗号化のとき、DEK を MasterKey のプロバイダーでラップする。OpenBao のプロバイダーは既定の接頭辞で OpenBao が受け付ける鍵を作ってラップする。
func TestHTTPTransitEngine_ReplaysOpenBaoTransit(t *testing.T) {
	rec := openBaoRecording(t).Select(t, "read-missing-key", "create-key", "encrypt-v1", "decrypt-v1")
	server := testing_httprecord.Replay{
		Headers:   []string{"Content-Type", "X-Vault-Token"},
		CheckBody: testing_httprecord.CompareJSON,
	}.Serve(t, rec.Exchanges...)
	provider := NewOpenBaoMasterKeyProvider(NewHTTPTransitEngine(server.URL, recordedToken, ""), DefaultKeyNamePrefix)
	ctx := context.Background()
	dek := recordedPlaintext(t, rec.Exchange(t, "encrypt-v1"))

	wrapped, masterKeyID, err := provider.WrapDataKey(ctx, "tenant-a", dek)
	if err != nil {
		t.Fatal(err)
	}
	unwrapped, err := provider.UnwrapDataKey(ctx, "tenant-a", wrapped, masterKeyID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unwrapped, dek) {
		t.Fatalf("unwrapped DEK = %q, want the wrapped DEK", unwrapped)
	}
}

// 一つの応答で決まる失敗。どの操作も、OpenBao の 2xx 以外の応答と解釈できない本文を成功として扱わない。
func TestHTTPTransitEngine_RejectsUnsuccessfulStatus(t *testing.T) {
	denied := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"errors":["permission denied"]}`)
	}
	ciphertext := "vault:v1:" + base64.StdEncoding.EncodeToString([]byte("sealed"))
	tests := []struct {
		name    string
		respond func(w http.ResponseWriter, r *http.Request)
		call    func(ctx context.Context, e *HTTPTransitEngine) error
	}{
		{
			"鍵の読み取りが拒否される", func(w http.ResponseWriter, _ *http.Request) { denied(w) },
			func(ctx context.Context, e *HTTPTransitEngine) error { return e.EnsureKey(ctx, "k") },
		},
		{"鍵の作成が拒否される", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			denied(w)
		}, func(ctx context.Context, e *HTTPTransitEngine) error { return e.EnsureKey(ctx, "k") }},
		{
			"暗号化が拒否される", func(w http.ResponseWriter, _ *http.Request) { denied(w) },
			func(ctx context.Context, e *HTTPTransitEngine) error {
				_, err := e.EncryptDataKey(ctx, "k", []byte("dek"))
				return err
			},
		},
		{"暗号化の応答に暗号文がない", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"data":{}}`)
		}, func(ctx context.Context, e *HTTPTransitEngine) error {
			_, err := e.EncryptDataKey(ctx, "k", []byte("dek"))
			return err
		}},
		{
			"復号が拒否される", func(w http.ResponseWriter, _ *http.Request) { denied(w) },
			func(ctx context.Context, e *HTTPTransitEngine) error {
				_, err := e.DecryptDataKey(ctx, "k", ciphertext)
				return err
			},
		},
		{"復号の平文が base64 でない", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"data":{"plaintext":"not base64!"}}`)
		}, func(ctx context.Context, e *HTTPTransitEngine) error {
			_, err := e.DecryptDataKey(ctx, "k", ciphertext)
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.respond))
			defer server.Close()
			if err := tc.call(context.Background(), NewHTTPTransitEngine(server.URL, recordedToken, "")); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestEnsureKeyNoOpWhenKeyExists(t *testing.T) {
	var gotCreate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			gotCreate = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewHTTPTransitEngine(server.URL, "test-token", "")
	if err := engine.EnsureKey(context.Background(), "idmagic-datakeys-tenant-a"); err != nil {
		t.Fatalf("EnsureKey failed: %v", err)
	}
	if gotCreate {
		t.Fatal("expected EnsureKey not to recreate an existing key")
	}
}

func TestHealthyReflectsSysHealthEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewHTTPTransitEngine(server.URL, "test-token", "")
	if !engine.Healthy(context.Background()) {
		t.Fatal("expected Healthy to be true when /v1/sys/health returns 200")
	}
}

func TestHealthyFalseWhenUnreachable(t *testing.T) {
	engine := NewHTTPTransitEngine("http://127.0.0.1:1", "test-token", "")
	if engine.Healthy(context.Background()) {
		t.Fatal("expected Healthy to be false when OpenBao is unreachable")
	}
}
