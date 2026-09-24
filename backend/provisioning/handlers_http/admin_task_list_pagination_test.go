package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/provisioning"
	provisioningmemory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// newAdminTaskPaginationHandler は provisioning/handlers_http 配下に他の admin
// ハンドラのような共有 test helper が存在しないため新規に用意する。GetTask/RetryTask
// は connection_id (= application id) の文字列一致しか見ないため、実在する Application/
// ProvisioningConnection レコードは不要 (usecases/admin_test.go と同じ前提)。
func newAdminTaskPaginationHandler(t *testing.T) (*echo.Echo, *provisioningmemory.ProvisioningTaskRepository) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	taskRepo := provisioningmemory.NewProvisioningTaskRepository()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://idp.test",
		Emit:            func(spec.DomainEvent) {},
		PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		UserRepo:        users,
		GroupRepo:       groupmemory.NewGroupRepository(),
		Application: application.Module{
			Repo:                    appmemory.NewApplicationRepository(),
			IconStore:               appmemory.NewApplicationIconStore(),
			AssignmentRepo:          appmemory.NewApplicationAssignmentRepository(),
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
		},
		Provisioning: provisioning.Module{
			ConnectionRepo: provisioningmemory.NewProvisioningConnectionRepository(),
			RemoteLinkRepo: provisioningmemory.NewRemoteResourceLinkRepository(),
			TaskRepo:       taskRepo,
		},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, taskRepo
}

// defaultRealmPath は bare path を default テナントの正規ロケーション配下へ移す。
// 他パッケージ (例: idmanagement/handlers_http) の同名 helper を複製したもの
// (_test.go はパッケージを跨げないため、Phase 2 と同方針)。
func defaultRealmPath(path string) string {
	if strings.HasPrefix(path, "/realms/") {
		return path
	}
	return "/realms/default" + path
}

func adminTaskListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedTask(t *testing.T, repo *provisioningmemory.ProvisioningTaskRepository, id string, now time.Time) {
	t.Helper()
	// SourceID を id ごとに変える: Save の idempotency key は
	// (TenantID, ConnectionID, SourceType, SourceID, SourceVersion) なので、同一 SourceID/
	// SourceVersion で複数件シードすると 2件目以降が無視される (IdempotencyKey 参照)。
	if _, err := repo.Save(context.Background(), &provisioningdomain.ProvisioningTask{
		ID: id, TenantID: tenancydomain.DefaultTenantID, ConnectionID: "app-1",
		SourceType: provisioningdomain.SourceTypeUser, SourceID: id, SourceVersion: 1,
		Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.TaskPending,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}
}

func decodeTaskListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal tasks body: %v (body=%s)", err, body)
	}
	return parsed.Tasks
}

func TestAdminTaskListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newAdminTaskPaginationHandler(t)
	base := time.Now().UTC().Add(-time.Hour)
	for i, id := range []string{"charlie", "delta", "echo"} {
		seedTask(t, repo, id, base.Add(time.Duration(i)*time.Minute))
	}

	resp := adminTaskListRequest(e, "/api/admin/v1/applications/app-1/provisioning/tasks?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminTaskListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, repo := newAdminTaskPaginationHandler(t)
	seedTask(t, repo, "solo", time.Now().UTC())

	resp := adminTaskListRequest(e, "/api/admin/v1/applications/app-1/provisioning/tasks?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminTaskListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, repo := newAdminTaskPaginationHandler(t)
	base := time.Now().UTC().Add(-time.Hour)
	for i, id := range []string{"charlie", "delta", "echo"} {
		seedTask(t, repo, id, base.Add(time.Duration(i)*time.Minute))
	}

	first := adminTaskListRequest(e, "/api/admin/v1/applications/app-1/provisioning/tasks?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminTaskListRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeTaskListBody(t, first.Body.Bytes())
	secondBody := decodeTaskListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, d := range firstBody {
		seen[d["id"].(string)] = true
	}
	for _, d := range secondBody {
		id := d["id"].(string)
		if seen[id] {
			t.Fatalf("task id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one task")
	}
}

func TestAdminTaskListFiltersBySourceType(t *testing.T) {
	e, repo := newAdminTaskPaginationHandler(t)
	now := time.Now().UTC()
	seedTask(t, repo, "user-task", now)
	if _, err := repo.Save(context.Background(), &provisioningdomain.ProvisioningTask{
		ID: "group-task", TenantID: tenancydomain.DefaultTenantID, ConnectionID: "app-1",
		SourceType: provisioningdomain.SourceTypeGroup, SourceID: "group-1", SourceVersion: 1,
		Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.TaskPending,
		CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("seed group task: %v", err)
	}

	resp := adminTaskListRequest(e, "/api/admin/v1/applications/app-1/provisioning/tasks?source_type=group")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	tasks := decodeTaskListBody(t, resp.Body.Bytes())
	if len(tasks) != 1 || tasks[0]["id"] != "group-task" {
		t.Fatalf("unexpected source_type result: %+v", tasks)
	}
}

func TestAdminTaskListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminTaskPaginationHandler(t)
	resp := adminTaskListRequest(e, "/api/admin/v1/applications/app-1/provisioning/tasks?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if contentType := resp.Header().Get("Content-Type"); contentType != support.ProblemContentType {
		t.Fatalf("Content-Type=%q, want %q", contentType, support.ProblemContentType)
	}
	if !strings.Contains(resp.Body.String(), "urn:idmagic:error:invalid_request") {
		t.Fatalf("unexpected body=%s", resp.Body.String())
	}
}

//spec:covers REQ-PROVISIONING-015: 別テナントで発行したカーソルでプロビジョニングタスクの一覧を読むと、400 で拒否される。
func TestAdminTaskListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newAdminTaskPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	// provisioningTasksQueryHash はフィルタ無しのリクエスト (?limit=... のみ) では
	// 空文字列になる (cursor/limit を除いた残りの query をそのまま hash にするため)。
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminTaskListRequest(e, "/api/admin/v1/applications/app-1/provisioning/tasks?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
