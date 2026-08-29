package handlers_http_test

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
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userhttp "github.com/ambi/idmagic/backend/idmanagement/user/handlers_http"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func TestGetAdminUserImportUsesManagementCursorPaginationForArtifactErrors(t *testing.T) {
	repo := usermemory.NewUserRepository()
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	tenantID := tenancydomain.DefaultTenantID
	repo.Seed(&userdomain.User{ID: "admin", TenantID: tenantID, PreferredUsername: "admin", PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now})
	artifacts := idmmemory.NewCSVArtifactStore()
	errorArtifact, err := artifacts.PutCSVArtifactPages(context.Background(), tenantID, func(emit func([]byte) error) error {
		for pageNumber := range 3 {
			page := make([]userusecases.UserImportRowError, 0, userusecases.UserImportErrorArtifactPageSize)
			for index := range userusecases.UserImportErrorArtifactPageSize {
				ordinal := pageNumber*userusecases.UserImportErrorArtifactPageSize + index + 1
				if ordinal > 450 {
					break
				}
				page = append(page, userusecases.UserImportRowError{Row: ordinal + 1, Column: "email", Code: "invalid_email"})
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
		TenantID: tenantID, Kind: jobsdomain.KindUserImportPreview, Params: []byte(`{"source_sha256":"source"}`), MaxAttempts: 1,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := jobs.ClaimBatch(context.Background(), "worker", jobsdomain.LaneBulk, 1, time.Minute, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed=%+v err=%v", claimed, err)
	}
	result, err := json.Marshal(userusecases.UserImportResult{
		SourceSHA256: "source", RejectedRows: 450, ErrorTotal: 450,
		ErrorArtifactRef: errorArtifact.Ref, ErrorArtifactSHA: errorArtifact.SHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Complete(context.Background(), job.ID, "worker", result, now); err != nil {
		t.Fatal(err)
	}

	codec := support.NewCursorCodec([]byte("user-import-pagination-test-secret"))
	e := echo.New()
	d := httpdeps.Deps{
		Issuer: "http://idp.test", PaginationCodec: codec,
		Authenticator: &support.Authenticator{UserRepo: repo, AuthnResolver: authusecases.DemoHeaderResolver{}},
		UserRepo:      repo, JobRepo: jobs, CSVArtifacts: artifacts,
	}
	e.GET("/api/admin/v1/users/imports/:job_id", func(c *echo.Context) error { return userhttp.HandleGetAdminUserImport(d, c) })

	first := getImportResultPage(t, e, "/api/admin/v1/users/imports/"+job.ID+"?limit=100")
	if first.Code != http.StatusOK || first.Header().Get("Pagination-Total-Items") != "450" || first.Header().Get("Pagination-Total-Pages") != "5" || first.Header().Get("Pagination-Current-Page") != "1" {
		t.Fatalf("status=%d headers=%v body=%s", first.Code, first.Header(), first.Body.String())
	}
	var body struct {
		Errors []userusecases.UserImportRowError `json:"errors"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil || len(body.Errors) != 100 || body.Errors[0].Row != 2 || body.Errors[99].Row != 101 {
		t.Fatalf("errors=%+v err=%v", body.Errors, err)
	}
	nextURL := paginationLink(first.Header().Get("Link"), "next")
	if nextURL == "" {
		t.Fatalf("next link missing: %s", first.Header().Get("Link"))
	}
	second := getImportResultPage(t, e, nextURL)
	if second.Header().Get("Pagination-Current-Page") != "2" || !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("headers=%v", second.Header())
	}
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil || len(body.Errors) != 100 || body.Errors[0].Row != 102 {
		t.Fatalf("second errors=%+v err=%v", body.Errors, err)
	}
}

func getImportResultPage(t *testing.T, e *echo.Echo, target string) *httptest.ResponseRecorder {
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
