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
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/tenancy"
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

// 判定の種類、行番号、安定したエラーコード、preview が保存層を動かさないこと、
// そして適用が CSV を再送せず保存済みのプレビューだけを指すことを、1 本の経路で読む。
//
//spec:covers REQ-IDMANAGEMENT-004, EX-IDMANAGEMENT-004-01: 管理 API の preview/apply と worker の二段階で、preview が判定と行番号と安定コードを返して User を変えず、apply が有効な行だけを反映すること。
func TestAdminUserImportPrimaryUseCase_REQ_IDMANAGEMENT_004(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	artifacts := idmmemory.NewCSVArtifactStore()
	jobRepo := jobsmemory.NewJobRepository()
	hasher := testing_passwords.NewHasher()
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
	jobResult := func(id string) userusecases.UserImportResult {
		t.Helper()
		job, err := jobRepo.Get(context.Background(), id)
		if err != nil || job == nil {
			t.Fatalf("job %s = %+v, %v", id, job, err)
		}
		var result userusecases.UserImportResult
		if err := json.Unmarshal(job.Result, &result); err != nil {
			t.Fatalf("decode job result: %v", err)
		}
		return result
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

	// 判定は 4 種類を 1 本のファイルに並べる。1 種類だけのファイルでは、判定を
	// 1 つに固定する実装が通ってしまう。
	document := "id,preferred_username,email,roles\n" +
		",alice,alice@example.com,support\n" + // 2: created
		"admin,admin,admin@example.com,admin\n" + // 3: updated
		"unknown-id,ghost,ghost@example.com,\n" // 4: rejected (target_not_found)

	previewID := jobID(post("/realms/default/api/admin/v1/users/imports", document))
	runJob(userusecases.UserImportJobHandler(jobDeps, userusecases.UserImportModePreview))
	preview := jobResult(previewID)
	if preview.CreatedRows != 1 || preview.UpdatedRows != 1 || preview.RejectedRows != 1 || preview.TotalRows != 3 {
		t.Fatalf("preview = %+v, want 1 created / 1 updated / 1 rejected", preview)
	}
	// 「`User` は変更されない」。プレビューは判定を返すだけで保存層を動かさない。
	if created, err := users.FindByUsername(context.Background(), tenancydomain.DefaultTenantID, "alice"); err != nil || created != nil {
		t.Fatalf("プレビューが User を作った: %+v, %v", created, err)
	}
	if unchanged, err := users.FindBySub(context.Background(), "admin"); err != nil || unchanged == nil || unchanged.Email != nil {
		t.Fatalf("プレビューが既存 User を変えた: %+v, %v", unchanged, err)
	}
	// 安定したエラーコードは、行ごとの誤りとして成果物から読める。行番号も付く。
	errorPage, err := userusecases.ReadUserImportErrorRange(
		tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", ""),
		artifacts, tenancydomain.DefaultTenantID, preview, 1, 10,
	)
	if err != nil || len(errorPage) != 1 {
		t.Fatalf("errors = %+v, %v", errorPage, err)
	}
	if errorPage[0].Row != 4 || errorPage[0].Code != "target_not_found" {
		t.Fatalf("error = %+v, want row 4 with target_not_found", errorPage[0])
	}

	// 適用は CSV を再送しない。保存済みのプレビューペイロードだけを指す。
	applyResponse := post("/realms/default/api/admin/v1/users/imports/"+previewID+"/apply", "")
	if applyID := jobID(applyResponse); applyID == "" {
		t.Fatal("apply job id is empty")
	}
	runJob(userusecases.UserImportJobHandler(jobDeps, userusecases.UserImportModeApply))

	alice, err := users.FindByUsername(context.Background(), tenancydomain.DefaultTenantID, "alice")
	if err != nil || alice == nil || alice.Email == nil || *alice.Email != "alice@example.com" {
		t.Fatalf("alice=%+v err=%v", alice, err)
	}
	// 有効な行のプロフィールとロールは、同じ 1 つの書き込みとして残る。
	if len(alice.Roles) != 1 || alice.Roles[0] != "support" {
		t.Fatalf("alice roles=%v, want the row applied atomically", alice.Roles)
	}
	// 無効な行は `rejected` として残り、User を作らない。
	if ghost, err := users.FindByUsername(context.Background(), tenancydomain.DefaultTenantID, "ghost"); err != nil || ghost != nil {
		t.Fatalf("拒否された行が User を作った: %+v, %v", ghost, err)
	}
}
