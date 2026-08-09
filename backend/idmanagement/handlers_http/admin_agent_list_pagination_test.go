package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// newAdminAgentPaginationHandler は idmanagement/handlers_http_test の既存 helper
// (newIdentityTestHandler) と異なり PaginationCodec を設定し、Agent repo を直接返す。
// 既存 helper は cursor pagination のテストには使えない (Deps.PaginationCodec が nil のまま
// だと SetNextLink/DecodeForQuery が nil pointer dereference するため)。
func newAdminAgentPaginationHandler(t *testing.T) (*echo.Echo, *agentmemory.AgentRepository) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", PreferredUsername: "admin", PasswordHash: "unused",
		Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
		TenantID: tenancydomain.DefaultTenantID,
	})
	agents := agentmemory.NewAgentRepository()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps:          support.Deps{Issuer: "http://idp.test", PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret"))},
		UserRepo:      users,
		AuthnResolver: authusecases.DemoHeaderResolver{},
		AgentRepo:     agents,
		GroupRepo:     groupmemory.NewGroupRepository(),
	})
	return e, agents
}

func adminAgentListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedAgent(t *testing.T, repo *agentmemory.AgentRepository, id, name string, now time.Time) {
	t.Helper()
	if err := repo.Save(context.Background(), &agentdomain.Agent{
		ID: id, TenantID: tenancydomain.DefaultTenantID, Name: name,
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "admin", Status: idmdomain.AgentStatusActive,
		Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
}

func TestAdminAgentListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newAdminAgentPaginationHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		seedAgent(t, repo, name+"-id", name, now)
	}

	resp := adminAgentListRequest(e, "/api/admin/v1/agents?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminAgentListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, repo := newAdminAgentPaginationHandler(t)
	seedAgent(t, repo, "solo-id", "solo", time.Now().UTC())

	resp := adminAgentListRequest(e, "/api/admin/v1/agents?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminAgentListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, repo := newAdminAgentPaginationHandler(t)
	now := time.Now().UTC()
	for _, name := range []string{"charlie", "delta", "echo"} {
		seedAgent(t, repo, name+"-id", name, now)
	}

	first := adminAgentListRequest(e, "/api/admin/v1/agents?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminAgentListRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeAgentListBody(t, first.Body.Bytes())
	secondBody := decodeAgentListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, a := range firstBody {
		seen[a["id"].(string)] = true
	}
	for _, a := range secondBody {
		id := a["id"].(string)
		if seen[id] {
			t.Fatalf("agent id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one agent")
	}
}

func TestAdminAgentListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminAgentPaginationHandler(t)
	resp := adminAgentListRequest(e, "/api/admin/v1/agents?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminAgentListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newAdminAgentPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "ListAgents",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminAgentListRequest(e, "/api/admin/v1/agents?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func decodeAgentListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Agents []map[string]any `json:"agents"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal agents body: %v (body=%s)", err, body)
	}
	return parsed.Agents
}
