package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

type applicationSecretCredential struct {
	CredentialID string  `json:"credential_id"`
	CreatedAt    string  `json:"created_at"`
	ExpiresAt    *string `json:"expires_at"`
	RevokedAt    *string `json:"revoked_at"`
	Status       string  `json:"status"`
}

func createSecretApplication(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie) string {
	t.Helper()
	response := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications", csrf, cookie, map[string]any{
		"name": "Secret App", "type": "oidc", "redirect_uris": []string{"https://secret.example/callback"},
		"client_type": "confidential", "token_endpoint_auth_method": "client_secret_post",
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Application struct {
			ID string `json:"id"`
		} `json:"application"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Application.ID
}

func TestAdminApplicationClientSecretLifecycle(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createSecretApplication(t, e, csrf, cookie)

	detail := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
	var detailBody struct {
		OIDC struct {
			Credentials []applicationSecretCredential `json:"secret_credentials"`
		} `json:"oidc"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &detailBody); err != nil {
		t.Fatal(err)
	}
	if len(detailBody.OIDC.Credentials) != 1 || detailBody.OIDC.Credentials[0].Status != "Active" {
		t.Fatalf("initial credentials=%#v", detailBody.OIDC.Credentials)
	}
	legacyID := detailBody.OIDC.Credentials[0].CredentialID

	issued := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications/"+applicationID+"/oidc/client-secrets", csrf, cookie, map[string]any{
		"expires_in_days": 90,
	})
	if issued.Code != http.StatusCreated {
		t.Fatalf("issue status=%d body=%s", issued.Code, issued.Body.String())
	}
	if !strings.Contains(issued.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("issue cache-control=%q", issued.Header().Get("Cache-Control"))
	}
	var issuedBody struct {
		ClientSecret string                        `json:"client_secret"`
		Credential   applicationSecretCredential   `json:"credential"`
		Credentials  []applicationSecretCredential `json:"credentials"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &issuedBody); err != nil {
		t.Fatal(err)
	}
	if issuedBody.ClientSecret == "" || issuedBody.Credential.Status != "Active" ||
		issuedBody.Credential.ExpiresAt == nil || len(issuedBody.Credentials) != 2 {
		t.Fatalf("issued body=%#v", issuedBody)
	}

	limited := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications/"+applicationID+"/oidc/client-secrets", csrf, cookie, map[string]any{
		"expires_in_days": 90,
	})
	if limited.Code != http.StatusConflict {
		t.Fatalf("limit status=%d body=%s", limited.Code, limited.Body.String())
	}

	revoked := adminJSON(t, e, http.MethodDelete, "/api/admin/v1/applications/"+applicationID+"/oidc/client-secrets/"+legacyID, csrf, cookie, nil)
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke status=%d body=%s", revoked.Code, revoked.Body.String())
	}
	var revokedBody struct {
		Credentials []applicationSecretCredential `json:"credentials"`
	}
	if err := json.Unmarshal(revoked.Body.Bytes(), &revokedBody); err != nil {
		t.Fatal(err)
	}
	if len(revokedBody.Credentials) != 2 || revokedBody.Credentials[0].Status != "Revoked" {
		t.Fatalf("revoked credentials=%#v", revokedBody.Credentials)
	}

	repeated := adminJSON(t, e, http.MethodDelete, "/api/admin/v1/applications/"+applicationID+"/oidc/client-secrets/"+legacyID, csrf, cookie, nil)
	if repeated.Code != http.StatusOK {
		t.Fatalf("repeated revoke status=%d body=%s", repeated.Code, repeated.Body.String())
	}
}
