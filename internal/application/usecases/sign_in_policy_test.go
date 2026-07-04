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
