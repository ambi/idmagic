package server_http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/apitoken/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

func provisioningAPIRequest(t *testing.T, s *stack.Stack, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	return provisioningAPIRequestInRealm(t, s, "default", method, path, token, body)
}

func provisioningAPIRequestInRealm(t *testing.T, s *stack.Stack, realm, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "https://idp.example/realms/"+realm+path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, req)
	return recorder
}

//spec:covers EX-PROVISIONING-001-01: provisioning:read は接続一覧を参照でき、provisioning:write は接続を登録できる。
func TestProvisioningScopesAllowOnlyTheirNamedOperations(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithProvisioning())
	readToken, _ := s.IssueApiToken(t, "default", domain.ScopeProvisioningRead)
	read := provisioningAPIRequest(t, s, http.MethodGet, "/api/admin/v1/provisioning/connections", readToken, "")
	if read.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s, want 200", read.Code, read.Body.String())
	}

	writeToken, _ := s.IssueApiToken(t, "default", domain.ScopeProvisioningWrite)
	created := provisioningAPIRequest(t, s, http.MethodPost, "/api/admin/v1/applications/app-1/provisioning", writeToken,
		`{"base_url":"https://downstream.example/scim/v2","credential":{"auth_method":"bearer_token","bearer_token":"secret"}}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("write status=%d body=%s, want 201", created.Code, created.Body.String())
	}
	connection, err := s.ProvisioningConnections.Find(context.Background(), s.TenantID(t, "default"), "app-1")
	if err != nil || connection == nil {
		t.Fatalf("stored connection = (%+v, %v), want registered connection", connection, err)
	}
}

//spec:covers EX-PROVISIONING-001-02: provisioning:read だけのトークンは接続登録を拒否し、接続を作らない。
func TestProvisioningReadScopeCannotRegisterAConnection(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithProvisioning())
	readToken, _ := s.IssueApiToken(t, "default", domain.ScopeProvisioningRead)
	refused := provisioningAPIRequest(t, s, http.MethodPost, "/api/admin/v1/applications/app-1/provisioning", readToken,
		`{"base_url":"https://downstream.example/scim/v2","credential":{"auth_method":"bearer_token","bearer_token":"secret"}}`)
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	connection, err := s.ProvisioningConnections.Find(context.Background(), s.TenantID(t, "default"), "app-1")
	if err != nil || connection != nil {
		t.Fatalf("stored connection after refusal = (%+v, %v), want none", connection, err)
	}
}

// wi-37560: 実装は AccessDeniedError ではなく invalid_token を返す。規範の受け入れテストは同項目で持つ。
func TestForeignTenantProvisioningTokenIsRejectedAsInvalid(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithProvisioning())
	foreignToken, _ := s.IssueApiToken(t, stack.OtherRealm, domain.ScopeProvisioningWrite)
	refused := provisioningAPIRequest(t, s, http.MethodPost, "/api/admin/v1/applications/app-1/provisioning", foreignToken,
		`{"base_url":"https://downstream.example/scim/v2","credential":{"auth_method":"bearer_token","bearer_token":"secret"}}`)
	if refused.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", refused.Code, refused.Body.String())
	}
	connection, err := s.ProvisioningConnections.Find(context.Background(), s.TenantID(t, "default"), "app-1")
	if err != nil || connection != nil {
		t.Fatalf("default tenant connection after foreign refusal = (%+v, %v), want none", connection, err)
	}
}
