package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/provisioning/testing_scim"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	testingpasswords "github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

//spec:covers REQ-PROVISIONING-002, REQ-PROVISIONING-003, REQ-PROVISIONING-004: 公開の接続、属性対応、User と割り当てを PostgreSQL に保存し、本番の worker の起動と Jobs を通して下流の作成と無効化を観測する。
//verify:operation RegisterProvisioningConnection: 保存した接続の資格情報が下流への Authorization に使われる。
//verify:operation UpdateProvisioningConnection: 属性対応を更新し、下流の User の userName に反映する。
//verify:operation TestProvisioningConnection: 下流の対応機能を取得し、無効化の送信に PATCH を使う。
func TestPublicSetupReachesProvisioningWorker(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	values := map[string]string{"PERSISTENCE": "postgres", "DATABASE_URL": db.Config().ConnString(), "KEY_PROVIDER": "local"}
	loader := bootstrap.NewConfigLoader(func(key string) string { return values[key] })
	config := bootstrap.LoadSharedConfig(loader)
	if err := loader.Err(); err != nil {
		t.Fatal(err)
	}
	deps, err := bootstrap.Assemble(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(deps.Close)
	deps.Authentication.SessionManager = deps.Authentication.NewSessionManager()
	deps.Authentication.AuthnResolver = deps.Authentication.SessionManager
	deps.Authentication.PasswordHasher = testingpasswords.NewHasher()
	tenant, err := deps.Tenancy.TenantRepo.FindByID(ctx, tenancydomain.DefaultTenantID)
	if err != nil || tenant == nil {
		t.Fatalf("tenant=%v error=%v", tenant, err)
	}
	tenantCtx := tenantports.WithTenant(ctx, tenant, "", "")
	adminID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	// 初期管理者とセッションだけを基盤の fixture にし、業務の設定と対象 User は公開 API で作る。
	if err := deps.IdManagement.UserRepo.Save(tenantCtx, &userdomain.User{
		ID: adminID, TenantID: tenant.ID, PreferredUsername: "worker-admin-" + adminID,
		Roles: []string{"admin"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	session, err := deps.Authentication.SessionManager.Create(tenantCtx, adminID, []string{"pwd"}, now)
	if err != nil {
		t.Fatal(err)
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "public-setup-test", "test")
	emit := deps.NewEmitFunc(logger)
	type sent struct {
		method, path, authorization string
		body                        map[string]any
	}
	sends := make(chan sent, 10)
	downstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/scim+json")
		if r.Header.Get("Authorization") != "Bearer public-downstream-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/ServiceProviderConfig" {
			_, _ = w.Write([]byte(`{"patch":{"supported":true}}`))
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sends <- sent{r.Method, r.URL.Path, r.Header.Get("Authorization"), body}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
		_, _ = w.Write([]byte(`{"id":"remote-public-user"}`))
	}))
	t.Cleanup(downstream.Close)
	newClient := testing_scim.BearerClientFactory(downstream.Client())
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: config.Issuer, Contract: spec.CurrentRuntimeContract(), Emit: emit,
		TenantRepo: deps.Tenancy.TenantRepo, Tenancy: deps.Tenancy, IdManagement: deps.IdManagement,
		Authentication: deps.Authentication, OAuth2: deps.OAuth2, SigningKeys: deps.SigningKeys,
		DataKeys: deps.DataKeys, Application: deps.Application, Provisioning: deps.Provisioning,
		SharedSignals: deps.SharedSignals, Notification: deps.Notification,
		ProvisioningTargetClientFactory: newClient,
	})
	request := func(method, path string, body any, want int) map[string]any {
		t.Helper()
		var raw bytes.Buffer
		if body != nil {
			if err := json.NewEncoder(&raw).Encode(body); err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, config.Issuer+"/realms/default/api/admin/v1"+path, &raw)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", config.Issuer)
		req.Header.Set("X-Csrf-Token", "worker-csrf")
		req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "worker-csrf"})
		req.AddCookie(&http.Cookie{Name: authentication.SessionCookie, Value: session.SessionID})
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s: status=%d body=%s, want %d", method, path, rec.Code, rec.Body.String(), want)
		}
		result := map[string]any{}
		if rec.Body.Len() > 0 {
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
		}
		return result
	}
	app := request(http.MethodPost, "/applications", map[string]any{
		"name": "Worker public setup", "type": "oidc",
		"redirect_uris": []string{"https://worker.example/callback"}, "scope": "openid",
	}, http.StatusCreated)
	appID := app["application"].(map[string]any)["id"].(string)
	connectionPath := "/applications/" + appID + "/provisioning"
	request(http.MethodPost, connectionPath, map[string]any{
		"base_url": downstream.URL, "credential": map[string]any{"auth_method": "bearer_token", "bearer_token": "public-downstream-secret"},
	}, http.StatusCreated)
	request(http.MethodPatch, connectionPath, map[string]any{
		"attribute_mappings": []map[string]any{
			{"target_path": "userName", "source_kind": "attribute", "source_key": "preferred_username", "apply_on": "create_and_update", "required": true},
			{"target_path": "active", "source_kind": "attribute", "source_key": "active", "apply_on": "create_and_update"},
		},
	}, http.StatusOK)
	connectionTest := request(http.MethodPost, connectionPath+"/test", nil, http.StatusOK)
	if connectionTest["reachable"] != true {
		t.Fatalf("connection test=%v", connectionTest)
	}
	username := "public-worker-" + adminID
	user := request(http.MethodPost, "/users", map[string]any{"preferred_username": username, "password": "public-worker-password-1234"}, http.StatusCreated)
	userID := user["id"].(string)
	request(http.MethodPost, "/applications/"+appID+"/assignments", map[string]any{"subject_type": "user", "subject_id": userID}, http.StatusCreated)
	workerCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	handlers := jobs.NewHandlerRegistry()
	startProvisioningWorker(workerCtx, deps, logger, handlers, newClient, time.Hour)
	runner := jobs.NewRunner(jobs.RunnerConfig{
		WorkerID: "public-setup", Lane: jobsdomain.LaneDefault, PollInterval: 10 * time.Millisecond, Concurrency: 1,
	}, jobs.RunnerDeps{Repo: deps.Jobs.Repo, Handlers: handlers, Emit: emit, QuotaRepo: deps.Tenancy.QuotaRepo})
	done := make(chan error, 1)
	go func() { done <- runner.Run(workerCtx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("worker did not stop")
		}
	})
	await := func(active bool) {
		t.Helper()
		select {
		case got := <-sends:
			method, path := http.MethodPost, "/Users"
			values := got.body
			if !active {
				method, path = http.MethodPatch, "/Users/remote-public-user"
				operations, ok := got.body["Operations"].([]any)
				if !ok || len(operations) != 1 {
					t.Fatalf("PATCH operations=%v", got.body)
				}
				operation := operations[0].(map[string]any)
				if operation["op"] != "replace" {
					t.Fatalf("PATCH operation=%v", operation)
				}
				values = operation["value"].(map[string]any)
			}
			if got.method != method || got.path != path || got.authorization != "Bearer public-downstream-secret" || values["userName"] != username || values["active"] != active {
				t.Fatalf("downstream=%+v, want %s %s user=%s active=%v", got, method, path, username, active)
			}
		case <-time.After(8 * time.Second):
			t.Fatal("worker did not deliver the captured public mutation")
		}
	}
	await(true)
	// 外向きの送信の後にタスクとリンクの確定を待ち、次の変更を直列に起こす。
	deadline := time.Now().Add(5 * time.Second)
	for {
		tasks := request(http.MethodGet, connectionPath+"/tasks", nil, http.StatusOK)
		entries, _ := tasks["tasks"].([]any)
		if len(entries) == 1 && entries[0].(map[string]any)["status"] == "succeeded" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not complete: %v", tasks)
		}
		time.Sleep(10 * time.Millisecond)
	}
	request(http.MethodPost, "/users/"+userID+"/disable", nil, http.StatusNoContent)
	await(false)
}
