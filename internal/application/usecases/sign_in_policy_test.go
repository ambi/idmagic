package usecases

import (
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestEvaluateSignInPolicyRequiresMFA(t *testing.T) {
	policy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "rule-1", Name: "MFA", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
	}}}

	got := EvaluateSignInPolicy(policy, &authdomain.AuthenticationContext{
		Sub: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"},
	}, "", time.Now().UTC())

	if got.Decision != PolicyStepUpRequired {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyStepUpRequired)
	}
}

func TestEvaluateSignInPolicyMfaSatisfied(t *testing.T) {
	policy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "rule-1", Name: "MFA", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
	}}}

	got := EvaluateSignInPolicy(policy, &authdomain.AuthenticationContext{
		Sub: "alice", ACR: authusecases.ACRMFA, AMR: []string{"pwd", "otp"},
	}, "", time.Now().UTC())

	if got.Decision != PolicyAllow {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyAllow)
	}
}

func TestEvaluateSignInPolicyNetworkCIDRAllows(t *testing.T) {
	policy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "rule-1", Name: "Network", Enabled: true,
		Condition: spec.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	authn := &authdomain.AuthenticationContext{Sub: "alice", ACR: authusecases.ACRMFA, AMR: []string{"pwd", "otp"}}

	if got := EvaluateSignInPolicy(policy, authn, "10.1.2.3", time.Now().UTC()); got.Decision != PolicyAllow {
		t.Fatalf("in-range decision=%s, want %s", got.Decision, PolicyAllow)
	}
	if got := EvaluateSignInPolicy(policy, authn, "192.168.0.1", time.Now().UTC()); got.Decision != PolicyDeny {
		t.Fatalf("out-of-range decision=%s, want %s", got.Decision, PolicyDeny)
	}
	// クライアント IP を取得できない場合は fail-closed で拒否する。
	if got := EvaluateSignInPolicy(policy, authn, "", time.Now().UTC()); got.Decision != PolicyDeny {
		t.Fatalf("missing-ip decision=%s, want %s", got.Decision, PolicyDeny)
	}
}

func TestEvaluateSignInPolicyReauthMaxAge(t *testing.T) {
	maxAge := 300
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	policy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "rule-1", Name: "Fresh", Enabled: true,
		Condition: spec.AccessCondition{ReauthMaxAgeSeconds: &maxAge},
	}}}

	got := EvaluateSignInPolicy(policy, &authdomain.AuthenticationContext{
		Sub: "alice", AuthTime: now.Add(-10 * time.Minute).Unix(), StepUpAt: now.Add(-2 * time.Minute).Unix(),
	}, "", now)

	if got.Decision != PolicyAllow {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyAllow)
	}
}

func mfaRule(id string) spec.SignInRule {
	return spec.SignInRule{
		RuleID: id, Name: "MFA", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
	}
}

func passwordRule(id string) spec.SignInRule {
	return spec.SignInRule{
		RuleID: id, Name: "Password", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnPassword},
	}
}

func TestEffectiveSignInRulesAppOverridesDefault(t *testing.T) {
	def := &spec.TenantDefaultSignInPolicy{Rules: []spec.SignInRule{mfaRule("default-1")}}
	app := &spec.AppSignInPolicy{Rules: []spec.SignInRule{passwordRule("app-1")}}

	// アプリが独自ポリシーを持てばデフォルトを完全に置換する (合成しない)。
	got := EffectiveSignInRules(def, app)
	if len(got) != 1 || got[0].RuleID != "app-1" {
		t.Fatalf("override result=%v, want only [app-1]", got)
	}
}

func TestEffectiveSignInRulesFallsBackToDefault(t *testing.T) {
	def := &spec.TenantDefaultSignInPolicy{Rules: []spec.SignInRule{mfaRule("default-1")}}

	// アプリが独自ポリシーを持たなければデフォルトを適用する。
	got := EffectiveSignInRules(def, &spec.AppSignInPolicy{Rules: []spec.SignInRule{}})
	if len(got) != 1 || got[0].RuleID != "default-1" {
		t.Fatalf("fallback result=%v, want only [default-1]", got)
	}
}

func TestEffectivePolicyForEvaluationNoRulesReturnsNil(t *testing.T) {
	if p := EffectivePolicyForEvaluation(nil, &spec.AppSignInPolicy{}); p != nil {
		t.Fatalf("expected nil policy for empty effective rules, got %+v", p)
	}
}

func TestEffectivePolicyOverrideCanGoBelowDefault(t *testing.T) {
	def := &spec.TenantDefaultSignInPolicy{Rules: []spec.SignInRule{mfaRule("default-1")}}
	singleFactor := &authdomain.AuthenticationContext{Sub: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"}}

	// アプリ独自ポリシーが無ければデフォルトの MFA 要求が適用される。
	appNoPolicy := &spec.AppSignInPolicy{Rules: []spec.SignInRule{}}
	effective := EffectivePolicyForEvaluation(def, appNoPolicy)
	if got := EvaluateSignInPolicy(effective, singleFactor, "", time.Now().UTC()); got.Decision != PolicyStepUpRequired {
		t.Fatalf("default decision=%s, want %s", got.Decision, PolicyStepUpRequired)
	}

	// 上書きモデルではアプリ独自ポリシーがデフォルトより弱くても適用される (警告は別途)。
	appWeaker := &spec.AppSignInPolicy{Rules: []spec.SignInRule{passwordRule("app-1")}}
	if got := EvaluateSignInPolicy(EffectivePolicyForEvaluation(def, appWeaker), singleFactor, "", time.Now().UTC()); got.Decision != PolicyAllow {
		t.Fatalf("override decision=%s, want %s", got.Decision, PolicyAllow)
	}
}

func TestAppPolicyWeakerThanDefault(t *testing.T) {
	longer := 600
	shorter := 300
	def := &spec.TenantDefaultSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "d", Name: "d", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
		Condition:     spec.AccessCondition{ReauthMaxAgeSeconds: &shorter, NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}

	// 独自ポリシー無し → デフォルト適用なので弱くない。
	if AppPolicyWeakerThanDefault(def, &spec.AppSignInPolicy{}) {
		t.Fatal("unconfigured app must not be weaker")
	}
	// 認証強度の引き下げ。
	if !AppPolicyWeakerThanDefault(def, &spec.AppSignInPolicy{Rules: []spec.SignInRule{passwordRule("a")}}) {
		t.Fatal("password override must be weaker than mfa default")
	}
	// 再認証時間の緩和 (延長)。
	weakReauth := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "a", Name: "a", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
		Condition:     spec.AccessCondition{ReauthMaxAgeSeconds: &longer, NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	if !AppPolicyWeakerThanDefault(def, weakReauth) {
		t.Fatal("longer reauth window must be weaker")
	}
	// 許可ネットワークの緩和 (制限撤廃)。
	noCIDR := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "a", Name: "a", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
		Condition:     spec.AccessCondition{ReauthMaxAgeSeconds: &shorter},
	}}}
	if !AppPolicyWeakerThanDefault(def, noCIDR) {
		t.Fatal("removing network restriction must be weaker")
	}
	// 全項目でデフォルト以上 → 弱くない。
	strong := &spec.AppSignInPolicy{Rules: []spec.SignInRule{{
		RuleID: "a", Name: "a", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{Strength: spec.RequiredAuthnMfa},
		Condition:     spec.AccessCondition{ReauthMaxAgeSeconds: &shorter, NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	if AppPolicyWeakerThanDefault(def, strong) {
		t.Fatal("equal-strength override must not be weaker")
	}
}

func TestValidateSignInPolicyRulesRejectsInvalidCIDR(t *testing.T) {
	rules := []spec.SignInRule{{
		Name: "bad", Enabled: true,
		Condition: spec.AccessCondition{NetworkAllowCIDRs: []string{"not-a-cidr"}},
	}}
	if err := ValidateSignInPolicyRules(rules); err == nil {
		t.Fatal("expected invalid CIDR to be rejected")
	}
}

func TestValidateSignInPolicyRulesDefaultsStrengthAndNormalizesCIDR(t *testing.T) {
	rules := []spec.SignInRule{{
		Name: "ok", Enabled: true,
		Condition: spec.AccessCondition{NetworkAllowCIDRs: []string{" 10.1.2.3/8 ", ""}},
	}}
	if err := ValidateSignInPolicyRules(rules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules[0].RequiredAuthn.Strength != spec.RequiredAuthnPassword {
		t.Fatalf("strength=%q, want Password default", rules[0].RequiredAuthn.Strength)
	}
	if len(rules[0].Condition.NetworkAllowCIDRs) != 1 || rules[0].Condition.NetworkAllowCIDRs[0] != "10.0.0.0/8" {
		t.Fatalf("normalized CIDRs=%v, want [10.0.0.0/8]", rules[0].Condition.NetworkAllowCIDRs)
	}
	if rules[0].RuleID == "" {
		t.Fatal("expected rule_id to be generated")
	}
}
