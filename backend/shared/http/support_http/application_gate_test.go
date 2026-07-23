package support_http_test

import (
	"context"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

func TestApplicationAccessAllowedGatesUnassignedSubjects(t *testing.T) {
	ctx := context.Background()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	now := time.Now().UTC()
	app := &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", Name: "Payroll",
		Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "c1"},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	d := &support.ApplicationGate{ApplicationRepo: apps, ApplicationAssignmentRepo: assignments, GroupRepo: groupmemory.NewGroupRepository()}

	// catalog 外の client は gating 対象外。
	if allowed, err := d.ApplicationAccessAllowed(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "other", "alice"); err != nil || !allowed {
		t.Fatalf("client outside catalog must be allowed: allowed=%v err=%v", allowed, err)
	}

	// catalog 内・未割当は fail-closed で拒否。
	if allowed, err := d.ApplicationAccessAllowed(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "c1", "alice"); err != nil || allowed {
		t.Fatalf("unassigned subject must be denied: allowed=%v err=%v", allowed, err)
	}

	// 割当後は許可。
	if err := assignments.Save(ctx, &appdomain.ApplicationAssignment{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser,
		SubjectID: "alice", Visibility: appdomain.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if allowed, err := d.ApplicationAccessAllowed(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "c1", "alice"); err != nil || !allowed {
		t.Fatalf("assigned subject must be allowed: allowed=%v err=%v", allowed, err)
	}

	// disabled application は割当済みでも拒否。
	app.Status = appdomain.ApplicationDisabled
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	if allowed, err := d.ApplicationAccessAllowed(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "c1", "alice"); err != nil || allowed {
		t.Fatalf("disabled application must be denied: allowed=%v err=%v", allowed, err)
	}
}

func TestApplicationAccessEvaluatesSignInPolicy(t *testing.T) {
	ctx := context.Background()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	policies := appmemory.NewSignInPolicyRepository()
	now := time.Now().UTC()
	app := &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", Name: "App", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "c1"},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	if err := assignments.Save(ctx, &appdomain.ApplicationAssignment{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "alice",
		Visibility: appdomain.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := policies.Save(ctx, &appdomain.AppSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", UpdatedAt: now,
		Rules: []appdomain.SignInRule{{RuleID: "rule-1", Name: "MFA", Enabled: true, RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa}}},
	}); err != nil {
		t.Fatal(err)
	}
	d := &support.ApplicationGate{ApplicationRepo: apps, ApplicationAssignmentRepo: assignments, ApplicationSignInPolicyRepo: policies}

	decision, err := d.EvaluateApplicationAccess(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "c1", "alice", &authdomain.AuthenticationContext{
		UserID: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if !decision.StepUpRequired || decision.Allowed {
		t.Fatalf("decision=%+v, want step-up required", decision)
	}
}

// TestApplicationAccessAppliesTenantDefaultPolicy はアプリ個別ポリシー未設定のとき
// テナントデフォルトが適用され、個別ポリシーがあればそれで上書きされることを確認する (wi-115, ADR-081)。
func TestApplicationAccessAppliesTenantDefaultPolicy(t *testing.T) {
	ctx := context.Background()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	policies := appmemory.NewSignInPolicyRepository()
	defaults := appmemory.NewDefaultSignInPolicyRepository()
	now := time.Now().UTC()
	app := &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", Name: "App", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "c1"},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	if err := assignments.Save(ctx, &appdomain.ApplicationAssignment{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "alice",
		Visibility: appdomain.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	// テナントデフォルトで MFA を要求。アプリ個別ポリシーは未設定。
	if err := defaults.Save(ctx, &appdomain.TenantDefaultSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, UpdatedAt: now,
		Rules: []appdomain.SignInRule{{RuleID: "def-1", Name: "MFA", Enabled: true, RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnMfa}}},
	}); err != nil {
		t.Fatal(err)
	}
	d := &support.ApplicationGate{
		ApplicationRepo: apps, ApplicationAssignmentRepo: assignments,
		ApplicationSignInPolicyRepo: policies, DefaultSignInPolicyRepo: defaults,
	}
	singleFactor := &authdomain.AuthenticationContext{UserID: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"}}

	// 個別ポリシーが無ければデフォルトの MFA が適用される。
	decision, err := d.EvaluateApplicationAccess(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "c1", "alice", singleFactor, "")
	if err != nil {
		t.Fatal(err)
	}
	if !decision.StepUpRequired || decision.Allowed {
		t.Fatalf("default decision=%+v, want step-up required", decision)
	}

	// アプリ独自ポリシー (パスワードのみ) はデフォルトを上書きし、より弱くても適用される。
	if err := policies.Save(ctx, &appdomain.AppSignInPolicy{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", UpdatedAt: now,
		Rules: []appdomain.SignInRule{{RuleID: "app-1", Name: "Password", Enabled: true, RequiredAuthn: appdomain.RequiredAuthnLevel{Strength: appdomain.RequiredAuthnPassword}}},
	}); err != nil {
		t.Fatal(err)
	}
	decision, err = d.EvaluateApplicationAccess(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, "c1", "alice", singleFactor, "")
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed {
		t.Fatalf("override application decision=%+v, want allowed", decision)
	}
}
