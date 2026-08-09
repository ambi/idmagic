package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

func adminUserListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func decodeAdminUserListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Users []map[string]any `json:"users"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal users body: %v (body=%s)", err, body)
	}
	return parsed.Users
}

func linkURLForRel(t *testing.T, header, rel string) string {
	t.Helper()
	for part := range strings.SplitSeq(header, ", ") {
		if strings.Contains(part, `rel="`+rel+`"`) {
			return strings.TrimPrefix(part[strings.Index(part, "<")+1:strings.Index(part, ">")], "http://idp.test")
		}
	}
	t.Fatalf("Link header missing rel=%s: %q", rel, header)
	return ""
}

func TestAdminUserListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newAdminUserHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		repo.Seed(&userdomain.User{
			ID: name + "-id", PreferredUsername: name, PasswordHash: "unused",
			CreatedAt: now, UpdatedAt: now,
		})
	}

	resp := adminUserListRequest(e, "/api/admin/v1/users?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header when more pages exist")
	}
	if !strings.Contains(link, `rel="next"`) {
		t.Fatalf("Link header missing rel=next: %q", link)
	}
	if !strings.Contains(link, `rel="last"`) {
		t.Fatalf("Link header missing rel=last: %q", link)
	}
	if resp.Header().Get("Pagination-Total-Items") != "5" || resp.Header().Get("Pagination-Total-Pages") != "3" || resp.Header().Get("Pagination-Current-Page") != "1" || resp.Header().Get("Pagination-Page-Size") != "2" {
		t.Fatalf("unexpected pagination headers: %#v", resp.Header())
	}
	if !strings.Contains(link, "cursor=") {
		t.Fatalf("Link header missing cursor param: %q", link)
	}
	users := decodeAdminUserListBody(t, resp.Body.Bytes())
	if len(users) != 2 {
		t.Fatalf("expected 2 users on a limit=2 first page, got %d", len(users))
	}
	var body struct {
		TotalUsers int64 `json:"total_users"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil || body.TotalUsers != 5 {
		t.Fatalf("total_users=%d err=%v body=%s", body.TotalUsers, err, resp.Body.String())
	}
}

func TestAdminUserListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, _ := newAdminUserHandler(t)
	// newAdminUserHandler already seeds 2 users (admin, regular); limit is
	// large enough to cover both in one page.
	resp := adminUserListRequest(e, "/api/admin/v1/users?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminUserListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, repo := newAdminUserHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		repo.Seed(&userdomain.User{
			ID: name + "-id", PreferredUsername: name, PasswordHash: "unused",
			CreatedAt: now, UpdatedAt: now,
		})
	}

	first := adminUserListRequest(e, "/api/admin/v1/users?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminUserListRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	firstUsers := decodeAdminUserListBody(t, first.Body.Bytes())
	secondUsers := decodeAdminUserListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, u := range firstUsers {
		seen[u["preferred_username"].(string)] = true
	}
	for _, u := range secondUsers {
		name := u["preferred_username"].(string)
		if seen[name] {
			t.Fatalf("username %q appeared on both pages", name)
		}
	}
	if len(secondUsers) == 0 {
		t.Fatal("expected the second page to return at least one user")
	}
}

func TestAdminUserListPreviousLinkReturnsPriorPage(t *testing.T) {
	e, repo := newAdminUserHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		repo.Seed(&userdomain.User{ID: name + "-id", PreferredUsername: name, PasswordHash: "unused", CreatedAt: now, UpdatedAt: now})
	}
	first := adminUserListRequest(e, "/api/admin/v1/users?limit=2")
	second := adminUserListRequest(e, linkURLForRel(t, first.Header().Get("Link"), "next"))
	if second.Code != http.StatusOK {
		t.Fatalf("second status=%d body=%s", second.Code, second.Body.String())
	}
	previous := adminUserListRequest(e, linkURLForRel(t, second.Header().Get("Link"), "prev"))
	if previous.Code != http.StatusOK {
		t.Fatalf("previous status=%d body=%s", previous.Code, previous.Body.String())
	}
	firstUsers := decodeAdminUserListBody(t, first.Body.Bytes())
	previousUsers := decodeAdminUserListBody(t, previous.Body.Bytes())
	if len(firstUsers) != len(previousUsers) {
		t.Fatalf("previous page length=%d, want %d", len(previousUsers), len(firstUsers))
	}
	for i := range firstUsers {
		if firstUsers[i]["id"] != previousUsers[i]["id"] {
			t.Fatalf("previous page differs at %d: got=%v want=%v", i, previousUsers[i], firstUsers[i])
		}
	}
}

func TestAdminUserListSearchAndStatusApplyBeforePaging(t *testing.T) {
	e, repo := newAdminUserHandler(t)
	now := time.Now().UTC()
	repo.Seed(&userdomain.User{
		ID: "alice-id", PreferredUsername: "alice", Name: new("Alice Example"),
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})

	resp := adminUserListRequest(e, "/api/admin/v1/users?query=EXAMPLE&status=active&limit=1")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	users := decodeAdminUserListBody(t, resp.Body.Bytes())
	if len(users) != 1 || users[0]["preferred_username"] != "alice" {
		t.Fatalf("unexpected filtered users: %+v", users)
	}
	if resp.Header().Get("Pagination-Total-Items") != "1" {
		t.Fatalf("filtered total items = %q, want 1", resp.Header().Get("Pagination-Total-Items"))
	}
	var body struct {
		TotalUsers int64 `json:"total_users"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil || body.TotalUsers != 3 {
		t.Fatalf("total_users=%d err=%v body=%s", body.TotalUsers, err, resp.Body.String())
	}
}

func TestAdminUserListLastLinkReturnsOnlyRemainder(t *testing.T) {
	e, repo := newAdminUserHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		repo.Seed(&userdomain.User{ID: name + "-id", PreferredUsername: name, PasswordHash: "unused", CreatedAt: now, UpdatedAt: now})
	}
	first := adminUserListRequest(e, "/api/admin/v1/users?limit=2")
	last := adminUserListRequest(e, linkURLForRel(t, first.Header().Get("Link"), "last"))
	if last.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", last.Code, last.Body.String())
	}
	users := decodeAdminUserListBody(t, last.Body.Bytes())
	if len(users) != 1 {
		t.Fatalf("last page length=%d, want 1: %+v", len(users), users)
	}
	if last.Header().Get("Pagination-Current-Page") != "3" || last.Header().Get("Pagination-Total-Pages") != "3" {
		t.Fatalf("last page metadata: %#v", last.Header())
	}
	link := last.Header().Get("Link")
	if !strings.Contains(link, `rel="first"`) || !strings.Contains(link, `rel="prev"`) || strings.Contains(link, `rel="last"`) {
		t.Fatalf("last page links=%q", link)
	}
}

func TestAdminUserListRejectsCursorAfterQueryChanges(t *testing.T) {
	e, repo := newAdminUserHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		repo.Seed(&userdomain.User{ID: name + "-id", PreferredUsername: name, PasswordHash: "unused", CreatedAt: now, UpdatedAt: now})
	}
	first := adminUserListRequest(e, "/api/admin/v1/users?query=a&limit=1")
	nextURL, err := url.Parse(linkURLForRel(t, first.Header().Get("Link"), "next"))
	if err != nil {
		t.Fatalf("parse next link: %v", err)
	}
	query := nextURL.Query()
	query.Set("query", "e")
	nextURL.RawQuery = query.Encode()

	changed := adminUserListRequest(e, nextURL.String())
	if changed.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", changed.Code, changed.Body.String())
	}
}

func TestAdminUserListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminUserHandler(t)
	resp := adminUserListRequest(e, "/api/admin/v1/users?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminUserListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newAdminUserHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "ListAdminUsers",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminUserListRequest(e, "/api/admin/v1/users?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
