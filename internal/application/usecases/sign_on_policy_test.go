package usecases

import (
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestEvaluateSignOnPolicyRequiresMFA(t *testing.T) {
	policy := &spec.AppSignOnPolicy{Rules: []spec.SignOnRule{{
		RuleID: "rule-1", Name: "MFA", Enabled: true,
		RequiredAuthn: spec.RequiredAuthnLevel{ACR: authusecases.ACRMFA},
	}}}

	got := EvaluateSignOnPolicy(policy, &authdomain.AuthenticationContext{
		Sub: "alice", ACR: authusecases.ACRPassword, AMR: []string{"pwd"},
	}, time.Now().UTC())

	if got.Decision != PolicyStepUpRequired {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyStepUpRequired)
	}
}

func TestEvaluateSignOnPolicyDenyUnsupportedConditions(t *testing.T) {
	policy := &spec.AppSignOnPolicy{Rules: []spec.SignOnRule{{
		RuleID: "rule-1", Name: "Network", Enabled: true,
		Condition: spec.AccessCondition{Network: "corp"},
	}}}

	got := EvaluateSignOnPolicy(policy, &authdomain.AuthenticationContext{
		Sub: "alice", ACR: authusecases.ACRMFA, AMR: []string{"pwd", "otp"},
	}, time.Now().UTC())

	if got.Decision != PolicyDeny {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyDeny)
	}
}

func TestEvaluateSignOnPolicyReauthMaxAge(t *testing.T) {
	maxAge := 300
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	policy := &spec.AppSignOnPolicy{Rules: []spec.SignOnRule{{
		RuleID: "rule-1", Name: "Fresh", Enabled: true,
		Condition: spec.AccessCondition{ReauthMaxAgeSeconds: &maxAge},
	}}}

	got := EvaluateSignOnPolicy(policy, &authdomain.AuthenticationContext{
		Sub: "alice", AuthTime: now.Add(-10 * time.Minute).Unix(), StepUpAt: now.Add(-2 * time.Minute).Unix(),
	}, now)

	if got.Decision != PolicyAllow {
		t.Fatalf("decision=%s, want %s", got.Decision, PolicyAllow)
	}
}
