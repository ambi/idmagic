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

// newAdminDeliveryPaginationHandler は provisioning/handlers_http 配下に他の admin
// ハンドラのような共有 test helper が存在しないため新規に用意する。GetDelivery/RetryDelivery
// は connection_id (= application id) の文字列一致しか見ないため、実在する Application/
// ProvisioningConnection レコードは不要 (usecases/admin_test.go と同じ前提)。
func newAdminDeliveryPaginationHandler(t *testing.T) (*echo.Echo, *provisioningmemory.ProvisioningDeliveryRepository) {
	t.Helper()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	deliveryRepo := provisioningmemory.NewProvisioningDeliveryRepository()
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
			DeliveryRepo:   deliveryRepo,
		},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e, deliveryRepo
}

// defaultRealmPath は bare path を default テナントの正規ロケーション配下へ移す (ADR-144)。
// 他パッケージ (例: idmanagement/handlers_http) の同名 helper を複製したもの
// (_test.go はパッケージを跨げないため、ADR-130 Phase 2 と同方針)。
func defaultRealmPath(path string) string {
	if strings.HasPrefix(path, "/realms/") {
		return path
	}
	return "/realms/default" + path
}

func adminDeliveryListRequest(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func seedDelivery(t *testing.T, repo *provisioningmemory.ProvisioningDeliveryRepository, id string, now time.Time) {
	t.Helper()
	// SourceID を id ごとに変える: Save の idempotency key は
	// (TenantID, ConnectionID, SourceType, SourceID, SourceVersion) なので、同一 SourceID/
	// SourceVersion で複数件シードすると 2件目以降が無視される (IdempotencyKey 参照)。
	if _, err := repo.Save(context.Background(), &provisioningdomain.ProvisioningDelivery{
		ID: id, TenantID: tenancydomain.DefaultTenantID, ConnectionID: "app-1",
		SourceType: provisioningdomain.SourceTypeUser, SourceID: id, SourceVersion: 1,
		Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.DeliveryPending,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed delivery: %v", err)
	}
}

func decodeDeliveryListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Deliveries []map[string]any `json:"deliveries"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal deliveries body: %v (body=%s)", err, body)
	}
	return parsed.Deliveries
}

func TestAdminDeliveryListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	e, repo := newAdminDeliveryPaginationHandler(t)
	base := time.Now().UTC().Add(-time.Hour)
	for i, id := range []string{"charlie", "delta", "echo"} {
		seedDelivery(t, repo, id, base.Add(time.Duration(i)*time.Minute))
	}

	resp := adminDeliveryListRequest(e, "/api/admin/v1/applications/app-1/provisioning/deliveries?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
}

func TestAdminDeliveryListOmitsLinkHeaderOnLastPage(t *testing.T) {
	e, repo := newAdminDeliveryPaginationHandler(t)
	seedDelivery(t, repo, "solo", time.Now().UTC())

	resp := adminDeliveryListRequest(e, "/api/admin/v1/applications/app-1/provisioning/deliveries?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminDeliveryListNextPageContinuesWithoutOverlap(t *testing.T) {
	e, repo := newAdminDeliveryPaginationHandler(t)
	base := time.Now().UTC().Add(-time.Hour)
	for i, id := range []string{"charlie", "delta", "echo"} {
		seedDelivery(t, repo, id, base.Add(time.Duration(i)*time.Minute))
	}

	first := adminDeliveryListRequest(e, "/api/admin/v1/applications/app-1/provisioning/deliveries?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://idp.test")

	second := adminDeliveryListRequest(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeDeliveryListBody(t, first.Body.Bytes())
	secondBody := decodeDeliveryListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, d := range firstBody {
		seen[d["id"].(string)] = true
	}
	for _, d := range secondBody {
		id := d["id"].(string)
		if seen[id] {
			t.Fatalf("delivery id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one delivery")
	}
}

func TestAdminDeliveryListFiltersBySourceType(t *testing.T) {
	e, repo := newAdminDeliveryPaginationHandler(t)
	now := time.Now().UTC()
	seedDelivery(t, repo, "user-delivery", now)
	if _, err := repo.Save(context.Background(), &provisioningdomain.ProvisioningDelivery{
		ID: "group-delivery", TenantID: tenancydomain.DefaultTenantID, ConnectionID: "app-1",
		SourceType: provisioningdomain.SourceTypeGroup, SourceID: "group-1", SourceVersion: 1,
		Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.DeliveryPending,
		CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("seed group delivery: %v", err)
	}

	resp := adminDeliveryListRequest(e, "/api/admin/v1/applications/app-1/provisioning/deliveries?source_type=group")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	deliveries := decodeDeliveryListBody(t, resp.Body.Bytes())
	if len(deliveries) != 1 || deliveries[0]["id"] != "group-delivery" {
		t.Fatalf("unexpected source_type result: %+v", deliveries)
	}
}

func TestAdminDeliveryListRejectsInvalidCursor(t *testing.T) {
	e, _ := newAdminDeliveryPaginationHandler(t)
	resp := adminDeliveryListRequest(e, "/api/admin/v1/applications/app-1/provisioning/deliveries?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminDeliveryListRejectsCursorFromAnotherTenant(t *testing.T) {
	e, _ := newAdminDeliveryPaginationHandler(t)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	// provisioningDeliveriesQueryHash はフィルタ無しのリクエスト (?limit=... のみ) では
	// 空文字列になる (cursor/limit を除いた残りの query をそのまま hash にするため)。
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := adminDeliveryListRequest(e, "/api/admin/v1/applications/app-1/provisioning/deliveries?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
