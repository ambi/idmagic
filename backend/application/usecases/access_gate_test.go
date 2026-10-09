package usecases_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"

	appusecases "github.com/ambi/idmagic/backend/application/usecases"
)

func TestApplicationAccessAllowedGatesUnassignedSubjects(t *testing.T) {
	ctx := context.Background()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	now := time.Now().UTC()
	app := &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: "app-1", Name: "Payroll",
		Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "c1"},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	d := &appusecases.AccessGate{ApplicationRepo: apps, ApplicationAssignmentRepo: assignments}
	accessAllowed := func(bindingKey string) (bool, error) {
		decision, err := d.EvaluateApplicationAccess(ctx, tenancydomain.DefaultTenantID, appdomain.ApplicationProtocolOIDC, bindingKey, "alice", nil, "")
		return decision.Allowed, err
	}

	// catalog 外の client は gating 対象外。
	if allowed, err := accessAllowed("other"); err != nil || !allowed {
		t.Fatalf("client outside catalog must be allowed: allowed=%v err=%v", allowed, err)
	}

	// catalog 内・未割当は fail-closed で拒否。
	if allowed, err := accessAllowed("c1"); err != nil || allowed {
		t.Fatalf("unassigned subject must be denied: allowed=%v err=%v", allowed, err)
	}

	// 割当後は許可。
	if err := assignments.Save(ctx, &appdomain.ApplicationAssignment{
		TenantID: tenancydomain.DefaultTenantID, ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser,
		SubjectID: "alice", Visibility: appdomain.AssignmentVisible, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if allowed, err := accessAllowed("c1"); err != nil || !allowed {
		t.Fatalf("assigned subject must be allowed: allowed=%v err=%v", allowed, err)
	}

	// disabled application は割当済みでも拒否。
	app.Status = appdomain.ApplicationDisabled
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	if allowed, err := accessAllowed("c1"); err != nil || allowed {
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
		TenantID: tenancydomain.DefaultTenantID, ID: "app-1", Name: "App", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
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
	d := &appusecases.AccessGate{ApplicationRepo: apps, ApplicationAssignmentRepo: assignments, ApplicationSignInPolicyRepo: policies}

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
// テナントデフォルトが適用され、個別ポリシーがあればそれで上書きされることを確認する (wi-115)。
func TestApplicationAccessAppliesTenantDefaultPolicy(t *testing.T) {
	ctx := context.Background()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	policies := appmemory.NewSignInPolicyRepository()
	defaults := appmemory.NewDefaultSignInPolicyRepository()
	now := time.Now().UTC()
	app := &appdomain.Application{
		TenantID: tenancydomain.DefaultTenantID, ID: "app-1", Name: "App", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive,
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
	d := &appusecases.AccessGate{
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

func TestApplicationGateClientIP(t *testing.T) {
	g := &appusecases.AccessGate{GateTrustedForwardedHops: 1}
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	if got := g.ClientIP(req); got != "203.0.113.5" {
		t.Fatalf("got=%q", got)
	}

	zero := &appusecases.AccessGate{}
	if got := zero.ClientIP(req); got != "" {
		t.Fatalf("got=%q, want empty with GateTrustedForwardedHops=0", got)
	}
	if got := g.ClientIP(nil); got != "" {
		t.Fatalf("got=%q, want empty for a nil request", got)
	}
}
