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
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// newAdminConsentPaginationHandler は既存 newAdminConsentHandler と異なり PaginationCodec を
// 設定する。既存 helper は Deps.PaginationCodec が nil のままなので cursor pagination の
// テストには使えない。
func newAdminConsentPaginationHandler(t *testing.T) (*echo.Echo, *oauth2memory.ConsentRepository) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	consents := oauth2memory.NewConsentRepository()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer:          "http://idp.test",
			Emit:            func(spec.DomainEvent) {},
			PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		},
		UserRepo:      users,
		OAuth2:        oauth2.Module{ConsentRepo: consents},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, consents
}

func adminConsentListPageRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedConsent(t *testing.T, repo *oauth2memory.ConsentRepository, userID, clientID string, now time.Time) {
	t.Helper()
	if err := repo.Save(context.Background(), tenancydomain.DefaultTenantID, &oauthdomain.Consent{
		UserID: userID, ClientID: clientID, Scopes: []string{"openid"},
		State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("seed consent: %v", err)
	}
}

func decodeConsentListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Consents []map[string]any `json:"consents"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal consents body: %v (body=%s)", err, body)
	}
	return parsed.Consents
}

func TestAdminConsentListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newAdminConsentPaginationHandler(t)
	now := time.Now().UTC()
	for _, userID := range []string{"charlie", "delta", "echo"} {
		seedConsent(t, repo, userID, "client-1", now)
	}

	resp := adminConsentListPageRequest(e, "/api/admin/v1/consents?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminConsentListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, repo := newAdminConsentPaginationHandler(t)
	seedConsent(t, repo, "solo", "client-1", time.Now().UTC())

	resp := adminConsentListPageRequest(e, "/api/admin/v1/consents?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminConsentListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, repo := newAdminConsentPaginationHandler(t)
	now := time.Now().UTC()
	for _, userID := range []string{"charlie", "delta", "echo"} {
		seedConsent(t, repo, userID, "client-1", now)
	}

	first := adminConsentListPageRequest(e, "/api/admin/v1/consents?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminConsentListPageRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	firstBody := decodeConsentListBody(t, first.Body.Bytes())
	secondBody := decodeConsentListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, c := range firstBody {
		seen[c["user_id"].(string)] = true
	}
	for _, c := range secondBody {
		id := c["user_id"].(string)
		if seen[id] {
			t.Fatalf("user id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one consent")
	}
}

func TestAdminConsentListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminConsentPaginationHandler(t)
	resp := adminConsentListPageRequest(e, "/api/admin/v1/consents?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminConsentListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newAdminConsentPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "ListAdminConsents",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminConsentListPageRequest(e, "/api/admin/v1/consents?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
