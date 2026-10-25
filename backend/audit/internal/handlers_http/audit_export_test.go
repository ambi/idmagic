package handlers_http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

//spec:covers REQ-AUDIT-012: 同期の互換経路でも保存先の 1,000 件上限で切り詰めない。
//spec:covers REQ-SYSTEM-014: TypeSpec から生成した非推奨日と廃止日を正式な HTTP 応答に載せる。
//verify:operation ExportAdminAuditEvents: 二ページにまたがる結果を attachment の JSON として返す。
func TestLegacyAuditExportReadsBeyondOneStoragePage(t *testing.T) {
	var events []*auditports.AuditEventRecord
	for i := range 1001 {
		events = append(events, auditEvent("acme", "UserAuthenticated", fmt.Sprint(i), time.Now().Add(-time.Minute)))
	}
	e := newAuditAdminServer(t, auditUser("admin", "acme", []string{"admin"}), events)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/audit-events/export", http.NoBody))
	var body struct {
		Events []any `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || len(body.Events) != 1001 {
		t.Fatalf("status=%d events=%d, want 200 and 1001", rec.Code, len(body.Events))
	}
	if rec.Header().Get("Deprecation") != "@1792886400" || rec.Header().Get("Sunset") != "Mon, 25 Oct 2027 00:00:00 GMT" {
		t.Fatalf("deprecation headers=%v", rec.Header())
	}
}

func exportRequest(e *echo.Echo, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, http.NoBody)
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", "test-csrf")
	req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "test-csrf"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

//spec:covers REQ-AUDIT-009: 正式な両経路で、検索範囲と要求者を固定した非同期ジョブを返す。
//spec:covers REQ-AUDIT-011: 現在の権限と所有者を確認して状態、取消し、ダウンロードを提供する。
//verify:operation StartAdminAuditExport: 202 の待機ジョブを返す。
//verify:operation StartSystemAuditExport: 制御面の横断ジョブを返す。
//verify:operation ListAdminAuditExports: 自分のテナント内ジョブだけを返す。
//verify:operation ListSystemAuditExports: 自分の横断ジョブだけを返す。
//verify:operation GetAdminAuditExport: 所有者だけが現在状態を取得できる。
//verify:operation GetSystemAuditExport: 現在の制御面権限を照合する。
//verify:operation CancelAdminAuditExport: 未完了だけを取り消す。
//verify:operation CancelSystemAuditExport: 制御面の未完了ジョブを取り消す。
//verify:operation DownloadAdminAuditExport: 成功した期限内の成果物を attachment で返す。
//verify:operation DownloadSystemAuditExport: 他の所有者と範囲には成果物を返さない。
func TestAuditExportHTTPAuthorization(t *testing.T) {
	for _, system := range []bool{false, true} {
		t.Run(fmt.Sprint(system), func(t *testing.T) {
			ctx := context.Background()
			tenantID, realm, base, roles := "acme", "acme", "/api/admin/v1/audit-events/exports", []string{"admin"}
			if system {
				tenantID = tenantdomain.DefaultTenantID
				realm = tenantdomain.DefaultRealm
				base = "/api/admin/v1/system/audit-events/exports"
				roles = []string{"system_admin"}
			}
			actor := auditUser("owner", tenantID, roles)
			users := idmanagement.NewMemoryModule().UserRepo
			if err := users.Save(ctx, actor); err != nil {
				t.Fatal(err)
			}
			resolver := &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}}
			module := audit.NewMemoryModule(0)
			for _, id := range []string{"acme", tenantdomain.DefaultTenantID} {
				if err := module.AuditEventRepo.Append(ctx, auditEvent(id, "UserAuthenticated", "user", time.Now().Add(-time.Minute))); err != nil {
					t.Fatal(err)
				}
			}
			jobModule := jobs.NewMemoryModule()
			tenants := tenancy.NewMemoryModule()
			emit := func(spec.DomainEvent) {}
			jobModule.Enqueuer = jobs.NewEnqueuer(jobModule.Repo, tenants.QuotaRepo, emit)
			e := echo.New()
			httpadapter.Register(e, httpadapter.Deps{Issuer: "http://test", TenantRepo: newSingleTenantRepo(), UserRepo: users, AuthnResolver: resolver, Audit: module, Jobs: jobModule, Tenancy: tenants, Emit: emit})
			path := "/realms/" + realm + base
			rejected := httptest.NewRecorder()
			e.ServeHTTP(rejected, httptest.NewRequest("POST", path, http.NoBody))
			if rejected.Code != 403 {
				t.Fatalf("missing browser verification=%d", rejected.Code)
			}
			create := func() audit.ExportView {
				t.Helper()
				rec := exportRequest(e, "POST", path+"?type=UserAuthenticated")
				var view audit.ExportView
				if rec.Code != 202 || json.Unmarshal(rec.Body.Bytes(), &view) != nil || view.Status != "queued" {
					t.Fatalf("create=%d %s", rec.Code, rec.Body.String())
				}
				return view
			}
			view := create()
			for _, route := range []string{path, path + "/" + view.ID} {
				if rec := exportRequest(e, "GET", route); rec.Code != 200 {
					t.Fatalf("read=%d %s", rec.Code, rec.Body.String())
				}
			}
			if rec := exportRequest(e, "GET", path+"/"+view.ID+"/download"); rec.Code != 409 {
				t.Fatalf("queued download=%d", rec.Code)
			}
			claims, err := jobModule.Repo.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, time.Now().UTC())
			if err != nil || len(claims) != 1 {
				t.Fatalf("claim %v %v", claims, err)
			}
			registry := jobs.NewHandlerRegistry()
			module.RegisterExportHandler(registry, audit.ExportJobDeps{Jobs: jobModule.Repo, Emit: emit})
			handler, err := registry.Lookup(audit.KindAuditExport)
			if err != nil {
				t.Fatal(err)
			}
			result, err := handler(ctx, claims[0])
			if err != nil {
				t.Fatal(err)
			}
			if _, err := jobModule.Repo.Complete(ctx, view.ID, "worker", result, time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			rec := exportRequest(e, "GET", path+"/"+view.ID+"/download")
			var body struct {
				Events []auditports.AuditEventRecord `json:"events"`
			}
			wantCount := 1
			if system {
				wantCount = 2
			}
			if rec.Code != 200 || rec.Header().Get("Content-Disposition") == "" || json.Unmarshal(rec.Body.Bytes(), &body) != nil || len(body.Events) != wantCount {
				t.Fatalf("download=%d %s", rec.Code, rec.Body.String())
			}
			for _, event := range body.Events {
				if !system && event.Payload["tenantId"] != tenantID {
					t.Fatalf("other tenant event=%+v", event)
				}
			}
			if rec := exportRequest(e, "POST", path+"/"+view.ID+"/cancel"); rec.Code != 409 {
				t.Fatalf("terminal cancel=%d", rec.Code)
			}
			other := auditUser("other", tenantID, roles)
			if err := users.Save(ctx, other); err != nil {
				t.Fatal(err)
			}
			resolver.ctx.UserID = other.ID
			for _, suffix := range []string{"", "/download"} {
				if rec := exportRequest(e, "GET", path+"/"+view.ID+suffix); rec.Code != 404 {
					t.Fatalf("other owner=%d", rec.Code)
				}
			}
			if rec := exportRequest(e, "POST", path+"/"+view.ID+"/cancel"); rec.Code != 404 {
				t.Fatalf("other cancel=%d", rec.Code)
			}
			rec = exportRequest(e, "GET", path)
			var list struct {
				Exports []audit.ExportView `json:"exports"`
			}
			if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &list) != nil || len(list.Exports) != 0 {
				t.Fatalf("other list=%d %s", rec.Code, rec.Body.String())
			}
			resolver.ctx.UserID = actor.ID
			second := create()
			if rec := exportRequest(e, "POST", path+"/"+second.ID+"/cancel"); rec.Code != 200 {
				t.Fatalf("cancel=%d %s", rec.Code, rec.Body.String())
			}
			actor.Roles = nil
			if err := users.Save(ctx, actor); err != nil {
				t.Fatal(err)
			}
			for _, suffix := range []string{"", "/download"} {
				if rec := exportRequest(e, "GET", path+"/"+view.ID+suffix); rec.Code != 403 {
					t.Fatalf("lost role=%d", rec.Code)
				}
			}
		})
	}
}

//spec:covers REQ-AUDIT-009: 非同期の作成はレコードの応答を返さず 202 のジョブを作る。
func TestAuditExportCreationIsAsynchronous(t *testing.T) {
	e := newAuditAdminServer(t, auditUser("admin", "acme", []string{"admin"}), nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/realms/acme/api/admin/v1/audit-events/exports", http.NoBody)
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", "test-csrf")
	req.AddCookie(&http.Cookie{Name: "idmagic_csrf", Value: "test-csrf"})
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s, want 202", rec.Code, rec.Body.String())
	}
}
