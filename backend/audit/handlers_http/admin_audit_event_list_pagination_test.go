package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

// newAuditAdminPaginationServer は既存 newAuditAdminServer と異なり PaginationCodec を
// 設定する。既存 helper は Deps.PaginationCodec が nil のままなので cursor pagination の
// テストには使えない。
func newAuditAdminPaginationServer(t *testing.T, events []*auditports.AuditEventRecord) *echo.Echo {
	t.Helper()
	actor := auditUser("admin", "acme", []string{"admin"})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(actor)
	auditStore := auditmemory.NewAuditEventStore(0)
	for _, ev := range events {
		if err := auditStore.Append(t.Context(), ev); err != nil {
			t.Fatalf("append audit event: %v", err)
		}
	}
	resolver := &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
		UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
	}}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://test",
		TenantRepo:      newSingleTenantRepo(),
		PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		UserRepo:        userRepo,
		Audit:           audit.Module{AuditEventRepo: auditStore},
		AuthnResolver:   resolver,
	})
	return e
}

func decodeAuditEventListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal audit events body: %v (body=%s)", err, body)
	}
	return parsed.Events
}

func TestAdminAuditEventListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", base),
		auditEvent("acme", "TypeB", "admin", base.Add(time.Minute)),
		auditEvent("acme", "TypeC", "admin", base.Add(2*time.Minute)),
	}
	e := newAuditAdminPaginationServer(t, events)

	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?limit=2")
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

func TestAdminAuditEventListOmitsLinkHeaderOnLastPage(t *testing.T) {
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", time.Now().UTC()),
	}
	e := newAuditAdminPaginationServer(t, events)

	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

func TestAdminAuditEventListNextPageContinuesWithoutOverlap(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", base),
		auditEvent("acme", "TypeB", "admin", base.Add(time.Minute)),
		auditEvent("acme", "TypeC", "admin", base.Add(2*time.Minute)),
	}
	e := newAuditAdminPaginationServer(t, events)

	first := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://test")

	second := getAdminAuditEvents(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeAuditEventListBody(t, first.Body.Bytes())
	secondBody := decodeAuditEventListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, ev := range firstBody {
		seen[ev["id"].(string)] = true
	}
	for _, ev := range secondBody {
		id := ev["id"].(string)
		if seen[id] {
			t.Fatalf("event id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one event")
	}
}

func TestAdminAuditEventListRejectsInvalidCursor(t *testing.T) {
	e := newAuditAdminPaginationServer(t, nil)
	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminAuditEventListRejectsCursorFromAnotherTenant(t *testing.T) {
	e := newAuditAdminPaginationServer(t, nil)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	// auditEventQueryHash はフィルタ無しのリクエスト (?limit=... のみ) では空文字列になる
	// (cursor/limit を除いた残りの query をそのまま hash にするため)。
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}
