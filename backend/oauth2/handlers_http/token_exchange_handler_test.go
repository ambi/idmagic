package handlers_http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"

	"github.com/labstack/echo/v5"
)

const (
	exchClientID     = "exch-client"
	exchClientSecret = "exch-client-secret"
)

// newTokenExchangeServer は client_credentials と token-exchange を許可した
// confidential client を持つ最小サーバを返す。subject_token は client_credentials で発行する。
func newTokenExchangeServer(t *testing.T, options ...func(*httpadapter.Deps)) string {
	t.Helper()
	clientRepo := oauth2memory.NewClientRepository()
	secretHash := domain.HashClientSecret(exchClientSecret)
	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: exchClientID, ClientSecretHash: &secretHash, ClientType: spec.ClientConfidential,
		GrantTypes: []spec.GrantType{
			spec.GrantClientCredentials, spec.GrantTokenExchange,
		},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
		Scope:                   "read write",
		FapiProfile:             domain.FapiNone,
		CreatedAt:               time.Now().UTC(),
	})
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	tokenIssuer := tokens_jose.NewJWTSigner("http://test", keyStore)
	resourceServers := oauth2memory.NewMcpResourceServerRepository()
	resourceServers.Seed(&domain.McpResourceServer{
		ID: "rs-orders", Resource: "https://api.example/orders", Name: "Orders API",
		Scopes: []string{"read", "write"}, State: domain.McpResourceServerActive,
	})
	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantRepo.Save(context.Background(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		DisplayName: "Default", Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatalf("tenant repo: %v", err)
	}
	e := echo.New()
	deps := httpadapter.Deps{
		Issuer: "http://test", TenantRepo: tenantRepo,
		OAuth2: oauth2.Module{
			ClientRepo: clientRepo, ConsentRepo: oauth2memory.NewConsentRepository(), RefreshStore: oauth2memory.NewRefreshTokenStore(),
			McpResourceServerRepo: resourceServers,
		},
		UserRepo: usermemory.NewUserRepository(),
		KeyStore: keyStore, TokenIssuer: tokenIssuer, TokenIntrospector: tokenIssuer,
	}
	for _, option := range options {
		option(&deps)
	}
	httpadapter.Register(e, deps)
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	// bare path はテナントの正規ロケーションではないので、
	// default テナントの prefix まで含めた base を返す。
	return srv.URL + "/realms/default"
}

type recordingWorkloadVerifier struct {
	called bool
}

func (v *recordingWorkloadVerifier) VerifyWorkloadToken(_ context.Context, tenantID, subjectToken string, _ time.Time) (*workloaddomain.WorkloadIdentityGrant, error) {
	v.called = true
	if tenantID != tenancydomain.DefaultTenantID || subjectToken != "external-jwt-svid" {
		return nil, fmt.Errorf("unexpected workload verification input: tenant=%q token=%q", tenantID, subjectToken)
	}
	return &workloaddomain.WorkloadIdentityGrant{AgentID: "checkout-bot", ClientID: exchClientID, TrustBundleID: "prod-cluster", BindingID: "checkout-binding"}, nil
}

// REQ-WORKLOADIDENTITY-001: Token Exchange の HTTP 入口が workload verifier の結果を Agent 資格情報へ変換する。
func TestTokenExchangeIssuesWorkloadCredential(t *testing.T) {
	verifier := &recordingWorkloadVerifier{}
	base := newTokenExchangeServer(t, func(deps *httpadapter.Deps) {
		deps.OAuth2.WorkloadVerifier = verifier
	})
	resource := "https://api.example/orders"
	status, body := postToken(t, base, url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {"external-jwt-svid"},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"},
		"resource":           {resource},
	})
	if status != http.StatusOK || !verifier.called {
		t.Fatalf("workload exchange status=%d called=%v body=%v", status, verifier.called, body)
	}
	issued, _ := body["access_token"].(string)
	if issued == "" {
		t.Fatalf("workload exchange returned no access token: %v", body)
	}

	request, _ := http.NewRequest(http.MethodPost, base+"/introspect", strings.NewReader(url.Values{"token": {issued}}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(exchClientID, exchClientSecret)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var introspection map[string]any
	if err := json.NewDecoder(response.Body).Decode(&introspection); err != nil {
		t.Fatal(err)
	}
	if introspection["active"] != true || introspection["sub"] != exchClientID {
		t.Fatalf("workload credential introspection = %#v", introspection)
	}
}

func postToken(t *testing.T, base string, form url.Values) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, base+"/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(exchClientID, exchClientSecret)
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

func mintSubjectToken(t *testing.T, base string) string {
	t.Helper()
	status, body := postToken(t, base, url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {"read write"},
	})
	if status != http.StatusOK {
		t.Fatalf("client_credentials status=%d body=%v", status, body)
	}
	tok, _ := body["access_token"].(string)
	if tok == "" {
		t.Fatalf("no access_token in %v", body)
	}
	return tok
}

func TestTokenExchangeIssuesDelegatedToken(t *testing.T) {
	base := newTokenExchangeServer(t)
	subject := mintSubjectToken(t, base)

	resource := "https://api.example/orders"
	status, body := postToken(t, base, url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {subject},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"resource":           {resource},
	})
	if status != http.StatusOK {
		t.Fatalf("token-exchange status=%d body=%v", status, body)
	}
	if got := body["issued_token_type"]; got != "urn:ietf:params:oauth:token-type:access_token" {
		t.Fatalf("issued_token_type=%v", got)
	}
	issued, _ := body["access_token"].(string)
	if issued == "" {
		t.Fatalf("no access_token in %v", body)
	}

	// 発行トークンを introspect して aud / act を検証する。
	intReq, _ := http.NewRequest(http.MethodPost, base+"/introspect",
		strings.NewReader(url.Values{"token": {issued}}.Encode()))
	intReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	intReq.SetBasicAuth(exchClientID, exchClientSecret)
	intResp, err := http.DefaultClient.Do(intReq)
	if err != nil {
		t.Fatalf("POST /introspect: %v", err)
	}
	defer intResp.Body.Close()
	var introspection map[string]any
	intBody, _ := io.ReadAll(intResp.Body)
	if err := json.Unmarshal(intBody, &introspection); err != nil {
		t.Fatalf("introspect json: %v", err)
	}
	if active, _ := introspection["active"].(bool); !active {
		t.Fatalf("issued token not active: %v", introspection)
	}
	aud, _ := introspection["aud"].([]any)
	if len(aud) != 1 || aud[0] != resource {
		t.Fatalf("aud=%v, want [%s]", introspection["aud"], resource)
	}
	act, _ := introspection["act"].(map[string]any)
	if act == nil || act["sub"] != exchClientID {
		t.Fatalf("act=%v, want act.sub=%s", introspection["act"], exchClientID)
	}
	if mode := introspection["delegation_mode"]; mode != "autonomous" {
		t.Fatalf("delegation_mode=%v, want autonomous", mode)
	}
}

func TestTokenExchangeRejectsInvalidSubjectToken(t *testing.T) {
	base := newTokenExchangeServer(t)
	status, body := postToken(t, base, url.Values{
		"grant_type":    {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token": {"not-a-real-token"},
		"resource":      {"https://api.example/orders"},
	})
	if status == http.StatusOK {
		t.Fatalf("invalid subject_token was accepted: %v", body)
	}
}

func TestTokenExchangeRejectsMissingResource(t *testing.T) {
	base := newTokenExchangeServer(t)
	subject := mintSubjectToken(t, base)
	status, body := postToken(t, base, url.Values{
		"grant_type":    {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token": {subject},
	})
	if status == http.StatusOK {
		t.Fatalf("missing resource was accepted: %v", body)
	}
}
