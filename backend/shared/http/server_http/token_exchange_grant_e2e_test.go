package server_http_test

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	exchangeDownstream      = "https://downstream.example/mcp"
	exchangeOtherDownstream = "https://other-downstream.example/mcp"
	tokenEndpoint           = stack.Issuer + "/realms/default/token"
)

// grantFixture は DPoP の Agent のトークンと、評価と交換の両方を許したゲートウェイである。
type grantFixture struct {
	*resourceAccessFixture
	agentKey, gatewayKey *rsa.PrivateKey
	agentJWK, gatewayJWK map[string]any
	gatewayJKT           string
	token                string
	sequence             int
}

func newGrantFixture(t *testing.T) *grantFixture {
	t.Helper()
	f := &grantFixture{resourceAccessFixture: newResourceAccessFixture(t, "A1")}
	f.grantD1(t)
	ctx := context.Background()
	client, err := f.s.Clients.FindByID(ctx, tenancydomain.DefaultTenantID, evalResourceServerClient)
	if err != nil || client == nil {
		t.Fatalf("gateway client: %v %v", client, err)
	}
	client.GrantTypes = append(client.GrantTypes, spec.GrantTokenExchange)
	client.Scope = "tools:read"
	if err := f.s.Clients.Save(ctx, client); err != nil {
		t.Fatal(err)
	}
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("resource server: %v %v", server, err)
	}
	server.AccessRules[0].TokenExchangeResources = []string{exchangeDownstream}
	if err := f.s.McpResourceServers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	for _, resource := range []string{exchangeDownstream, exchangeOtherDownstream} {
		downstream := *server
		downstream.ID, downstream.Resource, downstream.ClientID, downstream.AccessRules = "", resource, "", nil
		downstream.ID = "downstream-" + strings.ReplaceAll(resource, "/", "-")
		if err := f.s.McpResourceServers.Save(ctx, &downstream); err != nil {
			t.Fatal(err)
		}
	}
	f.agentKey, f.agentJWK, _ = tpDPoPKey(t)
	f.gatewayKey, f.gatewayJWK, f.gatewayJKT = tpDPoPKey(t)
	f.token = f.resourceToken(t, agentClientID("A1"), agentRevocationResource, func(request *http.Request) {
		request.Header.Set("DPoP", tpDPoPProof(t, f.agentKey, f.agentJWK, tpDPoPProofInput{htm: http.MethodPost, htu: tokenEndpoint, jti: "issue", iat: time.Now().UTC()}))
	})
	return f
}

func (f *grantFixture) jti(prefix string) string {
	f.sequence++
	return prefix + "-" + strings.Repeat("x", f.sequence)
}

// agentProof は Agent がゲートウェイの公開 URL へ送る証明である。
func (f *grantFixture) agentProof(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(f.token))
	return tpDPoPProof(t, f.agentKey, f.agentJWK, tpDPoPProofInput{
		htm: http.MethodPost, htu: agentRevocationResource, jti: f.jti("agent"),
		ath: base64.RawURLEncoding.EncodeToString(sum[:]), iat: time.Now().UTC(),
	})
}

// evaluateForExchange は Agent の証明とともに評価する。proved が偽なら証明を付けない。
func (f *grantFixture) evaluateForExchange(t *testing.T, token, downstream string, proved bool) map[string]any {
	t.Helper()
	form := evalForm(token)
	form.Set("token_exchange_resource", downstream)
	if proved {
		sum := sha256.Sum256([]byte(token))
		form.Set("dpop_proof", tpDPoPProof(t, f.agentKey, f.agentJWK, tpDPoPProofInput{
			htm: http.MethodPost, htu: agentRevocationResource, jti: f.jti("evaluate"),
			ath: base64.RawURLEncoding.EncodeToString(sum[:]), iat: time.Now().UTC(),
		}))
		form.Set("htm", http.MethodPost)
		form.Set("htu", agentRevocationResource)
	}
	_, body := f.evaluate(t, evalResourceServerClient, evalResourceServerSecret, form)
	return body
}

func (f *grantFixture) exchange(t *testing.T, subject, downstream, grant string, proved bool) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"}, "subject_token": {subject},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"}, "resource": {downstream},
	}
	if grant != "" {
		form.Set("token_exchange_grant", grant)
	}
	request := httptest.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(evalResourceServerClient, evalResourceServerSecret)
	if proved {
		request.Header.Set("DPoP", tpDPoPProof(t, f.gatewayKey, f.gatewayJWK, tpDPoPProofInput{htm: http.MethodPost, htu: tokenEndpoint, jti: f.jti("exchange"), iat: time.Now().UTC()}))
	}
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	body := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder, body
}

func requireInvalidGrant(t *testing.T, recorder *httptest.ResponseRecorder, body map[string]any) {
	t.Helper()
	if recorder.Code != http.StatusBadRequest || body["error"] != "invalid_grant" || body["access_token"] != nil {
		t.Fatalf("status=%d body=%v, want invalid_grant", recorder.Code, body)
	}
}

func unverifiedClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{}
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

// 評価が許可した DPoP の Agent のトークンだけが、ゲートウェイの鍵へ束縛し直される。
//
//spec:covers REQ-OAUTH2-079, REQ-OAUTH2-056, REQ-OAUTH2-048: 許可の発行と一回限りの消費、ゲートウェイの鍵の cnf、短い有効期間、act の連鎖、再束縛の記録
func TestGatewayRebindsAPermittedAgentToken(t *testing.T) {
	f := newGrantFixture(t)
	body := f.evaluateForExchange(t, f.token, exchangeDownstream, true)
	grant, _ := body["token_exchange_grant"].(string)
	if body["decision"] != true || grant == "" {
		t.Fatalf("evaluate: %v", body)
	}

	recorder, issued := f.exchange(t, f.token, exchangeDownstream, grant, true)
	if recorder.Code != http.StatusOK || issued["token_type"] != "DPoP" {
		t.Fatalf("exchange status=%d body=%v", recorder.Code, issued)
	}
	if expires, _ := issued["expires_in"].(float64); expires <= 0 || expires > 60 {
		t.Fatalf("expires_in = %v", issued["expires_in"])
	}
	claims := unverifiedClaims(t, issued["access_token"].(string))
	if cnf, _ := claims["cnf"].(map[string]any); cnf["jkt"] != f.gatewayJKT {
		t.Fatalf("cnf = %v, want the gateway key", claims["cnf"])
	}
	if claims["aud"] != exchangeDownstream {
		t.Fatalf("aud = %v", claims["aud"])
	}
	act, _ := claims["act"].(map[string]any)
	if act["sub"] != evalResourceServerClient {
		t.Fatalf("act = %v, want the gateway outermost", claims["act"])
	}
	if chain, _ := claims["agent_chain"].([]any); len(chain) != 1 || chain[0] != "A1" {
		t.Fatalf("agent_chain = %v", claims["agent_chain"])
	}
	var exchanged *oauthdomain.TokenExchanged
	for _, event := range f.s.Events.All() {
		if e, ok := event.(*oauthdomain.TokenExchanged); ok {
			exchanged = e
		}
	}
	if exchanged == nil || !exchanged.SenderConstraintRebound {
		t.Fatalf("TokenExchanged = %+v", exchanged)
	}

	recorder, again := f.exchange(t, f.token, exchangeDownstream, grant, true)
	requireInvalidGrant(t, recorder, again)
}

// 盗んだトークンを持ち込むゲートウェイは、評価の証明を示せず、許可を得られない。
//
//spec:covers REQ-OAUTH2-056, REQ-OAUTH2-079: 許可なしの再束縛、証明のない評価、宣言のない宛先、許可の流用、証明のない交換の拒否と、拒否が許可を消費しないこと
func TestStolenTokenCannotBeRebound(t *testing.T) {
	f := newGrantFixture(t)

	t.Run("exchange without a grant", func(t *testing.T) {
		recorder, body := f.exchange(t, f.token, exchangeDownstream, "", true)
		requireInvalidGrant(t, recorder, body)
	})
	t.Run("evaluation without the agent proof issues no grant", func(t *testing.T) {
		body := f.evaluateForExchange(t, f.token, exchangeDownstream, false)
		requireReason(t, body, oauthdomain.ReasonSenderProofInvalid)
		if body["token_exchange_grant"] != nil {
			t.Fatalf("grant issued: %v", body)
		}
	})
	t.Run("undeclared destination", func(t *testing.T) {
		body := f.evaluateForExchange(t, f.token, exchangeOtherDownstream, true)
		requireReason(t, body, oauthdomain.ReasonTokenExchangeNotPermitted)
		if body["token_exchange_grant"] != nil {
			t.Fatalf("grant issued: %v", body)
		}
	})
	t.Run("made up grant", func(t *testing.T) {
		recorder, body := f.exchange(t, f.token, exchangeDownstream, "made-up", true)
		requireInvalidGrant(t, recorder, body)
	})

	grant, _ := f.evaluateForExchange(t, f.token, exchangeDownstream, true)["token_exchange_grant"].(string)
	if grant == "" {
		t.Fatal("no grant")
	}
	t.Run("grant used for another destination", func(t *testing.T) {
		recorder, body := f.exchange(t, f.token, exchangeOtherDownstream, grant, true)
		requireInvalidGrant(t, recorder, body)
	})
	t.Run("grant used with another subject", func(t *testing.T) {
		other := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
		recorder, body := f.exchange(t, other, exchangeDownstream, grant, true)
		requireInvalidGrant(t, recorder, body)
	})
	t.Run("grant without a proof cannot become bearer", func(t *testing.T) {
		recorder, body := f.exchange(t, f.token, exchangeDownstream, grant, false)
		requireInvalidGrant(t, recorder, body)
	})
	t.Run("refused attempts did not consume the grant", func(t *testing.T) {
		recorder, body := f.exchange(t, f.token, exchangeDownstream, grant, true)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status=%d body=%v", recorder.Code, body)
		}
	})
}

// 発行済みの許可は失効を越えない。失効の判定は通常の交換と同じ経路を通る。
//
//spec:covers REQ-OAUTH2-079: subject_token の失効と Agent の停止の後は、評価が許可を発行せず、発行済みの許可でも交換できない
func TestGrantDoesNotSurviveRevocation(t *testing.T) {
	t.Run("agent killed after the grant", func(t *testing.T) {
		f := newGrantFixture(t)
		grant, _ := f.evaluateForExchange(t, f.token, exchangeDownstream, true)["token_exchange_grant"].(string)
		if grant == "" {
			t.Fatal("no grant")
		}
		f.admin(t, http.MethodPost, "/api/admin/v1/agents/A1/kill", nil)
		recorder, body := f.exchange(t, f.token, exchangeDownstream, grant, true)
		requireInvalidGrant(t, recorder, body)
		evaluated := f.evaluateForExchange(t, f.token, exchangeDownstream, true)
		if evaluated["decision"] != false || evaluated["token_exchange_grant"] != nil {
			t.Fatalf("evaluate after kill: %v", evaluated)
		}
	})
	t.Run("subject token revoked after the grant", func(t *testing.T) {
		f := newGrantFixture(t)
		grant, _ := f.evaluateForExchange(t, f.token, exchangeDownstream, true)["token_exchange_grant"].(string)
		if status, body := f.post(t, "revoke", agentClientID("A1"), url.Values{"token": {f.token}, "token_type_hint": {"access_token"}}); status != http.StatusOK {
			t.Fatalf("revoke status=%d body=%v", status, body)
		}
		recorder, body := f.exchange(t, f.token, exchangeDownstream, grant, true)
		requireInvalidGrant(t, recorder, body)
	})
	t.Run("downstream token dies with the agent", func(t *testing.T) {
		f := newGrantFixture(t)
		grant, _ := f.evaluateForExchange(t, f.token, exchangeDownstream, true)["token_exchange_grant"].(string)
		recorder, issued := f.exchange(t, f.token, exchangeDownstream, grant, true)
		if recorder.Code != http.StatusOK {
			t.Fatalf("exchange status=%d body=%v", recorder.Code, issued)
		}
		f.admin(t, http.MethodPost, "/api/admin/v1/agents/A1/kill", nil)
		f.requireInactive(t, "exchanged token", issued["access_token"].(string))
	})
}
