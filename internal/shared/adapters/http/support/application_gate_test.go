package support_test

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestApplicationAccessAllowedGatesUnassignedSubjects(t *testing.T) {
	ctx := context.Background()
	apps := memory.NewApplicationRepository()
	assignments := memory.NewApplicationAssignmentRepository()
	now := time.Now().UTC()
	app := &spec.Application{
		TenantID: "default", ApplicationID: "app-1", Name: "Payroll",
		Kind: spec.ApplicationFederated, Status: spec.ApplicationActive,
		Bindings:  []spec.ProtocolBinding{{Type: spec.ProtocolBindingOIDC, ClientID: "c1"}},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	d := support.Deps{ApplicationRepo: apps, ApplicationAssignmentRepo: assignments, GroupRepo: memory.NewGroupRepository()}

	// catalog 外の client は gating 対象外。
	if allowed, err := d.ApplicationAccessAllowed(ctx, "default", spec.ProtocolBindingOIDC, "other", "alice"); err != nil || !allowed {
		t.Fatalf("client outside catalog must be allowed: allowed=%v err=%v", allowed, err)
	}

	// catalog 内・未割当は fail-closed で拒否。
	if allowed, err := d.ApplicationAccessAllowed(ctx, "default", spec.ProtocolBindingOIDC, "c1", "alice"); err != nil || allowed {
		t.Fatalf("unassigned subject must be denied: allowed=%v err=%v", allowed, err)
	}

	// 割当後は許可。
	if err := assignments.Save(ctx, &spec.ApplicationAssignment{
		TenantID: "default", ApplicationID: "app-1", SubjectType: spec.AssignmentSubjectUser,
		SubjectID: "alice", Visibility: spec.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if allowed, err := d.ApplicationAccessAllowed(ctx, "default", spec.ProtocolBindingOIDC, "c1", "alice"); err != nil || !allowed {
		t.Fatalf("assigned subject must be allowed: allowed=%v err=%v", allowed, err)
	}

	// disabled application は割当済みでも拒否。
	app.Status = spec.ApplicationDisabled
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	if allowed, err := d.ApplicationAccessAllowed(ctx, "default", spec.ProtocolBindingOIDC, "c1", "alice"); err != nil || allowed {
		t.Fatalf("disabled application must be denied: allowed=%v err=%v", allowed, err)
	}
}

func TestApplicationAccessEvaluatesSignInPolicy(t *testing.T) {
	ctx := context.Background()
	apps := memory.NewApplicationRepository()
	assignments := memory.NewApplicationAssignmentRepository()
	policies := memory.NewSignInPolicyRepository()
	now := time.Now().UTC()
	app := &spec.Application{
		TenantID: "default", ApplicationID: "app-1", Name: "App", Kind: spec.ApplicationFederated, Status: spec.ApplicationActive,
		Bindings:  []spec.ProtocolBinding{{Type: spec.ProtocolBindingOIDC, ClientID: "c1"}},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	if err := assignments.Save(ctx, &spec.ApplicationAssignment{
		TenantID: "default", ApplicationID: "app-1", SubjectType: spec.AssignmentSubjectUser, SubjectID: "alice",
		Visibility: spec.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := policies.Save(ctx, &spec.AppSignInPolicy{
		TenantID: "default", ApplicationID: "app-1", UpdatedAt: now,
		Rules: []spec.SignInRule{{RuleID: "rule-1", Name: "MFA", Enabled: true, RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa}}},
	}); err != nil {
		t.Fatal(err)
	}
	d := support.Deps{ApplicationRepo: apps, ApplicationAssignmentRepo: assignments, ApplicationSignInPolicyRepo: policies}

	decision, err := d.EvaluateApplicationAccess(ctx, "default", spec.ProtocolBindingOIDC, "c1", "alice", &authdomain.AuthenticationContext{
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
	apps := memory.NewApplicationRepository()
	assignments := memory.NewApplicationAssignmentRepository()
	policies := memory.NewSignInPolicyRepository()
	defaults := memory.NewDefaultSignInPolicyRepository()
	now := time.Now().UTC()
	app := &spec.Application{
		TenantID: "default", ApplicationID: "app-1", Name: "App", Kind: spec.ApplicationFederated, Status: spec.ApplicationActive,
		Bindings:  []spec.ProtocolBinding{{Type: spec.ProtocolBindingOIDC, ClientID: "c1"}},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	if err := assignments.Save(ctx, &spec.ApplicationAssignment{
		TenantID: "default", ApplicationID: "app-1", SubjectType: spec.AssignmentSubjectUser, SubjectID: "alice",
		Visibility: spec.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	// テナントデフォルトで MFA を要求。アプリ個別ポリシーは未設定。
	if err := defaults.Save(ctx, &spec.TenantDefaultSignInPolicy{
		TenantID: "default", UpdatedAt: now,
		Rules: []spec.SignInRule{{RuleID: "def-1", Name: "MFA", Enabled: true, RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa}}},
	}); err != nil {
		t.Fatal(err)
	}
	d := support.Deps{
		ApplicationRepo: apps, ApplicationAssignmentRepo: assignments,
		ApplicationSignInPolicyRepo: policies, DefaultSignInPolicyRepo: defaults,
	}
	singleFactor := &authdomain.AuthenticationContext{UserID: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"}}

	// 個別ポリシーが無ければデフォルトの MFA が適用される。
	decision, err := d.EvaluateApplicationAccess(ctx, "default", spec.ProtocolBindingOIDC, "c1", "alice", singleFactor, "")
	if err != nil {
		t.Fatal(err)
	}
	if !decision.StepUpRequired || decision.Allowed {
		t.Fatalf("default decision=%+v, want step-up required", decision)
	}

	// アプリ独自ポリシー (パスワードのみ) はデフォルトを上書きし、より弱くても適用される。
	if err := policies.Save(ctx, &spec.AppSignInPolicy{
		TenantID: "default", ApplicationID: "app-1", UpdatedAt: now,
		Rules: []spec.SignInRule{{RuleID: "app-1", Name: "Password", Enabled: true, RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnPassword}}},
	}); err != nil {
		t.Fatal(err)
	}
	decision, err = d.EvaluateApplicationAccess(ctx, "default", spec.ProtocolBindingOIDC, "c1", "alice", singleFactor, "")
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allowed {
		t.Fatalf("override application decision=%+v, want allowed", decision)
	}
}
