package handlers_http_test

import (
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
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// newAdminOAuth2ClientPaginationHandler は既存 newAdminOAuth2ClientHandler と異なり
// PaginationCodec を設定する。既存 helper は Deps.PaginationCodec が nil のままなので
// cursor pagination のテストには使えない。
func newAdminOAuth2ClientPaginationHandler(t *testing.T) (*echo.Echo, *oauth2memory.OAuth2ClientRepository) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	clients := oauth2memory.NewClientRepository()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://idp.test",
		Emit:            func(spec.DomainEvent) {},
		PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		UserRepo:        users,
		OAuth2:          oauth2.Module{ClientRepo: clients},
		AuthnResolver:   authusecases.DemoHeaderResolver{},
	})
	return e, clients
}

func adminClientListPageRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedOAuth2Client(t *testing.T, repo *oauth2memory.OAuth2ClientRepository, clientID string, now time.Time) {
	t.Helper()
	repo.Seed(&oauthdomain.OAuth2Client{
		TenantID: tenancydomain.DefaultTenantID, ClientID: clientID, ClientType: spec.ClientPublic,
		RedirectURIs:            []string{"https://" + clientID + ".example/callback"},
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: oauthdomain.AuthMethodNone, IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile: oauthdomain.FapiNone, CreatedAt: now,
	})
}

func decodeClientListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Clients []map[string]any `json:"clients"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal clients body: %v (body=%s)", err, body)
	}
	return parsed.Clients
}

func TestAdminOAuth2ClientListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newAdminOAuth2ClientPaginationHandler(t)
	now := time.Now().UTC()
	for _, clientID := range []string{"charlie", "delta", "echo"} {
		seedOAuth2Client(t, repo, clientID, now)
	}

	resp := adminClientListPageRequest(e, "/api/admin/v1/clients?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminOAuth2ClientListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, repo := newAdminOAuth2ClientPaginationHandler(t)
	seedOAuth2Client(t, repo, "solo", time.Now().UTC())

	resp := adminClientListPageRequest(e, "/api/admin/v1/clients?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminOAuth2ClientListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, repo := newAdminOAuth2ClientPaginationHandler(t)
	now := time.Now().UTC()
	for _, clientID := range []string{"charlie", "delta", "echo"} {
		seedOAuth2Client(t, repo, clientID, now)
	}

	first := adminClientListPageRequest(e, "/api/admin/v1/clients?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminClientListPageRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeClientListBody(t, first.Body.Bytes())
	secondBody := decodeClientListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, c := range firstBody {
		seen[c["client_id"].(string)] = true
	}
	for _, c := range secondBody {
		id := c["client_id"].(string)
		if seen[id] {
			t.Fatalf("client id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one client")
	}
}

func TestAdminOAuth2ClientListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminOAuth2ClientPaginationHandler(t)
	resp := adminClientListPageRequest(e, "/api/admin/v1/clients?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminOAuth2ClientListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newAdminOAuth2ClientPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "ListAdminOAuth2Clients",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminClientListPageRequest(e, "/api/admin/v1/clients?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
