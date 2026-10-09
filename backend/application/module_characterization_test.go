package application_test

// Application が組み立てる割り当てのゲートと表示名の解決のうち、規範 ID を引くテストが
// 固定していない分岐を、`Module` の公開メソッドを境界にして固定する特性化テストである。
// ゲートと解決器の型をどこへ置いても、`Module.Gate` と `Module.ClientDisplayNames` から
// 呼べる操作は変わらないので、判定の結果は書式化した値で比べる。

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

var errCharacterizedStore = errors.New("characterized store failure")

type faultyApplications struct {
	*appmemory.ApplicationRepository
	findErr error
}

func (r *faultyApplications) FindByProtocol(
	ctx context.Context, tenantID string, protocol appdomain.ApplicationProtocolType, key string,
) (*appdomain.Application, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.ApplicationRepository.FindByProtocol(ctx, tenantID, protocol, key)
}

type faultyAssignments struct {
	*appmemory.ApplicationAssignmentRepository
	listErr error
}

func (r *faultyAssignments) ListBySubjects(
	ctx context.Context, tenantID string, subjects []appports.SubjectRef,
) ([]*appdomain.ApplicationAssignment, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.ApplicationAssignmentRepository.ListBySubjects(ctx, tenantID, subjects)
}

type faultySignInPolicies struct {
	*appmemory.SignInPolicyRepository
	getErr error
}

func (r *faultySignInPolicies) Get(ctx context.Context, tenantID, applicationID string) (*appdomain.AppSignInPolicy, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.SignInPolicyRepository.Get(ctx, tenantID, applicationID)
}

type faultyDefaultSignInPolicies struct {
	*appmemory.DefaultSignInPolicyRepository
	getErr error
}

func (r *faultyDefaultSignInPolicies) Get(ctx context.Context, tenantID string) (*appdomain.TenantDefaultSignInPolicy, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	return r.DefaultSignInPolicyRepository.Get(ctx, tenantID)
}

type faultyGroups struct {
	*groupmemory.GroupRepository
	listErr error
}

func (r *faultyGroups) ListGroupsByUser(ctx context.Context, tenantID, userID string) ([]*groupdomain.Group, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.GroupRepository.ListGroupsByUser(ctx, tenantID, userID)
}

type gateFixture struct {
	apps        *faultyApplications
	assignments *faultyAssignments
	policies    *faultySignInPolicies
	defaults    *faultyDefaultSignInPolicies
	groups      *faultyGroups
	module      application.Module
}

// newGateFixture は "c1" で OIDC に結んだ有効なアプリを置き、alice を直接割り当てる。
func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	f := &gateFixture{
		apps:        &faultyApplications{ApplicationRepository: appmemory.NewApplicationRepository()},
		assignments: &faultyAssignments{ApplicationAssignmentRepository: appmemory.NewApplicationAssignmentRepository()},
		policies:    &faultySignInPolicies{SignInPolicyRepository: appmemory.NewSignInPolicyRepository()},
		defaults:    &faultyDefaultSignInPolicies{DefaultSignInPolicyRepository: appmemory.NewDefaultSignInPolicyRepository()},
		groups:      &faultyGroups{GroupRepository: groupmemory.NewGroupRepository()},
	}
	if err := f.apps.Save(ctx, &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: "app-1", Name: "Payroll",
		Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "c1"},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.assignments.Save(ctx, &appdomain.ApplicationAssignment{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser,
		SubjectID: "alice", Visibility: appdomain.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	f.module = application.Module{
		Repo: f.apps, AssignmentRepo: f.assignments,
		SignInPolicyRepo: f.policies, DefaultSignInPolicyRepo: f.defaults,
	}
	return f
}

func (f *gateFixture) evaluate(t *testing.T) (string, error) {
	t.Helper()
	decision, err := f.module.Gate(f.groups, 0).EvaluateApplicationAccess(
		context.Background(), tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "c1", "alice",
		&authdomain.AuthenticationContext{UserID: "alice", AMR: []string{"pwd"}}, "",
	)
	return fmt.Sprintf("%+v", decision), err
}

func TestCharacterizeApplicationGateDecision(t *testing.T) {
	cases := []struct {
		name      string
		configure func(*gateFixture)
		want      string
	}{
		{
			name:      "assigned subject",
			configure: func(*gateFixture) {},
			want:      "{Allowed:true StepUpRequired:false ApplicationID:app-1 Reason: TrustedDeviceAllowed:false}",
		},
		{
			name: "disabled application",
			configure: func(f *gateFixture) {
				app, _ := f.apps.FindByID(context.Background(), tenancydomain.DefaultTenantID, "app-1")
				app.Status = appdomain.ApplicationDisabled
				_ = f.apps.Save(context.Background(), app)
			},
			want: "{Allowed:false StepUpRequired:false ApplicationID:app-1 Reason:application is disabled TrustedDeviceAllowed:false}",
		},
		{
			name:      "assignments are not wired",
			configure: func(f *gateFixture) { f.module.AssignmentRepo = nil },
			want:      "{Allowed:false StepUpRequired:false ApplicationID:app-1 Reason:application assignments are unavailable TrustedDeviceAllowed:false}",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newGateFixture(t)
			tc.configure(f)
			got, err := f.evaluate(t)
			if err != nil || got != tc.want {
				t.Fatalf("decision=%s err=%v, want %s", got, err, tc.want)
			}
		})
	}
}

func TestCharacterizeApplicationGatePropagatesStoreFailures(t *testing.T) {
	cases := map[string]func(*gateFixture){
		"application lookup":    func(f *gateFixture) { f.apps.findErr = errCharacterizedStore },
		"group lookup":          func(f *gateFixture) { f.groups.listErr = errCharacterizedStore },
		"assignment lookup":     func(f *gateFixture) { f.assignments.listErr = errCharacterizedStore },
		"application policy":    func(f *gateFixture) { f.policies.getErr = errCharacterizedStore },
		"tenant default policy": func(f *gateFixture) { f.defaults.getErr = errCharacterizedStore },
	}
	for name, configure := range cases {
		t.Run(name, func(t *testing.T) {
			f := newGateFixture(t)
			configure(f)
			got, err := f.evaluate(t)
			if !errors.Is(err, errCharacterizedStore) || got != "{Allowed:false StepUpRequired:false ApplicationID: Reason: TrustedDeviceAllowed:false}" {
				t.Fatalf("decision=%s err=%v, want the store failure", got, err)
			}
		})
	}
}

type countingClients struct {
	*oauth2memory.OAuth2ClientRepository
	lookups int
}

func (r *countingClients) FindByID(ctx context.Context, tenantID, clientID string) (*oauthdomain.OAuth2Client, error) {
	r.lookups++
	return r.OAuth2ClientRepository.FindByID(ctx, tenantID, clientID)
}

func TestCharacterizeClientDisplayNamesResolveEachClientOnce(t *testing.T) {
	clients := &countingClients{OAuth2ClientRepository: oauth2memory.NewClientRepository()}
	module := application.Module{Repo: appmemory.NewApplicationRepository()}
	names := module.ClientDisplayNames(clients).ResolveAll(
		context.Background(), tenancydomain.DefaultTenantID, []string{"a", "b", "a"},
	)
	if got := fmt.Sprintf("%v lookups=%d", names, clients.lookups); got != "map[a:a b:b] lookups=2" {
		t.Fatalf("got %s", got)
	}
}
