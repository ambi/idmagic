package server_http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// テナントの停止、再開、正規ロケーションの切替は、管理発行の API アクセストークンでは行えない
// (REQ-TENANCY-011、REQ-TENANCY-027)。テナントの制御面の経路は API トークンを照合しないので、
// tenants:write を持つトークンでも 401 invalid_token で拒否され、テナントは変わらない。
//
//spec:covers REQ-TENANCY-011, REQ-TENANCY-027: tenants:write の API アクセストークンでのテナントの無効化、再開、正規ロケーションの切替が 401 invalid_token で拒否され、テナントの状態と正規ロケーションが変わらないことを固定する。
func TestTenantLifecycleRefusesApiTokens(t *testing.T) {
	s := stack.New(t, stack.WithApiTokens())
	now := time.Now().UTC()
	target := &tenancydomain.Tenant{
		ID: "api-token-target", Realm: "api-token-target", Status: tenancydomain.TenantStatusActive,
		EndpointStyle: tenancydomain.TenantEndpointStylePath, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.Tenants.Save(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	token, _ := s.IssueApiToken(t, tenancydomain.DefaultRealm, apitokendomain.ScopeTenantsWrite)

	for _, call := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/admin/v1/tenants/" + target.ID + "/disable", ""},
		{http.MethodPost, "/api/admin/v1/tenants/" + target.ID + "/enable", ""},
		{http.MethodPut, "/api/admin/v1/tenants/" + target.ID + "/endpoint-style", `{"endpoint_style":"subdomain"}`},
	} {
		request := httptest.NewRequestWithContext(t.Context(), call.method, stack.Issuer+"/realms/default"+call.path, strings.NewReader(call.body))
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		s.Echo.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), "urn:idmagic:error:invalid_token") {
			t.Fatalf("%s %s status=%d body=%s, want 401 invalid_token", call.method, call.path, recorder.Code, recorder.Body.String())
		}
	}
	stored, err := s.Tenants.FindByID(context.Background(), target.ID)
	if err != nil || stored == nil {
		t.Fatalf("target tenant: %v", err)
	}
	if stored.Status != tenancydomain.TenantStatusActive || stored.EndpointStyle != tenancydomain.TenantEndpointStylePath {
		t.Fatalf("tenant=%+v, want Active の path のまま", stored)
	}
}
