package handlers_http_test

// Group import の HTTP 契約。エラー一覧は管理一覧と同じ署名済みカーソルと
// `Pagination-*` で返り、ジョブの params と結果には CSV の本文もセル値も
// 現れない (REQ-IDMANAGEMENT-026)。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	httpdeps "github.com/ambi/idmagic/backend/idmanagement/deps_http"
	grouphttp "github.com/ambi/idmagic/backend/idmanagement/group/handlers_http"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func TestGetAdminGroupImportUsesManagementCursorPaginationForArtifactErrors(t *testing.T) {
	repo := usermemory.NewUserRepository()
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	tenantID := tenancydomain.DefaultTenantID
	repo.Seed(&userdomain.User{ID: "admin", TenantID: tenantID, PreferredUsername: "admin", PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now})
	artifacts := idmmemory.NewCSVArtifactStore()
	errorArtifact, err := artifacts.PutCSVArtifactPages(context.Background(), tenantID, func(emit func([]byte) error) error {
		for pageNumber := range 3 {
			page := make([]groupusecases.GroupImportRowError, 0, groupusecases.GroupImportErrorArtifactPageSize)
			for index := range groupusecases.GroupImportErrorArtifactPageSize {
				ordinal := pageNumber*groupusecases.GroupImportErrorArtifactPageSize + index + 1
				if ordinal > 450 {
					break
				}
				page = append(page, groupusecases.GroupImportRowError{Row: ordinal + 1, Column: "roles", Code: "invalid_roles"})
			}
			payload, err := json.Marshal(page)
			if err != nil {
				return err
			}
			if err := emit(payload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	jobs := jobsmemory.NewJobRepository()
	job, err := jobsusecases.Enqueue(context.Background(), jobsusecases.EnqueueDeps{Repo: jobs}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindGroupImportPreview, Params: []byte(`{"source_sha256":"source"}`), MaxAttempts: 1,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := jobs.ClaimBatch(context.Background(), "worker", jobsdomain.LaneBulk, 1, time.Minute, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed=%+v err=%v", claimed, err)
	}
	result, err := json.Marshal(groupusecases.GroupImportResult{
		SourceSHA256: "source", RejectedRows: 450, ErrorTotal: 450, DeletedRows: 2, DeletedMemberships: 7,
		ErrorArtifactRef: errorArtifact.Ref, ErrorArtifactSHA: errorArtifact.SHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Complete(context.Background(), job.ID, "worker", result, now); err != nil {
		t.Fatal(err)
	}

	codec := support.NewCursorCodec([]byte("group-import-pagination-test-secret"))
	e := echo.New()
	d := httpdeps.Deps{
		Issuer: "http://idp.test", PaginationCodec: codec,
		Authenticator: &support.Authenticator{UserRepo: repo, AuthnResolver: authusecases.DemoHeaderResolver{}},
		UserRepo:      repo, JobRepo: jobs, CSVArtifacts: artifacts,
	}
	e.GET("/api/admin/v1/groups/imports/:job_id", func(c *echo.Context) error { return grouphttp.HandleGetAdminGroupImport(d, c) })

	first := getGroupImportResultPage(t, e, "/api/admin/v1/groups/imports/"+job.ID+"?limit=100")
	if first.Code != http.StatusOK || first.Header().Get("Pagination-Total-Items") != "450" ||
		first.Header().Get("Pagination-Total-Pages") != "5" || first.Header().Get("Pagination-Current-Page") != "1" {
		t.Fatalf("status=%d headers=%v body=%s", first.Code, first.Header(), first.Body.String())
	}
	var body struct {
		Errors []groupusecases.GroupImportRowError `json:"errors"`
		Result groupusecases.GroupImportResult     `json:"result"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil || len(body.Errors) != 100 || body.Errors[0].Row != 2 || body.Errors[99].Row != 101 {
		t.Fatalf("errors=%+v err=%v", body.Errors, err)
	}
	// 削除は不可逆で cascade するため、件数は他の操作と分けて返る。
	if body.Result.DeletedRows != 2 || body.Result.DeletedMemberships != 7 {
		t.Fatalf("result = %+v, want the deletion counts reported apart from the other operations", body.Result)
	}
	// 行エラーは位置と安定コードだけを運び、セル値をレスポンスへ出さない。
	if strings.Contains(first.Body.String(), "catalog:read") || strings.Contains(first.Body.String(), "\"value\"") {
		t.Fatalf("the error page leaked a cell value: %s", first.Body.String())
	}

	nextURL := paginationLink(first.Header().Get("Link"), "next")
	if nextURL == "" {
		t.Fatalf("next link missing: %s", first.Header().Get("Link"))
	}
	second := getGroupImportResultPage(t, e, nextURL)
	if second.Header().Get("Pagination-Current-Page") != "2" || !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("headers=%v", second.Header())
	}
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil || len(body.Errors) != 100 || body.Errors[0].Row != 102 {
		t.Fatalf("second errors=%+v err=%v", body.Errors, err)
	}
}

// scenario REQ-IDMANAGEMENT-026: 別テナントのジョブ ID は解決せず、その拒否は
// エラー一覧も結果も返さない。
func TestGetAdminGroupImportRefusesAnotherTenantsJob(t *testing.T) {
	repo := usermemory.NewUserRepository()
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	repo.Seed(&userdomain.User{ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now})
	jobs := jobsmemory.NewJobRepository()
	foreign, err := jobsusecases.Enqueue(context.Background(), jobsusecases.EnqueueDeps{Repo: jobs}, jobsports.EnqueueInput{
		TenantID: "other-tenant", Kind: jobsdomain.KindGroupImportPreview, Params: []byte(`{"source_sha256":"source"}`), MaxAttempts: 1,
	}, now)
	if err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	d := httpdeps.Deps{
		Issuer: "http://idp.test", PaginationCodec: support.NewCursorCodec([]byte("group-import-tenant-test-secret")),
		Authenticator: &support.Authenticator{UserRepo: repo, AuthnResolver: authusecases.DemoHeaderResolver{}},
		UserRepo:      repo, JobRepo: jobs, CSVArtifacts: idmmemory.NewCSVArtifactStore(),
	}
	e.GET("/api/admin/v1/groups/imports/:job_id", func(c *echo.Context) error { return grouphttp.HandleGetAdminGroupImport(d, c) })

	response := getGroupImportResultPage(t, e, "/api/admin/v1/groups/imports/"+foreign.ID)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for another tenant's job", response.Code)
	}
	if strings.Contains(response.Body.String(), "\"errors\"") || strings.Contains(response.Body.String(), "\"result\"") {
		t.Fatalf("the refusal returned import data: %s", response.Body.String())
	}
}

func getGroupImportResultPage(t *testing.T, e *echo.Echo, target string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func paginationLink(header, relation string) string {
	pattern := regexp.MustCompile(`<([^>]+)>; rel="` + regexp.QuoteMeta(relation) + `"`)
	match := pattern.FindStringSubmatch(header)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}
