package server_http

// 主要ユースケース追跡: REQ-TENANCY-011、REQ-TENANCY-012。
//
// 制御面の状態変更を製品の外部入口 (Register が組み立てるルート木) から行い、その記録を
// 制御面主体がシステムの経路の監査イベント検索から読めることを確かめる。発行の有無だけを
// ハンドラーの単位で見ても、イベントが対象テナントへ帰属し、操作者の検索軸に載ることは
// 確かめられないため、書き込みと読み出しの両方を外部入口に置く。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	auditusecases "github.com/ambi/idmagic/backend/audit/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	controlPlaneAuditActor = "ops"
	// controlPlaneAuditTenantID と controlPlaneAuditRealm を別の値にするのは、イベントが
	// realm ではなくテナントの ID へ帰属することを、応答から区別して確かめるためである。
	controlPlaneAuditTenantID = "tenant-acme"
	controlPlaneAuditRealm    = "acme"
)

type controlPlaneAuditServer struct {
	e *echo.Echo
}

// newControlPlaneAuditServer は、発行したイベントを監査の記録へ写す配線を本番と同じ
// 形で組む。本番の写像は cmd の内部パッケージにあって参照できないため、テナントの帰属と
// 検索属性の抽出という、検索結果を左右する二つの手順だけを同じ部品で再現する。
func newControlPlaneAuditServer(t *testing.T) *controlPlaneAuditServer {
	t.Helper()
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{
			ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default",
			Status: tenancydomain.TenantStatusActive, CreatedAt: now,
		},
		{
			ID: controlPlaneAuditTenantID, Realm: controlPlaneAuditRealm, DisplayName: "Acme",
			Status: tenancydomain.TenantStatusActive, EndpointStyle: tenancydomain.TenantEndpointStylePath,
			CreatedAt: now,
		},
	} {
		if err := tenants.Save(t.Context(), tenant); err != nil {
			t.Fatal(err)
		}
	}

	users := usermemory.NewUserRepository()
	users.Seed(controlPlaneTestUser(controlPlaneAuditActor, tenancydomain.DefaultTenantID, "system_admin"))

	auditStore := auditmemory.NewAuditEventStore(0)
	emit := func(event spec.DomainEvent) {
		wire, err := spec.MarshalDomainEvent(event)
		if err != nil {
			t.Errorf("MarshalDomainEvent(%T) error = %v", event, err)
			return
		}
		var payload map[string]any
		if err := json.Unmarshal(wire, &payload); err != nil {
			t.Errorf("unmarshal %s: %v", wire, err)
			return
		}
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Errorf("NewUUIDv4: %v", err)
			return
		}
		rec := &auditports.AuditEventRecord{
			ID: id, Type: event.EventType(), OccurredAt: event.OccurredAt(), Payload: payload,
		}
		rec.TenantID, _ = payload["tenantId"].(string)
		rec.SearchAttributes = auditusecases.ExtractSearchAttributes(rec)
		if err := auditStore.Append(t.Context(), rec); err != nil {
			t.Errorf("Append(%s) error = %v", rec.Type, err)
		}
	}

	e := echo.New()
	Register(e, Deps{
		Issuer: quotaCsrfIssuer, Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, TenantBaseDomain: "idp.test", Emit: emit,
		UserRepo: users, AuthnResolver: &fixedAuthnResolver{sub: controlPlaneAuditActor},
		Tenancy: tenancy.Module{TenantRepo: tenants, QuotaRepo: tenancymemory.NewQuotaRepository()},
		Audit:   audit.Module{AuditEventRepo: auditStore},
	})
	return &controlPlaneAuditServer{e: e}
}

func (s *controlPlaneAuditServer) put(t *testing.T, path, body string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/realms/"+tenancydomain.DefaultRealm+path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	withSystemConsoleCSRF("control-plane-audit")(req)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	if rec.Code >= 300 {
		t.Fatalf("PUT %s status = %d, want 2xx; body = %s", path, rec.Code, rec.Body.String())
	}
}

// searchSystemAuditLog は、制御面主体がシステムの経路で種類と操作者を指定して検索した
// 結果を返す。操作者は検索属性 actor.id で絞るので、ペイロードの actorUserId が検索軸へ
// 載っていなければ見つからない。
func (s *controlPlaneAuditServer) searchSystemAuditLog(t *testing.T, eventType string) []map[string]any {
	t.Helper()
	query := url.Values{"type": {eventType}, "filter": {"actor.id:eq:" + controlPlaneAuditActor}}
	req := httptest.NewRequest(http.MethodGet,
		"/realms/"+tenancydomain.DefaultRealm+"/api/admin/v1/system/audit-events?"+query.Encode(), http.NoBody)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("system audit search status = %d; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Events
}

// assertSingleAuditEvent は、検索結果がちょうど 1 件で、対象テナントへ帰属し、ペイロードが
// want のすべての項目を持つことを確かめる。
func assertSingleAuditEvent(t *testing.T, events []map[string]any, want map[string]any) {
	t.Helper()
	if len(events) != 1 {
		t.Fatalf("audit events = %d, want 1: %v", len(events), events)
	}
	if got := events[0]["tenant_id"]; got != controlPlaneAuditTenantID {
		t.Errorf("tenant_id = %v, want %q", got, controlPlaneAuditTenantID)
	}
	payload, _ := events[0]["payload"].(map[string]any)
	for key, value := range want {
		if !reflect.DeepEqual(payload[key], value) {
			t.Errorf("payload.%s = %#v, want %#v (payload %v)", key, payload[key], value, payload)
		}
	}
}

// クォータの更新は経路のパス引数をテナントの ID として保存するので、ID を指定する。
//
//spec:covers REQ-TENANCY-012, EX-TENANCY-012-01: System 管理者がクォータを更新すると、システムの経路の監査イベント検索で、操作者、対象テナント、要求したリソースの名前を載せた TenantQuotaUpdated が 1 件読める
func TestTenantQuotaUpdateIsReadableFromTheSystemAuditLog(t *testing.T) {
	srv := newControlPlaneAuditServer(t)

	srv.put(t, "/api/admin/v1/tenants/"+controlPlaneAuditTenantID+"/quota", `{"users":20000,"groups":5}`)

	assertSingleAuditEvent(t, srv.searchSystemAuditLog(t, "TenantQuotaUpdated"), map[string]any{
		"actorUserId":   controlPlaneAuditActor,
		"tenantId":      controlPlaneAuditTenantID,
		"changedFields": []any{"users", "groups"},
	})
}

//spec:covers REQ-TENANCY-011, EX-TENANCY-011-01: System 管理者が正規ロケーションを切り替えると、システムの経路の監査イベント検索で、操作者、対象テナント、切り替える前と後の endpoint_style を載せた TenantEndpointStyleChanged が 1 件読める
func TestTenantEndpointStyleSwitchIsReadableFromTheSystemAuditLog(t *testing.T) {
	srv := newControlPlaneAuditServer(t)

	srv.put(t, "/api/admin/v1/tenants/"+controlPlaneAuditRealm+"/endpoint-style", `{"endpoint_style":"subdomain"}`)

	assertSingleAuditEvent(t, srv.searchSystemAuditLog(t, "TenantEndpointStyleChanged"), map[string]any{
		"actorUserId":           controlPlaneAuditActor,
		"tenantId":              controlPlaneAuditTenantID,
		"previousEndpointStyle": "path",
		"endpointStyle":         "subdomain",
	})
}
