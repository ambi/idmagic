package server_http_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ambi/idmagic/backend/mcpgateway"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	downstreamClient = "downstream-resource-server"
	downstreamSecret = "downstream-secret"
)

// exchangeGatewayFixture は、自身で提示トークンを評価する下流の手前に置いたゲートウェイである。
type exchangeGatewayFixture struct {
	*grantFixture
	gateway      *httptest.Server
	decisions    atomic.Int32
	permitted    atomic.Int32
	lastUpstream atomic.Value
}

func newExchangeGatewayFixture(t *testing.T) *exchangeGatewayFixture {
	t.Helper()
	f := &exchangeGatewayFixture{grantFixture: newGrantFixture(t)}
	ctx := context.Background()
	now := time.Now().UTC()
	hash := oauthdomain.HashClientSecret(downstreamSecret)
	if err := f.s.Clients.Save(ctx, &oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: downstreamClient, ClientType: spec.ClientConfidential,
		TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic, ClientSecretHash: &hash,
		GrantTypes: []spec.GrantType{spec.GrantClientCredentials}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, exchangeDownstream)
	if err != nil || server == nil {
		t.Fatalf("downstream: %v %v", server, err)
	}
	server.ClientID = downstreamClient
	server.AccessRules = []oauthdomain.ResourceAccessRule{{ResourceType: "document", Relation: "viewer", Scopes: []string{"tools:read"}}}
	if err := f.s.McpResourceServers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		f.lastUpstream.Store(r.Header.Clone())
		f.decisions.Add(1)
		scheme, token, _ := strings.Cut(r.Header.Get("Authorization"), " ")
		form := url.Values{
			"token": {token}, "resource_type": {"document"}, "resource_id": {"d1"}, "relation": {"viewer"},
			"dpop_proof": {r.Header.Get("DPoP")}, "htm": {http.MethodPost}, "htu": {exchangeDownstream},
		}
		_, body := f.evaluate(t, downstreamClient, downstreamSecret, form)
		if scheme != "DPoP" || body["decision"] != true {
			t.Errorf("downstream evaluation scheme=%q body=%v", scheme, body)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		f.permitted.Add(1)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":7,"result":{}}`)
	}))
	t.Cleanup(upstream.Close)

	dir := t.TempDir()
	write := func(name string, content []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	idmagic := httptest.NewTLSServer(issuerFront(t, f.s.Echo, key))
	t.Cleanup(idmagic.Close)
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	config := write("config.yaml", fmt.Appendf(nil, `listen: 127.0.0.1:0
resource: %s
upstream: %s/
idmagic:
  realm_url: %s/realms/default
  ca_file: %s
  client_id: %s
  client_secret_file: %s
downstream:
  resource: %s
  dpop_key_file: %s
mapping:
  tools:
    read_document: { resource_type: document, relation: viewer, resource_id_argument: document_id }
    other_document: { resource_type: document, relation: other, resource_id_argument: document_id }
`, agentRevocationResource, upstream.URL, idmagic.URL,
		write("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idmagic.Certificate().Raw})),
		evalResourceServerClient, write("secret", []byte(evalResourceServerSecret+"\n")),
		exchangeDownstream, write("key.pem", pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))))
	audit := &bytes.Buffer{}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("gateway audit:\n%s", audit)
		}
	})
	gateway, err := mcpgateway.Load(config, audit)
	if err != nil {
		t.Fatal(err)
	}
	f.gateway = httptest.NewServer(gateway.Handler())
	t.Cleanup(f.gateway.Close)
	return f
}

func (f *exchangeGatewayFixture) call(t *testing.T) *http.Response {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"read_document","arguments":{"document_id":"d1"},` +
		`"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.gateway.URL+"/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "DPoP "+f.token)
	request.Header.Set("DPoP", f.agentProof(t))
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", "read_document")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// 下流は自身宛のゲートウェイの鍵のトークンを通常の評価で受け、元のトークンを受け取らない。呼び出しごとに別のトークンになる。
//
//spec:covers REQ-MCPGATEWAY-008, REQ-OAUTH2-079: DPoP の Agent から下流の評価の許可まで、呼び出しごとの差し替えと元のトークンの不到達
func TestMcpGatewayForwardsADownstreamToken(t *testing.T) {
	f := newExchangeGatewayFixture(t)
	if response := f.call(t); response.StatusCode != http.StatusOK {
		t.Fatalf("first call status = %d", response.StatusCode)
	}
	first := f.lastUpstream.Load().(http.Header).Get("Authorization")
	if strings.Contains(first, f.token) {
		t.Fatal("the agent token reached the upstream")
	}
	if response := f.call(t); response.StatusCode != http.StatusOK {
		t.Fatalf("second call status = %d", response.StatusCode)
	}
	if second := f.lastUpstream.Load().(http.Header).Get("Authorization"); second == first {
		t.Fatal("the downstream token was reused")
	}
	if f.permitted.Load() != 2 {
		t.Fatalf("downstream permits = %d", f.permitted.Load())
	}
}

// 宛先の宣言がなくなった規則では、許可にならず下流は呼ばれない。
//
//spec:covers REQ-MCPGATEWAY-008: 交換の許可を得られない呼び出しは上流へ送らない
func TestMcpGatewayDoesNotForwardWithoutAGrant(t *testing.T) {
	f := newExchangeGatewayFixture(t)
	ctx := context.Background()
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("resource server: %v %v", server, err)
	}
	server.AccessRules[0].TokenExchangeResources = nil
	if err := f.s.McpResourceServers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	if response := f.call(t); response.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if f.decisions.Load() != 0 {
		t.Fatal("the upstream was called")
	}
}

// issuerFront は、試験の TLS の URL 宛のゲートウェイの証明を、製品が照合する発行者の URL へ読み替える。
// 製品は htu を発行者から導くが、試験では発行者の名前を解決できない。同じ鍵で署名し直すので、鍵の束縛は本物のまま確かめる。
// 発行者の URL の違いだけはここで隠れるので、htu の値は exchange_http の単体テストが確かめる。
func issuerFront(t *testing.T, next http.Handler, key *ecdsa.PrivateKey) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proof := r.Header.Get("DPoP"); proof != "" {
			claims := jwt.MapClaims{}
			parsed, _, err := jwt.NewParser().ParseUnverified(proof, claims)
			if err != nil {
				t.Errorf("gateway proof: %v", err)
				return
			}
			claims["htu"] = stack.Issuer + r.URL.Path
			token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
			token.Header = parsed.Header
			signed, err := token.SignedString(key)
			if err != nil {
				t.Errorf("sign: %v", err)
				return
			}
			r.Header.Set("DPoP", signed)
		}
		next.ServeHTTP(w, r)
	})
}
