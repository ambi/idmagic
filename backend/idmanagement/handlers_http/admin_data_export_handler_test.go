package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type exportTestHandler struct {
	echo      *echo.Echo
	users     *usermemory.UserRepository
	groups    *groupmemory.GroupRepository
	jobRepo   *jobsmemory.JobRepository
	artifacts *idmmemory.CSVArtifactStore
}

func newExportTestHandler(t *testing.T, options ...func(*httpadapter.Deps)) exportTestHandler {
	t.Helper()
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	jobRepo := jobsmemory.NewJobRepository()
	artifacts := idmmemory.NewCSVArtifactStore()
	now := time.Now().UTC()
	email := "alice@example.com"
	users.Seed(&userdomain.User{ID: "admin", PreferredUsername: "admin", PasswordHash: "unused", Roles: []string{"admin"}, TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now})
	users.Seed(&userdomain.User{ID: "u1", PreferredUsername: "alice", PasswordHash: "unused", Email: &email, TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now})
	_ = groups.Save(context.Background(), &groupdomain.Group{ID: "g1", TenantID: tenancydomain.DefaultTenantID, Name: "engineering", Roles: []string{"dev"}, MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now})
	_, _ = groups.AddMember(context.Background(), &groupdomain.GroupMember{GroupID: "g1", UserID: "u1", Source: groupdomain.MembershipSourceManual, CreatedAt: now})

	e := echo.New()
	deps := httpadapter.Deps{
		Issuer: "http://idp.test", UserRepo: users, PasswordHasher: passwords_argon2id.NewArgon2idPasswordHasher(),
		AuthnResolver: authusecases.DemoHeaderResolver{},
		AgentRepo:     agentmemory.NewAgentRepository(),
		GroupRepo:     groups,
		Jobs:          jobs.Module{Repo: jobRepo},
		IdManagement:  idmanagement.Module{UserRepo: users, GroupRepo: groups, CSVArtifacts: artifacts},
		EmailSender:   mockEmailSender{},
	}
	for _, option := range options {
		option(&deps)
	}
	httpadapter.Register(e, deps)
	return exportTestHandler{echo: e, users: users, groups: groups, jobRepo: jobRepo, artifacts: artifacts}
}

// runExportJob drives the queued export Job to succeeded the way the worker
// would, so the HTTP download path can be exercised without a live worker.
func (h exportTestHandler) runExportJob(t *testing.T, exportID string) {
	t.Helper()
	ctx := context.Background()
	job, err := h.jobRepo.Get(ctx, exportID)
	if err != nil {
		t.Fatal(err)
	}
	deps := idmusecases.DataExportDeps{
		UserRepo: h.users, GroupRepo: h.groups, JobRepo: h.jobRepo, CSVArtifacts: h.artifacts,
		UserCSVExporter: userusecases.UserCSVExporter{
			Deps:   userusecases.UserCSVExportDeps{UserRepo: h.users, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, Artifacts: h.artifacts},
			Policy: idmdomain.DefaultCSVTransferPolicy(),
		},
	}
	raw, err := idmusecases.DataExportHandler(deps)(ctx, job)
	if err != nil {
		t.Fatalf("run export job: %v", err)
	}
	if _, err := h.jobRepo.ClaimBatch(ctx, "w1", jobsdomain.LaneBulk, 10, time.Minute, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := h.jobRepo.Complete(ctx, exportID, "w1", raw, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

func TestDataExportHTTP_UserFullFlow(t *testing.T) {
	h := newExportTestHandler(t)
	e := h.echo
	csrf, cookie := adminCSRF(t, e)

	// Start (202, queued). Target is the /users/ path, not a body field.
	start := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/users/exports", csrf, cookie, map[string]any{
		"columns": []string{"preferred_username", "email"},
	})
	if start.Code != http.StatusAccepted {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body.String())
	}
	var started struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		Target       string `json:"target"`
		Downloadable bool   `json:"downloadable"`
	}
	_ = json.Unmarshal(start.Body.Bytes(), &started)
	if started.ID == "" || started.Status != "queued" || started.Target != "user" || started.Downloadable {
		t.Fatalf("unexpected start body: %s", start.Body.String())
	}

	// List under /users/exports contains it.
	list := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/users/exports", csrf, cookie, nil)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), started.ID) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}

	// A user export id must not resolve under /groups/exports (per-type isolation).
	cross := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/groups/exports/"+started.ID, csrf, cookie, nil)
	if cross.Code != http.StatusNotFound {
		t.Fatalf("cross-type get status=%d, want 404", cross.Code)
	}

	// Download before completion is rejected.
	early := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/users/exports/"+started.ID+"/file", csrf, cookie, nil)
	if early.Code != http.StatusConflict {
		t.Fatalf("early download status=%d, want 409", early.Code)
	}

	// Drive the job to succeeded, then download.
	h.runExportJob(t, started.ID)
	get := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/users/exports/"+started.ID, csrf, cookie, nil)
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"status":"succeeded"`) || !strings.Contains(get.Body.String(), `"downloadable":true`) {
		t.Fatalf("get status=%d body=%s", get.Code, get.Body.String())
	}
	dl := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/users/exports/"+started.ID+"/file", csrf, cookie, nil)
	if dl.Code != http.StatusOK {
		t.Fatalf("download status=%d body=%s", dl.Code, dl.Body.String())
	}
	if ct := dl.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("content-type=%q, want text/csv", ct)
	}
	if cd := dl.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") || !strings.Contains(cd, ".csv") {
		t.Errorf("content-disposition=%q", cd)
	}
	if !strings.Contains(dl.Body.String(), "alice") {
		t.Errorf("csv missing seeded user: %s", dl.Body.String())
	}
}

func TestDataExportHTTP_GroupMemberFlow(t *testing.T) {
	h := newExportTestHandler(t)
	e := h.echo
	csrf, cookie := adminCSRF(t, e)

	// Member export is nested under a specific group; group_id comes from the path.
	start := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups/g1/members/exports", csrf, cookie, map[string]any{
		"columns": []string{"user_id", "source"},
	})
	if start.Code != http.StatusAccepted {
		t.Fatalf("member start status=%d body=%s", start.Code, start.Body.String())
	}
	var started struct {
		ID     string `json:"id"`
		Target string `json:"target"`
	}
	_ = json.Unmarshal(start.Body.Bytes(), &started)
	if started.Target != "group_membership" {
		t.Fatalf("unexpected member export target: %s", start.Body.String())
	}

	// The same export must not resolve under a different group.
	cross := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/groups/other/members/exports/"+started.ID, csrf, cookie, nil)
	if cross.Code != http.StatusNotFound {
		t.Fatalf("cross-group member get status=%d, want 404", cross.Code)
	}

	h.runExportJob(t, started.ID)
	dl := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/groups/g1/members/exports/"+started.ID+"/file", csrf, cookie, nil)
	if dl.Code != http.StatusOK {
		t.Fatalf("member download status=%d body=%s", dl.Code, dl.Body.String())
	}
}

func TestDataExportHTTP_RejectsInvalidColumns(t *testing.T) {
	h := newExportTestHandler(t)
	e := h.echo
	csrf, cookie := adminCSRF(t, e)
	resp := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/users/exports", csrf, cookie, map[string]any{
		"columns": []string{"password_hash"},
	})
	if resp.Code != http.StatusUnprocessableEntity ||
		!strings.Contains(resp.Body.String(), "urn:idmagic:error:invalid_columns") {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

// 実行中ジョブ数の上限と、テナント資源クォータは別の概念である。前者は待てば通る
// 一時的な状態 (RFC 6585 の 429)、後者は解析できた内容の業務規則違反 (422) で、
// クライアントが再試行すべきかは code だけで判断できなければならない。
// 両者が `quota_exceeded` を共有していると、その判断ができない。
func TestDataExportHTTP_ActiveJobCeilingHasItsOwnCode(t *testing.T) {
	quotaRepo := tenancymemory.NewQuotaRepository()
	none := 0
	if err := quotaRepo.SetQuota(context.Background(), tenancydomain.DefaultTenantID,
		&tenancydomain.TenantQuota{ActiveJobs: &none}); err != nil {
		t.Fatal(err)
	}
	h := newExportTestHandler(t, func(deps *httpadapter.Deps) {
		deps.Tenancy.QuotaRepo = quotaRepo
	})
	e := h.echo
	csrf, cookie := adminCSRF(t, e)
	resp := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/users/exports", csrf, cookie, map[string]any{
		"columns": []string{"preferred_username", "email"},
	})

	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s, want 429", resp.Code, resp.Body.String())
	}
	var problem struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v (body=%s)", err, resp.Body.String())
	}
	if problem.Type != "urn:idmagic:error:active_job_quota_exceeded" {
		t.Fatalf("type=%q, want urn:idmagic:error:active_job_quota_exceeded", problem.Type)
	}
	// テナント資源クォータの 422 quota_exceeded と取り違えられないこと。
	if strings.Contains(resp.Body.String(), "urn:idmagic:error:quota_exceeded") {
		t.Fatalf("active job ceiling must not reuse the tenant resource quota code: %s", resp.Body.String())
	}
	// 拒否が何も通していないこと。実行待ちの export ジョブは 1 件も残っていない。
	jobsList, err := h.jobRepo.ClaimBatch(context.Background(), "w1", jobsdomain.LaneBulk, 10, time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobsList) != 0 {
		t.Fatalf("refused export left %d runnable job(s): %+v", len(jobsList), jobsList)
	}
}

func TestDataExportHTTP_NotFoundForUnknownID(t *testing.T) {
	h := newExportTestHandler(t)
	e := h.echo
	csrf, cookie := adminCSRF(t, e)
	resp := adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/users/exports/00000000-0000-0000-0000-000000000000", csrf, cookie, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestDataExportHTTP_RequiresAdmin(t *testing.T) {
	h := newExportTestHandler(t)
	// A non-admin authenticated user must be denied (u1 is a regular user).
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/users/exports", http.NoBody)
	request.Header.Set("X-Demo-Sub", "u1")
	resp := httptest.NewRecorder()
	h.echo.ServeHTTP(resp, request)
	if resp.Code == http.StatusOK {
		t.Fatalf("non-admin list must not succeed, got %d body=%s", resp.Code, resp.Body.String())
	}
}
