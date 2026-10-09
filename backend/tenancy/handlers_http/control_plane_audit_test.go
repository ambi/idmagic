package handlers_http_test

// 主要ユースケース追跡: REQ-TENANCY-011、REQ-TENANCY-012。
//
// 制御面のテナント操作が、状態を変えたときにだけ監査イベントを発行することを、
// RegisterControlPlaneRoutes が組み立てる経路の境界で確かめる。発行は Deps.Emit という
// 出力ポートだけを通るので、そのポートの呼び出しを観測する。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	authhttpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	idmhttpdeps "github.com/ambi/idmagic/backend/idmanagement/deps_http"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/handlers_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/handlers_http"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	"github.com/labstack/echo/v5"
)

const (
	auditTestActor = "ops"
	// auditTestTenantID と auditTestRealm を別の値にするのは、イベントが realm ではなく
	// テナントの ID を載せることを区別して確かめるためである。
	auditTestTenantID = "tenant-acme"
	auditTestRealm    = "acme"
)

// quotaSaveFailingRepository は上書きの保存だけを失敗させる。
type quotaSaveFailingRepository struct {
	*memory.QuotaRepository
}

func (quotaSaveFailingRepository) SetQuota(context.Context, string, *domain.TenantQuota) error {
	return errors.New("quota store unavailable")
}

type auditTestServer struct {
	e      *echo.Echo
	events []spec.DomainEvent
}

type auditTestOptions struct {
	tenantBaseDomain string
	failQuotaSave    bool
}

func newAuditTestServer(t *testing.T, opts auditTestOptions) *auditTestServer {
	t.Helper()
	now := time.Now().UTC()

	tenantRepo := memory.NewTenantRepository()
	for _, tenant := range []*domain.Tenant{
		{
			ID: domain.DefaultTenantID, Realm: domain.DefaultRealm, DisplayName: "Default",
			Status: domain.TenantStatusActive, CreatedAt: now,
		},
		{
			ID: auditTestTenantID, Realm: auditTestRealm, DisplayName: "Acme",
			Status: domain.TenantStatusActive, EndpointStyle: domain.TenantEndpointStylePath, CreatedAt: now,
		},
	} {
		if err := tenantRepo.Save(testing_tenant.Default(context.Background()), tenant); err != nil {
			t.Fatal(err)
		}
	}

	userRepo := usermemory.NewUserRepository()
	actor := &userdomain.User{
		ID: auditTestActor, PreferredUsername: auditTestActor, TenantID: domain.DefaultTenantID,
		Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	}
	userRepo.Seed(actor)

	deps := tenancyhttp.Deps{
		Deps: support.Deps{
			Issuer: quotaTestIssuer, Contract: spec.CurrentRuntimeContract(), TenantRepo: tenantRepo,
			TenantBaseDomain: opts.tenantBaseDomain,
		},
		Authenticator: &support.Authenticator{
			Principals: idmhttpdeps.Principals{Users: userRepo},
			Sessions: authhttpdeps.SessionAuthentications{Resolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
				UserID: actor.ID, AuthTime: now.Unix(), AMR: []string{"pwd"},
			}}},
			AccessTokens: oauth2http.ResourceAccessTokens{Introspector: staticIntrospector{sub: actor.ID, scope: "idmagic.admin"}},
		},
		TenantRepo: tenantRepo,
		UserRepo:   userRepo,
		QuotaRepo:  memory.NewQuotaRepository(),
	}
	if opts.failQuotaSave {
		deps.QuotaRepo = quotaSaveFailingRepository{QuotaRepository: memory.NewQuotaRepository()}
	}
	srv := &auditTestServer{e: echo.New()}
	deps.Emit = func(event spec.DomainEvent) { srv.events = append(srv.events, event) }
	tenancyhttp.RegisterControlPlaneRoutes(srv.e.Group("", testing_tenant.ResolveDefault), deps)
	return srv
}

// send は Authorization ヘッダーの資格情報で要求する。周囲資格情報ではないので CSRF の
// 検査を通り、発行の有無だけを観測できる。
func (s *auditTestServer) send(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer access-token")
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

// wireFields は発行されたイベントを監査が保存するワイヤ表現で返す。
func wireFields(t *testing.T, event spec.DomainEvent) map[string]any {
	t.Helper()
	wire, err := spec.MarshalDomainEvent(event)
	if err != nil {
		t.Fatalf("MarshalDomainEvent(%T) error = %v", event, err)
	}
	var fields map[string]any
	if err := json.Unmarshal(wire, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

// controlPlaneStateChange は、制御面の状態を変える経路と、成功したときに発行すべき
// イベントの組である。
type controlPlaneStateChange struct {
	path, body string
	eventType  string
	// tenantID は空なら、新しく作ったテナントの ID であることだけを確かめる。
	tenantID string
}

// controlPlaneStateChanges は RegisterControlPlaneRoutes が登録する、状態を変える経路の
// 棚卸しである。経路を足したらここへ足し、成功時の監査イベントを決める。
var controlPlaneStateChanges = map[string]controlPlaneStateChange{
	"POST /api/admin/v1/tenants": {
		path: "/api/admin/v1/tenants", body: `{"realm":"newco","display_name":"New Co"}`,
		eventType: "TenantCreated",
	},
	"PATCH /api/admin/v1/tenants/:target_tenant_id": {
		path: "/api/admin/v1/tenants/" + auditTestRealm, body: `{"display_name":"Acme Renamed"}`,
		eventType: "TenantUpdated", tenantID: auditTestTenantID,
	},
	"PUT /api/admin/v1/tenants/:target_tenant_id/endpoint-style": {
		path: "/api/admin/v1/tenants/" + auditTestRealm + "/endpoint-style", body: `{"endpoint_style":"subdomain"}`,
		eventType: "TenantEndpointStyleChanged", tenantID: auditTestTenantID,
	},
	"POST /api/admin/v1/tenants/:target_tenant_id/disable": {
		path:      "/api/admin/v1/tenants/" + auditTestRealm + "/disable",
		eventType: "TenantDisabled", tenantID: auditTestTenantID,
	},
	"POST /api/admin/v1/tenants/:target_tenant_id/enable": {
		path:      "/api/admin/v1/tenants/" + auditTestRealm + "/enable",
		eventType: "TenantEnabled", tenantID: auditTestTenantID,
	},
	// クォータの更新はパス引数をテナントの ID として扱う。
	"PUT /api/admin/v1/tenants/:target_tenant_id/quota": {
		path: "/api/admin/v1/tenants/" + auditTestTenantID + "/quota", body: `{"users":20000}`,
		eventType: "TenantQuotaUpdated", tenantID: auditTestTenantID,
	},
}

// 棚卸しにない経路を登録すると、その経路の監査イベントが決まっていないとして落ちる。
//
//spec:covers REQ-TENANCY-011, REQ-TENANCY-012: 制御面に登録した状態を変える経路は、成功するたびに操作者と対象テナントを載せた監査イベントをちょうど一つ発行する
func TestEveryControlPlaneStateChangeEmitsAnAuditEvent(t *testing.T) {
	registered := []string{}
	for _, route := range newAuditTestServer(t, auditTestOptions{}).e.Router().Routes() {
		switch route.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			registered = append(registered, route.Method+" "+route.Path)
		}
	}
	inventoried := []string{}
	for route := range controlPlaneStateChanges {
		inventoried = append(inventoried, route)
	}
	slices.Sort(registered)
	slices.Sort(inventoried)
	if !slices.Equal(registered, inventoried) {
		t.Fatalf("state-changing control-plane routes = %v, inventory = %v; decide the audit event of every new route", registered, inventoried)
	}

	for route, change := range controlPlaneStateChanges {
		t.Run(route, func(t *testing.T) {
			srv := newAuditTestServer(t, auditTestOptions{tenantBaseDomain: "idp.test"})
			method, _, _ := strings.Cut(route, " ")

			rec := srv.send(method, change.path, change.body)

			if rec.Code >= 300 {
				t.Fatalf("status = %d, want 2xx; body = %s", rec.Code, rec.Body.String())
			}
			if len(srv.events) != 1 {
				t.Fatalf("events = %d, want 1 (%v)", len(srv.events), srv.events)
			}
			fields := wireFields(t, srv.events[0])
			if fields["type"] != change.eventType {
				t.Errorf("type = %v, want %s", fields["type"], change.eventType)
			}
			if fields["actorUserId"] != auditTestActor {
				t.Errorf("actorUserId = %v, want %s", fields["actorUserId"], auditTestActor)
			}
			tenantID, _ := fields["tenantId"].(string)
			if tenantID == "" || (change.tenantID != "" && tenantID != change.tenantID) {
				t.Errorf("tenantId = %q, want %q", tenantID, change.tenantID)
			}
		})
	}
}

//spec:covers REQ-TENANCY-011: 前と同じ値への切替も、切り替える前と後に同じ値を載せた TenantEndpointStyleChanged を発行する
func TestEndpointStyleSwitchToTheSameStyleStillEmitsAnEvent(t *testing.T) {
	srv := newAuditTestServer(t, auditTestOptions{})

	rec := srv.send(http.MethodPut, "/api/admin/v1/tenants/"+auditTestRealm+"/endpoint-style", `{"endpoint_style":"path"}`)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if len(srv.events) != 1 {
		t.Fatalf("events = %d, want 1", len(srv.events))
	}
	fields := wireFields(t, srv.events[0])
	if fields["previousEndpointStyle"] != "path" || fields["endpointStyle"] != "path" {
		t.Errorf("previousEndpointStyle = %v, endpointStyle = %v, want path and path", fields["previousEndpointStyle"], fields["endpointStyle"])
	}
}

//spec:covers REQ-TENANCY-011, EX-TENANCY-011-02: 不正な endpoint_style、tenant_base_domain のないデプロイでの subdomain、存在しない realm で拒否した切替は、TenantEndpointStyleChanged を発行しない
func TestRefusedEndpointStyleSwitchEmitsNoEvent(t *testing.T) {
	for _, tc := range []struct {
		name, realm, body, baseDomain string
		status                        int
	}{
		{name: "unknown style", realm: auditTestRealm, body: `{"endpoint_style":"bogus"}`, baseDomain: "idp.test", status: http.StatusBadRequest},
		{name: "subdomain without base domain", realm: auditTestRealm, body: `{"endpoint_style":"subdomain"}`, status: http.StatusBadRequest},
		{name: "unknown realm", realm: "missing", body: `{"endpoint_style":"subdomain"}`, baseDomain: "idp.test", status: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newAuditTestServer(t, auditTestOptions{tenantBaseDomain: tc.baseDomain})

			rec := srv.send(http.MethodPut, "/api/admin/v1/tenants/"+tc.realm+"/endpoint-style", tc.body)

			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.status, rec.Body.String())
			}
			if len(srv.events) != 0 {
				t.Errorf("events = %v, want none", srv.events)
			}
		})
	}
}

//spec:covers REQ-TENANCY-012, REQ-TENANCY-037: 負の上限で拒否した更新と、上書きを保存できなかった更新は、TenantQuotaUpdated を発行しない
func TestRefusedOrUnsavedQuotaUpdateEmitsNoEvent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		opts   auditTestOptions
		status int
	}{
		{name: "negative limit", body: `{"users":-1}`, status: http.StatusBadRequest},
		{name: "save failed", body: `{"users":20000}`, opts: auditTestOptions{failQuotaSave: true}, status: http.StatusInternalServerError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newAuditTestServer(t, tc.opts)

			rec := srv.send(http.MethodPut, "/api/admin/v1/tenants/"+auditTestTenantID+"/quota", tc.body)

			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, tc.status, rec.Body.String())
			}
			if len(srv.events) != 0 {
				t.Errorf("events = %v, want none", srv.events)
			}
		})
	}
}
