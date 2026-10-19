package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-APPLICATION-006, REQ-APPLICATION-007: 所有者の資格情報を管理用の項目だけへ写し、保存上の識別子とハッシュを応答へ混ぜない。
func TestOIDCSecretResponsesExposeOnlyManagementMetadata(t *testing.T) {
	f := newSecretExampleFixture(t)
	applicationID, _, _ := f.createApplication(t)
	path := "/api/admin/v1/applications/" + applicationID

	detail := adminJSON(t, f.e, http.MethodGet, path, f.csrf, f.cookie, nil)
	var body struct {
		OIDC struct {
			Credentials []json.RawMessage `json:"secret_credentials"`
		} `json:"oidc"`
	}
	if detail.Code != http.StatusOK || json.Unmarshal(detail.Body.Bytes(), &body) != nil || len(body.OIDC.Credentials) != 1 {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	assertManagementCredential(t, body.OIDC.Credentials[0])

	issued := adminJSON(t, f.e, http.MethodPost, path+"/oidc/client-secrets", f.csrf, f.cookie, map[string]any{"expires_in_days": 30})
	var issue struct {
		Credential  json.RawMessage   `json:"credential"`
		Credentials []json.RawMessage `json:"credentials"`
	}
	if issued.Code != http.StatusCreated || json.Unmarshal(issued.Body.Bytes(), &issue) != nil || len(issue.Credentials) != 2 {
		t.Fatalf("issue status=%d body=%s", issued.Code, issued.Body.String())
	}
	credentialID := assertManagementCredential(t, issue.Credential)
	for _, credential := range issue.Credentials {
		assertManagementCredential(t, credential)
	}

	for _, operation := range []struct {
		method, path string
		body         any
	}{
		{http.MethodDelete, path + "/oidc/client-secrets/" + credentialID, nil},
		{http.MethodPost, path + "/oidc/rotate-secret", map[string]any{"grace_days": 1}},
	} {
		response := adminJSON(t, f.e, operation.method, operation.path, f.csrf, f.cookie, operation.body)
		var result struct {
			Credentials []json.RawMessage `json:"credentials"`
		}
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &result) != nil || len(result.Credentials) == 0 {
			t.Fatalf("%s status=%d body=%s", operation.path, response.Code, response.Body.String())
		}
		for _, credential := range result.Credentials {
			assertManagementCredential(t, credential)
		}
	}
}

func assertManagementCredential(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for key := range fields {
		switch key {
		case "credential_id", "created_at", "expires_at", "revoked_at", "status":
		default:
			t.Fatalf("credential exposed storage field %q: %s", key, raw)
		}
	}
	var id, status string
	if json.Unmarshal(fields["credential_id"], &id) != nil || id == "" || json.Unmarshal(fields["status"], &status) != nil || status == "" {
		t.Fatalf("credential lacks management identity or status: %s", raw)
	}
	return id
}

//spec:covers REQ-APPLICATION-002: 管理の HTTP が受け取った公開鍵と TLS の設定を、所有者の登録へ渡す。
func TestOIDCCatalogRegistrationKeepsPublicKeySettings(t *testing.T) {
	f := newSecretExampleFixture(t)
	uri, dn := "https://rp.example/jwks", "CN=client"
	response := adminJSON(t, f.e, http.MethodPost, "/api/admin/v1/applications", f.csrf, f.cookie, map[string]any{
		"name": "Key client", "type": "oidc", "client_type": "confidential",
		"redirect_uris":              []string{"https://rp.example/callback"},
		"token_endpoint_auth_method": "private_key_jwt", "jwks_uri": uri, "tls_client_auth_subject_dn": dn,
	})
	var body struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &body) != nil || body.ClientID == "" || body.ClientSecret != "" {
		t.Fatalf("registration status=%d body=%s", response.Code, response.Body.String())
	}
	client, err := f.clients.FindByID(t.Context(), tenancydomain.DefaultTenantID, body.ClientID)
	if err != nil || client == nil || client.JwksURI == nil || *client.JwksURI != uri || client.TlsClientAuthSubjectDN == nil || *client.TlsClientAuthSubjectDN != dn {
		t.Fatalf("client=%+v err=%v", client, err)
	}
}

//spec:covers REQ-APPLICATION-007: クォータの拒否で Application を作らず、クライアント作成と Application 作成の既存の確定順を保つ。
func TestOIDCCatalogQuotaRefusalPreservesTheExistingCommitOrder(t *testing.T) {
	for _, tc := range []struct {
		name        string
		quota       tenancydomain.TenantQuota
		wantClients int
	}{
		{"client quota", tenancydomain.TenantQuota{OAuth2Clients: new(0)}, 0},
		{"application quota", tenancydomain.TenantQuota{Applications: new(0)}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			quotas := tenancy.NewMemoryModule().QuotaRepo
			if err := quotas.SetQuota(t.Context(), tenancydomain.DefaultTenantID, &tc.quota); err != nil {
				t.Fatal(err)
			}
			f := newSecretExampleFixtureWithQuota(t, quotas)
			response := adminJSON(t, f.e, http.MethodPost, "/api/admin/v1/applications", f.csrf, f.cookie, map[string]any{"name": "Quota client", "type": "service"})
			assertApplicationProblem(t, response, http.StatusUnprocessableEntity, "quota_exceeded")
			listed := adminJSON(t, f.e, http.MethodGet, "/api/admin/v1/applications", f.csrf, f.cookie, nil)
			var body struct {
				Applications []json.RawMessage `json:"applications"`
			}
			if listed.Code != http.StatusOK || json.Unmarshal(listed.Body.Bytes(), &body) != nil || len(body.Applications) != 0 {
				t.Fatalf("refusal created an application: status=%d body=%s", listed.Code, listed.Body.String())
			}
			clients, err := f.clients.FindAll(t.Context(), tenancydomain.DefaultTenantID)
			if err != nil || len(clients) != tc.wantClients {
				t.Fatalf("clients=%+v err=%v, want %d", clients, err, tc.wantClients)
			}
			usage, err := quotas.GetUsage(t.Context(), tenancydomain.DefaultTenantID)
			if err != nil || usage.Applications != 0 || usage.OAuth2Clients != tc.wantClients {
				t.Fatalf("usage=%+v err=%v", usage, err)
			}
			if f.countEvents("QuotaExceeded") != 1 || f.countEvents("ApplicationCreated") != 0 || f.countEvents("AdminOAuth2ClientCreated") != tc.wantClients {
				t.Fatalf("events=%v", f.eventTypes())
			}
		})
	}
}

//spec:covers REQ-APPLICATION-007: 関連付けたクライアントを削除し、両方のクォータを一度だけ解放して再作成できる。
func TestOIDCCatalogDeleteReleasesQuotaAndRemovesTheOwnedClient(t *testing.T) {
	quotas := tenancy.NewMemoryModule().QuotaRepo
	if err := quotas.SetQuota(t.Context(), tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Applications: new(1), OAuth2Clients: new(1)}); err != nil {
		t.Fatal(err)
	}
	f := newSecretExampleFixtureWithQuota(t, quotas)
	applicationID, clientID, _ := f.createApplication(t)
	response := adminJSON(t, f.e, http.MethodDelete, "/api/admin/v1/applications/"+applicationID, f.csrf, f.cookie, nil)
	assertApplicationNoContent(t, response)
	client, err := f.clients.FindByID(t.Context(), tenancydomain.DefaultTenantID, clientID)
	if err != nil || client != nil {
		t.Fatalf("deleted client=%+v err=%v", client, err)
	}
	usage, err := quotas.GetUsage(t.Context(), tenancydomain.DefaultTenantID)
	if err != nil || usage.Applications != 0 || usage.OAuth2Clients != 0 {
		t.Fatalf("usage=%+v err=%v", usage, err)
	}
	f.createApplication(t)
}
