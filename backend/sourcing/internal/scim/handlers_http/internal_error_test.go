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

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
	scimhttp "github.com/ambi/idmagic/backend/sourcing/internal/scim/handlers_http"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	"github.com/labstack/echo/v5"
)

type failingScimRepository struct {
	scimports.ScimRepository
	failure error
}

type failingScimAuthenticator struct{ failure error }

func (f failingScimAuthenticator) Authenticate(context.Context, string) (apitokendomain.Principal, error) {
	return apitokendomain.Principal{}, f.failure
}

//spec:covers REQ-SYSTEM-034: 認証ポートの内部障害は500で診断し、既知の無効トークンは401のまま、ラップの詳細を公開しない。
func TestScimAuthenticationErrors(t *testing.T) {
	for _, tc := range []struct {
		failure error
		status  int
	}{
		{errors.New("private-marker: token repository SQL error"), http.StatusInternalServerError},
		{fmt.Errorf("private-marker: %w", apitokendomain.ErrInvalidToken), http.StatusUnauthorized},
	} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			var logs bytes.Buffer
			previous := logging.Default()
			logging.SetDefault(logging.New(&logs, slog.LevelError, "test", ""))
			t.Cleanup(func() { logging.SetDefault(previous) })
			e := echo.New()
			scimhttp.RegisterRoutes(e.Group("", testing_tenant.ResolveDefault), scimhttp.Deps{
				ApiTokenAuthenticator: failingScimAuthenticator{failure: tc.failure},
			})
			request := httptest.NewRequest(http.MethodGet, "/scim/v2/ServiceProviderConfig", http.NoBody)
			request.Header.Set("Authorization", "Bearer access-token")
			request = request.WithContext(logging.ContextWithRequestID(request.Context(), "scim-auth-request"))
			response := httptest.NewRecorder()
			e.ServeHTTP(response, request)
			if response.Code != tc.status || strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), "private-marker") {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if tc.status == http.StatusInternalServerError && (!strings.Contains(logs.String(), "private-marker") || !strings.Contains(logs.String(), "scim-auth-request")) {
				t.Fatalf("missing diagnostic: %s", logs.String())
			}
			if tc.status == http.StatusUnauthorized && response.Header().Get("WWW-Authenticate") != `Bearer error="invalid_token"` {
				t.Fatalf("missing Bearer challenge: %v", response.Header())
			}
		})
	}
}

func (r failingScimRepository) FindUserRefByScimID(context.Context, string, string) (*scimports.ScimUserRef, error) {
	return nil, r.failure
}

func (r failingScimRepository) FindGroupRefByScimID(context.Context, string, string) (*scimports.ScimGroupRef, error) {
	return nil, r.failure
}

func (r failingScimRepository) FindUserRefByUserID(context.Context, string, string) (*scimports.ScimUserRef, error) {
	return nil, r.failure
}

func (r failingScimRepository) FindGroupRefByGroupID(context.Context, string, string) (*scimports.ScimGroupRef, error) {
	return nil, r.failure
}

//spec:covers REQ-SYSTEM-034: SCIMの読取、更新、削除と一覧の内部障害は安全な500となり、原因を記録し、UserとGroupを変えない。
func TestScimInternalErrors(t *testing.T) {
	for _, failure := range []error{
		errors.New("private-marker: SQL constraint scim_refs_pkey"),
		errors.New("private-marker: runtime error: index out of range"),
		errors.New("private-marker: cipher: message authentication failed"),
		errors.New("private-marker: goroutine 1 [running]:\n/internal/scim.go:42"),
		fmt.Errorf("read: %w", errors.New("private-marker: dial tcp 10.0.0.5:5432")),
	} {
		t.Run(failure.Error(), func(t *testing.T) {
			var logs bytes.Buffer
			previous := logging.Default()
			logging.SetDefault(logging.New(&logs, slog.LevelError, "test", ""))
			t.Cleanup(func() { logging.SetDefault(previous) })
			h := newScimHarness()
			token := issueAllScimToken(t, h.apiTokens)
			userID := createScimUserForTest(t, h.echo, token, map[string]any{"userName": "alice", "active": true})
			groupID := createScimGroupForTest(t, h, token, map[string]any{"displayName": "Workforce"})
			for _, target := range []struct{ collection, id string }{{"Users", userID}, {"Groups", groupID}} {
				path := "/scim/v2/" + target.collection + "/" + target.id
				_, before := doScimGet(t, h.echo, token, path)
				for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, "LIST"} {
					t.Run(target.collection+"/"+method, func(t *testing.T) {
						h.usecases.ScimRepo = failingScimRepository{ScimRepository: h.scimRepo, failure: failure}
						requestPath, requestMethod := path, method
						if method == "LIST" {
							requestPath, requestMethod = "/scim/v2/"+target.collection, http.MethodGet
						}
						request := httptest.NewRequest(requestMethod, requestPath, strings.NewReader(`{"userName":"changed","displayName":"Changed","Operations":[{"op":"replace","path":"displayName","value":"Changed"}]}`))
						request.Header.Set("Authorization", "Bearer "+token)
						request.Header.Set("Content-Type", "application/scim+json")
						request = request.WithContext(logging.ContextWithRequestID(request.Context(), "scim-request"))
						response := httptest.NewRecorder()
						h.echo.ServeHTTP(response, request)
						var body map[string]any
						if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || response.Code != http.StatusInternalServerError || body["status"] != "500" || response.Header().Get("Content-Type") != "application/scim+json" {
							t.Fatalf("status=%d body=%s: %v", response.Code, response.Body.String(), err)
						}
						if schemas, ok := body["schemas"].([]any); !ok || len(schemas) != 1 || schemas[0] != "urn:ietf:params:scim:api:messages:2.0:Error" {
							t.Fatalf("invalid SCIM envelope: %v", body)
						}
						if strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), "private-marker") || !strings.Contains(logs.String(), "private-marker") || !strings.Contains(logs.String(), "scim-request") {
							t.Errorf("response=%s logs=%s", response.Body.String(), logs.String())
						}
						h.usecases.ScimRepo = h.scimRepo
						_, after := doScimGet(t, h.echo, token, path)
						if !reflect.DeepEqual(before, after) {
							t.Fatalf("resource changed: before=%v after=%v", before, after)
						}
					})
				}
			}
		})
	}
}
