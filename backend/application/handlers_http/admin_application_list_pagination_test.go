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

// newApplicationPaginationHandler は application/handlers_http_test の既存 newApplicationHandler
// と異なり PaginationCodec を設定し、Application repo を直接返す。既存 helper は
// Deps.PaginationCodec が nil のままなので cursor pagination のテストには使えない。
func newApplicationPaginationHandler(t *testing.T) (*echo.Echo, *appmemory.ApplicationRepository) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	appRepo := appmemory.NewApplicationRepository()
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
			AssignmentRepo:          appmemory.NewApplicationAssignmentRepository(),
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
		},
		Saml:          saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository()},
		OAuth2:        oauth2.Module{ClientRepo: oauth2memory.NewClientRepository()},
		WsFederation:  wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, appRepo
}

func adminApplicationListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedApplication(t *testing.T, repo *appmemory.ApplicationRepository, id, name string, now time.Time) {
	t.Helper()
	if err := repo.Save(context.Background(), &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: id, Name: name,
		Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive,
		LaunchURL: "https://example.com/" + id, CategoryIDs: []string{},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed application: %v", err)
	}
}

func decodeApplicationListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Applications []map[string]any `json:"applications"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal applications body: %v (body=%s)", err, body)
	}
	return parsed.Applications
}

func TestAdminApplicationListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newApplicationPaginationHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		seedApplication(t, repo, name+"-id", name, now)
	}

	resp := adminApplicationListRequest(e, "/api/admin/v1/applications?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminApplicationListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, repo := newApplicationPaginationHandler(t)
	seedApplication(t, repo, "solo-id", "solo", time.Now().UTC())

	resp := adminApplicationListRequest(e, "/api/admin/v1/applications?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminApplicationListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, repo := newApplicationPaginationHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		seedApplication(t, repo, name+"-id", name, now)
	}

	first := adminApplicationListRequest(e, "/api/admin/v1/applications?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminApplicationListRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeApplicationListBody(t, first.Body.Bytes())
	secondBody := decodeApplicationListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, a := range firstBody {
		seen[a["id"].(string)] = true
	}
	for _, a := range secondBody {
		id := a["id"].(string)
		if seen[id] {
			t.Fatalf("application id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one application")
	}
}

func TestAdminApplicationListRejectsInvalidCursor(t *testing.T) {
	e, _ := newApplicationPaginationHandler(t)
	resp := adminApplicationListRequest(e, "/api/admin/v1/applications?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminApplicationListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newApplicationPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "ListAdminApplications",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminApplicationListRequest(e, "/api/admin/v1/applications?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
