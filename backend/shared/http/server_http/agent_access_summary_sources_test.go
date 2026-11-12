package server_http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

var errSummarySourceFailure = errors.New("source unavailable")

type summaryAuditSource struct {
	auditports.AuditEventRepository
	err error
}

func (s summaryAuditSource) LatestByAgent(context.Context, string, []string, []string) (map[string]time.Time, error) {
	return nil, s.err
}

type summaryFailingClients struct {
	oauthports.OAuth2ClientRepository
}

func (summaryFailingClients) FindByID(context.Context, string, string) (*oauthdomain.OAuth2Client, error) {
	return nil, errSummarySourceFailure
}

type summaryFailingConsents struct{ oauthports.ConsentRepository }

func (summaryFailingConsents) ListDelegationsForAgent(context.Context, string, string) ([]*oauthdomain.Consent, error) {
	return nil, errSummarySourceFailure
}

type summaryFailingAgents struct {
	agentports.AgentRepository
	bindingFailure bool
}

func (s summaryFailingAgents) FindByID(ctx context.Context, tenant, id string) (*agentdomain.Agent, error) {
	if !s.bindingFailure {
		return nil, errSummarySourceFailure
	}
	return s.AgentRepository.FindByID(ctx, tenant, id)
}

func (summaryFailingAgents) ListBindings(context.Context, string, string) ([]*agentdomain.AgentCredentialBinding, error) {
	return nil, errSummarySourceFailure
}

func summarySourceDeps(t *testing.T) Deps {
	t.Helper()
	d := Deps{IdManagement: idmanagement.NewMemoryModule(), OAuth2: oauth2.NewMemoryModule()}
	d.Audit.AuditEventRepo = summaryAuditSource{}
	now := time.Now().UTC()
	if err := d.IdManagement.UserRepo.Save(t.Context(), &userdomain.User{ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", Roles: []string{"admin"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}}); err != nil {
		t.Fatal(err)
	}
	if err := d.IdManagement.AgentRepo.Save(t.Context(), &agentdomain.Agent{ID: "agent", TenantID: tenancydomain.DefaultTenantID, Name: "agent", OwnerUserID: "admin", Kind: idmdomain.AgentKindAutonomous, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.IdManagement.AgentRepo.AddBinding(t.Context(), &agentdomain.AgentCredentialBinding{AgentID: "agent", ClientID: "missing-client", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	return d
}

func readSummarySources(t *testing.T, d Deps) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	registerAgentAccessSummary(e.Group(""), d, &support.Authenticator{
		Principals: d.IdManagement.Principals(),
		Sessions:   authentication.Module{AuthnResolver: authentication.DemoHeaderResolver{}}.SessionAuthentications(),
	})
	request := httptest.NewRequest(http.MethodGet, "/api/admin/v1/agents/agent/access-summary", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	request = request.WithContext(tenantports.WithTenant(request.Context(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: "default"}, "", ""))
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	return recorder
}

//spec:covers REQ-IDMANAGEMENT-106: 材料の失敗と未配線は部分成功を返さない。
func TestAgentAccessSummaryRefusesIncompleteSources(t *testing.T) {
	for name, change := range map[string]func(*Deps){
		"agent read": func(d *Deps) {
			d.IdManagement.AgentRepo = summaryFailingAgents{AgentRepository: d.IdManagement.AgentRepo}
		},
		"bindings read": func(d *Deps) {
			d.IdManagement.AgentRepo = summaryFailingAgents{AgentRepository: d.IdManagement.AgentRepo, bindingFailure: true}
		},
		"client read":      func(d *Deps) { d.OAuth2.ClientRepo = summaryFailingClients{} },
		"consent read":     func(d *Deps) { d.OAuth2.ConsentRepo = summaryFailingConsents{} },
		"audit read":       func(d *Deps) { d.Audit.AuditEventRepo = summaryAuditSource{err: errSummarySourceFailure} },
		"missing agents":   func(d *Deps) { d.IdManagement.AgentRepo = nil },
		"missing clients":  func(d *Deps) { d.OAuth2.ClientRepo = nil },
		"missing consents": func(d *Deps) { d.OAuth2.ConsentRepo = nil },
		"missing audit":    func(d *Deps) { d.Audit.AuditEventRepo = nil },
	} {
		t.Run(name, func(t *testing.T) {
			d := summarySourceDeps(t)
			change(&d)
			response := readSummarySources(t, d)
			if response.Code != http.StatusInternalServerError {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

//spec:covers REQ-IDMANAGEMENT-106: 多数の無関係な付与を含むテナントでも対象の材料だけを返し、欠落した束縛先を示す。
func TestAgentAccessSummaryWithLargeTenant(t *testing.T) {
	d := summarySourceDeps(t)
	now := time.Now().UTC()
	for i := range 10000 {
		if err := d.OAuth2.ConsentRepo.SaveAgentDelegation(t.Context(), tenancydomain.DefaultTenantID, &oauthdomain.Consent{UserID: fmt.Sprintf("user-%d", i), AgentID: "unrelated-agent", Scopes: []string{"private"}, GrantedAt: now, ExpiresAt: now.Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	started := time.Now()
	response := readSummarySources(t, d)
	t.Logf("10000 unrelated grants: response took %s, %d bytes", time.Since(started), response.Body.Len())
	var body agentAccessSummary
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &body) != nil {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if len(body.Delegations) != 0 || len(body.Clients) != 1 || !body.Clients[0].Missing || len(body.Clients[0].Scopes) != 0 {
		t.Fatalf("materials=%+v", body)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache-control=%q", response.Header().Get("Cache-Control"))
	}
}
