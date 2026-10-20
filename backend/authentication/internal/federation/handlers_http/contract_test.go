package handlers_http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	oidcprotocol "github.com/ambi/idmagic/backend/authentication/internal/federation/protocol_oidc"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

func federationContractJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if rec.Code != status || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body == nil {
		t.Fatalf("JSON=%v err=%v", body, err)
	}
	return body
}

func federationContractConnection(t *testing.T, body map[string]any, name string) {
	t.Helper()
	if id, ok := body["id"].(string); !ok || id == "" {
		t.Fatalf("connection id=%v", body)
	}
	if body["tenant_id"] != tenancydomain.DefaultTenantID || body["display_name"] != name || body["protocol"] != "oidc" || body["status"] != "disabled" || body["issuer"] != "https://contract.example" || body["client_id"] != "contract-client" || body["client_secret_configured"] != false || body["jit_provisioning"] != false || body["linking_policy"] != "none" {
		t.Fatalf("connection=%v", body)
	}
	if _, present := body["secret_reference"]; present {
		t.Fatal("connection disclosed secret material")
	}
	for _, key := range []string{"saml_signing_certificates", "allowed_email_domains"} {
		if _, present := body[key]; present {
			t.Fatalf("%s=%v", key, body[key])
		}
	}
	for _, key := range []string{"created_at", "updated_at"} {
		value, ok := body[key].(string)
		if parsed, err := time.Parse(time.RFC3339Nano, value); !ok || err != nil || parsed.IsZero() {
			t.Fatalf("%s=%v", key, body[key])
		}
	}
}

//verify:operation CreateIdentityProviderConnection: 接続の作成、更新、mapping preview、metadata refresh の HTTP 応答と保存結果
//verify:operation UpdateIdentityProviderConnection: 接続の作成、更新、mapping preview、metadata refresh の HTTP 応答と保存結果
//verify:operation PreviewIdentityProviderMapping: 接続の作成、更新、mapping preview、metadata refresh の HTTP 応答と保存結果
//verify:operation PreviewIdentityProviderMetadata: issuer の要求から検証済みのエンドポイントを返し、保存済みの接続を変えない。
func TestIdentityProviderAdminHTTPContracts(t *testing.T) {
	client := oidcprotocol.Client{HTTPClient: &http.Client{Transport: fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.String() {
		case "https://contract.example/.well-known/openid-configuration":
			body = `{"issuer":"https://contract.example","authorization_endpoint":"https://contract.example/new-auth","token_endpoint":"https://contract.example/new-token","jwks_uri":"https://contract.example/jwks"}`
		case "https://contract.example/jwks":
			body = `{"keys":[{"kty":"RSA","alg":"RS256","kid":"contract","n":"AQAB","e":"AQAB"}]}`
		default:
			t.Fatalf("unexpected upstream URL %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	e, repos := newAdminServer(t, &client)
	send := func(method, path string, body any) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, adminRequest(t, method, path, body))
		return rec
	}
	input := map[string]any{
		"display_name": "Contract provider", "protocol": "oidc", "issuer": "https://contract.example", "client_id": "contract-client",
		"authorization_endpoint": "https://contract.example/auth", "token_endpoint": "https://contract.example/token", "jwks_uri": "https://contract.example/jwks",
		"claim_mapping":  map[string]string{"subject": "sub", "username": "email", "email": "email", "email_verified": "email_verified", "name": "name"},
		"linking_policy": "none", "jit_provisioning": false, "saml_signing_certificates": []string{}, "allowed_email_domains": []string{},
	}
	created := federationContractJSON(t, send(http.MethodPost, "/api/admin/v1/identity-providers", input), http.StatusCreated)
	federationContractConnection(t, created, "Contract provider")
	id := created["id"].(string)
	input["display_name"] = "Contract renamed"
	updated := federationContractJSON(t, send(http.MethodPut, "/api/admin/v1/identity-providers/"+id, input), http.StatusOK)
	federationContractConnection(t, updated, "Contract renamed")
	if updated["id"] != id || updated["created_at"] != created["created_at"] || !reflect.DeepEqual(updated["claim_mapping"], created["claim_mapping"]) {
		t.Fatalf("updated=%v", updated)
	}
	preview := federationContractJSON(t, send(http.MethodPost, "/api/admin/v1/identity-providers/"+id+"/mapping-preview", map[string]any{"claims": map[string]any{"sub": "external-sub", "email": "alice@example.test", "email_verified": true, "name": "Alice"}}), http.StatusOK)
	if !reflect.DeepEqual(preview["preview"], map[string]any{"subject": "external-sub", "username": "alice@example.test", "email": "alice@example.test", "email_verified": true, "name": "Alice"}) {
		t.Fatalf("preview=%v", preview)
	}
	refreshed := federationContractJSON(t, send(http.MethodPost, "/api/admin/v1/identity-providers/metadata-preview", map[string]string{"issuer": "https://contract.example"}), http.StatusOK)
	if len(refreshed) != 4 || refreshed["issuer"] != "https://contract.example" || refreshed["authorization_endpoint"] != "https://contract.example/new-auth" || refreshed["token_endpoint"] != "https://contract.example/new-token" || refreshed["jwks_uri"] != "https://contract.example/jwks" {
		t.Fatalf("refreshed=%v", refreshed)
	}
	stored, err := repos.Connections.Find(testing_tenant.Default(t.Context()), tenancydomain.DefaultTenantID, id)
	if err != nil || stored == nil || stored.AuthorizationEndpoint != updated["authorization_endpoint"] || stored.MetadataRefreshedAt != nil {
		t.Fatalf("stored=%v err=%v", stored, err)
	}
}

//verify:operation CreateIdentityProviderConnection: CSRF 拒否の 403 Problem Details
//verify:operation UpdateIdentityProviderConnection: CSRF 拒否の 403 Problem Details
//verify:operation PreviewIdentityProviderMapping: CSRF 拒否の 403 Problem Details
//verify:operation PreviewIdentityProviderMetadata: CSRF 拒否の 403 Problem Details
func TestIdentityProviderAdminHTTPContractsRefuseCSRF(t *testing.T) {
	e, _ := newAdminServer(t, nil)
	for _, request := range []struct{ method, path string }{
		{http.MethodPost, "/api/admin/v1/identity-providers"},
		{http.MethodPut, "/api/admin/v1/identity-providers/missing"},
		{http.MethodPost, "/api/admin/v1/identity-providers/missing/mapping-preview"},
		{http.MethodPost, "/api/admin/v1/identity-providers/metadata-preview"},
	} {
		req := adminRequest(t, request.method, request.path, nil)
		req.Header.Del("X-Csrf-Token")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden || rec.Header().Get("Content-Type") != "application/problem+json" {
			t.Fatalf("%s: status=%d body=%s", request.path, rec.Code, rec.Body.String())
		}
		var problem map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil || problem["type"] != "urn:idmagic:error:csrf_failed" || problem["status"] != float64(http.StatusForbidden) {
			t.Fatalf("Problem=%v err=%v", problem, err)
		}
	}
}
