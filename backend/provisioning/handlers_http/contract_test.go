package handlers_http_test

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func provisioningContractJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	media, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if rec.Code != status || err != nil || media != "application/json" {
		t.Fatalf("status=%d content-type=%q body=%s, want %d JSON", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String(), status)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func assertProvisioningProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if rec.Code != status || rec.Header().Get("Content-Type") != support.ProblemContentType || problem.Status != status || problem.Type != "urn:idmagic:error:"+code {
		t.Fatalf("status=%d content-type=%q body=%s, want %d %s", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String(), status, code)
	}
}

func assertProvisioningTimes(t *testing.T, body map[string]any, fields ...string) {
	t.Helper()
	for _, field := range fields {
		value, _ := body[field].(string)
		if at, err := time.Parse(time.RFC3339Nano, value); err != nil || at.IsZero() {
			t.Fatalf("%s=%q: %v", field, value, err)
		}
	}
}

func assertProvisioningConnection(t *testing.T, body map[string]any, scope string) {
	t.Helper()
	if body["application_id"] != "app-1" || body["tenant_id"] != tenancydomain.DefaultTenantID || body["status"] != "active" || body["health"] != "ok" || body["scope"] != scope || body["base_url"] != "https://downstream.example/scim/v2" {
		t.Fatalf("connection=%v", body)
	}
	credential, _ := body["credential"].(map[string]any)
	if id, _ := credential["credential_id"].(string); id == "" || credential["auth_method"] != "bearer_token" {
		t.Fatalf("credential=%v", credential)
	}
	for _, field := range []string{"bearer_token", "oauth2_client_secret"} {
		if _, exists := credential[field]; exists {
			t.Fatalf("response exposes %s: %v", field, credential)
		}
	}
	assertProvisioningTimes(t, credential, "created_at")
	assertProvisioningTimes(t, body, "created_at", "updated_at")
}

//spec:covers REQ-PROVISIONING-002, REQ-PROVISIONING-011, REQ-PROVISIONING-012, REQ-PROVISIONING-013, REQ-PROVISIONING-010: 接続とタスクの管理は保存済みの状態を HTTP 本文と再読で返す。
//verify:operation RegisterProvisioningConnection: 201 JSON は新しい資格情報の公開メタデータと接続を返し、秘密を含めない。
//verify:operation GetProvisioningConnection: 200 JSON は同じ接続と保存した能力、設定を返す。
//verify:operation UpdateProvisioningConnection: 200 JSON は保存した all_users の適用範囲を返す。
//verify:operation DeleteProvisioningConnection: 削除は空の 204 を返し、再読は 404 の Problem Details となる。
//verify:operation TestProvisioningConnection: 200 JSON は reachable と取得した能力、取得時刻を返す。
//verify:operation ListTenantProvisioningConnections: 200 JSON の connections は保存した同じ接続を返す。
//verify:operation ResumeProvisioningConnection: 200 JSON は隔離を解除した ok を返し、再度の再開は 400 の Problem Details となる。
//verify:operation ProvisionOnDemand: 201 JSON は採番した pending のタスクを返し、適用範囲外は 409 の Problem Details となる。
//verify:operation StartFullResync: 200 JSON は二人の対象へ作った enqueued_count 2 を返す。
//verify:operation ListProvisioningTasks: 200 JSON の tasks は指定した接続の同じタスクの属性を返す。
//verify:operation GetProvisioningTask: 200 JSON は同じタスクの主体、操作、状態と時刻を返す。
//verify:operation RetryProvisioningTask: 200 JSON は dead_letter を pending へ戻し、job_id と last_error を除く。
func TestProvisioningManagementHTTPContracts(t *testing.T) {
	request, connections, tasks := newProvisioningContractServer(t)
	path := provisioningConnectionPath
	created := request("admin", http.MethodPost, path, map[string]any{
		"base_url": "https://downstream.example/scim/v2", "credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "contract-secret"},
	})
	assertProvisioningConnection(t, provisioningContractJSON(t, created, http.StatusCreated), "assigned_only")
	if strings.Contains(created.Body.String(), "contract-secret") {
		t.Fatalf("registration exposes the secret: %s", created.Body.String())
	}
	assertProvisioningProblem(t, request("admin", http.MethodPost, path+"/on-demand", map[string]any{"subject_type": "user", "subject_id": "member"}), http.StatusConflict, "provisioning_conflict")
	tested := provisioningContractJSON(t, request("admin", http.MethodPost, path+"/test", nil), http.StatusOK)
	caps, _ := tested["capabilities"].(map[string]any)
	if tested["reachable"] != true || caps["supports_patch"] != true || caps["supports_filter"] != true || caps["supports_bulk"] != false {
		t.Fatalf("test result=%v", tested)
	}
	assertProvisioningTimes(t, caps, "discovered_at")
	updated := provisioningContractJSON(t, request("admin", http.MethodPatch, path, map[string]any{"scope": "all_users"}), http.StatusOK)
	assertProvisioningConnection(t, updated, "all_users")
	got := provisioningContractJSON(t, request("admin", http.MethodGet, path, nil), http.StatusOK)
	assertProvisioningConnection(t, got, "all_users")
	if !reflect.DeepEqual(got["capabilities"], caps) || !reflect.DeepEqual(got, updated) {
		t.Fatalf("connection differs: got=%v updated=%v capabilities=%v", got, updated, caps)
	}
	listed := provisioningContractJSON(t, request("admin", http.MethodGet, "/api/admin/v1/provisioning/connections", nil), http.StatusOK)
	items, _ := listed["connections"].([]any)
	if len(items) != 1 || !reflect.DeepEqual(items[0], got) {
		t.Fatalf("connections=%v, want %v", listed, got)
	}
	conn, err := connections.Find(context.Background(), tenancydomain.DefaultTenantID, "app-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Quarantine("contract quarantine", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := connections.Update(context.Background(), conn, nil); err != nil {
		t.Fatal(err)
	}
	resumed := provisioningContractJSON(t, request("admin", http.MethodPost, path+"/resume", nil), http.StatusOK)
	assertProvisioningConnection(t, resumed, "all_users")
	assertProvisioningProblem(t, request("admin", http.MethodPost, path+"/resume", nil), http.StatusBadRequest, "invalid_request")
	onDemand := provisioningContractJSON(t, request("admin", http.MethodPost, path+"/on-demand", map[string]any{"subject_type": "user", "subject_id": "member"}), http.StatusCreated)
	id, _ := onDemand["id"].(string)
	if id == "" || onDemand["tenant_id"] != tenancydomain.DefaultTenantID || onDemand["connection_id"] != "app-1" || onDemand["source_type"] != "user" || onDemand["source_id"] != "member" || onDemand["status"] != "pending" || onDemand["operation"] != "update" {
		t.Fatalf("on demand task=%v", onDemand)
	}
	if version, ok := onDemand["source_version"].(float64); !ok || version <= 0 {
		t.Fatalf("source_version=%v", onDemand)
	}
	assertProvisioningTimes(t, onDemand, "created_at", "updated_at")
	taskPath := path + "/tasks/" + id
	detail := provisioningContractJSON(t, request("admin", http.MethodGet, taskPath, nil), http.StatusOK)
	if !reflect.DeepEqual(detail, onDemand) {
		t.Fatalf("task reread=%v, want %v", detail, onDemand)
	}
	taskList := provisioningContractJSON(t, request("admin", http.MethodGet, path+"/tasks", nil), http.StatusOK)
	items, _ = taskList["tasks"].([]any)
	if len(items) != 1 || !reflect.DeepEqual(items[0], onDemand) {
		t.Fatalf("tasks=%v", taskList)
	}
	if _, err := tasks.AttachJob(context.Background(), tenancydomain.DefaultTenantID, id, "contract-job"); err != nil {
		t.Fatal(err)
	}
	lastError := "downstream_unavailable"
	if err := tasks.UpdateStatus(context.Background(), tenancydomain.DefaultTenantID, id, domain.TaskDeadLetter, &lastError); err != nil {
		t.Fatal(err)
	}
	retried := provisioningContractJSON(t, request("admin", http.MethodPost, taskPath+"/retry", nil), http.StatusOK)
	if retried["id"] != id || retried["status"] != "pending" {
		t.Fatalf("retried=%v", retried)
	}
	for _, field := range []string{"job_id", "last_error"} {
		if _, present := retried[field]; present {
			t.Fatalf("retry retains %s: %v", field, retried)
		}
	}
	assertProvisioningProblem(t, request("admin", http.MethodPost, taskPath+"/retry", nil), http.StatusConflict, "provisioning_conflict")
	resync := provisioningContractJSON(t, request("admin", http.MethodPost, path+"/full-resync", nil), http.StatusOK)
	if resync["enqueued_count"] != float64(2) {
		t.Fatalf("resync=%v", resync)
	}
	deleted := request("admin", http.MethodDelete, path, nil)
	if deleted.Code != http.StatusNoContent || deleted.Body.Len() != 0 {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	assertProvisioningProblem(t, request("admin", http.MethodGet, path, nil), http.StatusNotFound, "provisioning_not_found")
}

//spec:covers REQ-PROVISIONING-001: 管理者でない主体は接続とタスクの管理を 403 access_denied で拒否される。
//verify:operation RegisterProvisioningConnection: 非管理者の登録は 403 の Problem Details で拒否する。
//verify:operation GetProvisioningConnection: 非管理者の参照は 403 の Problem Details で拒否する。
//verify:operation UpdateProvisioningConnection: 非管理者の更新は 403 の Problem Details で拒否する。
//verify:operation DeleteProvisioningConnection: 非管理者の削除は 403 の Problem Details で拒否する。
//verify:operation TestProvisioningConnection: 非管理者のテストは 403 の Problem Details で拒否する。
//verify:operation ResumeProvisioningConnection: 非管理者の再開は 403 の Problem Details で拒否する。
//verify:operation ProvisionOnDemand: 非管理者の On-Demand は 403 の Problem Details で拒否する。
//verify:operation StartFullResync: 非管理者のフル同期は 403 の Problem Details で拒否する。
//verify:operation ListProvisioningTasks: 非管理者のタスク一覧は 403 の Problem Details で拒否する。
//verify:operation GetProvisioningTask: 非管理者のタスク参照は 403 の Problem Details で拒否する。
//verify:operation RetryProvisioningTask: 非管理者のタスク再試行は 403 の Problem Details で拒否する。
//verify:operation ListTenantProvisioningConnections: 非管理者の接続一覧は 403 の Problem Details で拒否する。
func TestProvisioningManagementHTTPContractsRefuseNonAdmin(t *testing.T) {
	request := newProvisioningAdminServer(t)
	registerProvisioningConnection(t, request)
	for _, operation := range []struct{ method, path string }{
		{http.MethodPost, provisioningConnectionPath},
		{http.MethodGet, provisioningConnectionPath},
		{http.MethodPatch, provisioningConnectionPath},
		{http.MethodDelete, provisioningConnectionPath},
		{http.MethodPost, provisioningConnectionPath + "/test"},
		{http.MethodPost, provisioningConnectionPath + "/resume"},
		{http.MethodPost, provisioningConnectionPath + "/on-demand"},
		{http.MethodPost, provisioningConnectionPath + "/full-resync"},
		{http.MethodGet, provisioningConnectionPath + "/tasks"},
		{http.MethodGet, provisioningConnectionPath + "/tasks/task"},
		{http.MethodPost, provisioningConnectionPath + "/tasks/task/retry"},
		{http.MethodGet, "/api/admin/v1/provisioning/connections"},
	} {
		assertProvisioningProblem(t, request("member", operation.method, operation.path, map[string]any{}), http.StatusForbidden, "access_denied")
	}
	assertProvisioningConnection(t, provisioningContractJSON(t, request("admin", http.MethodGet, provisioningConnectionPath, nil), http.StatusOK), "assigned_only")
}
