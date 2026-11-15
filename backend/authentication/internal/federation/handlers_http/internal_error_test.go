package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/internal/federation/handlers_http"
	federationports "github.com/ambi/idmagic/backend/authentication/internal/federation/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

type failingFederationRepository struct {
	federationports.ConnectionRepository
	failure error
}

func (r *failingFederationRepository) Save(ctx context.Context, connection *federationdomain.IdentityProviderConnection) error {
	if r.failure != nil {
		return r.failure
	}
	return r.ConnectionRepository.Save(ctx, connection)
}

//spec:covers REQ-SYSTEM-023: 作成と更新の保存障害は500で詳細を公開せず、相関ID付きの原因を残し、保存済みの設定を変えない。
func TestFederationInternalErrors(t *testing.T) {
	for _, failure := range []error{
		errors.New("private-marker: SQL constraint identity_provider_connections_pkey"),
		errors.New("private-marker: runtime error: index out of range"),
		errors.New("private-marker: cipher: message authentication failed"),
		errors.New("private-marker: goroutine 1 [running]:\n/internal/federation.go:42"),
		fmt.Errorf("save: %w", errors.New("private-marker: dial tcp 10.0.0.5:5432")),
	} {
		for _, method := range []string{http.MethodPost, http.MethodPut} {
			t.Run(method+"/"+failure.Error(), func(t *testing.T) {
				var logs bytes.Buffer
				repo := &failingFederationRepository{}
				e, repos := newAdminServer(t, nil, func(d *federationhttp.Deps, e *echo.Echo) {
					repo.ConnectionRepository = d.Broker.Connections
					d.Broker.Connections = repo
					e.HTTPErrorHandler = support.ErrorHandler(logging.New(&logs, slog.LevelError, "test", ""), nil)
				})
				body := validFederationInput()
				path := "/api/admin/v1/identity-providers"
				if method == http.MethodPut {
					created := httptest.NewRecorder()
					e.ServeHTTP(created, adminRequest(t, http.MethodPost, path, body))
					var connection federationdomain.IdentityProviderConnection
					if err := json.Unmarshal(created.Body.Bytes(), &connection); err != nil || created.Code != http.StatusCreated {
						t.Fatalf("create: %d %s: %v", created.Code, created.Body.String(), err)
					}
					path += "/" + connection.ID
				}
				before, _ := repos.Connections.List(t.Context(), tenancydomain.DefaultTenantID)
				body["display_name"] = "Changed"
				body["secret_reference"] = "replacement-secret"
				repo.failure = failure
				request := adminRequest(t, method, path, body)
				request = request.WithContext(logging.ContextWithRequestID(request.Context(), "federation-request"))
				response := httptest.NewRecorder()
				e.ServeHTTP(response, request)
				var problem support.Problem
				if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil || response.Code != http.StatusInternalServerError || problem.Type != "urn:idmagic:error:internal_server_error" {
					t.Fatalf("status=%d body=%s: %v", response.Code, response.Body.String(), err)
				}
				if strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), "private-marker") || !strings.Contains(logs.String(), "private-marker") || !strings.Contains(logs.String(), "federation-request") {
					t.Fatalf("response=%s logs=%s", response.Body.String(), logs.String())
				}
				after, _ := repos.Connections.List(t.Context(), tenancydomain.DefaultTenantID)
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("saved state changed: before=%+v after=%+v", before, after)
				}
			})
		}
	}
}

func validFederationInput() map[string]any {
	return map[string]any{
		"display_name": "Workforce", "protocol": "oidc", "issuer": "https://accounts.example.com",
		"client_id": "client-1", "authorization_endpoint": "https://accounts.example.com/authorize",
		"token_endpoint": "https://accounts.example.com/token", "jwks_uri": "https://accounts.example.com/jwks",
		"claim_mapping": map[string]string{"subject": "sub", "username": "email"}, "linking_policy": "none",
	}
}

//spec:covers REQ-SYSTEM-023: 入力検証の拒否はSaveの障害と区別し、説明を残して設定を保存しない。
func TestFederationInvalidInputRemainsActionable(t *testing.T) {
	e, repos := newAdminServer(t, nil)
	body := validFederationInput()
	body["display_name"] = ""
	response := httptest.NewRecorder()
	e.ServeHTTP(response, adminRequest(t, http.MethodPost, "/api/admin/v1/identity-providers", body))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "display name") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if all, err := repos.Connections.List(t.Context(), tenancydomain.DefaultTenantID); err != nil || len(all) != 0 {
		t.Fatalf("invalid connection saved: %v %v", all, err)
	}
}
