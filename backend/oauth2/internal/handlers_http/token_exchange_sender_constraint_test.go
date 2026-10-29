package handlers_http_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
)

const exchTokenHTU = "http://test/realms/default/token"

type exchDPoPKey struct {
	key *rsa.PrivateKey
	jwk map[string]any
	jkt string
}

func newExchDPoPKey(t *testing.T) exchDPoPKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := resourceTestJWK(&key.PublicKey)
	return exchDPoPKey{key: key, jwk: jwk, jkt: resourceTestJKT(t, jwk)}
}

func (k exchDPoPKey) proof(t *testing.T, jti string) string {
	t.Helper()
	return resourceTestDPoPProof(t, k.key, k.jwk, http.MethodPost, exchTokenHTU, jti, "", time.Now().UTC())
}

func newDPoPTokenExchangeServer(t *testing.T) string {
	t.Helper()
	return newTokenExchangeServer(t, func(deps *httpadapter.Deps) {
		deps.OAuth2.DpopReplayStore = oauth2memory.NewDpopReplayStore()
	})
}

func postTokenWithDPoP(t *testing.T, base string, form url.Values, proof string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, base+"/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(exchClientID, exchClientSecret)
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /token: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	return resp.StatusCode, parsed
}

func mintDPoPSubjectToken(t *testing.T, base string, key exchDPoPKey) string {
	t.Helper()
	status, body := postTokenWithDPoP(t, base, url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"read write"},
	}, key.proof(t, "mint-"+key.jkt))
	if status != http.StatusOK || body["token_type"] != "DPoP" {
		t.Fatalf("client_credentials with DPoP status=%d body=%v", status, body)
	}
	tok, _ := body["access_token"].(string)
	return tok
}

func exchangeForm(subject string) url.Values {
	return url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {subject},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"resource":           {"https://api.example/orders"},
	}
}

func introspectCnf(t *testing.T, base, token string) map[string]any {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, base+"/introspect", strings.NewReader(url.Values{"token": {token}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(exchClientID, exchClientSecret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var introspection map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&introspection); err != nil {
		t.Fatal(err)
	}
	if introspection["active"] != true {
		t.Fatalf("issued token is not active: %v", introspection)
	}
	cnf, _ := introspection["cnf"].(map[string]any)
	return cnf
}

//spec:covers REQ-OAUTH2-056: 正式な /token で、K1 に束縛した subject_token を K1 の Proof で交換すると、DPoP のトークンを返し、イントロスペクションの cnf.jkt が K1 のままである。
func TestTokenExchangeKeepsSubjectKeyWhenProofMatches(t *testing.T) {
	base := newDPoPTokenExchangeServer(t)
	k1 := newExchDPoPKey(t)
	subject := mintDPoPSubjectToken(t, base, k1)

	status, body := postTokenWithDPoP(t, base, exchangeForm(subject), k1.proof(t, "exchange-k1"))
	if status != http.StatusOK || body["token_type"] != "DPoP" {
		t.Fatalf("exchange with the subject key status=%d body=%v", status, body)
	}
	issued, _ := body["access_token"].(string)
	if cnf := introspectCnf(t, base, issued); cnf["jkt"] != k1.jkt {
		t.Fatalf("cnf=%v, want jkt=%s", cnf, k1.jkt)
	}
}

//spec:covers REQ-OAUTH2-056: 正式な /token で、K1 に束縛した subject_token を K2 の Proof で交換すると、400 と invalid_grant を返し、アクセストークンを返さない。
func TestTokenExchangeRejectsRebindingToAnotherKey(t *testing.T) {
	base := newDPoPTokenExchangeServer(t)
	k1, k2 := newExchDPoPKey(t), newExchDPoPKey(t)
	subject := mintDPoPSubjectToken(t, base, k1)

	status, body := postTokenWithDPoP(t, base, exchangeForm(subject), k2.proof(t, "exchange-k2"))
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" || body["access_token"] != nil {
		t.Fatalf("exchange with another key status=%d body=%v, want 400 invalid_grant without a token", status, body)
	}
}

//spec:covers REQ-OAUTH2-056: 正式な /token で、K1 に束縛した subject_token を Proof なしで交換すると、400 と invalid_grant を返し、Bearer のトークンを返さない。
func TestTokenExchangeRejectsDroppingTheSubjectConstraint(t *testing.T) {
	base := newDPoPTokenExchangeServer(t)
	k1 := newExchDPoPKey(t)
	subject := mintDPoPSubjectToken(t, base, k1)

	status, body := postTokenWithDPoP(t, base, exchangeForm(subject), "")
	if status != http.StatusBadRequest || body["error"] != "invalid_grant" || body["access_token"] != nil {
		t.Fatalf("exchange without a proof status=%d body=%v, want 400 invalid_grant without a token", status, body)
	}
}
