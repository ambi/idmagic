package handlers_http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func adminGroupListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func TestAdminGroupListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newAdminGroupHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"Charlie", "Delta", "Echo"} {
		_ = repo.Save(t.Context(), &groupdomain.Group{
			ID: name + "-id", TenantID: tenancydomain.DefaultTenantID, Name: name, Roles: []string{},
			CreatedAt: now, UpdatedAt: now,
		})
	}

	resp := adminGroupListRequest(e, "/api/admin/v1/groups?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
	if !strings.Contains(link, `rel="last"`) || resp.Header().Get("Pagination-Total-Items") != "3" || resp.Header().Get("Pagination-Total-Pages") != "2" || resp.Header().Get("Pagination-Current-Page") != "1" || resp.Header().Get("Pagination-Page-Size") != "2" {
		t.Fatalf("unexpected pagination contract: Link=%q headers=%#v", link, resp.Header())
	}
}

func TestAdminGroupListSecondPageExposesPreviousDirection(t *testing.T) {
	e, repo := newAdminGroupHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"Charlie", "Delta", "Echo"} {
		_ = repo.Save(t.Context(), &groupdomain.Group{
			ID: name + "-id", TenantID: tenancydomain.DefaultTenantID, Name: name, Roles: []string{},
			CreatedAt: now, UpdatedAt: now,
		})
	}
	first := adminGroupListRequest(e, "/api/admin/v1/groups?limit=2")
	link := first.Header().Get("Link")
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")
	second := adminGroupListRequest(e, nextPath)
	if second.Code != http.StatusOK || !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second status=%d Link=%q body=%s", second.Code, second.Header().Get("Link"), second.Body.String())
	}
}

func TestAdminGroupListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, _ := newAdminGroupHandler(t)
	resp := adminGroupListRequest(e, "/api/admin/v1/groups?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminGroupListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminGroupHandler(t)
	resp := adminGroupListRequest(e, "/api/admin/v1/groups?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
