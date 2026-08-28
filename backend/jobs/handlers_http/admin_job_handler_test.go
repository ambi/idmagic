package handlers_http_test

// REQ-JOBS-012 / REQ-JOBS-013 / REQ-JOBS-014 を
// /api/admin/v1/jobs 経由で検証する (wi-157)。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	"github.com/ambi/idmagic/backend/jobs/domain"
	jobports "github.com/ambi/idmagic/backend/jobs/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type fakeAuthnResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (f *fakeAuthnResolver) Resolve(_ context.Context, _ authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return f.ctx, nil
}

// jobsTenantRepo は "acme" と制御面テナントだけを解決する最小の TenantRepository。
type jobsTenantRepo struct{}

func acmeTenant() *tenancydomain.Tenant {
	return &tenancydomain.Tenant{ID: "acme", Realm: "acme", Status: tenancydomain.TenantStatusActive}
}

func defaultTenant() *tenancydomain.Tenant {
	return &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm,
		Status: tenancydomain.TenantStatusActive,
	}
}

func (jobsTenantRepo) FindByID(_ context.Context, id string) (*tenancydomain.Tenant, error) {
	switch id {
	case "acme":
		return acmeTenant(), nil
	case tenancydomain.DefaultTenantID:
		return defaultTenant(), nil
	}
	return nil, nil
}

func (jobsTenantRepo) FindByRealm(_ context.Context, realm string) (*tenancydomain.Tenant, error) {
	switch realm {
	case "acme":
		return acmeTenant(), nil
	case tenancydomain.DefaultRealm:
		return defaultTenant(), nil
	}
	return nil, nil
}

func (jobsTenantRepo) FindAll(_ context.Context) ([]*tenancydomain.Tenant, error) {
	return []*tenancydomain.Tenant{acmeTenant(), defaultTenant()}, nil
}
func (jobsTenantRepo) Save(_ context.Context, _ *tenancydomain.Tenant) error { return nil }
func (jobsTenantRepo) Delete(_ context.Context, _ string) error              { return nil }

func jobsAdminUser(sub, tenantID string, roles []string) *userdomain.User {
	now := time.Now().UTC()
	return &userdomain.User{
		ID: sub, PreferredUsername: sub, TenantID: tenantID, Roles: roles,
		CreatedAt: now, UpdatedAt: now,
	}
}

type jobsTestServer struct {
	e        *echo.Echo
	repo     *jobsmemory.JobRepository
	emitted  []spec.DomainEvent
	acmeJob  *domain.Job
	otherJob *domain.Job
}

// newJobsAdminServer は "acme" に 2 件、制御面テナントに 1 件の Job を持つサーバーを作る。
func newJobsAdminServer(t *testing.T, actor *userdomain.User) *jobsTestServer {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	if actor != nil {
		userRepo.Seed(actor)
	}
	repo := jobsmemory.NewJobRepository()
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	enqueue := func(tenantID string, kind domain.JobKind, offset time.Duration) *domain.Job {
		t.Helper()
		lane, ok := domain.LaneFor(kind)
		if !ok {
			t.Fatalf("no lane registered for %q", kind)
		}
		job, _, err := repo.Enqueue(context.Background(), jobports.EnqueueInput{
			TenantID: tenantID, Kind: kind, Lane: lane,
			Params:      json.RawMessage(`{"email":"alice@example.test"}`),
			MaxAttempts: 3, RunAt: base.Add(offset), Now: base.Add(offset),
		})
		if err != nil {
			t.Fatalf("seed enqueue: %v", err)
		}
		return job
	}
	srv := &jobsTestServer{repo: repo}
	enqueue("acme", domain.KindUserImportApply, 0)
	srv.acmeJob = enqueue("acme", domain.KindNoopEcho, time.Minute)
	srv.otherJob = enqueue(tenancydomain.DefaultTenantID, domain.KindNoopEcho, 2*time.Minute)

	resolver := &fakeAuthnResolver{}
	if actor != nil {
		resolver.ctx = &authdomain.AuthenticationContext{
			UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", TenantRepo: jobsTenantRepo{},
		Emit:          func(e spec.DomainEvent) { srv.emitted = append(srv.emitted, e) },
		UserRepo:      userRepo,
		IdManagement:  idmanagement.Module{UserRepo: userRepo},
		AuthnResolver: resolver,
		Jobs:          jobs.Module{Repo: repo},
	})
	srv.e = e
	return srv
}

func (s *jobsTestServer) get(path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

// csrf は state-changing なブラウザー経路に必要なトークンと Cookie を取る。
func (s *jobsTestServer) csrf(t *testing.T, realmPath string) (string, *http.Cookie) {
	t.Helper()
	rec := s.get(realmPath + "/api/auth/account")
	if rec.Code != http.StatusOK {
		t.Fatalf("account status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	return body.CSRFToken, cookies[0]
}

func (s *jobsTestServer) cancel(t *testing.T, jobID string) *httptest.ResponseRecorder {
	t.Helper()
	const realmPath = "/realms/acme"
	token, cookie := s.csrf(t, realmPath)
	req := httptest.NewRequest(http.MethodPost, realmPath+"/api/admin/v1/jobs/"+jobID+"/cancel", http.NoBody)
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", token)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

type jobListBody struct {
	Jobs []struct {
		ID          string  `json:"id"`
		TenantID    string  `json:"tenant_id"`
		Kind        string  `json:"kind"`
		Lane        string  `json:"lane"`
		Status      string  `json:"status"`
		Attempts    int     `json:"attempts"`
		MaxAttempts int     `json:"max_attempts"`
		Error       *string `json:"error"`
	} `json:"jobs"`
	NextCursor string `json:"next_cursor"`
}

func decodeJobList(t *testing.T, rec *httptest.ResponseRecorder) jobListBody {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body jobListBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

// REQ-JOBS-012: 一覧は自テナントに閉じる。
func TestListJobsStaysInsideTheTenant(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	body := decodeJobList(t, srv.get("/realms/acme/api/admin/v1/jobs"))
	if len(body.Jobs) != 2 {
		t.Fatalf("got %d jobs, want 2", len(body.Jobs))
	}
	for _, j := range body.Jobs {
		if j.TenantID != "acme" {
			t.Fatalf("job %s belongs to %q, want acme only", j.ID, j.TenantID)
		}
	}
}

// REQ-JOBS-014: params / result / dedup_key は応答に現れない。
func TestListJobsOmitsHandlerInputAndOutput(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	rec := srv.get("/realms/acme/api/admin/v1/jobs")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var raw struct {
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, job := range raw.Jobs {
		for _, forbidden := range []string{"params", "result", "dedup_key"} {
			if _, present := job[forbidden]; present {
				t.Fatalf("response carries %q: %v", forbidden, job)
			}
		}
	}
	// シードした params の中身が本文のどこにも漏れていないこと。
	if strings.Contains(rec.Body.String(), "alice@example.test") {
		t.Fatalf("handler params leaked into the response: %s", rec.Body.String())
	}
}

// REQ-JOBS-012: admin ロールを持たない実行者は拒否される。
func TestListJobsRequiresAdminRole(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("nobody", "acme", []string{}))
	if rec := srv.get("/realms/acme/api/admin/v1/jobs"); rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// REQ-JOBS-012: 絞り込みは許可された語彙に限り、未知の値は無視せず拒否する。
func TestListJobsRejectsAnUnknownFilterValue(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	for _, query := range []string{"status=nonsense", "kind=nonsense", "lane=nonsense"} {
		rec := srv.get("/realms/acme/api/admin/v1/jobs?" + query)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status=%d body=%s", query, rec.Code, rec.Body.String())
		}
	}
}

// REQ-JOBS-012: 種別で絞り込める。
func TestListJobsFiltersByKind(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	body := decodeJobList(t, srv.get("/realms/acme/api/admin/v1/jobs?kind=user_import_apply"))
	if len(body.Jobs) != 1 || body.Jobs[0].Kind != "user_import_apply" {
		t.Fatalf("kind filter returned %+v", body.Jobs)
	}
}

// REQ-JOBS-012: 横断は system_admin かつ制御面テナントの経路でのみ認める。
func TestListJobsAllTenantsRequiresSystemAdminOnTheControlPlane(t *testing.T) {
	// 通常の admin が all_tenants を指定しても自テナントに閉じる。
	tenantAdmin := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	body := decodeJobList(t, tenantAdmin.get("/realms/acme/api/admin/v1/jobs?all_tenants=true"))
	for _, j := range body.Jobs {
		if j.TenantID != "acme" {
			t.Fatalf("a tenant admin saw tenant %q", j.TenantID)
		}
	}

	sysAdmin := newJobsAdminServer(t, jobsAdminUser("root", tenancydomain.DefaultTenantID, []string{"admin", "system_admin"}))
	crossed := decodeJobList(t, sysAdmin.get("/realms/default/api/admin/v1/jobs?all_tenants=true"))
	if len(crossed.Jobs) != 3 {
		t.Fatalf("system_admin saw %d jobs, want every tenant's 3", len(crossed.Jobs))
	}
}

// REQ-JOBS-012: 他テナントの Job は id を知っていても存在しないものとして扱う。
func TestGetJobHidesAnotherTenant(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	rec := srv.get("/realms/acme/api/admin/v1/jobs/" + srv.otherJob.ID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	// 存在しない id と同じ応答であること。
	missing := srv.get("/realms/acme/api/admin/v1/jobs/does-not-exist")
	if missing.Code != rec.Code {
		t.Fatalf("an unknown id answered %d while another tenant's answered %d", missing.Code, rec.Code)
	}
}

// REQ-JOBS-013: 終端に達していない Job を取り消せ、JobCanceled が発行される。
func TestCancelJob(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	rec := srv.cancel(t, srv.acmeJob.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != string(domain.StatusCanceled) {
		t.Fatalf("status = %q, want canceled", body.Status)
	}
	var canceled int
	for _, e := range srv.emitted {
		if _, ok := e.(*domain.JobCanceled); ok {
			canceled++
		}
	}
	if canceled != 1 {
		t.Fatalf("emitted %d JobCanceled events, want 1", canceled)
	}
}

// REQ-JOBS-013: 終端に達した Job の取り消しは成功として黙認せず 409 で拒否する。
func TestCancelJobRefusesATerminalJob(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	if rec := srv.cancel(t, srv.acmeJob.ID); rec.Code != http.StatusOK {
		t.Fatalf("first cancel status=%d body=%s", rec.Code, rec.Body.String())
	}
	rec := srv.cancel(t, srv.acmeJob.ID)
	if rec.Code != http.StatusConflict {
		t.Fatalf("second cancel status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// REQ-JOBS-013: 他テナントの Job は取り消せず、状態も変わらない。
func TestCancelJobHidesAnotherTenant(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	if rec := srv.cancel(t, srv.otherJob.ID); rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	got, err := srv.repo.Get(context.Background(), srv.otherJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusQueued {
		t.Fatalf("a refused cross-tenant cancel changed the job to %q", got.Status)
	}
}

// 状態を変える経路は CSRF を伴わない要求を受け付けない。
func TestCancelJobRequiresBrowserVerification(t *testing.T) {
	srv := newJobsAdminServer(t, jobsAdminUser("admin", "acme", []string{"admin"}))
	req := httptest.NewRequest(http.MethodPost, "/realms/acme/api/admin/v1/jobs/"+srv.acmeJob.ID+"/cancel", http.NoBody)
	rec := httptest.NewRecorder()
	srv.e.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("a request without CSRF succeeded: body=%s", rec.Body.String())
	}
	got, err := srv.repo.Get(context.Background(), srv.acmeJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusQueued {
		t.Fatalf("the job changed to %q despite a refused request", got.Status)
	}
}
