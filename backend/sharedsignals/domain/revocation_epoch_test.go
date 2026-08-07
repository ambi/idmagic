package domain_test

import (
	"testing"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func validRevocationEpoch() ssdomain.AgentRevocationEpoch {
	now := time.Now().UTC()
	return ssdomain.AgentRevocationEpoch{
		AgentID:    "agent_1",
		TenantID:   "tenant-a",
		Epoch:      now,
		Reason:     ssdomain.RevocationReasonAgentKilled,
		AdvancedAt: now,
	}
}

// TestAgentRevocationEpochValidateHappyAndFailure — scenario
// `kill-switchは既発行トークンをintrospectionで即時無効化する` の前提となる
// AgentRevocationEpoch の構造的妥当性を検証する。
func TestAgentRevocationEpochValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ssdomain.AgentRevocationEpoch)
		wantErr bool
	}{
		{"ok", func(*ssdomain.AgentRevocationEpoch) {}, false},
		{"missing agent id", func(e *ssdomain.AgentRevocationEpoch) { e.AgentID = "" }, true},
		{"missing tenant", func(e *ssdomain.AgentRevocationEpoch) { e.TenantID = "" }, true},
		{"bad reason", func(e *ssdomain.AgentRevocationEpoch) { e.Reason = ssdomain.RevocationReason("x") }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := validRevocationEpoch()
			c.mutate(&e)
			err := e.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
		})
	}
}

// TestAgentRevocationEpochSupersedes — Introspect の fail-closed 判定
// (issued_at < epoch なら失効) の前提となる。
func TestAgentRevocationEpochSupersedes(t *testing.T) {
	epoch := validRevocationEpoch()
	before := epoch.Epoch.Add(-time.Minute)
	after := epoch.Epoch.Add(time.Minute)

	if !epoch.Supersedes(before) {
		t.Fatal("token issued before epoch must be superseded (revoked)")
	}
	if epoch.Supersedes(after) {
		t.Fatal("token issued after epoch must not be superseded")
	}
	if epoch.Supersedes(epoch.Epoch) {
		t.Fatal("token issued exactly at epoch must not be superseded")
	}
}
