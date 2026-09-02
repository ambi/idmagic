package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	passwordsargon2id "github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type unmanagedImportUsers struct{}

func (unmanagedImportUsers) SourceManagedUserIDs(_ context.Context, _ string, ids []string) (map[string]bool, error) {
	managed := make(map[string]bool, len(ids))
	for _, id := range ids {
		managed[id] = false
	}
	return managed, nil
}

// REQ-IDMANAGEMENT-004: 管理 API の preview/apply と worker の二段階を通し、有効な CSV 行が User repository へ反映されることを確認する。
func TestAdminUserImportPrimaryUseCase_REQ_IDMANAGEMENT_004(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	artifacts := idmmemory.NewCSVArtifactStore()
	jobRepo := jobsmemory.NewJobRepository()
	hasher := passwordsargon2id.NewArgon2idPasswordHasher()
	committer := usermemory.UserImportRowCommitter{Users: users}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test",
		Authentication: authentication.Module{
			AuthnResolver: authusecases.DemoHeaderResolver{}, PasswordHasher: hasher,
		},
		IdManagement: idmanagement.Module{
			UserRepo: users, CSVArtifacts: artifacts, UserImportCommitter: committer,
		},
		Jobs: jobs.Module{Repo: jobRepo},
	})

	account := httptest.NewRequest(http.MethodGet, "/realms/default/api/auth/account", http.NoBody)
	account.Header.Set("X-Demo-Sub", "admin")
	accountResponse := httptest.NewRecorder()
	e.ServeHTTP(accountResponse, account)
	if accountResponse.Code != http.StatusOK {
		t.Fatalf("account status=%d body=%s", accountResponse.Code, accountResponse.Body.String())
	}
	var accountBody struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(accountResponse.Body.Bytes(), &accountBody); err != nil {
		t.Fatal(err)
	}
	cookies := accountResponse.Result().Cookies()
	if len(cookies) == 0 || accountBody.CSRFToken == "" {
		t.Fatal("csrf session was not issued")
	}

	post := func(path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "text/csv")
		request.Header.Set("Origin", "http://idp.test")
		request.Header.Set("X-Csrf-Token", accountBody.CSRFToken)
		request.Header.Set("X-Demo-Sub", "admin")
		request.AddCookie(cookies[0])
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)
		return response
	}
	jobID := func(response *httptest.ResponseRecorder) string {
		t.Helper()
		if response.Code != http.StatusAccepted {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
		var body struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.ID
	}

	planDeps := userusecases.UserImportPlanDeps{
		UserRepo: users, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, OwnershipGuard: unmanagedImportUsers{},
	}
	jobDeps := userusecases.UserImportJobDeps{
		Artifacts: artifacts, Jobs: jobRepo, Plan: planDeps,
		Apply: userusecases.UserImportApplyDeps{Plan: planDeps, Committer: committer, PasswordHasher: hasher},
	}
	runJob := func(handler func(context.Context, *jobsdomain.Job) (json.RawMessage, error)) {
		t.Helper()
		// Job を投入するのは HTTP ハンドラーで、その RunAt は実時計から取る。取得側も
		// 同じ時計で、かつ投入より後の時刻を使う必要がある。ClaimBatch は RunAt が
		// 取得時刻より後の Job を候補から外すためである。
		claimAt := time.Now().UTC()
		claimed, err := jobRepo.ClaimBatch(context.Background(), "worker", jobsdomain.LaneBulk, 1, time.Minute, claimAt)
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim=%+v err=%v", claimed, err)
		}
		result, err := handler(context.Background(), claimed[0])
		if err != nil {
			t.Fatal(err)
		}
		if _, err := jobRepo.Complete(context.Background(), claimed[0].ID, "worker", result, claimAt.Add(time.Second)); err != nil {
			t.Fatal(err)
		}
	}

	previewID := jobID(post("/realms/default/api/admin/v1/users/imports", "preferred_username,email\nalice,alice@example.com\n"))
	runJob(userusecases.UserImportJobHandler(jobDeps, userusecases.UserImportModePreview))
	applyID := jobID(post("/realms/default/api/admin/v1/users/imports/"+previewID+"/apply", ""))
	runJob(userusecases.UserImportJobHandler(jobDeps, userusecases.UserImportModeApply))

	if applyID == "" {
		t.Fatal("apply job id is empty")
	}
	alice, err := users.FindByUsername(context.Background(), tenancydomain.DefaultTenantID, "alice")
	if err != nil || alice == nil || alice.Email == nil || *alice.Email != "alice@example.com" {
		t.Fatalf("alice=%+v err=%v", alice, err)
	}
}
