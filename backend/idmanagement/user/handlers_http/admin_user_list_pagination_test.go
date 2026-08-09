package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	if !strings.Contains(link, "cursor=") {
		t.Fatalf("Link header missing cursor param: %q", link)
	}
	users := decodeAdminUserListBody(t, resp.Body.Bytes())
	if len(users) != 2 {
		t.Fatalf("expected 2 users on a limit=2 first page, got %d", len(users))
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
