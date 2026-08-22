package spec

import (
	"slices"
	"testing"
	"time"
)

func TestActionNameForCapability(t *testing.T) {
	action, ok := ActionNameForCapability("AdminUserRead")
	if !ok || action != ActionAdminUserRead {
		t.Fatalf("ActionNameForCapability(known) = (%q, %v)", action, ok)
	}
	if _, ok := ActionNameForCapability("NoSuchCapability"); ok {
		t.Fatal("expected ok=false for an unknown capability name")
	}
}

func TestRulesForAction(t *testing.T) {
	rules, ok := RulesForAction(ActionTokenGrantAuthorizationCode)
	if !ok || len(rules) == 0 {
		t.Fatalf("RulesForAction(known) = (%v, %v)", rules, ok)
	}
	rules[0] = "mutated"
	original, _ := RulesForAction(ActionTokenGrantAuthorizationCode)
	if original[0] == "mutated" {
		t.Fatal("RulesForAction must return a clone, not the backing slice")
	}
	if _, ok := RulesForAction("no-such-action"); ok {
		t.Fatal("expected ok=false for an unknown action")
	}
}

func TestGrantFromAction(t *testing.T) {
	cases := map[string]GrantType{
		ActionTokenGrantAuthorizationCode: GrantAuthorizationCode,
		ActionTokenGrantRefresh:           GrantRefreshToken,
		ActionTokenGrantClientCredentials: GrantClientCredentials,
		ActionTokenGrantDeviceCode:        GrantDeviceCode,
		ActionTokenGrantTokenExchange:     GrantTokenExchange,
		ActionTokenIntrospect:             "",
	}
	for action, want := range cases {
		if got := grantFromAction(action); got != want {
			t.Fatalf("grantFromAction(%q) = %q, want %q", action, got, want)
		}
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if isExpired(time.Time{}, now) {
		t.Fatal("zero expiresAt must never be expired")
	}
	if !isExpired(now.Add(-time.Minute), now) {
		t.Fatal("expiresAt in the past must be expired")
	}
	if isExpired(now.Add(time.Minute), now) {
		t.Fatal("expiresAt in the future must not be expired")
	}
	// A zero `now` falls back to time.Now(); a long-past expiresAt must still
	// register as expired under that fallback.
	if !isExpired(now.Add(-24*time.Hour), time.Time{}) {
		t.Fatal("zero now must fall back to the real current time")
	}
}

func TestEvaluateRejectsUndefinedAction(t *testing.T) {
	result := Evaluate(AuthZRequest{Action: "no-such-action"})
	if result.Permit {
		t.Fatal("expected undefined action to be denied")
	}
	if len(result.Reasons) != 1 {
		t.Fatalf("expected exactly one reason, got %v", result.Reasons)
	}
}

func TestAllRuleIDsAndImplementedRuleIDsAreConsistent(t *testing.T) {
	all := AllRuleIDs()
	if len(all) == 0 {
		t.Fatal("AllRuleIDs() must not be empty")
	}
	implemented := ImplementedRuleIDs()
	if len(implemented) == 0 {
		t.Fatal("ImplementedRuleIDs() must not be empty")
	}
	for _, id := range all {
		if !slices.Contains(implemented, id) {
			t.Fatalf("rule %q referenced by an action has no evaluator", id)
		}
	}
}
