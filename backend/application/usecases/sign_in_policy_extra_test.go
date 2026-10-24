package usecases_test

// sign-in policy の取得・更新・評価とテナントデフォルトの未カバー分岐を補う (wi-129)。

import (
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	"github.com/ambi/idmagic/backend/signinpolicy"
	policydomain "github.com/ambi/idmagic/backend/signinpolicy/domain"
	policyusecases "github.com/ambi/idmagic/backend/signinpolicy/usecases"

	"github.com/ambi/idmagic/backend/application/domain"
	appmemory "github.com/ambi/idmagic/backend/application/internal/db_memory"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
)

func newPolicyDeps() (policyusecases.SignInPolicyDeps, appusecases.ApplicationDeps) {
	apps := appmemory.NewApplicationRepository()
	appDeps := appusecases.ApplicationDeps{Repo: apps, AssignmentRepo: appmemory.NewApplicationAssignmentRepository()}
	policyDeps := policyusecases.SignInPolicyDeps{
		AppRepo:     application.Module{Repo: apps},
		PolicyRepo:  signinpolicy.NewMemoryModule().ApplicationRepo,
		DefaultRepo: signinpolicy.NewMemoryModule().DefaultRepo,
	}
	return policyDeps, appDeps
}

func mfaRule() policydomain.SignInRule {
	return policydomain.SignInRule{
		Name: "MFA", Enabled: true,
		RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnMfa},
	}
}

func TestGetAndUpdateSignInPolicy(t *testing.T) {
	ctx := tenantContext()
	policyDeps, appDeps := newPolicyDeps()
	app := seedApp(ctx, t, appDeps, "Payroll")
	policyDeps.AppRepo = application.Module{Repo: appDeps.Repo}

	// 未設定なら空ポリシー。
	empty, err := policyusecases.GetSignInPolicy(ctx, policyDeps, app.ID)
	if err != nil {
		t.Fatalf("get empty: %v", err)
	}
	if len(empty.Rules) != 0 {
		t.Fatalf("expected empty rules, got %v", empty.Rules)
	}

	// 更新して再取得すると 1 ルール。
	saved, err := policyusecases.UpdateSignInPolicy(ctx, policyDeps, policyusecases.UpdateSignInPolicyInput{
		ActorUserID: "admin", ApplicationID: app.ID, Rules: []policydomain.SignInRule{mfaRule()},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(saved.Rules) != 1 || saved.Rules[0].RuleID == "" {
		t.Fatalf("rule id not assigned: %+v", saved.Rules)
	}
	got, err := policyusecases.GetSignInPolicy(ctx, policyDeps, app.ID)
	if err != nil || len(got.Rules) != 1 {
		t.Fatalf("get after update = %+v err=%v", got, err)
	}

	// 存在しないアプリは not found。
	if _, err := policyusecases.GetSignInPolicy(ctx, policyDeps, "ghost"); !errors.Is(err, policyusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
	if _, err := policyusecases.UpdateSignInPolicy(ctx, policyDeps, policyusecases.UpdateSignInPolicyInput{
		ApplicationID: "ghost", Rules: []policydomain.SignInRule{mfaRule()},
	}); !errors.Is(err, policyusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
	// 名前空のルールは検証エラー。
	if _, err := policyusecases.UpdateSignInPolicy(ctx, policyDeps, policyusecases.UpdateSignInPolicyInput{
		ApplicationID: app.ID, Rules: []policydomain.SignInRule{{Enabled: true}},
	}); !errors.Is(err, policyusecases.ErrInvalidSignInPolicy) {
		t.Fatalf("expected ErrInvalidSignInPolicy, got %v", err)
	}
}

func TestGetAndUpdateDefaultSignInPolicy(t *testing.T) {
	ctx := tenantContext()
	policyDeps, _ := newPolicyDeps()

	def, err := policyusecases.GetDefaultSignInPolicy(ctx, policyDeps)
	if err != nil || len(def.Rules) != 0 {
		t.Fatalf("empty default = %+v err=%v", def, err)
	}
	saved, err := policyusecases.UpdateDefaultSignInPolicy(ctx, policyDeps, policyusecases.UpdateDefaultSignInPolicyInput{
		ActorUserID: "admin", Rules: []policydomain.SignInRule{mfaRule()},
	})
	if err != nil || len(saved.Rules) != 1 {
		t.Fatalf("update default = %+v err=%v", saved, err)
	}
	got, err := policyusecases.GetDefaultSignInPolicy(ctx, policyDeps)
	if err != nil || len(got.Rules) != 1 {
		t.Fatalf("get default = %+v err=%v", got, err)
	}
	// 不正 CIDR は検証エラー。
	badCIDR := policydomain.SignInRule{
		Name: "Net", Enabled: true, RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnPassword},
		Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"not-a-cidr"}},
	}
	if _, err := policyusecases.UpdateDefaultSignInPolicy(ctx, policyDeps, policyusecases.UpdateDefaultSignInPolicyInput{
		Rules: []policydomain.SignInRule{badCIDR},
	}); !errors.Is(err, policyusecases.ErrInvalidSignInPolicy) {
		t.Fatalf("expected ErrInvalidSignInPolicy for bad cidr, got %v", err)
	}
}

func TestSignInPolicyNilRepoFallbacks(t *testing.T) {
	ctx := tenantContext()
	apps := appmemory.NewApplicationRepository()
	deps := policyusecases.SignInPolicyDeps{AppRepo: application.Module{Repo: apps}}
	app, err := appusecases.CreateApplication(ctx, appusecases.ApplicationDeps{Repo: apps, AssignmentRepo: appmemory.NewApplicationAssignmentRepository()},
		appusecases.CreateApplicationInput{ActorUserID: "admin", Name: "X", Kind: domain.ApplicationFederated, Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "test-client"}})
	if err != nil {
		t.Fatal(err)
	}
	// PolicyRepo nil -> 空ポリシー。
	got, err := policyusecases.GetSignInPolicy(ctx, deps, app.ID)
	if err != nil || len(got.Rules) != 0 {
		t.Fatalf("nil policy repo get = %+v err=%v", got, err)
	}
	// DefaultRepo nil -> 空デフォルト。
	def, err := policyusecases.GetDefaultSignInPolicy(ctx, deps)
	if err != nil || len(def.Rules) != 0 {
		t.Fatalf("nil default repo get = %+v err=%v", def, err)
	}
	// UpdateSignInPolicy は PolicyRepo nil でも event 発火して policy を返す。
	if _, err := policyusecases.UpdateSignInPolicy(ctx, deps, policyusecases.UpdateSignInPolicyInput{
		ApplicationID: app.ID, Rules: []policydomain.SignInRule{mfaRule()},
	}); err != nil {
		t.Fatalf("update with nil policy repo: %v", err)
	}
}

func TestEmptyPolicyBuilders(t *testing.T) {
	at := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	p := policyusecases.EmptySignInPolicy("acme", "app-1", at)
	if p.TenantID != "acme" || p.ApplicationID != "app-1" || p.Rules == nil || !p.CreatedAt.Equal(at) {
		t.Fatalf("EmptySignInPolicy = %+v", p)
	}
	d := policyusecases.EmptyDefaultSignInPolicy("acme", at)
	if d.TenantID != "acme" || d.Rules == nil || !d.CreatedAt.Equal(at) {
		t.Fatalf("EmptyDefaultSignInPolicy = %+v", d)
	}
}

func TestEvaluateSignInPolicyDecisions(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// nil policy -> allow。
	if got := policyusecases.EvaluateSignInPolicy(nil, nil, "", now); got.Decision != policyusecases.PolicyAllow {
		t.Fatalf("nil policy decision = %v", got.Decision)
	}

	mfaPolicy := &policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{mfaRule()}}
	// authn 無し -> deny (fail-closed)。
	if got := policyusecases.EvaluateSignInPolicy(mfaPolicy, nil, "", now); got.Decision != policyusecases.PolicyDeny {
		t.Fatalf("nil authn decision = %v", got.Decision)
	}
	// MFA 未満 -> step-up。
	authn := &authdomain.AuthenticationContext{AuthTime: now.Unix(), ACR: "urn:pwd"}
	if got := policyusecases.EvaluateSignInPolicy(mfaPolicy, authn, "", now); got.Decision != policyusecases.PolicyStepUpRequired {
		t.Fatalf("mfa unmet decision = %v", got.Decision)
	}

	// ネットワーク条件: 許可外 IP -> deny。
	netPolicy := &policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{
		Name: "Net", Enabled: true, RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnPassword},
		Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	if got := policyusecases.EvaluateSignInPolicy(netPolicy, authn, "192.168.1.1", now); got.Decision != policyusecases.PolicyDeny {
		t.Fatalf("network denied decision = %v", got.Decision)
	}
	// 許可内 IP -> allow。
	if got := policyusecases.EvaluateSignInPolicy(netPolicy, authn, "10.1.2.3", now); got.Decision != policyusecases.PolicyAllow {
		t.Fatalf("network allowed decision = %v", got.Decision)
	}

	// 再認証時間超過 -> step-up。
	maxAge := 60
	reauthPolicy := &policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{
		Name: "Reauth", Enabled: true, RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnPassword},
		Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: &maxAge},
	}}}
	stale := &authdomain.AuthenticationContext{AuthTime: now.Add(-time.Hour).Unix(), ACR: "urn:pwd"}
	if got := policyusecases.EvaluateSignInPolicy(reauthPolicy, stale, "", now); got.Decision != policyusecases.PolicyStepUpRequired {
		t.Fatalf("reauth exceeded decision = %v", got.Decision)
	}
}

func TestSignInPolicySettingsFromRulesAndClientIP(t *testing.T) {
	ctx := tenantContext()
	policyDeps, appDeps := newPolicyDeps()
	app := seedApp(ctx, t, appDeps, "Payroll")
	policyDeps.AppRepo = application.Module{Repo: appDeps.Repo}

	// settingsFromRules: no enabled rules
	weaker := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Name: "MFA", Enabled: false}}},
		&policydomain.AppSignInPolicy{ApplicationID: app.ID, Rules: []policydomain.SignInRule{{Name: "MFA", Enabled: false}}},
	)
	if weaker {
		t.Fatalf("should not be weaker because no enabled rules")
	}

	// evaluateSignInPolicy error cases
	// clientIP parse error
	policy := &policydomain.AppSignInPolicy{
		ApplicationID: app.ID,
		Rules: []policydomain.SignInRule{{
			Name: "Network Limit", Enabled: true,
			Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"192.168.1.0/24"}},
		}},
	}
	eval := policyusecases.EvaluateSignInPolicy(
		policy,
		&authdomain.AuthenticationContext{AuthTime: time.Now().Unix(), ACR: "urn:pwd"},
		"invalid-ip",
		time.Now(),
	)
	if eval.Decision != policyusecases.PolicyDeny {
		t.Fatalf("expected PolicyDeny for invalid IP, got %v", eval.Decision)
	}

	// invalid CIDR format in rule
	policyBadCIDR := &policydomain.AppSignInPolicy{
		ApplicationID: app.ID,
		Rules: []policydomain.SignInRule{{
			Name: "Network Limit", Enabled: true,
			Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"invalid-cidr"}},
		}},
	}
	eval2 := policyusecases.EvaluateSignInPolicy(
		policyBadCIDR,
		&authdomain.AuthenticationContext{AuthTime: time.Now().Unix(), ACR: "urn:pwd"},
		"192.168.1.1",
		time.Now(),
	)
	if eval2.Decision != policyusecases.PolicyDeny {
		t.Fatalf("expected PolicyDeny for invalid CIDR entry, got %v", eval2.Decision)
	}
}

func TestEnsureApplicationExistsErrors(t *testing.T) {
	ctx := tenantContext()
	policyDeps, _ := newPolicyDeps()

	// repo is nil
	policyDeps.AppRepo = nil
	_, err := policyusecases.GetSignInPolicy(ctx, policyDeps, "some-app")
	if !errors.Is(err, policyusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound when Repo is nil, got %v", err)
	}
}

func TestValidateSignInPolicyRulesExtraErrors(t *testing.T) {
	// 1. Duplicate RuleID
	dupRule := policydomain.SignInRule{
		RuleID: "rule-1", Name: "Rule 1", Enabled: true,
	}
	dupRule2 := policydomain.SignInRule{
		RuleID: "rule-1", Name: "Rule 2", Enabled: true,
	}
	err := policyusecases.ValidateSignInPolicyRules([]policydomain.SignInRule{dupRule, dupRule2})
	if !errors.Is(err, policyusecases.ErrInvalidSignInPolicy) {
		t.Fatalf("expected ErrInvalidSignInPolicy for duplicate RuleID, got %v", err)
	}

	// 2. Invalid Strength
	badStrength := policydomain.SignInRule{
		Name: "Rule 1", Enabled: true,
		RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: "SuperStrong"},
	}
	err = policyusecases.ValidateSignInPolicyRules([]policydomain.SignInRule{badStrength})
	if !errors.Is(err, policyusecases.ErrInvalidSignInPolicy) {
		t.Fatalf("expected ErrInvalidSignInPolicy for invalid strength, got %v", err)
	}

	// 3. ReauthMaxAgeSeconds <= 0
	badReauth := 0
	badReauthRule := policydomain.SignInRule{
		Name: "Rule 1", Enabled: true,
		Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: &badReauth},
	}
	err = policyusecases.ValidateSignInPolicyRules([]policydomain.SignInRule{badReauthRule})
	if !errors.Is(err, policyusecases.ErrInvalidSignInPolicy) {
		t.Fatalf("expected ErrInvalidSignInPolicy for non-positive reauth max age, got %v", err)
	}

	// 4. normalizeCIDRs with empty entry
	emptyCIDRRule := policydomain.SignInRule{
		Name: "Rule 1", Enabled: true,
		Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"", "192.168.1.0/24"}},
	}
	err = policyusecases.ValidateSignInPolicyRules([]policydomain.SignInRule{emptyCIDRRule})
	if err != nil {
		t.Fatalf("expected no error for empty CIDR entry, got %v", err)
	}
}

func TestUpdateDefaultSignInPolicyUpdatesExisting(t *testing.T) {
	ctx := tenantContext()
	policyDeps, _ := newPolicyDeps()

	// 1回目
	_, err := policyusecases.UpdateDefaultSignInPolicy(ctx, policyDeps, policyusecases.UpdateDefaultSignInPolicyInput{
		ActorUserID: "admin", Rules: []policydomain.SignInRule{mfaRule()},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 2回目 (existing が存在)
	_, err = policyusecases.UpdateDefaultSignInPolicy(ctx, policyDeps, policyusecases.UpdateDefaultSignInPolicyInput{
		ActorUserID: "admin", Rules: []policydomain.SignInRule{mfaRule()},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetDefaultSignInPolicyNullRules(t *testing.T) {
	ctx := tenantContext()
	policyDeps, _ := newPolicyDeps()

	// Rules が nil の policy を seed
	policy := &policydomain.TenantDefaultSignInPolicy{
		TenantID: "acme",
		Rules:    nil,
	}
	_ = policyDeps.DefaultRepo.Save(ctx, policy)

	got, err := policyusecases.GetDefaultSignInPolicy(ctx, policyDeps)
	if err != nil {
		t.Fatal(err)
	}
	if got.Rules == nil {
		t.Fatalf("rules should be normalized to empty slice, got nil")
	}
}

func TestAppPolicyWeakerThanDefaultExtraWeaker(t *testing.T) {
	maxAgeDef := 3600
	maxAgeAppWeaker := 7200
	maxAgeAppStronger := 1800

	// 1. Reauth: Def settings has maxAge, App has none -> weaker: true
	weaker1 := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: &maxAgeDef}}}},
		&policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: nil}}}},
	)
	if !weaker1 {
		t.Fatalf("expected weaker due to missing reauth max age in app")
	}

	// 2. Reauth: App maxAge is larger than Def -> weaker: true
	weaker2 := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: &maxAgeDef}}}},
		&policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: &maxAgeAppWeaker}}}},
	)
	if !weaker2 {
		t.Fatalf("expected weaker due to larger reauth max age in app")
	}

	// 3. Reauth: App maxAge is smaller than Def -> weaker: false
	weaker3 := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: &maxAgeDef}}}},
		&policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{ReauthMaxAgeSeconds: &maxAgeAppStronger}}}},
	)
	if weaker3 {
		t.Fatalf("expected not weaker since app reauth is stronger")
	}

	// 4. Network: Def has CIDR restrictions, App has none -> weaker: true
	weaker4 := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8"}}}}},
		&policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{NetworkAllowCIDRs: nil}}}},
	)
	if !weaker4 {
		t.Fatalf("expected weaker due to missing network restrictions in app")
	}

	// 5. Network: App has CIDR restrictions, Def has none -> weaker: false
	weaker5 := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{NetworkAllowCIDRs: nil}}}},
		&policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8"}}}}},
	)
	if weaker5 {
		t.Fatalf("expected not weaker since default has no network restrictions")
	}

	// 6. MFA: Def requires MFA, App does not -> weaker: true
	weaker6 := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnMfa}}}},
		&policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, RequiredAuthn: policydomain.RequiredAuthnLevel{Strength: policydomain.RequiredAuthnPassword}}}},
	)
	if !weaker6 {
		t.Fatalf("expected weaker due to app missing MFA under MFA default")
	}

	// 7. Network: App allows CIDR outside Def -> weaker: true
	weaker7 := policyusecases.AppPolicyWeakerThanDefault(
		&policydomain.TenantDefaultSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8"}}}}},
		&policydomain.AppSignInPolicy{Rules: []policydomain.SignInRule{{Enabled: true, Condition: policydomain.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8", "192.168.1.0/24"}}}}},
	)
	if !weaker7 {
		t.Fatalf("expected weaker because app allows extra CIDR outside default")
	}
}
