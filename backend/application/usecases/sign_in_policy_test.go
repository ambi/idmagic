package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application/domain"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
)

func TestValidateSignInPolicyRulesAtMfaEnrollment(t *testing.T) {
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	grace := 900
	rules := []domain.SignInRule{{
		Name: "mfa", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
		MfaEnrollment: &domain.MfaEnrollmentPolicy{EnforcementStartAt: &start, GracePeriodSeconds: &grace, AllowAdminBypass: true},
	}}
	if err := ValidateSignInPolicyRulesAt(rules, now); err != nil {
		t.Fatalf("valid enrollment policy: %v", err)
	}
	past := now.Add(-time.Second)
	rules[0].MfaEnrollment.EnforcementStartAt = &past
	if err := ValidateSignInPolicyRulesAt(rules, now); !errors.Is(err, ErrInvalidSignInPolicy) {
		t.Fatalf("past start: %v", err)
	}
}

func TestEvaluateSignInPolicyRequiresMFA(t *testing.T) {
	policy := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "rule-1", Name: "MFA", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
	}}}

	got := EvaluateSignInPolicy(policy, &authdomain.AuthenticationContext{
		UserID: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"},
	}, "", time.Now().UTC())

	if got.Decision != PolicyStepUpRequired {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyStepUpRequired)
	}
}

func TestEvaluateSignInPolicyMfaSatisfied(t *testing.T) {
	policy := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "rule-1", Name: "MFA", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
	}}}

	got := EvaluateSignInPolicy(policy, &authdomain.AuthenticationContext{
		UserID: "alice", ACR: authusecases.ACRMFA, AMR: []string{"pwd", "otp"},
	}, "", time.Now().UTC())

	if got.Decision != PolicyAllow {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyAllow)
	}
}

func TestEvaluateSignInPolicyNetworkCIDRAllows(t *testing.T) {
	policy := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "rule-1", Name: "Network", Enabled: true,
		Condition: domain.AccessCondition{NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	authn := &authdomain.AuthenticationContext{UserID: "alice", ACR: authusecases.ACRMFA, AMR: []string{"pwd", "otp"}}

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
	policy := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "rule-1", Name: "Fresh", Enabled: true,
		Condition: domain.AccessCondition{ReauthMaxAgeSeconds: &maxAge},
	}}}

	got := EvaluateSignInPolicy(policy, &authdomain.AuthenticationContext{
		UserID: "alice", AuthTime: now.Add(-10 * time.Minute).Unix(), StepUpAt: now.Add(-2 * time.Minute).Unix(),
	}, "", now)

	if got.Decision != PolicyAllow {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyAllow)
	}
}

func mfaRule(id string) domain.SignInRule {
	return domain.SignInRule{
		RuleID: id, Name: "MFA", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
	}
}

func passwordRule(id string) domain.SignInRule {
	return domain.SignInRule{
		RuleID: id, Name: "Password", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnPassword},
	}
}

func TestEffectiveSignInRulesAppOverridesDefault(t *testing.T) {
	def := &domain.TenantDefaultSignInPolicy{Rules: []domain.SignInRule{mfaRule("default-1")}}
	app := &domain.AppSignInPolicy{Rules: []domain.SignInRule{passwordRule("app-1")}}

	// アプリが独自ポリシーを持てばデフォルトを完全に置換する (合成しない)。
	got := EffectiveSignInRules(def, app)
	if len(got) != 1 || got[0].RuleID != "app-1" {
		t.Fatalf("override result=%v, want only [app-1]", got)
	}
}

func TestEffectiveSignInRulesFallsBackToDefault(t *testing.T) {
	def := &domain.TenantDefaultSignInPolicy{Rules: []domain.SignInRule{mfaRule("default-1")}}

	// アプリが独自ポリシーを持たなければデフォルトを適用する。
	got := EffectiveSignInRules(def, &domain.AppSignInPolicy{Rules: []domain.SignInRule{}})
	if len(got) != 1 || got[0].RuleID != "default-1" {
		t.Fatalf("fallback result=%v, want only [default-1]", got)
	}
}

func TestEffectivePolicyForEvaluationNoRulesReturnsNil(t *testing.T) {
	if p := EffectivePolicyForEvaluation(nil, &domain.AppSignInPolicy{}); p != nil {
		t.Fatalf("expected nil policy for empty effective rules, got %+v", p)
	}
}

func TestEffectivePolicyOverrideCanGoBelowDefault(t *testing.T) {
	def := &domain.TenantDefaultSignInPolicy{Rules: []domain.SignInRule{mfaRule("default-1")}}
	singleFactor := &authdomain.AuthenticationContext{UserID: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"}}

	// アプリ独自ポリシーが無ければデフォルトの MFA 要求が適用される。
	appNoPolicy := &domain.AppSignInPolicy{Rules: []domain.SignInRule{}}
	effective := EffectivePolicyForEvaluation(def, appNoPolicy)
	if got := EvaluateSignInPolicy(effective, singleFactor, "", time.Now().UTC()); got.Decision != PolicyStepUpRequired {
		t.Fatalf("default decision=%s, want %s", got.Decision, PolicyStepUpRequired)
	}

	// 上書きモデルではアプリ独自ポリシーがデフォルトより弱くても適用される (警告は別途)。
	appWeaker := &domain.AppSignInPolicy{Rules: []domain.SignInRule{passwordRule("app-1")}}
	if got := EvaluateSignInPolicy(EffectivePolicyForEvaluation(def, appWeaker), singleFactor, "", time.Now().UTC()); got.Decision != PolicyAllow {
		t.Fatalf("override decision=%s, want %s", got.Decision, PolicyAllow)
	}
}

func TestAppPolicyWeakerThanDefault(t *testing.T) {
	longer := 600
	shorter := 300
	def := &domain.TenantDefaultSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "d", Name: "d", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
		Condition:     domain.AccessCondition{ReauthMaxAgeSeconds: &shorter, NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}

	// 独自ポリシー無し → デフォルト適用なので弱くない。
	if AppPolicyWeakerThanDefault(def, &domain.AppSignInPolicy{}) {
		t.Fatal("unconfigured app must not be weaker")
	}
	// 認証強度の引き下げ。
	if !AppPolicyWeakerThanDefault(def, &domain.AppSignInPolicy{Rules: []domain.SignInRule{passwordRule("a")}}) {
		t.Fatal("password override must be weaker than mfa default")
	}
	// 再認証時間の緩和 (延長)。
	weakReauth := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "a", Name: "a", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
		Condition:     domain.AccessCondition{ReauthMaxAgeSeconds: &longer, NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	if !AppPolicyWeakerThanDefault(def, weakReauth) {
		t.Fatal("longer reauth window must be weaker")
	}
	// 許可ネットワークの緩和 (制限撤廃)。
	noCIDR := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "a", Name: "a", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
		Condition:     domain.AccessCondition{ReauthMaxAgeSeconds: &shorter},
	}}}
	if !AppPolicyWeakerThanDefault(def, noCIDR) {
		t.Fatal("removing network restriction must be weaker")
	}
	// 全項目でデフォルト以上 → 弱くない。
	strong := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
		RuleID: "a", Name: "a", Enabled: true,
		RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnMfa},
		Condition:     domain.AccessCondition{ReauthMaxAgeSeconds: &shorter, NetworkAllowCIDRs: []string{"10.0.0.0/8"}},
	}}}
	if AppPolicyWeakerThanDefault(def, strong) {
		t.Fatal("equal-strength override must not be weaker")
	}
}

func TestValidateSignInPolicyRulesRejectsInvalidCIDR(t *testing.T) {
	rules := []domain.SignInRule{{
		Name: "bad", Enabled: true,
		Condition: domain.AccessCondition{NetworkAllowCIDRs: []string{"not-a-cidr"}},
	}}
	if err := ValidateSignInPolicyRules(rules); err == nil {
		t.Fatal("expected invalid CIDR to be rejected")
	}
}

func TestValidateSignInPolicyRulesDefaultsStrengthAndNormalizesCIDR(t *testing.T) {
	rules := []domain.SignInRule{{
		Name: "ok", Enabled: true,
		Condition: domain.AccessCondition{NetworkAllowCIDRs: []string{" 10.1.2.3/8 ", ""}},
	}}
	if err := ValidateSignInPolicyRules(rules); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rules[0].RequiredAuthn.Strength != domain.RequiredAuthnPassword {
		t.Fatalf("strength=%q, want Password default", rules[0].RequiredAuthn.Strength)
	}
	if len(rules[0].Condition.NetworkAllowCIDRs) != 1 || rules[0].Condition.NetworkAllowCIDRs[0] != "10.0.0.0/8" {
		t.Fatalf("normalized CIDRs=%v, want [10.0.0.0/8]", rules[0].Condition.NetworkAllowCIDRs)
	}
	if rules[0].RuleID == "" {
		t.Fatal("expected rule_id to be generated")
	}
}

// REQ-AUTHENTICATION-027: 記憶済みデバイスによる昇格 (amr=tdev) は既定では MFA 要件を
// 満たすが、allow_trusted_device=false のルールでは満たさない (wi-91)。
func TestEvaluateSignInPolicyTrustedDeviceSatisfiesMfaOnlyWhenAllowed(t *testing.T) {
	deny := false
	trustedOnly := &authdomain.AuthenticationContext{
		UserID: "alice", ACR: authusecases.ACRMFA, AMR: []string{"pwd", "tdev"},
	}
	realFactor := &authdomain.AuthenticationContext{
		UserID: "alice", ACR: authusecases.ACRMFA, AMR: []string{"pwd", "otp", "tdev"},
	}
	now := time.Now().UTC()

	allowing := &domain.AppSignInPolicy{Rules: []domain.SignInRule{mfaRule("rule-1")}}
	if got := EvaluateSignInPolicy(allowing, trustedOnly, "", now); got.Decision != PolicyAllow {
		t.Fatalf("decision=%s, want %s when the rule allows trusted devices", got.Decision, PolicyAllow)
	}

	everyTime := &domain.AppSignInPolicy{Rules: []domain.SignInRule{mfaRule("rule-1")}}
	everyTime.Rules[0].AllowTrustedDevice = &deny
	if got := EvaluateSignInPolicy(everyTime, trustedOnly, "", now); got.Decision != PolicyStepUpRequired {
		t.Fatalf("decision=%s, want %s when the rule requires MFA every time", got.Decision, PolicyStepUpRequired)
	}
	// 本物の第二要素があれば、同じセッションに tdev が併存していても満たされている。
	if got := EvaluateSignInPolicy(everyTime, realFactor, "", now); got.Decision != PolicyAllow {
		t.Fatalf("decision=%s, want %s when a genuine second factor was presented", got.Decision, PolicyAllow)
	}
}

// 「毎回 MFA」のテナントデフォルトを、記憶済みデバイスを許すアプリ設定で緩めた場合は
// weaker_than_default として警告する (wi-91)。
func TestAppPolicyWeakerThanDefaultDetectsTrustedDeviceRelaxation(t *testing.T) {
	deny := false
	allow := true
	def := &domain.TenantDefaultSignInPolicy{Rules: []domain.SignInRule{mfaRule("default-1")}}
	def.Rules[0].AllowTrustedDevice = &deny
	app := &domain.AppSignInPolicy{Rules: []domain.SignInRule{mfaRule("app-1")}}
	app.Rules[0].AllowTrustedDevice = &allow

	if !AppPolicyWeakerThanDefault(def, app) {
		t.Fatal("allowing trusted devices where the default forbids them is a weaker policy")
	}
	app.Rules[0].AllowTrustedDevice = &deny
	if AppPolicyWeakerThanDefault(def, app) {
		t.Fatal("matching the default must not be reported as weaker")
	}
}

// 既存の保存済みルール (allow_trusted_device 未設定) は既定の true として読む (wi-91)。
func TestTrustedDeviceAllowedDefaultsToTrue(t *testing.T) {
	if !(domain.SignInRule{}).TrustedDeviceAllowed() {
		t.Fatal("an unset allow_trusted_device must read as the declared default (true)")
	}
	if !TrustedDeviceAllowedByRules([]domain.SignInRule{mfaRule("rule-1")}) {
		t.Fatal("rules that do not set the field must allow trusted devices")
	}
	deny := false
	rules := []domain.SignInRule{mfaRule("rule-1"), mfaRule("rule-2")}
	rules[1].AllowTrustedDevice = &deny
	if TrustedDeviceAllowedByRules(rules) {
		t.Fatal("a single every-time-MFA rule must win over the others")
	}
}
