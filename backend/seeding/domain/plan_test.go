package domain

import "testing"

func TestRequestValidateRejectsUnsafeOrAmbiguousRequests(t *testing.T) {
	tests := []Request{
		{Environment: EnvironmentProduction, Profile: ProfileDevelopment, Mode: ModeDryRun},
		{Environment: EnvironmentDevelopment, Profile: ProfileBootstrap, Mode: ModeApply, Count: 1},
		{Environment: EnvironmentTest, Profile: ProfilePerformance, Mode: ModeApply, Count: -1},
		{Environment: EnvironmentProduction, Profile: ProfileBootstrap, Mode: ModeApply},
		{Environment: EnvironmentDevelopment, Profile: ProfilePerformance, Mode: ModeApply, Count: DefaultPerformanceCountLimit + 1},
		{Environment: EnvironmentDevelopment, Profile: ProfileTest, Mode: ModeApply},
		{Environment: EnvironmentDevelopment, Profile: ProfileBootstrap, Mode: ModeApply, BatchSize: MaximumBatchSize + 1},
	}
	for _, request := range tests {
		if err := request.Validate(); err == nil {
			t.Errorf("Validate(%+v) succeeded, want error", request)
		}
	}
}

func TestRequestEffectiveBatchSize(t *testing.T) {
	if got := (Request{}).EffectiveBatchSize(); got != DefaultBatchSize {
		t.Fatalf("default batch size = %d, want %d", got, DefaultBatchSize)
	}
	if got := (Request{BatchSize: 17}).EffectiveBatchSize(); got != 17 {
		t.Fatalf("configured batch size = %d, want 17", got)
	}
}

func TestRequestValidateAcceptsProductionBootstrapAndPerformanceCount(t *testing.T) {
	for _, request := range []Request{
		{Environment: EnvironmentProduction, Profile: ProfileBootstrap, Mode: ModeDryRun, FirstPartyRedirectURIs: []string{"https://id.example/callback"}},
		{Environment: EnvironmentDevelopment, Profile: ProfilePerformance, Mode: ModeApply, Count: 1_000},
		{Environment: EnvironmentDevelopment, Profile: ProfilePerformance, Mode: ModeApply, Count: DefaultPerformanceCountLimit + 1, AllowLarge: true},
	} {
		if err := request.Validate(); err != nil {
			t.Errorf("Validate(%+v) error = %v", request, err)
		}
	}
}

func TestPlanCount(t *testing.T) {
	plan := Plan{Operations: []Operation{{Kind: OperationCreate}, {Kind: OperationNoop}, {Kind: OperationCreate}}}
	if got := plan.Count(OperationCreate); got != 2 {
		t.Fatalf("create count = %d, want 2", got)
	}
}
