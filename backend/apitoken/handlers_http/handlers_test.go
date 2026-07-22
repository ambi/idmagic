package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/apitoken/db_memory"
	apitokenhttp "github.com/ambi/idmagic/backend/apitoken/handlers_http"
	"github.com/ambi/idmagic/backend/apitoken/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func newHandler(t *testing.T) (*echo.Echo, *usecases.Service) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	service := usecases.New(db_memory.NewRepository())
	deps := support.Deps{Issuer: "http://idp.test"}
	authenticator := &support.Authenticator{UserRepo: users, AuthnResolver: authusecases.DemoHeaderResolver{}}
	e := echo.New()
	apitokenhttp.RegisterRoutes(e.Group("", deps.ResolveDefaultTenant), apitokenhttp.Deps{
		Deps: deps, Authenticator: authenticator, Service: service,
	})
	return e, service
}

func request(t *testing.T, e *echo.Echo, method, path string, body any, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if admin {
		req.Header.Set("X-Demo-Sub", "admin")
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// SCL scenario: 管理者はAPIアクセストークンを発行・失効できる。
func TestAdminApiTokenLifecycle(t *testing.T) {
	e, service := newHandler(t)
	if rec := request(t, e, http.MethodGet, "/api/admin/api-tokens", nil, false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d body=%s", rec.Code, rec.Body.String())
	}

	issued := request(t, e, http.MethodPost, "/api/admin/api-tokens", map[string]any{
		"description": "SCIM", "scopes": []string{"scim:users:read"}, "expiry_days": 7,
	}, true)
	if issued.Code != http.StatusCreated || issued.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("issue status=%d cache=%q body=%s", issued.Code, issued.Header().Get("Cache-Control"), issued.Body.String())
	}
	var issueBody struct {
		Token string `json:"token"`
		Meta  struct {
			ID     string   `json:"id"`
			Scopes []string `json:"scopes"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &issueBody); err != nil {
		t.Fatal(err)
	}
	if len(issueBody.Token) != len("idmagic_pat_")+64 || issueBody.Meta.ID == "" || len(issueBody.Meta.Scopes) != 1 {
		t.Fatalf("issue response = %+v", issueBody)
	}

	listed := request(t, e, http.MethodGet, "/api/admin/api-tokens", nil, true)
	if listed.Code != http.StatusOK || bytes.Contains(listed.Body.Bytes(), []byte("token_hash")) || bytes.Contains(listed.Body.Bytes(), []byte(issueBody.Token)) {
		t.Fatalf("list status=%d body=%s", listed.Code, listed.Body.String())
	}

	revoked := request(t, e, http.MethodDelete, "/api/admin/api-tokens/"+issueBody.Meta.ID, nil, true)
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s", revoked.Code, revoked.Body.String())
	}
	if _, err := service.Authenticate(context.Background(), issueBody.Token); err == nil {
		t.Fatal("revoked token authenticated")
	}
}

func TestIssueApiTokenRejectsInvalidRequest(t *testing.T) {
	e, _ := newHandler(t)
	for _, body := range []map[string]any{
		{"description": "bad expiry", "scopes": []string{"scim:users:read"}, "expiry_days": 0},
		{"description": "bad scope", "scopes": []string{"scim:unknown"}, "expiry_days": 7},
	} {
		rec := request(t, e, http.MethodPost, "/api/admin/api-tokens", body, true)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}
