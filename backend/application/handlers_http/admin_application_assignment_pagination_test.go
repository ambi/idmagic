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
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"
	"github.com/labstack/echo/v5"
)

// newApplicationAssignmentPaginationHandler は Application/Assignment 両方の memory repo を
// 露出させた上で PaginationCodec を設定する独立 helper。newApplicationPaginationHandler は
// AssignmentRepo を内部生成して外に返さないため、assignment 一覧の pagination テストには
// 使えない (シードした行が見えない)。対象 application はここで1件事前作成する。
func newApplicationAssignmentPaginationHandler(t *testing.T) (*echo.Echo, *appmemory.ApplicationAssignmentRepository, string) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	appRepo := appmemory.NewApplicationRepository()
	assignmentRepo := appmemory.NewApplicationAssignmentRepository()
	appID := "app-1"
	if err := appRepo.Save(context.Background(), &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: appID, Name: "target-app",
		Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive,
		LaunchURL: "https://example.com/target", CategoryIDs: []string{},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed application: %v", err)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:          "http://idp.test",
			Emit:            func(spec.DomainEvent) {},
			PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		},
		UserRepo:  users,
		GroupRepo: groupmemory.NewGroupRepository(),
		Application: application.Module{
			Repo:                    appRepo,
			IconStore:               appmemory.NewApplicationIconStore(),
			AssignmentRepo:          assignmentRepo,
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
		},
		Saml:          saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository()},
		OAuth2:        oauth2.Module{ClientRepo: oauth2memory.NewClientRepository()},
		WsFederation:  wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, assignmentRepo, appID
}

func adminApplicationAssignmentListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedApplicationAssignment(t *testing.T, repo *appmemory.ApplicationAssignmentRepository, appID, subjectID string, now time.Time) {
	t.Helper()
	if err := repo.Save(context.Background(), &appdomain.ApplicationAssignment{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: appID,
		SubjectType: appdomain.AssignmentSubjectGroup, SubjectID: subjectID,
		Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed application assignment: %v", err)
	}
}

func decodeApplicationAssignmentListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Assignments []map[string]any `json:"assignments"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal assignments body: %v (body=%s)", err, body)
	}
	return parsed.Assignments
}

func TestAdminApplicationAssignmentListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, assignmentRepo, appID := newApplicationAssignmentPaginationHandler(t)
	now := time.Now().UTC()
	for _, subjectID := range []string{"charlie-id", "delta-id", "echo-id"} {
		seedApplicationAssignment(t, assignmentRepo, appID, subjectID, now)
	}

	resp := adminApplicationAssignmentListRequest(e, "/api/admin/v1/applications/"+appID+"/assignments?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminApplicationAssignmentListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, assignmentRepo, appID := newApplicationAssignmentPaginationHandler(t)
	seedApplicationAssignment(t, assignmentRepo, appID, "solo-id", time.Now().UTC())

	resp := adminApplicationAssignmentListRequest(e, "/api/admin/v1/applications/"+appID+"/assignments?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminApplicationAssignmentListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, assignmentRepo, appID := newApplicationAssignmentPaginationHandler(t)
	now := time.Now().UTC()
	for _, subjectID := range []string{"charlie-id", "delta-id", "echo-id"} {
		seedApplicationAssignment(t, assignmentRepo, appID, subjectID, now)
	}

	first := adminApplicationAssignmentListRequest(e, "/api/admin/v1/applications/"+appID+"/assignments?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminApplicationAssignmentListRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeApplicationAssignmentListBody(t, first.Body.Bytes())
	secondBody := decodeApplicationAssignmentListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, a := range firstBody {
		seen[a["subject_id"].(string)] = true
	}
	for _, a := range secondBody {
		id := a["subject_id"].(string)
		if seen[id] {
			t.Fatalf("subject id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one assignment")
	}
}

func TestAdminApplicationAssignmentListRejectsInvalidCursor(t *testing.T) {
	e, _, appID := newApplicationAssignmentPaginationHandler(t)
	resp := adminApplicationAssignmentListRequest(e, "/api/admin/v1/applications/"+appID+"/assignments?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminApplicationAssignmentListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _, appID := newApplicationAssignmentPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "ListApplicationAssignments",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminApplicationAssignmentListRequest(e, "/api/admin/v1/applications/"+appID+"/assignments?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
