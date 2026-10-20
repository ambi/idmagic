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

	federationmemory "github.com/ambi/idmagic/backend/authentication/internal/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/internal/federation/protocol_oidc"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-AUTHENTICATION-041: 入力支援の取得は接続を保存せず、その間の無効化と編集を保持する。
//verify:operation PreviewIdentityProviderMetadata: issuer の JSON 要求に対して検証済みの候補を 200 で返し、保存済みの接続を変えない。
func TestMetadataPreviewPreservesConcurrentConfigurationChanges(t *testing.T) {
	var connections *federationmemory.ConnectionRepository
	var edited *federationdomain.IdentityProviderConnection
	client := oidcprotocol.Client{HTTPClient: &http.Client{Transport: fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("upstream method=%s, want GET", r.Method)
		}
		var body string
		switch r.URL.String() {
		case "https://preview.example/.well-known/openid-configuration":
			var err error
			edited, err = connections.Find(t.Context(), tenancydomain.DefaultTenantID, "preview-provider")
			if err != nil || edited == nil {
				t.Fatalf("before edit=(%+v, %v)", edited, err)
			}
			edited.Disable(time.Date(2026, 10, 21, 1, 0, 0, 0, time.UTC))
			edited.DisplayName = "edited while fetching"
			edited.SecretReference = "replacement-secret"
			edited.Issuer = "https://replacement.example"
			edited.AuthorizationEndpoint = "https://replacement.example/auth"
			if err := connections.Save(t.Context(), edited); err != nil {
				t.Fatal(err)
			}
			body = `{"issuer":"https://preview.example","authorization_endpoint":"https://preview.example/new-auth","token_endpoint":"https://preview.example/new-token","jwks_uri":"https://preview.example/jwks"}`
		case "https://preview.example/jwks":
			body = `{"keys":[{"kty":"RSA","alg":"RS256","kid":"preview","n":"AQAB","e":"AQAB"}]}`
		default:
			t.Fatalf("unexpected upstream URL %s", r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
	e, repos := newAdminServer(t, &client)
	connections = repos.Connections
	now := time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)
	original := &federationdomain.IdentityProviderConnection{
		ID: "preview-provider", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Preview provider",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://preview.example", ClientID: "preview-client", SecretReference: "original-secret",
		AuthorizationEndpoint: "https://preview.example/auth", TokenEndpoint: "https://preview.example/token", JWKSURI: "https://preview.example/jwks",
		ClaimMapping: federationdomain.ClaimMapping{Subject: "sub", Username: "email"}, LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := connections.Save(t.Context(), original); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, adminRequest(t, http.MethodPost, "/api/admin/v1/identity-providers/metadata-preview", map[string]string{"issuer": original.Issuer}))
	got := federationContractJSON(t, rec, http.StatusOK)
	want := map[string]any{
		"issuer": "https://preview.example", "authorization_endpoint": "https://preview.example/new-auth",
		"token_endpoint": "https://preview.example/new-token", "jwks_uri": "https://preview.example/jwks",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("preview=%v, want %v", got, want)
	}
	stored, err := connections.Find(t.Context(), tenancydomain.DefaultTenantID, original.ID)
	if err != nil || edited == nil || !reflect.DeepEqual(stored, edited) {
		t.Fatalf("stored=(%+v, %v), want concurrent edits only: %+v", stored, err, edited)
	}
}

//spec:covers REQ-AUTHENTICATION-041: 不正な入力と検証できない上流は 400 で拒否し、接続を保存しない。
//verify:operation PreviewIdentityProviderMetadata: 不正な入力は invalid_request、取得先と issuer と鍵の検証失敗は metadata_invalid の 400 Problem Details を返す。
func TestMetadataPreviewRejectsInvalidInputsWithoutSaving(t *testing.T) {
	for _, tc := range []struct {
		name, body, document, keys, code string
	}{
		{name: "invalid JSON", body: "{", code: "invalid_request"},
		{name: "missing issuer", body: `{}`, code: "invalid_request"},
		{name: "blank issuer", body: `{"issuer":" "}`, code: "invalid_request"},
		{name: "HTTP issuer", body: `{"issuer":"http://idp.example"}`, code: "metadata_invalid"},
		{name: "local issuer", body: `{"issuer":"https://127.0.0.1"}`, code: "metadata_invalid"},
		{name: "issuer mismatch", body: `{"issuer":"https://preview.example"}`, document: `{"issuer":"https://different.example"}`, code: "metadata_invalid"},
		{
			name: "bad key", body: `{"issuer":"https://preview.example"}`,
			document: `{"issuer":"https://preview.example","authorization_endpoint":"https://preview.example/auth","token_endpoint":"https://preview.example/token","jwks_uri":"https://preview.example/jwks"}`,
			keys:     `{"keys":[]}`, code: "metadata_invalid",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := oidcprotocol.Client{HTTPClient: &http.Client{Transport: fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
				body := tc.document
				if r.URL.Path == "/jwks" {
					body = tc.keys
				}
				if body == "" {
					t.Fatalf("unexpected upstream request: %s", r.URL)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			e, repos := newAdminServer(t, &client)
			req := adminRequest(t, http.MethodPost, "/api/admin/v1/identity-providers/metadata-preview", nil)
			req.Body = io.NopCloser(strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			var problem struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil || rec.Code != http.StatusBadRequest || problem.Type != "urn:idmagic:error:"+tc.code || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/problem+json") {
				t.Fatalf("status=%d headers=%v body=%s, want 400 %s", rec.Code, rec.Header(), rec.Body.String(), tc.code)
			}
			stored, err := repos.Connections.List(t.Context(), tenancydomain.DefaultTenantID)
			if err != nil || len(stored) != 0 {
				t.Fatalf("stored=(%+v, %v), want no connection", stored, err)
			}
		})
	}
}
