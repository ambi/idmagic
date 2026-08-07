package domain_test

import (
	"errors"
	"testing"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

func validBinding() workloaddomain.AgentWorkloadBinding {
	now := time.Now().UTC()
	return workloaddomain.AgentWorkloadBinding{
		ID:             "binding_1",
		TenantID:       "tenant-a",
		TrustBundleID:  "bundle_1",
		SubjectPattern: "spiffe://example.org/ns/prod/sa/*",
		AgentID:        "agent_1",
		Status:         workloaddomain.AgentWorkloadBindingStatusEnabled,
		CreatedAt:      now,
	}
}

// TestBindingValidateHappyAndFailure — AgentWorkloadBinding constraints
// (spec/contexts/workloadidentity.yaml)。
func TestBindingValidateHappyAndFailure(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*workloaddomain.AgentWorkloadBinding)
		wantErr bool
	}{
		{"ok", func(*workloaddomain.AgentWorkloadBinding) {}, false},
		{"missing id", func(b *workloaddomain.AgentWorkloadBinding) { b.ID = "" }, true},
		{"missing tenant", func(b *workloaddomain.AgentWorkloadBinding) { b.TenantID = "" }, true},
		{"missing trust_bundle_id", func(b *workloaddomain.AgentWorkloadBinding) { b.TrustBundleID = "" }, true},
		{"missing subject_pattern", func(b *workloaddomain.AgentWorkloadBinding) { b.SubjectPattern = "" }, true},
		{"missing agent_id", func(b *workloaddomain.AgentWorkloadBinding) { b.AgentID = "" }, true},
		{"bad status", func(b *workloaddomain.AgentWorkloadBinding) {
			b.Status = workloaddomain.AgentWorkloadBindingStatus("x")
		}, true},
		{"malformed glob pattern", func(b *workloaddomain.AgentWorkloadBinding) { b.SubjectPattern = "[unclosed" }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := validBinding()
			c.mutate(&b)
			err := b.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
		})
	}
}

func TestBindingIsEnabled(t *testing.T) {
	b := validBinding()
	if !b.IsEnabled() {
		t.Fatal("enabled binding must report IsEnabled() == true")
	}
	b.Status = workloaddomain.AgentWorkloadBindingStatusDisabled
	if b.IsEnabled() {
		t.Fatal("disabled binding must report IsEnabled() == false")
	}
}

// TestMatchAgent_ExactMatch — scenario
// `登録済みtrustbundle経由でワークロードトークンをAgent資格情報に交換できる`。
func TestMatchAgent_ExactMatch(t *testing.T) {
	b := validBinding()
	match, err := workloaddomain.MatchAgent([]workloaddomain.AgentWorkloadBinding{b}, "spiffe://example.org/ns/prod/sa/worker-1")
	if err != nil {
		t.Fatalf("MatchAgent: %v", err)
	}
	if match == nil || match.ID != b.ID {
		t.Fatalf("MatchAgent = %+v, want binding %q", match, b.ID)
	}
}

// TestMatchAgent_NoMatch — pattern に一致する subject が無ければ fail-closed で拒否する。
func TestMatchAgent_NoMatch(t *testing.T) {
	b := validBinding()
	_, err := workloaddomain.MatchAgent([]workloaddomain.AgentWorkloadBinding{b}, "spiffe://example.org/ns/staging/sa/worker-1")
	if !errors.Is(err, workloaddomain.ErrNoBindingMatch) {
		t.Fatalf("MatchAgent err = %v, want ErrNoBindingMatch", err)
	}
}

// TestMatchAgent_AmbiguousMatch — scenario `複数bindingに曖昧にマッチするsubjectは拒否される`
// (binding collision, fail-closed)。
func TestMatchAgent_AmbiguousMatch(t *testing.T) {
	a := validBinding()
	a.ID = "binding_a"
	a.SubjectPattern = "spiffe://example.org/ns/prod/sa/*"
	a.AgentID = "agent_a"

	b := validBinding()
	b.ID = "binding_b"
	b.SubjectPattern = "spiffe://example.org/ns/prod/sa/worker-*"
	b.AgentID = "agent_b"

	_, err := workloaddomain.MatchAgent([]workloaddomain.AgentWorkloadBinding{a, b}, "spiffe://example.org/ns/prod/sa/worker-1")
	if !errors.Is(err, workloaddomain.ErrAmbiguousBindingMatch) {
		t.Fatalf("MatchAgent err = %v, want ErrAmbiguousBindingMatch", err)
	}
}

// TestMatchAgent_ExcludesDisabled — disabled binding lifecycle: 無効化された binding は
// マッチ対象から除外される (ambiguity にも数えない)。
func TestMatchAgent_ExcludesDisabled(t *testing.T) {
	enabled := validBinding()
	enabled.ID = "binding_enabled"

	disabled := validBinding()
	disabled.ID = "binding_disabled"
	disabled.SubjectPattern = "spiffe://example.org/ns/prod/sa/worker-*"
	disabled.Status = workloaddomain.AgentWorkloadBindingStatusDisabled

	match, err := workloaddomain.MatchAgent([]workloaddomain.AgentWorkloadBinding{enabled, disabled}, "spiffe://example.org/ns/prod/sa/worker-1")
	if err != nil {
		t.Fatalf("MatchAgent: %v", err)
	}
	if match == nil || match.ID != enabled.ID {
		t.Fatalf("MatchAgent = %+v, want enabled binding only", match)
	}
}

func TestNewAgentWorkloadBindingID(t *testing.T) {
	id, err := workloaddomain.NewAgentWorkloadBindingID()
	if err != nil {
		t.Fatalf("NewAgentWorkloadBindingID: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("NewAgentWorkloadBindingID = %q, want UUID", id)
	}
}
