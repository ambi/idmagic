package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authnmemory "github.com/ambi/idmagic/backend/authentication/db_memory"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// newAdminAuthEventBucketPaginationHandler は authentication/handlers_http 配下に他の
// admin ハンドラのような共有 test helper が存在しないため新規に用意する。RequireAuditReader
// (admin/system_admin 許可) を通す admin actor と、PaginationCodec を設定した Deps を返す。
func newAdminAuthEventBucketPaginationHandler(t *testing.T) (*echo.Echo, *authnmemory.AuthEventBucketStore) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	store := authnmemory.NewAuthEventBucketStore()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		UserRepo:      users,
		AuthnResolver: authusecases.DemoHeaderResolver{},
		Authentication: authentication.Module{
			AuthEventBucketStore: store,
		},
	})
	return e, store
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

func adminAuthEventBucketListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedAuthEventBucket(t *testing.T, store *authnmemory.AuthEventBucketStore, keyHash string, now time.Time) {
	t.Helper()
	if _, err := store.Record(t.Context(), authnports.AuthEventBucketFailedLogin, tenancydomain.DefaultTenantID, keyHash, now); err != nil {
		t.Fatalf("record auth event bucket: %v", err)
	}
}

func decodeAuthEventBucketListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Buckets []map[string]any `json:"buckets"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal buckets body: %v (body=%s)", err, body)
	}
	return parsed.Buckets
}

func TestAdminAuthEventBucketListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, store := newAdminAuthEventBucketPaginationHandler(t)
	now := time.Now().UTC()
	for _, keyHash := range []string{"charlie", "delta", "echo"} {
		seedAuthEventBucket(t, store, keyHash, now)
	}

	resp := adminAuthEventBucketListRequest(e, "/api/admin/v1/authentication_event_buckets?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminAuthEventBucketListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, store := newAdminAuthEventBucketPaginationHandler(t)
	seedAuthEventBucket(t, store, "solo", time.Now().UTC())

	resp := adminAuthEventBucketListRequest(e, "/api/admin/v1/authentication_event_buckets?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminAuthEventBucketListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, store := newAdminAuthEventBucketPaginationHandler(t)
	now := time.Now().UTC()
	for _, keyHash := range []string{"charlie", "delta", "echo"} {
		seedAuthEventBucket(t, store, keyHash, now)
	}

	first := adminAuthEventBucketListRequest(e, "/api/admin/v1/authentication_event_buckets?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminAuthEventBucketListRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeAuthEventBucketListBody(t, first.Body.Bytes())
	secondBody := decodeAuthEventBucketListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, b := range firstBody {
		seen[b["key_hash"].(string)] = true
	}
	for _, b := range secondBody {
		key := b["key_hash"].(string)
		if seen[key] {
			t.Fatalf("key_hash %q appeared on both pages", key)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one bucket")
	}
}

func TestAdminAuthEventBucketListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminAuthEventBucketPaginationHandler(t)
	resp := adminAuthEventBucketListRequest(e, "/api/admin/v1/authentication_event_buckets?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminAuthEventBucketListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newAdminAuthEventBucketPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "ListAuthenticationEventBuckets",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminAuthEventBucketListRequest(e, "/api/admin/v1/authentication_event_buckets?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
