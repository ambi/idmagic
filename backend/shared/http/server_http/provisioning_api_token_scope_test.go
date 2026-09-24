package server_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// テナントの一致しない API アクセストークンは、アプリケーションの接続、テナントの接続、
// プロビジョニングタスクの操作のどれでも 401 invalid_token で拒否する。管理発行トークンはリクエスト先テナントで
// 照合され、見つからなければ RFC 7662 の非開示に従って無効として扱う。403 にすると
// 「このトークン自体は有効だが、このテナントでは使えない」ことを提示者へ伝えてしまう。
// 参照と変更、接続とプロビジョニングタスクの組を並べ、一部の経路だけが配線された実装を見分ける。
//
//spec:covers EX-PROVISIONING-001-03: トークンのテナントとリクエスト先のテナントが一致しないとき、接続とプロビジョニングタスクの操作を 401 の InvalidAccessTokenError で拒否し、接続を作らない。
func TestForeignTenantProvisioningTokenIsRejectedAsInvalid(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithProvisioning())
	foreignRead, _ := s.IssueApiToken(t, stack.OtherRealm, domain.ScopeProvisioningRead)
	foreignWrite, _ := s.IssueApiToken(t, stack.OtherRealm, domain.ScopeProvisioningWrite)

	// テナントの接続一覧とプロビジョニングタスク一覧の参照は届かない。
	assertProvisioningInvalidToken(t, provisioningAPIRequest(t, s, http.MethodGet, "/api/admin/v1/provisioning/connections", foreignRead, ""))
	assertProvisioningInvalidToken(t, provisioningAPIRequest(t, s, http.MethodGet, "/api/admin/v1/applications/app-1/provisioning/tasks", foreignRead, ""))

	// アプリケーションの接続の登録と全件再同期は届かず、接続は作られない。
	assertProvisioningInvalidToken(t, provisioningAPIRequest(t, s, http.MethodPost, "/api/admin/v1/applications/app-1/provisioning", foreignWrite,
		`{"base_url":"https://downstream.example/scim/v2","credential":{"auth_method":"bearer_token","bearer_token":"secret"}}`))
	assertProvisioningInvalidToken(t, provisioningAPIRequest(t, s, http.MethodPost, "/api/admin/v1/applications/app-1/provisioning/full-resync", foreignWrite, ""))
	connection, err := s.ProvisioningConnections.Find(context.Background(), s.TenantID(t, "default"), "app-1")
	if err != nil || connection != nil {
		t.Fatalf("default tenant connection after foreign refusal = (%+v, %v), want none", connection, err)
	}

	// 対照: 同じテナントで発行したトークンなら接続一覧を参照できる。拒否したのがテナントの
	// 食い違いであって、スタックの配線そのものではないと示す。
	sameRealmRead, _ := s.IssueApiToken(t, "default", domain.ScopeProvisioningRead)
	if allowed := provisioningAPIRequest(t, s, http.MethodGet, "/api/admin/v1/provisioning/connections", sameRealmRead, ""); allowed.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 同一テナントの参照が status=%d body=%s", allowed.Code, allowed.Body.String())
	}
}

func assertProvisioningInvalidToken(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", recorder.Code, recorder.Body.String())
	}
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem body %s: %v", recorder.Body.String(), err)
	}
	if problem.Type != "urn:idmagic:error:invalid_token" {
		t.Fatalf("problem type=%q, want invalid_token", problem.Type)
	}
	if got := recorder.Header().Get("WWW-Authenticate"); !strings.Contains(got, `Bearer error="invalid_token"`) {
		t.Fatalf("WWW-Authenticate=%q, want invalid_token challenge", got)
	}
}
