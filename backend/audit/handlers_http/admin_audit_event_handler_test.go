package handlers_http_test

// 主要ユースケース追跡: REQ-AUDIT-001。

// SCL scenario "管理者は所属テナントの監査イベントを参照できるが別テナントは公開しない" を
// /api/admin/v1/audit_events 経由で検証する。テナント管理経路は要求元の所属テナントへ閉じ、
// 全テナント横断は /api/admin/v1/system/audit_events の制御面主体だけが到達できる
// (REQ-AUDIT-001 / REQ-AUDIT-007)。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	audithttp "github.com/ambi/idmagic/backend/audit/handlers_http"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	auditusecases "github.com/ambi/idmagic/backend/audit/usecases"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type fakeAuthnResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (f *fakeAuthnResolver) Resolve(_ context.Context, _ authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return f.ctx, nil
}

// singleTenantRepo は指定 ID の Active テナントだけを返す最小の TenantRepository。
type singleTenantRepo struct {
	tenant *tenancydomain.Tenant
}

func newSingleTenantRepo() *singleTenantRepo {
	now := time.Now().UTC()
	return &singleTenantRepo{tenant: &tenancydomain.Tenant{
		ID: "acme", Realm: "acme", Status: tenancydomain.TenantStatusActive, CreatedAt: now,
	}}
}

func (r *singleTenantRepo) FindByID(_ context.Context, id string) (*tenancydomain.Tenant, error) {
	if r.tenant.ID == id {
		return r.tenant, nil
	}
	if id == tenancydomain.DefaultTenantID {
		return &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive}, nil
	}
	return nil, nil
}

func (r *singleTenantRepo) FindByRealm(_ context.Context, realm string) (*tenancydomain.Tenant, error) {
	if r.tenant.Realm == realm {
		return r.tenant, nil
	}
	if realm == tenancydomain.DefaultRealm {
		return &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive}, nil
	}
	return nil, nil
}

func (r *singleTenantRepo) FindAll(_ context.Context) ([]*tenancydomain.Tenant, error) {
	return []*tenancydomain.Tenant{r.tenant}, nil
}

func (r *singleTenantRepo) Save(_ context.Context, _ *tenancydomain.Tenant) error { return nil }
func (r *singleTenantRepo) Delete(_ context.Context, _ string) error              { return nil }

func newAuditAdminServer(t *testing.T, actor *userdomain.User, events []*auditports.AuditEventRecord) *echo.Echo {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	if actor != nil {
		userRepo.Seed(actor)
	}
	auditStore := auditmemory.NewAuditEventStore(0)
	for _, ev := range events {
		_ = auditStore.Append(context.Background(), ev)
	}
	resolver := &fakeAuthnResolver{}
	if actor != nil {
		resolver.ctx = &authdomain.AuthenticationContext{
			UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test",
		// カーソルを発行させる検査 (範囲ごとの束縛) がここの Link ヘッダーを読む。
		PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		TenantRepo:      newSingleTenantRepo(), UserRepo: userRepo,
		Audit: audit.Module{AuditEventRepo: auditStore}, AuthnResolver: resolver,
	})
	return e
}

func auditUser(sub, tenantID string, roles []string) *userdomain.User {
	now := time.Now().UTC()
	return &userdomain.User{
		ID: sub, PreferredUsername: sub, TenantID: tenantID, Roles: roles,
		CreatedAt: now, UpdatedAt: now,
	}
}

func auditEvent(tenantID, typ, sub string, occurredAt time.Time) *auditports.AuditEventRecord {
	rec := &auditports.AuditEventRecord{
		ID:       tenantID + ":" + typ + ":" + sub + ":" + occurredAt.Format(time.RFC3339Nano),
		TenantID: tenantID, Type: typ, OccurredAt: occurredAt,
		Payload: map[string]any{"userId": sub, "tenantId": tenantID, "type": typ},
	}
	rec.SearchAttributes = auditusecases.ExtractSearchAttributes(rec)
	return rec
}

func getAdminAuditEvents(e *echo.Echo, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAdminAuditEventsResolvesUsernameToUserID(t *testing.T) {
	// wi-147: 実アカウントが常に確定するイベント (UserAuthenticated 等) は payload に
	// username を持たないため、username=... は検索時に user_id へ解決してから絞り込む。
	admin := auditUser("user_admin", "acme", []string{"admin"})
	target := auditUser("alice", "acme", []string{})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(admin)
	userRepo.Seed(target)
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "UserAuthenticated", "alice", now),
		auditEvent("acme", "UserAuthenticated", "user_admin", now),
	}
	auditStore := auditmemory.NewAuditEventStore(0)
	for _, ev := range events {
		_ = auditStore.Append(context.Background(), ev)
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", TenantRepo: newSingleTenantRepo(),
		UserRepo:      userRepo,
		Audit:         audit.Module{AuditEventRepo: auditStore},
		AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{UserID: admin.ID, AuthTime: now.Unix(), AMR: []string{"pwd"}}},
	})

	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?username=alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].Payload["userId"] != "alice" {
		t.Fatalf("expected only alice's event, got %+v", body.Events)
	}
}

func TestAdminAuditEventsUnknownUsernameReturnsEmptyNotError(t *testing.T) {
	// wi-147: 該当ユーザーが存在しない username は 0 件を返す。フィルタ無視で全件返す・
	// エラーにするのどちらでもない。
	admin := auditUser("user_admin", "acme", []string{"admin"})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(admin)
	now := time.Now().UTC()
	auditStore := auditmemory.NewAuditEventStore(0)
	_ = auditStore.Append(context.Background(), auditEvent("acme", "UserAuthenticated", "alice", now))
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", TenantRepo: newSingleTenantRepo(),
		UserRepo:      userRepo,
		Audit:         audit.Module{AuditEventRepo: auditStore},
		AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{UserID: admin.ID, AuthTime: now.Unix(), AMR: []string{"pwd"}}},
	})

	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?username=no-such-user")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 0 {
		t.Fatalf("expected 0 events for unknown username, got %+v", body.Events)
	}
}

func TestAdminAuditEventsRequiresAdminRole(t *testing.T) {
	// 認証はあるが admin/system_admin ロールが無い → 403。
	user := auditUser("user_alice", "acme", []string{})
	e := newAuditAdminServer(t, user, nil)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events")
	if rec.Code != http.StatusForbidden ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"type":"urn:idmagic:error:access_denied"`)) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminAuditEventsScopesToOwnTenant(t *testing.T) {
	user := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "UserAuthenticated", "alice", now.Add(-time.Minute)),
		auditEvent(tenancydomain.DefaultTenantID, "UserAuthenticated", "ops", now.Add(-30*time.Second)),
		auditEvent("acme", "AccessTokenIssued", "alice", now),
	}
	e := newAuditAdminServer(t, user, events)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Events) != 2 {
		t.Fatalf("acme admin must see 2 events, got %d", len(body.Events))
	}
	for _, ev := range body.Events {
		if ev.TenantID != "acme" {
			t.Fatalf("cross-tenant leak: %+v", ev)
		}
	}
}

func TestAdminAuditEventsAllTenantsRequiresSystemAdminOnDefaultTenant(t *testing.T) {
	// admin (acme) が all_tenants=true を渡しても自テナント限定で動く。
	admin := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "X", "a", now),
		auditEvent(tenancydomain.DefaultTenantID, "X", "b", now),
	}
	e := newAuditAdminServer(t, admin, events)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?all_tenants=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].TenantID != "acme" {
		t.Fatalf("admin must not escape own tenant: %+v", body.Events)
	}
}

func TestAdminAuditEventsAllTenantsRejectsSystemAdminOutsideControlPlaneTenant(t *testing.T) {
	sysAdmin := auditUser("tenant_system_admin", "acme", []string{"system_admin"})
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "X", "a", now),
		auditEvent(tenancydomain.DefaultTenantID, "X", "b", now),
	}
	e := newAuditAdminServer(t, sysAdmin, events)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?all_tenants=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].TenantID != "acme" {
		t.Fatalf("別テナントの system_admin が横断した: %+v", body.Events)
	}
}

// 同じ範囲で答える。REQ-AUDIT-007。
//
//spec:covers EX-AUDIT-007-01: システム経路は全テナントの監査イベントを返し、エクスポートも 1 件参照も
func TestListSystemAuditEventsSpansEveryTenant(t *testing.T) {
	sysAdmin := auditUser("user_system_admin", tenancydomain.DefaultTenantID, []string{"system_admin"})
	now := time.Now().UTC()
	foreign := auditEvent("acme", "X", "a", now)
	events := []*auditports.AuditEventRecord{
		foreign,
		auditEvent(tenancydomain.DefaultTenantID, "X", "b", now),
	}
	e := newAuditAdminServer(t, sysAdmin, events)

	rec := getAdminAuditEvents(e, "/realms/default/api/admin/v1/system/audit_events")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 2 {
		t.Fatalf("system route must span every tenant's 2 events, got %d", len(body.Events))
	}

	export := getAdminAuditEvents(e, "/realms/default/api/admin/v1/system/audit_events/export")
	if export.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", export.Code, export.Body.String())
	}
	if !strings.Contains(export.Body.String(), foreign.ID) {
		t.Fatalf("export omitted another tenant's event %q: %s", foreign.ID, export.Body.String())
	}

	detail := getAdminAuditEvents(e, "/realms/default/api/admin/v1/system/audit_events/"+foreign.ID)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}
}

// 制御面主体はテナント管理経路でも所属テナントが制御面テナントなので、カーソルの指紋に
// 範囲を混ぜていなければ両経路の指紋が一致し、テナント内で発行したカーソルが横断検索の
// 続きとして黙って読めてしまう。ここで見ているのはその取り違えである。
//
//spec:covers EX-AUDIT-007-03: テナント管理経路で発行したカーソルはシステム経路の続きにならない。
func TestTenantAuditCursorDoesNotContinueTheSystemSearch(t *testing.T) {
	sysAdmin := auditUser("user_system_admin", tenancydomain.DefaultTenantID, []string{"system_admin", "admin"})
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{}
	for i := range 3 {
		events = append(events,
			auditEvent(tenancydomain.DefaultTenantID, "TypeA", "admin", base.Add(time.Duration(i)*time.Minute)))
	}
	e := newAuditAdminServer(t, sysAdmin, events)

	// テナント管理経路で 1 ページ目を引き、その rel="next" のカーソルを取り出す。
	first := getAdminAuditEvents(e, "/realms/default/api/admin/v1/audit_events?limit=2")
	if first.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", first.Code, first.Body.String())
	}
	link := first.Header().Get("Link")
	if !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://test")
	cursor := ""
	if parsed, err := url.Parse(nextPath); err == nil {
		cursor = parsed.Query().Get("cursor")
	}
	if cursor == "" {
		t.Fatalf("no cursor in the next link %q", nextPath)
	}

	// 同じカーソルをシステム経路へ持ち込む。続きとして読まず、拒否すること。
	crossed := getAdminAuditEvents(e,
		"/realms/default/api/admin/v1/system/audit_events?limit=2&cursor="+url.QueryEscape(cursor))
	if crossed.Code != http.StatusBadRequest {
		t.Fatalf("a tenant cursor continued the system search: status=%d body=%s",
			crossed.Code, crossed.Body.String())
	}
}

//spec:covers EX-AUDIT-007-02: システム経路は制御面主体でない実行者を拒否し、どのテナントの記録も返さない。
func TestSystemAuditEventRoutesRefuseNonControlPlaneActor(t *testing.T) {
	now := time.Now().UTC()
	foreign := auditEvent("acme", "X", "a", now)
	events := []*auditports.AuditEventRecord{foreign, auditEvent(tenancydomain.DefaultTenantID, "X", "b", now)}

	for name, actor := range map[string]*userdomain.User{
		// 制御面テナントの経路だが system_admin を持たない。
		"tenant admin at the control plane": auditUser("user_admin", tenancydomain.DefaultTenantID, []string{"admin"}),
		// system_admin だが所属も経路も制御面テナントではない。
		"system_admin outside the control plane": auditUser("user_system_admin", "acme", []string{"system_admin"}),
	} {
		e := newAuditAdminServer(t, actor, events)
		realm := "default"
		if actor.TenantID == "acme" {
			realm = "acme"
		}
		for _, path := range []string{
			"/realms/" + realm + "/api/admin/v1/system/audit_events",
			"/realms/" + realm + "/api/admin/v1/system/audit_events/export",
			"/realms/" + realm + "/api/admin/v1/system/audit_events/" + foreign.ID,
		} {
			rec := getAdminAuditEvents(e, path)
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s: %s status=%d want %d body=%s", name, path, rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), foreign.ID) {
				t.Errorf("%s: %s leaked event %q: %s", name, path, foreign.ID, rec.Body.String())
			}
		}
	}
}

// EX-JOBS-012-04 と同じ趣旨を監査側で確かめる。テナント管理経路は制御面主体でも、
// 横断を求める入力を添えても、要求元の所属テナントへ閉じる。REQ-AUDIT-001。
func TestAdminAuditEventsIgnoreAnyCrossTenantInput(t *testing.T) {
	sysAdmin := auditUser("user_system_admin", tenancydomain.DefaultTenantID, []string{"system_admin", "admin"})
	now := time.Now().UTC()
	foreign := auditEvent("acme", "X", "a", now)
	events := []*auditports.AuditEventRecord{
		foreign,
		auditEvent(tenancydomain.DefaultTenantID, "X", "b", now),
	}
	e := newAuditAdminServer(t, sysAdmin, events)
	rec := getAdminAuditEvents(e, "/realms/default/api/admin/v1/audit_events?all_tenants=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].TenantID != tenancydomain.DefaultTenantID {
		t.Fatalf("tenant route crossed a tenant boundary: %+v", body.Events)
	}
	detail := getAdminAuditEvents(e, "/realms/default/api/admin/v1/audit_events/"+foreign.ID)
	if detail.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant detail status=%d want %d body=%s", detail.Code, http.StatusNotFound, detail.Body.String())
	}

	// エクスポートは検索と同じ範囲でなければならない。一覧だけを閉じてエクスポートを
	// 開いたままにする誤りは、一覧を読むテストでは見つからない。
	export := getAdminAuditEvents(e, "/realms/default/api/admin/v1/audit_events/export?all_tenants=true")
	if export.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", export.Code, export.Body.String())
	}
	if strings.Contains(export.Body.String(), foreign.ID) {
		t.Fatalf("tenant export crossed a tenant boundary: %s", export.Body.String())
	}
}

func TestAdminAuditEventsGetReturns404ForCrossTenant(t *testing.T) {
	user := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	foreign := auditEvent(tenancydomain.DefaultTenantID, "X", "alice", now)
	e := newAuditAdminServer(t, user, []*auditports.AuditEventRecord{foreign})
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events/"+foreign.ID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-tenant event, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminAuditEventsFilterByTypeAndSub(t *testing.T) {
	user := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "UserAuthenticated", "alice", now),
		auditEvent("acme", "UserAuthenticated", "bob", now.Add(-time.Second)),
		auditEvent("acme", "AccessTokenIssued", "alice", now.Add(-2*time.Second)),
	}
	e := newAuditAdminServer(t, user, events)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?type=UserAuthenticated&user_id=alice")
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 ||
		body.Events[0].Type != "UserAuthenticated" ||
		body.Events[0].Payload["userId"] != "alice" {
		t.Fatalf("filter mismatch: %+v", body.Events)
	}
}

// wi-44 統合: 監査ログ検索のイベントカテゴリ絞り込み (認証サブ分類 + 管理操作) を検証する。
func TestAdminAuditEventsFilterByCategory(t *testing.T) {
	user := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "UserAuthenticated", "alice", now),
		auditEvent("acme", "AuthenticationFailed", "", now.Add(-time.Second)),
		auditEvent("acme", "PasswordChanged", "alice", now.Add(-2*time.Second)),        // user カテゴリ
		auditEvent("acme", "AdminOAuth2ClientCreated", "ops", now.Add(-3*time.Second)), // client カテゴリ
		auditEvent("acme", "ClientSecretIssued", "ops", now.Add(-4*time.Second)),       // client credential lifecycle
		auditEvent("acme", "ClientSecretRevoked", "ops", now.Add(-5*time.Second)),      // client credential lifecycle
	}
	e := newAuditAdminServer(t, user, events)

	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?category=fail")
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].Type != "AuthenticationFailed" {
		t.Fatalf("category=fail mismatch: %+v", body.Events)
	}

	// authentication は成功 + 失敗 (PasswordChanged / AdminOAuth2ClientCreated は対象外)。
	rec = getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?category=authentication")
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 2 {
		t.Fatalf("category=authentication must return 2, got %d: %+v", len(body.Events), body.Events)
	}

	// 管理操作カテゴリ (認証以外) も絞り込めること。
	rec = getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?category=client")
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 3 || body.Events[0].Type != "AdminOAuth2ClientCreated" ||
		body.Events[1].Type != "ClientSecretIssued" || body.Events[2].Type != "ClientSecretRevoked" {
		t.Fatalf("category=client mismatch: %+v", body.Events)
	}

	rec = getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?category=bogus")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown category must be 400, got %d", rec.Code)
	}
}

func TestAdminAuditEventsFilterAndQUseSearchAttributes(t *testing.T) {
	user := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "UserAuthenticated", "alice", now),
		auditEvent("acme", "AuthenticationFailed", "bob", now.Add(-time.Second)),
		auditEvent("acme", "AccessTokenIssued", "alice", now.Add(-2*time.Second)),
	}
	e := newAuditAdminServer(t, user, events)

	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?filter=outcome:eq:failure")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].Type != "AuthenticationFailed" {
		t.Fatalf("filter outcome mismatch: %+v", body.Events)
	}

	rec = getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?q=access")
	if rec.Code != http.StatusOK {
		t.Fatalf("q status=%d body=%s", rec.Code, rec.Body.String())
	}
	body.Events = nil
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].Type != "AccessTokenIssued" {
		t.Fatalf("q mismatch: %+v", body.Events)
	}
}

func TestAdminAuditEventsFiltersPlaintextUsernameAndIP(t *testing.T) {
	// actor.username / client.ip は平文のまま
	// 完全一致で検索する。サーバ側の hash/truncate transform はしない。
	user := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	ev := auditEvent("acme", "AuthenticationFailed", "", now)
	ev.Payload["username"] = "alice"
	ev.Payload["ip"] = "203.0.113.9"
	ev.SearchAttributes = auditusecases.ExtractSearchAttributes(ev)
	e := newAuditAdminServer(t, user, []*auditports.AuditEventRecord{ev})

	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?filter=actor.username:eq:alice&filter=client.ip:eq:203.0.113.9")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 || body.Events[0].Type != "AuthenticationFailed" {
		t.Fatalf("plaintext filter mismatch: %+v", body.Events)
	}
}

func TestAdminAuditEventsRejectsUnknownFilterField(t *testing.T) {
	user := auditUser("user_admin", "acme", []string{"admin"})
	e := newAuditAdminServer(t, user, nil)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events?filter=payload.any:eq:value")
	if rec.Code != http.StatusBadRequest ||
		rec.Header().Get("Content-Type") != support.ProblemContentType ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"type":"urn:idmagic:error:invalid_request"`)) {
		t.Fatalf("unknown filter must be 400 invalid_request, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminAuditEventSearchOptionsRequiresAuditReader(t *testing.T) {
	user := auditUser("user_alice", "acme", []string{})
	e := newAuditAdminServer(t, user, nil)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events/search_options")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminAuditEventSearchOptionsReturnsAllowlist(t *testing.T) {
	// wi-147: event.type / outcome の選択肢は Go 側の単一の正 (auditEventCategoryTypes /
	// eventOutcome) から機械的に導出され、UI のハードコードとの drift を防ぐ。
	user := auditUser("user_admin", "acme", []string{"admin"})
	e := newAuditAdminServer(t, user, nil)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events/search_options")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body audithttp.AuditEventSearchOptionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Outcomes) != 2 {
		t.Fatalf("expected 2 outcome choices, got %v", body.Outcomes)
	}
	found := map[string]bool{}
	for _, ty := range body.EventTypes {
		found[ty] = true
	}
	for _, want := range []string{"UserAuthenticated", "AuthenticationFailed", "AccessTokenIssued", "ConsentGranted"} {
		if !found[want] {
			t.Fatalf("expected event_types to contain %q, got %v", want, body.EventTypes)
		}
	}
}

func TestAdminAuditEventsExportSetsAttachment(t *testing.T) {
	user := auditUser("user_admin", "acme", []string{"admin"})
	now := time.Now().UTC()
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "UserAuthenticated", "alice", now),
	}
	e := newAuditAdminServer(t, user, events)
	rec := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit_events/export?category=authentication")
	if rec.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", rec.Code, rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); cd == "" {
		t.Fatal("export must set Content-Disposition")
	}
	var body struct {
		Events []audithttp.AdminAuditEventResponse `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body.Events) != 1 {
		t.Fatalf("export must return 1 event, got %d", len(body.Events))
	}
}
