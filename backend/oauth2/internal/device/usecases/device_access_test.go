package usecases

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	deviceports "github.com/ambi/idmagic/backend/oauth2/internal/device/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//spec:covers REQ-APPLICATION-011, REQ-APPLICATION-012, REQ-OAUTH2-027: コードのクライアントの関門を評価し、未割り当てと無効の拒否では記録を変えず、直接割り当て、hidden、カタログ外なら承認と交換を許す。
func TestApproveUserCodeEnforcesApplicationAccess(t *testing.T) {
	for _, condition := range []string{"unassigned", "disabled", "assigned", "hidden", "outside catalog", "assignment in another tenant"} {
		t.Run(condition, func(t *testing.T) {
			ctx := testing_tenant.Default(t.Context())
			f := newDeviceFixture()
			now := time.Now().UTC()
			issued, err := RequestDeviceAuthorization(ctx, f.requestDeps, DeviceAuthorizationInput{ClientID: "device-client", Scope: "openid"}, now)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := f.verifyDeps.DeviceCodeStore.FindByUserCode(ctx, domain.NormalizeUserCode(issued.UserCode))
			app := application.NewMemoryModule()
			record := &appdomain.Application{
				TenantID: before.TenantID, ID: "app", Name: "Device", Kind: appdomain.ApplicationFederated,
				Status: appdomain.ApplicationActive, Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: before.ClientID}, CreatedAt: now, UpdatedAt: now,
			}
			if condition == "disabled" {
				record.Status = appdomain.ApplicationDisabled
			}
			if condition != "outside catalog" {
				if err := app.Repo.Save(ctx, record); err != nil {
					t.Fatal(err)
				}
			}
			if condition != "unassigned" {
				assignment := &appdomain.ApplicationAssignment{
					TenantID: before.TenantID, ApplicationID: record.ID,
					SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "user", Visibility: appdomain.AssignmentVisible, CreatedAt: now,
				}
				if condition == "hidden" {
					assignment.Visibility = appdomain.AssignmentHidden
				}
				if condition == "assignment in another tenant" {
					assignment.TenantID = "other"
				}
				if err := app.AssignmentRepo.Save(ctx, assignment); err != nil {
					t.Fatal(err)
				}
			}
			var events []spec.DomainEvent
			f.verifyDeps.Emit = func(event spec.DomainEvent) { events = append(events, event) }
			decision, err := ApproveUserCode(ctx, f.verifyDeps, app.Gate(nil, nil, nil), ApproveUserCodeInput{
				UserCode: issued.UserCode, Authn: &authdomain.AuthenticationContext{UserID: "user"}, ClientIP: "203.0.113.7",
			}, now)
			wantAllowed := condition == "assigned" || condition == "hidden" || condition == "outside catalog"
			if err != nil || decision.Allowed != wantAllowed {
				t.Fatalf("decision=%+v err=%v, want allowed=%v", decision, err, wantAllowed)
			}
			after, _ := f.verifyDeps.DeviceCodeStore.FindByUserCode(ctx, before.UserCode)
			if !wantAllowed {
				if !reflect.DeepEqual(before, after) {
					t.Fatalf("refused approval changed record: before=%+v after=%+v", before, after)
				}
				for _, event := range events {
					if event.EventType() == "DeviceAuthorizationApproved" {
						t.Fatal("refused approval emitted DeviceAuthorizationApproved")
					}
				}
				return
			}
			if after.State != spec.DeviceFlowApproved || after.UserID == nil || *after.UserID != "user" || len(events) != 1 || events[0].EventType() != "DeviceAuthorizationApproved" {
				t.Fatalf("approved record=%+v events=%v", after, events)
			}
			tokens, err := ExchangeDeviceCode(ctx, f.deps, ExchangeDeviceCodeInput{ClientID: "device-client", DeviceCode: issued.DeviceCode}, now)
			if err != nil || tokens.AccessToken == "" || tokens.IDToken == "" {
				t.Fatalf("approved exchange=%+v err=%v", tokens, err)
			}
		})
	}
}

type failingApplicationGate struct{ err error }

func (g failingApplicationGate) EvaluateApplicationAccess(context.Context, string, appdomain.ApplicationProtocolType, string, string, policydomain.SignInAuthentication, string) (appdomain.ApplicationAccessDecision, error) {
	return appdomain.ApplicationAccessDecision{}, g.err
}

//spec:covers REQ-APPLICATION-011, REQ-OAUTH2-027: 関門の未配線と評価障害、他テナントと期限切れのコードは承認状態も承認イベントも残さない。
func TestApproveUserCodeRefusesWhenAccessCannotBeEvaluated(t *testing.T) {
	failure := errors.New("application lookup failed")
	for _, condition := range []string{"missing gate", "gate failure", "foreign tenant", "expired"} {
		t.Run(condition, func(t *testing.T) {
			ctx := testing_tenant.Default(t.Context())
			f := newDeviceFixture()
			now := time.Now().UTC()
			issued, err := RequestDeviceAuthorization(ctx, f.requestDeps, DeviceAuthorizationInput{ClientID: "device-client"}, now)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := f.verifyDeps.DeviceCodeStore.FindByUserCode(ctx, domain.NormalizeUserCode(issued.UserCode))
			var gate deviceports.ApplicationGate = application.NewMemoryModule().Gate(nil, nil, nil)
			switch condition {
			case "missing gate":
				gate = nil
			case "gate failure":
				gate = failingApplicationGate{err: failure}
			case "foreign tenant":
				ctx = tenantContext("other")
			case "expired":
				now = now.Add(domain.DeviceCodeTTL + time.Second)
			}
			var events []spec.DomainEvent
			f.verifyDeps.Emit = func(event spec.DomainEvent) { events = append(events, event) }
			decision, err := ApproveUserCode(ctx, f.verifyDeps, gate, ApproveUserCodeInput{UserCode: issued.UserCode, Authn: &authdomain.AuthenticationContext{UserID: "user"}}, now)
			if err == nil || decision.Allowed || (condition == "gate failure" && !errors.Is(err, failure)) {
				t.Fatalf("decision=%+v err=%v", decision, err)
			}
			after, _ := f.verifyDeps.DeviceCodeStore.FindByUserCode(ctx, before.UserCode)
			if !reflect.DeepEqual(before, after) || len(events) != 0 {
				t.Fatalf("refused approval changed record=%+v events=%v", after, events)
			}
		})
	}
}
