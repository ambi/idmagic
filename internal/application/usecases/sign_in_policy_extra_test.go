package usecases_test

// sign-in policy の取得・更新・評価とテナントデフォルトの未カバー分岐を補う (wi-129, ADR-081)。

import (
	"errors"
	"testing"
	"time"

	appusecases "github.com/ambi/idmagic/internal/application/usecases"
	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func newPolicyDeps() (appusecases.SignInPolicyDeps, appusecases.ApplicationDeps) {
	apps := memory.NewApplicationRepository()
	appDeps := appusecases.ApplicationDeps{Repo: apps, AssignmentRepo: memory.NewApplicationAssignmentRepository()}
	policyDeps := appusecases.SignInPolicyDeps{
		AppRepo:     apps,
		PolicyRepo:  memory.NewSignInPolicyRepository(),
		DefaultRepo: memory.NewDefaultSignInPolicyRepository(),
	}
	return policyDeps, appDeps
}

func mfaRule() spec.SignInRule {
	return spec.SignInRule{
		Name: "MFA", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
	}
}

func TestGetAndUpdateSignInPolicy(t *testing.T) {
	ctx := tenantContext()
	policyDeps, appDeps := newPolicyDeps()
	app := seedApp(ctx, t, appDeps, "Payroll")
	policyDeps.AppRepo = appDeps.Repo

	// 未設定なら空ポリシー。
	empty, err := appusecases.GetSignInPolicy(ctx, policyDeps, app.ApplicationID)
	if err != nil {
		t.Fatalf("get empty: %v", err)
	}
	if len(empty.Rules) != 0 {
		t.Fatalf("expected empty rules, got %v", empty.Rules)
	}

	// 更新して再取得すると 1 ルール。
	saved, err := appusecases.UpdateSignInPolicy(ctx, policyDeps, appusecases.UpdateSignInPolicyInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Rules: []spec.SignInRule{mfaRule()},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(saved.Rules) != 1 || saved.Rules[0].RuleID == "" {
		t.Fatalf("rule id not assigned: %+v", saved.Rules)
	}
	got, err := appusecases.GetSignInPolicy(ctx, policyDeps, app.ApplicationID)
	if err != nil || len(got.Rules) != 1 {
		t.Fatalf("get after update = %+v err=%v", got, err)
	}

	// 存在しないアプリは not found。
	if _, err := appusecases.GetSignInPolicy(ctx, policyDeps, "ghost"); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
	if _, err := appusecases.UpdateSignInPolicy(ctx, policyDeps, appusecases.UpdateSignInPolicyInput{
		ApplicationID: "ghost", Rules: []spec.SignInRule{mfaRule()},
	}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
	// 名前空のルールは検証エラー。
	if _, err := appusecases.UpdateSignInPolicy(ctx, policyDeps, appusecases.UpdateSignInPolicyInput{
		ApplicationID: app.ApplicationID, Rules: []spec.SignInRule{{Enabled: true}},
	}); !errors.Is(err, appusecases.ErrInvalidSignInPolicy) {
		t.Fatalf("expected ErrInvalidSignInPolicy, got %v", err)
	}
}

func TestGetAndUpdateDefaultSignInPolicy(t *testing.T) {
	ctx := tenantContext()
	policyDeps, _ := newPolicyDeps()

	def, err := appusecases.GetDefaultSignInPolicy(ctx, policyDeps)
	if err != nil || len(def.Rules) != 0 {
		t.Fatalf("empty default = %+v err=%v", def, err)
	}
	saved, err := appusecases.UpdateDefaultSignInPolicy(ctx, policyDeps, appusecases.UpdateDefaultSignInPolicyInput{
		ActorUserID: "admin", Rules: []spec.SignInRule{mfaRule()},
	})
	if err != nil || len(saved.Rules) != 1 {
		t.Fatalf("update default = %+v err=%v", saved, err)
	}
	got, err := appusecases.GetDefaultSignInPolicy(ctx, policyDeps)
	if err != nil || len(got.Rules) != 1 {
		t.Fatalf("get default = %+v err=%v", got, err)
	}
	// 不正 CIDR は検証エラー。
	badCIDR := spec.SignInRule{
		Name: "Net", Enabled: true, RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnPassword},
		Condition: spec.AccessCondition{NetworkAllowCIDRs: []string{"not-a-cidr"}},
	}
	if _, err := appusecases.UpdateDefaultSignInPolicy(ctx, policyDeps, appusecases.UpdateDefaultSignInPolicyInput{
		Rules: []spec.SignInRule{badCIDR},
	}); !errors.Is(err, appusecases.ErrInvalidSignInPolicy) {
		t.Fatalf("expected ErrInvalidSignInPolicy for bad cidr, got %v", err)
	}
}

func TestSignInPolicyNilRepoFallbacks(t *testing.T) {
	ctx := tenantContext()
	apps := memory.NewApplicationRepository()
	deps := appusecases.SignInPolicyDeps{AppRepo: apps}
	app, err := appusecases.CreateApplication(ctx, appusecases.ApplicationDeps{Repo: apps, AssignmentRepo: memory.NewApplicationAssignmentRepository()},
		appusecases.CreateApplicationInput{ActorUserID: "admin", Name: "X", Kind: spec.ApplicationFederated})
	if err != nil {
		t.Fatal(err)
	}
	// PolicyRepo nil -> 空ポリシー。
	got, err := appusecases.GetSignInPolicy(ctx, deps, app.ApplicationID)
	if err != nil || len(got.Rules) != 0 {
		t.Fatalf("nil policy repo get = %+v err=%v", got, err)
	}
	// DefaultRepo nil -> 空デフォルト。
	def, err := appusecases.GetDefaultSignInPolicy(ctx, deps)
	if err != nil || len(def.Rules) != 0 {
		t.Fatalf("nil default repo get = %+v err=%v", def, err)
	}
	// UpdateSignInPolicy は PolicyRepo nil でも event 発火して policy を返す。
	if _, err := appusecases.UpdateSignInPolicy(ctx, deps, appusecases.UpdateSignInPolicyInput{
		ApplicationID: app.ApplicationID, Rules: []spec.SignInRule{mfaRule()},
	}); err != nil {
		t.Fatalf("update with nil policy repo: %v", err)
	}
}

func TestEmptyPolicyBuilders(t *testing.T) {
	at := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	p := appusecases.EmptySignInPolicy("acme", "app-1", at)
	if p.TenantID != "acme" || p.ApplicationID != "app-1" || p.Rules == nil || !p.CreatedAt.Equal(at) {
		t.Fatalf("EmptySignInPolicy = %+v", p)
	}
	d := appusecases.EmptyDefaultSignInPolicy("acme", at)
	if d.TenantID != "acme" || d.Rules == nil || !d.CreatedAt.Equal(at) {
		t.Fatalf("EmptyDefaultSignInPolicy = %+v", d)
	}
}

func TestEvaluateSignInPolicyDecisions(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// nil policy -> allow。
	if got := appusecases.EvaluateSignInPolicy(nil, nil, "", now); got.Decision != appusecases.PolicyAllow {
		t.Fatalf("nil policy decision = %v", got.Decision)
	}

	mfaPolicy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{mfaRule()}}
	// authn 無し -> deny (fail-closed)。
	if got := appusecases.EvaluateSignInPolicy(mfaPolicy, nil, "", now); got.Decision != appusecases.PolicyDeny {
		t.Fatalf("nil authn decision = %v", got.Decision)
	}
	// MFA 未満 -> step-up。
	authn := &authdomain.AuthenticationContext{AuthTime: now.Unix(), ACR: "urn:pwd"}
	if got := appusecases.EvaluateSignInPolicy(mfaPolicy, authn, "", now); got.Decision != appusecases.PolicyStepUpRequired {
		t.Fatalf("mfa unmet decision = %v", got.Decision)
	}

	// ネットワーク条件: 許可外 IP -> deny。
	netPolicy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		Name: "Net", Enabled: true, RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnPassword},
		Condition: spec.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	if got := appusecases.EvaluateSignInPolicy(netPolicy, authn, "192.168.1.1", now); got.Decision != appusecases.PolicyDeny {
		t.Fatalf("network denied decision = %v", got.Decision)
	}
	// 許可内 IP -> allow。
	if got := appusecases.EvaluateSignInPolicy(netPolicy, authn, "10.1.2.3", now); got.Decision != appusecases.PolicyAllow {
		t.Fatalf("network allowed decision = %v", got.Decision)
	}

	// 再認証時間超過 -> step-up。
	maxAge := 60
	reauthPolicy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		Name: "Reauth", Enabled: true, RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnPassword},
		Condition: spec.AccessCondition{ReauthMaxAgeSeconds: &maxAge},
	}}}
	stale := &authdomain.AuthenticationContext{AuthTime: now.Add(-time.Hour).Unix(), ACR: "urn:pwd"}
	if got := appusecases.EvaluateSignInPolicy(reauthPolicy, stale, "", now); got.Decision != appusecases.PolicyStepUpRequired {
		t.Fatalf("reauth exceeded decision = %v", got.Decision)
	}
}
