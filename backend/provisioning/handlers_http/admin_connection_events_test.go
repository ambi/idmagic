package handlers_http_test

// 主要ユースケース追跡: REQ-PROVISIONING-002、REQ-PROVISIONING-011、REQ-PROVISIONING-014。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	provisioningmemory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	provisioninghttp "github.com/ambi/idmagic/backend/provisioning/handlers_http"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// 管理 API の発行ポートは support.Deps.Emit であり、本番では EventSink と監査へ書く。
// ここでは同じ位置に記録器を置き、監査が保存するのと同じワイヤ表現で中身を確かめる。
//
//spec:covers EX-PROVISIONING-002-01: 管理 API で接続を登録すると ProvisioningConnectionRegistered が発行され、接続テストで capabilities がキャッシュされる。
func TestAdminConnectionAPIEmitsLifecycleEvents(t *testing.T) {
	now := time.Now().UTC()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	connections := provisioningmemory.NewProvisioningConnectionRepository()
	var events []spec.DomainEvent
	e := echo.New()
	e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
	provisioninghttp.RegisterRoutes(e.Group(""), provisioninghttp.Deps{
		Issuer: "http://idp.test",
		Emit:   func(event spec.DomainEvent) { events = append(events, event) },
		Authenticator: &support.Authenticator{
			UserRepo: users, AuthnResolver: authusecases.DemoHeaderResolver{},
		},
		ConnectionRepo: connections,
		NewTargetClient: func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error) {
			return discoveryTarget{}, nil
		},
	})
	request := adminRequester(t, e)
	const base = "/api/admin/v1/applications/app-1/provisioning"

	if got := request(http.MethodPost, base, map[string]any{
		"base_url":   "https://downstream.example/scim/v2",
		"credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "secret"},
	}); got.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(http.MethodPost, base+"/test", nil); got.Code != http.StatusOK {
		t.Fatalf("test status=%d body=%s", got.Code, got.Body.String())
	}
	saved, _ := connections.Find(context.Background(), tenancydomain.DefaultTenantID, "app-1")
	if saved == nil || saved.Capabilities == nil {
		t.Fatalf("connection after test = %+v, want cached capabilities", saved)
	}
	if got := request(http.MethodPatch, base, map[string]any{
		"credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "rotated"},
	}); got.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s", got.Code, got.Body.String())
	}
	saved, _ = connections.Find(context.Background(), tenancydomain.DefaultTenantID, "app-1")
	if err := saved.Quarantine("too many failures", now); err != nil {
		t.Fatal(err)
	}
	if err := connections.Update(context.Background(), saved, nil); err != nil {
		t.Fatal(err)
	}
	if got := request(http.MethodPost, base+"/resume", nil); got.Code != http.StatusOK {
		t.Fatalf("resume status=%d body=%s", got.Code, got.Body.String())
	}
	if got := request(http.MethodPost, base+"/resume", nil); got.Code == http.StatusOK {
		t.Fatalf("second resume status=%d, want a refusal", got.Code)
	}

	want := []map[string]any{
		{"type": "ProvisioningConnectionRegistered", "tenantId": tenancydomain.DefaultTenantID, "applicationId": "app-1"},
		{"type": "ProvisioningCredentialRotated", "tenantId": tenancydomain.DefaultTenantID, "applicationId": "app-1", "credentialId": saved.Credential.CredentialID},
		{"type": "ProvisioningConnectionQuarantineCleared", "tenantId": tenancydomain.DefaultTenantID, "applicationId": "app-1"},
	}
	if len(events) != len(want) {
		t.Fatalf("events = %d, want %d (%v)", len(events), len(want), eventTypes(events))
	}
	for i, event := range events {
		wire, err := spec.MarshalDomainEvent(event)
		if err != nil {
			t.Fatalf("MarshalDomainEvent(%T) error = %v", event, err)
		}
		var fields map[string]any
		if err := json.Unmarshal(wire, &fields); err != nil {
			t.Fatal(err)
		}
		for key, value := range want[i] {
			if fields[key] != value {
				t.Errorf("events[%d] %s = %v, want %v (wire %s)", i, key, fields[key], value, wire)
			}
		}
	}
}

func adminRequester(t *testing.T, e *echo.Echo) func(method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	const csrf = "csrf-token"
	return func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var payload []byte
		if body != nil {
			var err error
			if payload, err = json.Marshal(body); err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Demo-Sub", "admin")
		if method != http.MethodGet {
			req.Header.Set("Origin", "http://idp.test")
			req.Header.Set("X-Csrf-Token", csrf)
			req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: csrf})
		}
		response := httptest.NewRecorder()
		e.ServeHTTP(response, req)
		return response
	}
}

func eventTypes(events []spec.DomainEvent) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.EventType())
	}
	return types
}
