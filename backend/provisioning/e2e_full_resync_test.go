package provisioning_test

// 主要ユースケース追跡: REQ-PROVISIONING-013。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// startFullResyncOverHTTP は本番と同じ Module.Register の経路で管理 API の Full Resync を呼ぶ。
func (r *lifecycleRun) startFullResyncOverHTTP() {
	r.h.t.Helper()
	now := time.Now().UTC()
	r.h.userRepo.Seed(&userdomain.User{
		ID: "resync-admin", TenantID: r.h.tenantID, PreferredUsername: "resync-admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
	r.module.Register(e.Group(""), support.Deps{Issuer: "http://idp.test", Emit: r.emit},
		&support.Authenticator{UserRepo: r.h.userRepo, AuthnResolver: authusecases.DemoHeaderResolver{}},
		appmemory.NewApplicationAssignmentRepository(), r.h.userRepo, r.h.groupRepo)

	const csrf = "csrf-token"
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/applications/"+r.h.connectionID+"/provisioning/full-resync", http.NoBody)
	req.Header.Set("X-Demo-Sub", "resync-admin")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: csrf})
	response := httptest.NewRecorder()
	e.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		r.h.t.Fatalf("full resync status=%d body=%s", response.Code, response.Body.String())
	}
}

func (r *lifecycleRun) completedResyncs() []*domain.FullResyncCompleted {
	var completed []*domain.FullResyncCompleted
	for _, event := range r.events {
		if c, ok := event.(*domain.FullResyncCompleted); ok {
			completed = append(completed, c)
		}
	}
	return completed
}

//spec:covers EX-PROVISIONING-013-01: 管理 API の Full Resync は scope 内の全 subject にプロビジョニングタスクを作り、worker がすべてを終端にしたときだけ FullResyncCompleted を一度発行する。
func TestE2E_FullResyncEmitsCompletedOnceAllTasksSettle(t *testing.T) {
	h := newE2EHarness(t)
	run := newLifecycleRun(h)
	h.createUser("alice-resync")
	run.drainAt(time.Now().UTC())
	// 捕捉を通らない User は下流と乖離している。
	now := time.Now().UTC()
	h.userRepo.Seed(&userdomain.User{
		ID: "carol-resync", TenantID: h.tenantID, PreferredUsername: "carol-resync", PasswordHash: "unused",
		CreatedAt: now, UpdatedAt: now,
	})
	run.events = nil
	requestsBefore := h.downstream.count()

	run.startFullResyncOverHTTP()
	run.dispatch()
	jobs, err := run.jobs.ClaimBatch(context.Background(), "worker-e2e", jobsdomain.LaneDefault, 10, time.Minute, time.Now().UTC())
	if err != nil || len(jobs) != 3 {
		t.Fatalf("ClaimBatch() = %d jobs, %v; want one per subject (admin, alice, carol)", len(jobs), err)
	}
	for i, job := range jobs {
		if got := run.completedResyncs(); len(got) != 0 {
			t.Fatalf("FullResyncCompleted after %d of %d tasks = %+v, want none before the last task settles", i, len(jobs), got)
		}
		if err := run.handle(job); err != nil {
			t.Fatalf("handle(%s) error = %v", job.ID, err)
		}
	}
	// 同じジョブの再実行（リースの喪失など）は完了を二度発行しない。
	if err := run.handle(jobs[len(jobs)-1]); err != nil {
		t.Fatalf("handle() rerun error = %v", err)
	}

	completed := run.completedResyncs()
	if len(completed) != 1 {
		t.Fatalf("FullResyncCompleted = %d events (%v), want exactly one", len(completed), run.types())
	}
	fields := wire(t, completed[0])
	want := map[string]any{
		"type": "FullResyncCompleted", "tenantId": h.tenantID, "applicationId": h.connectionID,
		"totalSubjects": float64(3), "succeededCount": float64(3), "failedCount": float64(0),
	}
	for key, value := range want {
		if fields[key] != value {
			t.Errorf("FullResyncCompleted %s = %v, want %v (wire %v)", key, fields[key], value, fields)
		}
	}
	// 下流に無い管理者と carol は作成し、反映済みの alice は更新する。
	var posts, puts int
	for _, request := range h.downstream.snapshot()[requestsBefore:] {
		switch {
		case request.method == http.MethodPost && request.path == "/Users":
			posts++
		case request.method == http.MethodPut:
			puts++
		}
	}
	if posts != 2 || puts != 1 {
		t.Errorf("resync downstream requests = %v, want the two diverged Users created and alice updated", h.downstream.snapshot()[requestsBefore:])
	}
}
