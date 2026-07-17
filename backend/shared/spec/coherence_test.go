package spec_test

// SCL ↔ Go バインディングの coherence test。
// 仕様核 (spec/scl.yaml) と Go 実装の双子定義が乖離していないことを検証する。

import (
	"slices"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestUserStatusPendingDeletionWireValue(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatal(err)
	}
	if got := s.ToWire("PendingDeletion"); got != string(idmdomain.UserStatusPendingDeletion) {
		t.Fatalf("SCL PendingDeletion wire=%q, Go UserStatusPendingDeletion=%q", got, idmdomain.UserStatusPendingDeletion)
	}
}

func TestMfaFactorTypeMatchesSCL(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	got, err := s.EnumWireValues("MfaFactorType")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		string(spec.MfaFactorTOTP),
		string(spec.MfaFactorWebAuthn),
		string(spec.MfaFactorHWK),
		string(spec.MfaFactorSWK),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SCL MfaFactorType=%v, Go=%v", got, want)
	}
}

func TestAgentStatusMatchesSCL(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	got, err := s.EnumWireValues("AgentStatus")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		string(idmdomain.AgentStatusActive),
		string(idmdomain.AgentStatusDisabled),
		string(idmdomain.AgentStatusKilled),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SCL AgentStatus=%v, Go=%v", got, want)
	}
}

func TestAgentKindMatchesSCL(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	got, err := s.EnumWireValues("AgentKind")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		string(idmdomain.AgentKindAutonomous),
		string(idmdomain.AgentKindSupervised),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("SCL AgentKind=%v, Go=%v", got, want)
	}
}

func TestGrantTypeTokenExchangeMatchesSCL(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	got, err := s.EnumWireValues("GrantType")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(got, string(spec.GrantTokenExchange)) {
		t.Fatalf("SCL GrantType=%v は Go の GrantTokenExchange=%q を含みません", got, spec.GrantTokenExchange)
	}
	if !spec.GrantTokenExchange.Valid() {
		t.Fatal("GrantTokenExchange.Valid() が false です")
	}
}

func TestStandardsAuthorizationAndFlowsLoadFromSCL(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	rfc9700, ok := s.Standards["RFC9700"]
	if !ok {
		t.Fatal("standards.RFC9700 is missing")
	}
	if len(rfc9700.Requirements) == 0 {
		t.Fatal("standards.RFC9700.requirements is empty")
	}
	if len(s.AuthorizationByContext["Authentication"].Policies) == 0 {
		t.Fatal("authentication authorization policies are missing")
	}
	if _, ok := s.Flows["Login"]; !ok {
		t.Fatal("flows.Login is missing")
	}
}

func TestCurrentSCLLoadsAllNormativeSections(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	sections := []struct {
		name string
		size int
	}{
		{"context_map", len(s.ContextMap)},
		{"standards", len(s.Standards)},
		{"vocabulary", len(s.Vocabulary)},
		{"models", len(s.Models)},
		{"interfaces", len(s.Interfaces)},
		{"states", len(s.States)},
		{"scenarios", len(s.Scenarios)},
		{"authorization contexts", len(s.AuthorizationByContext)},
		{"objectives", len(s.Objectives)},
		{"flows", len(s.Flows)},
	}
	for _, section := range sections {
		if section.size == 0 {
			t.Errorf("%s was not loaded", section.name)
		}
	}
	if len(s.Annotations) != 0 {
		t.Errorf("top-level annotations must remain non-normative and are currently unexpected: %v", s.Annotations)
	}
}

func TestCurrentSCLIsInternallyCoherent(t *testing.T) {
	s, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	if err := s.ValidateCoherence(); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeSCLRejectsUnknownFields(t *testing.T) {
	_, err := spec.DecodeSCL([]byte(`
system: example
spec_version: "1.0"
unknown_section: {}
`))
	if err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}
