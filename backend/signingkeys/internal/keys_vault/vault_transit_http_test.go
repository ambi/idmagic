package keys_vault_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
	adaptercrypto "github.com/ambi/idmagic/backend/signingkeys/internal/keys_vault"
)

// recordedToken は、記録が X-Vault-Token に置いた値である。記録に使った root トークンはこの値へ置き換えてある。
const recordedToken = "recorded-root-token"

// vaultRecording は、mise run record-transit -- vault が Vault の開発用サーバーから取った記録を読む。
func vaultRecording(t *testing.T) testing_httprecord.Recording {
	t.Helper()
	data, err := os.ReadFile("testdata/vault-transit.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatalf("decode Vault recording: %v", err)
	}
	return rec
}

// replayVault は記録の往復を順に返す。要求は方式、パス、Content-Type、X-Vault-Token、JSON の本文の一致まで照合する。
func replayVault(t *testing.T, exchanges ...testing_httprecord.Exchange) *httptest.Server {
	t.Helper()
	return testing_httprecord.Replay{
		Headers:   []string{"Content-Type", "X-Vault-Token"},
		CheckBody: testing_httprecord.CompareJSON,
	}.Serve(t, exchanges...)
}

// recordedDigest は、記録した署名の要求が送ったダイジェストを返す。
func recordedDigest(t *testing.T, ex testing_httprecord.Exchange) []byte {
	t.Helper()
	var body struct {
		Input string `json:"input"`
	}
	if err := json.Unmarshal(ex.Request.Body, &body); err != nil {
		t.Fatal(err)
	}
	digest, err := base64.StdEncoding.DecodeString(body.Input)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

// 鍵ストアの初回取得、PS256 の署名、ローテーション、回転後の PKCS #1 v1.5 の署名を、Vault から取った記録に対して行う。
// 署名は Vault が実際に返した値であり、PS256 は RFC 7518 §3.5 の salt 長（ハッシュ長）で検証する。
//
//spec:covers RFC7518-SIGNATURE-ALGORITHMS: Vault の Transit で署名した PS256 の署名は、salt 長をハッシュ長に固定した検証器が受け入れる。
func TestHTTPTransitEngine_ReplaysVaultTransit(t *testing.T) {
	rec := vaultRecording(t)
	server := replayVault(t, rec.Exchanges...)
	store := adaptercrypto.NewVaultKeyStore(adaptercrypto.NewHTTPTransitEngine(server.URL, recordedToken, ""), "")
	ctx := tenantCtx()
	digest := recordedDigest(t, rec.Exchange(t, "sign-v1-pss"))

	first, err := store.GetActiveKey(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	signature, err := first.PrivateKey.(crypto.Signer).Sign(rand.Reader, digest, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	if err != nil {
		t.Fatal(err)
	}
	if err := rsa.VerifyPSS(first.PublicKey.(*rsa.PublicKey), crypto.SHA256, digest, signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}); err != nil {
		t.Fatalf("PS256 signature from Vault must verify with a hash-length salt: %v", err)
	}

	second, err := store.Rotate(ctx, time.Now().UTC(), 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if second.Kid == first.Kid {
		t.Fatal("rotation must publish the public key of the new Vault version")
	}
	signature, err = second.PrivateKey.(crypto.Signer).Sign(rand.Reader, digest, crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if err := rsa.VerifyPKCS1v15(second.PublicKey.(*rsa.PublicKey), crypto.SHA256, digest, signature); err != nil {
		t.Fatalf("PKCS #1 v1.5 signature from the rotated Vault version must verify: %v", err)
	}
}

// 呼び出し元の PSS の salt 長を Vault の salt_length へ写す。PS256 の署名者は PSSSaltLengthEqualsHash を渡す。
func TestHTTPTransitEngine_SendsCallerPSSSaltLength(t *testing.T) {
	tests := []struct {
		name string
		opts crypto.SignerOpts
		want any // salt_length の値。nil なら欄を送らない
	}{
		{"ハッシュ長", &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256}, "hash"},
		{"自動", &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto, Hash: crypto.SHA256}, "auto"},
		{"バイト数", &rsa.PSSOptions{SaltLength: 20, Hash: crypto.SHA256}, "20"},
		{"PKCS #1 v1.5", crypto.SHA256, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&body)
				_, _ = io.WriteString(w, `{"data":{"signature":"vault:v1:c2ln"}}`)
			}))
			defer server.Close()
			engine := adaptercrypto.NewHTTPTransitEngine(server.URL, recordedToken, "")
			if _, err := engine.Sign(context.Background(), "k", 1, make([]byte, 32), tc.opts); err != nil {
				t.Fatal(err)
			}
			if got, present := body["salt_length"]; got != tc.want || (tc.want == nil && present) {
				t.Fatalf("salt_length = %v (present %v), want %v", got, present, tc.want)
			}
		})
	}
}

// 一つの応答で決まる失敗。どの操作も、Vault の 2xx 以外の応答と解釈できない本文を成功として扱わない。
func TestHTTPTransitEngine_RejectsUnsuccessfulStatus(t *testing.T) {
	denied := func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"errors":["2 errors occurred:\n\t* permission denied\n\t* invalid token\n\n"]}`)
	}
	tests := []struct {
		name    string
		respond func(w http.ResponseWriter, r *http.Request)
		call    func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error
	}{
		{
			"鍵の読み取りが拒否される", func(w http.ResponseWriter, _ *http.Request) { denied(w) },
			func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error { return e.EnsureKey(ctx, "k") },
		},
		{"鍵の作成が拒否される", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			denied(w)
		}, func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error { return e.EnsureKey(ctx, "k") }},
		{
			"回転が拒否される", func(w http.ResponseWriter, _ *http.Request) { denied(w) },
			func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error { return e.RotateKey(ctx, "k") },
		},
		{
			"公開鍵の読み取りが拒否される", func(w http.ResponseWriter, _ *http.Request) { denied(w) },
			func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error {
				_, _, err := e.LatestPublicKey(ctx, "k")
				return err
			},
		},
		{"最新バージョンの公開鍵がない", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"data":{"latest_version":2,"keys":{"1":{"public_key":"-----BEGIN PUBLIC KEY-----"}}}}`)
		}, func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error {
			_, _, err := e.LatestPublicKey(ctx, "k")
			return err
		}},
		{
			"署名が拒否される", func(w http.ResponseWriter, _ *http.Request) { denied(w) },
			func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error {
				_, err := e.Sign(ctx, "k", 1, make([]byte, 32), crypto.SHA256)
				return err
			},
		},
		{"署名の形が vault:vN:<base64> でない", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"data":{"signature":"c2ln"}}`)
		}, func(ctx context.Context, e *adaptercrypto.HTTPTransitEngine) error {
			_, err := e.Sign(ctx, "k", 1, make([]byte, 32), crypto.SHA256)
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.respond))
			defer server.Close()
			if err := tc.call(context.Background(), adaptercrypto.NewHTTPTransitEngine(server.URL, recordedToken, "")); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestHTTPTransitEngine_HealthyOnSysHealth200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sys/health" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	if !adaptercrypto.NewHTTPTransitEngine(server.URL, recordedToken, "").Healthy(context.Background()) {
		t.Fatal("engine must report healthy on 200 sys/health")
	}
}
