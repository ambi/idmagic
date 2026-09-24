package server_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

const (
	isolatedApplicationID = "app-1"
	isolatedTaskID        = "task-1"
	isolatedBaseURL       = "https://downstream.example/scim/v2"
)

// seedDefaultTenantProvisioning は default テナントに接続を 1 件と、その接続に属するプロビジョニングタスクを
// 1 件置く。越境した参照が漏らしうる値は接続先 URL とプロビジョニングタスク id である。
func seedDefaultTenantProvisioning(t *testing.T, s *stack.Stack) {
	t.Helper()
	ctx := context.Background()
	tenantID := s.TenantID(t, "default")
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	if err := s.ProvisioningConnections.Register(ctx, &domain.ProvisioningConnection{
		ApplicationID: isolatedApplicationID, TenantID: tenantID,
		Status: domain.ConnectionActive, BaseURL: isolatedBaseURL,
		Credential: domain.ProvisioningConnectionCredentialMetadata{
			CredentialID: "credential-1", AuthMethod: domain.AuthBearerToken, CreatedAt: now,
		},
		Scope: domain.ScopeAssignedOnly,
		DeprovisionPolicy: domain.DeprovisionPolicy{
			OnUnassign: domain.DeprovisionDeactivate, OnDelete: domain.DeprovisionDeactivate,
		},
		RateLimitPerMinute: 60, MaxAttempts: 8, QuarantineAfterConsecutiveFailure: 10,
		Health:    domain.HealthOK,
		CreatedAt: now, UpdatedAt: now,
	}, "secret"); err != nil {
		t.Fatalf("seed connection: %v", err)
	}
	if _, err := s.ProvisioningTasks.Save(ctx, &domain.ProvisioningTask{
		ID: isolatedTaskID, TenantID: tenantID, ConnectionID: isolatedApplicationID,
		SourceType: domain.SourceTypeUser, SourceID: stack.UserID, SourceVersion: 1,
		Operation: domain.OperationCreate, Status: domain.TaskPending,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}
}

func adminSessionGet(t *testing.T, s *stack.Stack, realm, path string, session *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, stack.Issuer+"/realms/"+realm+path, http.NoBody)
	request.AddCookie(session)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

type provisioningProblem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

func decodeProvisioningProblem(t *testing.T, recorder *httptest.ResponseRecorder) provisioningProblem {
	t.Helper()
	var problem provisioningProblem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("problem body %s: %v", recorder.Body.String(), err)
	}
	return problem
}

// 別テナントの管理者が同じ application id と task id を指定しても、接続とプロビジョニングタスクは
// どちらも存在しない id を指定したときと同じ 404 になる。状態コードと problem の
// type、title、detail まで一致させ、別テナントに同じ id があることを推測させない。
// 接続とプロビジョニングタスクを別々に観測し、一方の参照だけが要求先テナントで絞り込む実装を見分ける。
// 同じ URL を default の管理者が参照できることを先に確かめ、拒否が URL や配線の誤り
// ではないことを示す。
//
//spec:covers REQ-PROVISIONING-015, EX-PROVISIONING-015-01: 他テナントの管理者が同じ id で接続とプロビジョニングタスクを参照すると、存在しない id と同じ 404 provisioning_not_found になり、接続先 URL とプロビジョニングタスク id を返さない。
func TestForeignTenantAdminSeesProvisioningAsMissing(t *testing.T) {
	s := stack.New(t, stack.WithAuthorizationCodeFlow(), stack.WithProvisioning())
	seedDefaultTenantProvisioning(t, s)
	ownerSession := s.AdminSessionCookie(t, "default")
	foreignSession := s.AdminSessionCookie(t, stack.OtherRealm)

	cases := []struct {
		name        string
		path        string
		missingPath string
		leaked      string
	}{
		{
			name:        "接続",
			path:        "/api/admin/v1/applications/" + isolatedApplicationID + "/provisioning",
			missingPath: "/api/admin/v1/applications/app-missing/provisioning",
			leaked:      isolatedBaseURL,
		},
		{
			name:        "プロビジョニングタスク",
			path:        "/api/admin/v1/applications/" + isolatedApplicationID + "/provisioning/tasks/" + isolatedTaskID,
			missingPath: "/api/admin/v1/applications/" + isolatedApplicationID + "/provisioning/tasks/task-missing",
			leaked:      isolatedTaskID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owned := adminSessionGet(t, s, "default", tc.path, ownerSession)
			if owned.Code != http.StatusOK || !strings.Contains(owned.Body.String(), tc.leaked) {
				t.Fatalf("前提が壊れている: 所有テナントの参照が status=%d body=%s", owned.Code, owned.Body.String())
			}

			foreign := adminSessionGet(t, s, stack.OtherRealm, tc.path, foreignSession)
			if foreign.Code != http.StatusNotFound {
				t.Fatalf("越境の参照 status=%d body=%s, want 404", foreign.Code, foreign.Body.String())
			}
			if strings.Contains(foreign.Body.String(), tc.leaked) {
				t.Fatalf("越境の参照が %q を返した: %s", tc.leaked, foreign.Body.String())
			}
			got := decodeProvisioningProblem(t, foreign)
			if got.Type != "urn:idmagic:error:provisioning_not_found" {
				t.Fatalf("越境の参照 type=%q, want provisioning_not_found", got.Type)
			}

			missing := adminSessionGet(t, s, stack.OtherRealm, tc.missingPath, foreignSession)
			if missing.Code != foreign.Code {
				t.Fatalf("存在しない id の status=%d, 越境の status=%d", missing.Code, foreign.Code)
			}
			if want := decodeProvisioningProblem(t, missing); got != want {
				t.Fatalf("越境の problem=%+v, 存在しない id の problem=%+v", got, want)
			}
		})
	}
}
