package handlers_http_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signinpolicy"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-APPLICATION-011, REQ-APPLICATION-009, REQ-OAUTH2-027: 未割り当て、無効なアプリケーション、追加認証とネットワークの条件による拒否は承認を保存せず、イベントもトークンも発行しない。
//verify:operation SubmitBrowserDevice: 関門の拒否は 403 access_denied の Problem を返し、コードを承認しない。
func TestDeviceApprovalRefusesApplicationAccess(t *testing.T) {
	for _, condition := range []string{"unassigned", "disabled", "mfa without a second factor", "network"} {
		t.Run(condition, func(t *testing.T) {
			ctx := tenantContext(tenancydomain.DefaultTenantID)
			app := application.NewMemoryModule()
			policies := signinpolicy.NewMemoryModule()
			now := time.Now().UTC()
			status := appdomain.ApplicationActive
			if condition == "disabled" {
				status = appdomain.ApplicationDisabled
			}
			if err := app.Repo.Save(ctx, &appdomain.Application{
				ID: "device-app", TenantID: tenancydomain.DefaultTenantID, Name: "Device",
				Kind: appdomain.ApplicationFederated, Status: status,
				Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "device-client"},
				CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				t.Fatal(err)
			}
			if condition != "unassigned" {
				if err := app.AssignmentRepo.Save(ctx, &appdomain.ApplicationAssignment{
					TenantID: tenancydomain.DefaultTenantID, ApplicationID: "device-app",
					SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "user-1", CreatedAt: now,
				}); err != nil {
					t.Fatal(err)
				}
			}
			if condition == "mfa without a second factor" || condition == "network" {
				rule := policydomain.SignInRule{RuleID: "rule", Name: "Policy", Enabled: true, RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnMfa}}
				if condition == "network" {
					rule.RequiredAuthn.Strength = policydomain.RequiredAuthnPassword
					rule.Condition.NetworkAllowCIDRs = []string{"198.51.100.0/24"}
				}
				if err := policies.ApplicationRepo.Save(ctx, &policydomain.AppSignInPolicy{
					TenantID: tenancydomain.DefaultTenantID, ApplicationID: "device-app", Rules: []policydomain.SignInRule{rule}, UpdatedAt: now,
				}); err != nil {
					t.Fatal(err)
				}
			}
			var events []spec.DomainEvent
			f := newDeviceServer(t, func(d *httpadapter.Deps) {
				d.Application, d.SignInPolicy = app, policies
				d.Emit = func(event spec.DomainEvent) { events = append(events, event) }
			})
			issued := deviceAuthorizationForTest(t, f)
			response := approveDeviceForTest(f, issued["user_code"].(string))
			oauthContractProblem(t, response, http.StatusForbidden, "access_denied")
			stored, err := f.deviceStore.FindByUserCode(ctx, domain.NormalizeUserCode(issued["user_code"].(string)))
			if err != nil || stored.State != spec.DeviceFlowIssued || stored.UserID != nil || stored.AuthTime != nil {
				t.Fatalf("refused approval changed the record: %+v err=%v", stored, err)
			}
			for _, event := range events {
				if event.EventType() == "DeviceAuthorizationApproved" || event.EventType() == "AccessTokenIssued" {
					t.Fatalf("refused approval emitted %s", event.EventType())
				}
			}
			form := url.Values{"client_id": {"device-client"}, "device_code": {issued["device_code"].(string)}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}
			req := httptest.NewRequest(http.MethodPost, "/realms/default/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			f.e.ServeHTTP(rec, req)
			oauthContractError(t, rec, http.StatusBadRequest, "authorization_pending")
		})
	}
}

func deviceAuthorizationForTest(t *testing.T, f deviceFixture) map[string]any {
	t.Helper()
	form := url.Values{"client_id": {"device-client"}, "scope": {"openid"}}
	req := httptest.NewRequest(http.MethodPost, "/realms/default/device_authorization", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	return oauthContractJSON(t, rec, http.StatusOK)
}

func approveDeviceForTest(f deviceFixture, code string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/realms/default/api/auth/device", strings.NewReader(`{"user_code":"`+code+`","action":"allow"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://test")
	req.Header.Set("X-Csrf-Token", "csrf-val")
	req.Header.Set("Cookie", "idmagic_csrf=csrf-val")
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	return rec
}
