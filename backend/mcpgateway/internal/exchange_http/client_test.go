package exchange_http

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
)

const resource = "https://downstream.example/mcp"

func proofClaims(t *testing.T, proof string, key *ecdsa.PrivateKey) map[string]any {
	t.Helper()
	claims, err := tokens_jose.VerifyJWT(proof, "ES256", func(map[string]any) (crypto.PublicKey, error) { return &key.PublicKey, nil },
		tokens_jose.JWTExpectation{Now: time.Now()})
	if err != nil {
		t.Fatalf("proof: %v", err)
	}
	return claims
}

//spec:covers REQ-MCPGATEWAY-008: 交換の要求の形、/token の URL を htu にした証明、下流の resource と ath を持つ転送の証明
func TestExchangeSendsTheGrantAndProvesBothCalls(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var tokenProof string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenProof = r.Header.Get("DPoP")
		if r.FormValue("token_exchange_grant") != "grant" || r.FormValue("subject_token") != "subject" || r.FormValue("resource") != resource {
			t.Errorf("form = %v", r.PostForm)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "downstream", "token_type": "DPoP"})
	}))
	defer server.Close()
	client := Client{TokenURL: server.URL + "/token", ClientID: "c", ClientSecret: "s", Resource: resource, Key: key, HTTPClient: server.Client()}

	credential, err := client.Exchange(t.Context(), "subject", "grant", "id")
	if err != nil {
		t.Fatal(err)
	}
	if got := proofClaims(t, tokenProof, key); got["htu"] != client.TokenURL || got["htm"] != http.MethodPost || got["ath"] != nil {
		t.Fatalf("token proof = %v", got)
	}
	sum := sha256.Sum256([]byte("downstream"))
	got := proofClaims(t, credential.Proof, key)
	if credential.Token != "downstream" || got["htu"] != resource || got["ath"] != base64.RawURLEncoding.EncodeToString(sum[:]) {
		t.Fatalf("credential = %+v proof = %v", credential, got)
	}
}

//spec:covers REQ-MCPGATEWAY-008: 200 以外と DPoP でない結果は資格情報にしない
func TestExchangeRefusesWhatItCannotPresent(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for name, handler := range map[string]http.HandlerFunc{
		"rejected": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) },
		"bearer": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"access_token":"t","token_type":"Bearer"}`))
		},
		"broken": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{")) },
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()
			client := Client{TokenURL: server.URL, Resource: resource, Key: key, HTTPClient: server.Client()}
			credential, err := client.Exchange(t.Context(), "subject", "grant", "id")
			if err == nil || credential.Token != "" {
				t.Fatalf("credential=%+v err=%v", credential, err)
			}
		})
	}
}
