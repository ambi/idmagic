package domain

import (
	"errors"
	"testing"
	"time"
)

func reasons(findings []AgentGuardrailFinding) map[string]AgentGuardrailViolationAction {
	out := map[string]AgentGuardrailViolationAction{}
	for _, f := range findings {
		out[f.Reason] = f.Action
	}
	return out
}

// 判定の両側と境界値 (REQ-OAUTH2-074)。
//
//spec:covers REQ-OAUTH2-074
func TestAgentGuardrailEvaluateBothSides(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		limits AgentGuardrailLimits
		req    AgentGuardrailRequest
		want   map[string]AgentGuardrailViolationAction
	}{
		{name: "未設定は制限しない", req: AgentGuardrailRequest{Resources: []string{"https://b"}, DelegationDepth: 99}, want: map[string]AgentGuardrailViolationAction{}},
		{name: "許可した資源", limits: AgentGuardrailLimits{AllowedResources: []string{"https://a"}}, req: AgentGuardrailRequest{Resources: []string{"https://a"}}, want: map[string]AgentGuardrailViolationAction{}},
		{name: "許可しない資源", limits: AgentGuardrailLimits{AllowedResources: []string{"https://a"}}, req: AgentGuardrailRequest{Resources: []string{"https://b"}}, want: map[string]AgentGuardrailViolationAction{AgentGuardrailReasonResource: AgentGuardrailDeny}},
		{name: "aud のない発行", limits: AgentGuardrailLimits{AllowedResources: []string{"https://a"}}, want: map[string]AgentGuardrailViolationAction{AgentGuardrailReasonResource: AgentGuardrailDeny}},
		{name: "空の許可は何も許さない", limits: AgentGuardrailLimits{AllowedResources: []string{}}, req: AgentGuardrailRequest{Resources: []string{"https://a"}}, want: map[string]AgentGuardrailViolationAction{AgentGuardrailReasonResource: AgentGuardrailDeny}},
		{name: "詳細のない発行", limits: AgentGuardrailLimits{AllowedAuthorizationDetailTypes: []string{"pay"}}, want: map[string]AgentGuardrailViolationAction{}},
		{name: "許可しない詳細を降格", limits: AgentGuardrailLimits{AllowedAuthorizationDetailTypes: []string{"pay"}, OnViolation: AgentGuardrailOnViolation{AllowedAuthorizationDetailTypes: AgentGuardrailRequireApproval}}, req: AgentGuardrailRequest{DetailTypes: []string{"wire"}}, want: map[string]AgentGuardrailViolationAction{AgentGuardrailReasonDetailType: AgentGuardrailRequireApproval}},
		{name: "期限の直前", limits: AgentGuardrailLimits{ExpiresAt: new(now.Add(time.Second))}, want: map[string]AgentGuardrailViolationAction{}},
		{name: "期限ちょうど", limits: AgentGuardrailLimits{ExpiresAt: new(now)}, want: map[string]AgentGuardrailViolationAction{AgentGuardrailReasonExpired: AgentGuardrailDeny}},
		{name: "深さの上限ちょうど", limits: AgentGuardrailLimits{MaxDelegationDepth: new(1)}, req: AgentGuardrailRequest{DelegationDepth: 1}, want: map[string]AgentGuardrailViolationAction{}},
		{name: "深さの超過", limits: AgentGuardrailLimits{MaxDelegationDepth: new(1)}, req: AgentGuardrailRequest{DelegationDepth: 2}, want: map[string]AgentGuardrailViolationAction{AgentGuardrailReasonDelegationDepth: AgentGuardrailDeny}},
		{name: "常に承認", limits: AgentGuardrailLimits{RequireApproval: true}, want: map[string]AgentGuardrailViolationAction{AgentGuardrailReasonApproval: AgentGuardrailRequireApproval}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reasons(tt.limits.Evaluate(tt.req, now))
			if len(got) != len(tt.want) {
				t.Fatalf("findings = %v, want %v", got, tt.want)
			}
			for reason, action := range tt.want {
				if got[reason] != action {
					t.Fatalf("findings = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

//spec:covers REQ-OAUTH2-074
func TestAgentGuardrailTokenLifetime(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	if got := (AgentGuardrailLimits{}).TokenLifetimeSeconds(600, now); got != 600 {
		t.Fatalf("no limit = %d", got)
	}
	if got := (AgentGuardrailLimits{MaxTokenTTLSeconds: new(60)}).TokenLifetimeSeconds(600, now); got != 60 {
		t.Fatalf("ttl = %d", got)
	}
	if got := (AgentGuardrailLimits{MaxTokenTTLSeconds: new(900)}).TokenLifetimeSeconds(600, now); got != 600 {
		t.Fatalf("ttl must not extend = %d", got)
	}
	if got := (AgentGuardrailLimits{ExpiresAt: new(now.Add(30 * time.Second))}).TokenLifetimeSeconds(600, now); got != 30 {
		t.Fatalf("expires_at = %d", got)
	}
	if got := (AgentGuardrailLimits{ExpiresAt: new(now.Add(-time.Second))}).TokenLifetimeSeconds(600, now); got != 600 {
		t.Fatalf("past expires_at = %d", got)
	}
}

//spec:covers REQ-OAUTH2-077
func TestAgentGuardrailValidateAndVersion(t *testing.T) {
	bad := []AgentGuardrailLimits{
		{IssuanceRate: &AgentGuardrailIssuanceRate{MaxIssuances: 0, WindowSeconds: 60}},
		{IssuanceRate: &AgentGuardrailIssuanceRate{MaxIssuances: 1, WindowSeconds: 86401}},
		{MaxTokenTTLSeconds: new(0)},
		{MaxDelegationDepth: new(17)},
		{AllowedResources: []string{""}},
		{OnViolation: AgentGuardrailOnViolation{IssuanceRate: "allow"}},
	}
	for i, limits := range bad {
		if err := limits.Validate(); !errors.Is(err, ErrInvalidAgentGuardrail) {
			t.Fatalf("bad[%d] = %v", i, err)
		}
	}
	a := AgentGuardrailLimits{AllowedResources: []string{"https://a"}}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if a.Version() != (AgentGuardrailLimits{AllowedResources: []string{"https://a"}}).Version() {
		t.Fatal("same content must have the same version")
	}
	// nil（制限なし）と空（何も許さない）は別のバージョンである。
	if (AgentGuardrailLimits{}).Version() == (AgentGuardrailLimits{AllowedResources: []string{}}).Version() {
		t.Fatal("nil and empty must differ")
	}
}
