package handlers_http_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioninghttp "github.com/ambi/idmagic/backend/provisioning/internal/handlers_http"
	"github.com/ambi/idmagic/backend/provisioning/internal/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

type failingConnectionRepository struct {
	ports.ProvisioningConnectionRepository
	failure error
}

type failingDiscoveryTarget struct {
	discoveryTarget
	failure error
}

func (d failingDiscoveryTarget) Discover(context.Context) (domain.ProvisioningCapabilities, error) {
	return domain.ProvisioningCapabilities{}, d.failure
}

func (r *failingConnectionRepository) Register(ctx context.Context, conn *domain.ProvisioningConnection, secret string) error {
	if r.failure != nil {
		return r.failure
	}
	return r.ProvisioningConnectionRepository.Register(ctx, conn, secret)
}

func (r *failingConnectionRepository) Update(ctx context.Context, conn *domain.ProvisioningConnection, secret *string) error {
	if r.failure != nil {
		return r.failure
	}
	return r.ProvisioningConnectionRepository.Update(ctx, conn, secret)
}

//spec:covers REQ-SYSTEM-023: 接続の登録と更新で保存障害の詳細を公開せず、診断と相関IDを残し、接続と資格情報とイベントを変えない。
func TestProvisioningInternalErrors(t *testing.T) {
	for _, failure := range []error{
		errors.New(`private-marker: SQL constraint provisioning_connections_pkey`),
		errors.New("private-marker: runtime error: invalid memory address"),
		errors.New("private-marker: cipher: message authentication failed"),
		errors.New("private-marker: goroutine 1 [running]:\n/internal/store.go:42"),
		fmt.Errorf("storage: %w", errors.New("private-marker: dial tcp 10.0.0.5:5432")),
	} {
		for _, method := range []string{http.MethodPost, http.MethodPatch} {
			t.Run(method+"/"+failure.Error(), func(t *testing.T) {
				var logs bytes.Buffer
				var events []spec.DomainEvent
				repo := &failingConnectionRepository{}
				request, stored, _ := newProvisioningContractServer(t, func(d *provisioninghttp.Deps, e *echo.Echo) {
					repo.ProvisioningConnectionRepository = d.ConnectionRepo
					d.ConnectionRepo = repo
					d.Emit = func(event spec.DomainEvent) { events = append(events, event) }
					e.HTTPErrorHandler = support.ErrorHandler(logging.New(&logs, slog.LevelError, "test", ""), nil)
					e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
						return func(c *echo.Context) error {
							r := c.Request()
							c.SetRequest(r.WithContext(logging.ContextWithRequestID(r.Context(), "provisioning-request")))
							return next(c)
						}
					})
				})
				if method == http.MethodPatch {
					registerProvisioningConnection(t, request)
				}
				before, _ := stored.Find(t.Context(), tenancydomain.DefaultTenantID, "app-1")
				secretBefore, _ := stored.CredentialSecret(t.Context(), tenancydomain.DefaultTenantID, "app-1")
				eventCount := len(events)
				repo.failure = failure
				response := request("admin", method, provisioningConnectionPath, map[string]any{
					"base_url": "https://changed.example/scim/v2", "credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "replacement"},
				})
				assertProvisioningProblem(t, response, http.StatusInternalServerError, "internal_server_error")
				if strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), "private-marker") {
					t.Fatalf("disclosed error: %s %v", response.Body.String(), response.Header())
				}
				if !strings.Contains(logs.String(), "private-marker") || !strings.Contains(logs.String(), "provisioning-request") {
					t.Fatalf("missing diagnostic: %s", logs.String())
				}
				after, _ := stored.Find(t.Context(), tenancydomain.DefaultTenantID, "app-1")
				secretAfter, _ := stored.CredentialSecret(t.Context(), tenancydomain.DefaultTenantID, "app-1")
				if !reflect.DeepEqual(before, after) || secretBefore != secretAfter || len(events) != eventCount {
					t.Fatalf("state changed: before=%+v after=%+v events=%d", before, after, len(events))
				}
			})
		}
	}
}

//spec:covers REQ-SYSTEM-023: 入力不正と既知の競合は修正可能な説明で拒否し、内部障害へ変えない。
func TestProvisioningKnownRefusals(t *testing.T) {
	request, connections, _ := newProvisioningContractServer(t)
	for _, input := range []map[string]any{
		{"base_url": "http://downstream.example", "credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "secret"}},
		{"base_url": "https://downstream.example", "credential": map[string]any{"auth_method": "bearer_token"}},
	} {
		response := request("admin", http.MethodPost, provisioningConnectionPath, input)
		assertProvisioningProblem(t, response, http.StatusBadRequest, "invalid_request")
		if !strings.Contains(response.Body.String(), "provisioning:") {
			t.Fatalf("missing validation explanation: %s", response.Body.String())
		}
	}
	if all, err := connections.ListAll(t.Context(), tenancydomain.DefaultTenantID); err != nil || len(all) != 0 {
		t.Fatalf("invalid input saved: %v %v", all, err)
	}
	registerProvisioningConnection(t, request)
	assertProvisioningProblem(t, request("admin", http.MethodPost, provisioningConnectionPath, map[string]any{
		"base_url": "https://downstream.example", "credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "secret"},
	}), http.StatusConflict, "provisioning_conflict")
	assertProvisioningProblem(t, request("admin", http.MethodPatch, provisioningConnectionPath, map[string]any{"scope": "unknown"}), http.StatusBadRequest, "invalid_request")
}

//spec:covers REQ-SYSTEM-023, REQ-PROVISIONING-002: 下流の到達性テストの失敗は安全な説明と診断を残し、検出済みの能力と資格情報を変えない。
func TestProvisioningDiscoveryErrorIsNotDisclosed(t *testing.T) {
	var logs bytes.Buffer
	previous := logging.Default()
	logging.SetDefault(logging.New(&logs, slog.LevelError, "test", ""))
	t.Cleanup(func() { logging.SetDefault(previous) })
	request, connections, _ := newProvisioningContractServer(t, func(d *provisioninghttp.Deps, _ *echo.Echo) {
		d.NewTargetClient = func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error) {
			return failingDiscoveryTarget{failure: errors.New("private-marker: dial tcp 10.0.0.5:443")}, nil
		}
	})
	registerProvisioningConnection(t, request)
	before, _ := connections.Find(t.Context(), tenancydomain.DefaultTenantID, "app-1")
	response := request("admin", http.MethodPost, provisioningConnectionPath+"/test", nil)
	body := provisioningContractJSON(t, response, http.StatusOK)
	if body["reachable"] != false || body["error"] == "" || strings.Contains(response.Body.String()+fmt.Sprint(response.Header()), "private-marker") {
		t.Fatalf("unsafe test result: %v", body)
	}
	if !strings.Contains(logs.String(), "private-marker") {
		t.Fatalf("missing diagnostic: %s", logs.String())
	}
	after, _ := connections.Find(t.Context(), tenancydomain.DefaultTenantID, "app-1")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("connection changed: before=%+v after=%+v", before, after)
	}
}
