package envelope_openbao

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestEnsureKeyCreatesMissingKey verifies the OpenBao Transit-compatible
// create-if-missing flow, mirroring backend/signingkeys/keys_vault's
// EnsureKey pattern: GET first, POST create only on 404.
func TestEnsureKeyCreatesMissingKey(t *testing.T) {
	var gotCreate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Vault-Token"); got != "test-token" {
			t.Fatalf("unexpected token header: %s", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/transit/keys/idmagic/datakeys/tenant-a":
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/transit/keys/idmagic/datakeys/tenant-a":
			gotCreate = true
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["type"] != "aes256-gcm96" {
				t.Fatalf("unexpected key type: %v", body["type"])
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	engine := NewHTTPTransitEngine(server.URL, "test-token", "")
	if err := engine.EnsureKey(context.Background(), "idmagic/datakeys/tenant-a"); err != nil {
		t.Fatalf("EnsureKey failed: %v", err)
	}
	if !gotCreate {
		t.Fatal("expected EnsureKey to create the missing key")
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
	if err := engine.EnsureKey(context.Background(), "idmagic/datakeys/tenant-a"); err != nil {
		t.Fatalf("EnsureKey failed: %v", err)
	}
	if gotCreate {
		t.Fatal("expected EnsureKey not to recreate an existing key")
	}
}

// TestEncryptDecryptRoundTrip exercises the wire format: base64 plaintext in
// the request, "vault:vN:<base64>" ciphertext in the response, and the
// reverse for decrypt.
func TestEncryptDecryptRoundTrip(t *testing.T) {
	const plaintext = "0123456789abcdef0123456789abcdef"
	encoded := base64.StdEncoding.EncodeToString([]byte(plaintext))
	vaultCiphertext := "vault:v1:" + encoded

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/transit/encrypt/idmagic/datakeys/tenant-a":
			var body struct {
				Plaintext string `json:"plaintext"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Plaintext != encoded {
				t.Fatalf("unexpected plaintext in request: %s", body.Plaintext)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"ciphertext": vaultCiphertext},
			})
		case "/v1/transit/decrypt/idmagic/datakeys/tenant-a":
			var body struct {
				Ciphertext string `json:"ciphertext"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Ciphertext != vaultCiphertext {
				t.Fatalf("unexpected ciphertext in request: %s", body.Ciphertext)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"plaintext": encoded},
			})
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	engine := NewHTTPTransitEngine(server.URL, "test-token", "")
	ciphertext, err := engine.EncryptDataKey(context.Background(), "idmagic/datakeys/tenant-a", []byte(plaintext))
	if err != nil {
		t.Fatalf("EncryptDataKey failed: %v", err)
	}
	if ciphertext != vaultCiphertext {
		t.Fatalf("unexpected ciphertext: %s", ciphertext)
	}

	decrypted, err := engine.DecryptDataKey(context.Background(), "idmagic/datakeys/tenant-a", ciphertext)
	if err != nil {
		t.Fatalf("DecryptDataKey failed: %v", err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("unexpected decrypted plaintext: %s", decrypted)
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
