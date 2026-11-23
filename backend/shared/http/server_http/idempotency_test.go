package server_http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-SYSTEM-036: 正式な管理 API への同じキーの再送は同じ JWT とメタデータを返し、発行記録は一件だけになる。
func TestAdminIdempotencyReplaysTheSameToken(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithIdempotency())
	ctx := s.RealmContext(t, "default")
	token, _, err := s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{TenantID: s.TenantID(t, "default"), ClientID: "admin-portal"},
		Sub:    stack.AdminUserID, Scopes: []string{"idmagic.admin"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	post := func(body string) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
			stack.Issuer+"/realms/default/api/admin/v1/api-tokens", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", `"token-creation"`)
		recorder := httptest.NewRecorder()
		s.Echo.ServeHTTP(recorder, request)
		return recorder
	}
	first := post(`{"description":"automation","scopes":["users:read"],"expiry_days":1}`)
	second := post(`{ "description": "automation", "scopes": ["users:read"], "expiry_days": 1 }`)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("first=%d %s second=%d %s", first.Code, first.Body, second.Code, second.Body)
	}
	var issued struct {
		Token string `json:"token"`
		Meta  struct {
			ID string `json:"id"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &issued); err != nil || issued.Token == "" || issued.Meta.ID == "" {
		t.Fatalf("token response=%s error=%v", first.Body, err)
	}
	if !bytes.Equal(first.Body.Bytes(), second.Body.Bytes()) {
		t.Fatal("retry returned a different token or metadata")
	}
	stored, err := s.ApiTokens.List(ctx, s.TenantID(t, "default"))
	if err != nil || len(stored) != 1 || stored[0].ID != issued.Meta.ID {
		t.Fatalf("stored tokens=%v error=%v, want the one returned token", stored, err)
	}
	conflict := post(`{"description":"other","scopes":["users:read"],"expiry_days":1}`)
	if conflict.Code != http.StatusUnprocessableEntity || !strings.Contains(conflict.Body.String(), "urn:idmagic:error:idempotency_key_reused") {
		t.Fatalf("mismatched retry=%d %s", conflict.Code, conflict.Body)
	}
	user, err := s.Users.FindBySub(ctx, stack.AdminUserID)
	if err != nil {
		t.Fatal(err)
	}
	user.Lifecycle.Status = idmdomain.UserStatusDisabled
	if err := s.Users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	refused := post(`{"description":"automation","scopes":["users:read"],"expiry_days":1}`)
	if refused.Code != http.StatusUnauthorized || strings.Contains(refused.Body.String(), issued.Token) {
		t.Fatalf("disabled principal replay=%d %s", refused.Code, refused.Body)
	}
}

//spec:covers REQ-SYSTEM-036: 対象の全操作で認可後、デコードと作用の前にキーの照合へ到達する。
func TestEveryDeclaredAdminIdempotencyOperationUsesTheLedger(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens(), stack.WithIdempotency())
	ctx := s.RealmContext(t, "default")
	tenantID := s.TenantID(t, "default")
	token, _, err := s.Signer.SignAccessToken(ctx, oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{TenantID: s.TenantID(t, "default"), ClientID: "admin-portal"},
		Sub:    stack.AdminUserID, Scopes: []string{"idmagic.admin"}, ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	operations := []string{
		"IssueApiToken", "CreateAdminOAuth2Client", "CreateAdminApplication", "CreateApplicationCategory",
		"IssueApplicationClientSecret", "RotateApplicationClientSecret", "CreateSamlIdentityProviderProfile",
		"CreateIdentityProviderConnection", "IssueMfaEnrollmentBypass", "StartUserCsvExport", "StartGroupCsvExport",
		"StartGroupMemberCsvExport", "ImportAdminUsers", "ImportAdminGroups", "ImportAdminGroupMembers",
	}
	for _, name := range operations {
		t.Run(name, func(t *testing.T) {
			operation, ok := spec.CurrentRuntimeContract().Operation(name)
			if !ok {
				t.Fatal("contract operation is missing")
			}
			if _, _, err := s.Idempotency.Claim(ctx, tenantID, stack.AdminUserID, name, "another-request", time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			path := strings.NewReplacer("{id}", "target", "{user_id}", "target", "{group_id}", "target").Replace(operation.Path)
			request := httptest.NewRequestWithContext(ctx, operation.Method, stack.Issuer+"/realms/default"+path, strings.NewReader(`{}`))
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", `"`+name+`"`)
			response := httptest.NewRecorder()
			s.Echo.ServeHTTP(response, request)
			if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "urn:idmagic:error:idempotency_key_reused") {
				t.Fatalf("operation=%s status=%d body=%s", name, response.Code, response.Body)
			}
		})
	}
}
