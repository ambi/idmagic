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

func TestManifestValidateRejectsInvalidShapeAndDuplicateLogicalKeys(t *testing.T) {
	valid := Manifest{
		SchemaVersion: CurrentManifestSchemaVersion,
		Profile:       ProfileDevelopment,
		Resources: []Resource{
			{Kind: ResourceKindFirstPartyClients, LogicalKey: "first-party-portals", Clients: []FirstPartyClientSeed{{ID: "client", Name: "Client", Scope: "openid"}}},
			{Kind: ResourceKindDevelopmentDemo, LogicalKey: "development-demo", Demo: &DevelopmentDemoSeed{ClientID: "demo", Users: []DemoUserSeed{{ID: "user"}}}, Secrets: map[string]SecretReference{
				"client_secret": {Provider: SecretProviderEnv, Locator: "DEMO_CLIENT_SECRET", Version: "v1"},
				"user_password": {Provider: SecretProviderFile, Locator: "demo-password", Version: "v1"},
			}},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid manifest error = %v", err)
	}

	tests := []Manifest{
		{SchemaVersion: "2", Profile: ProfileDevelopment},
		{SchemaVersion: CurrentManifestSchemaVersion, Profile: Profile("unknown")},
		{SchemaVersion: CurrentManifestSchemaVersion, Profile: ProfileDevelopment, Resources: []Resource{{Kind: ResourceKind("unknown"), LogicalKey: "x"}}},
		{SchemaVersion: CurrentManifestSchemaVersion, Profile: ProfileDevelopment, Resources: []Resource{{Kind: ResourceKindFirstPartyClients, LogicalKey: "same"}, {Kind: ResourceKindDevelopmentDemo, LogicalKey: "same"}}},
		{SchemaVersion: CurrentManifestSchemaVersion, Profile: ProfileDevelopment, Resources: []Resource{{Kind: ResourceKindDevelopmentDemo, LogicalKey: "demo", Secrets: map[string]SecretReference{"password": {Provider: SecretProviderEnv, Locator: "", Version: "v1"}}}}},
	}
	for _, manifest := range tests {
		if err := manifest.Validate(); err == nil {
			t.Errorf("Validate(%+v) succeeded, want error", manifest)
		}
	}
}

func TestManifestValidateForRequestEnforcesProfileAndProductionSecretPolicy(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: CurrentManifestSchemaVersion,
		Profile:       ProfileBootstrap,
		Resources: []Resource{{
			Kind:       ResourceKindFirstPartyClients,
			LogicalKey: "first-party-portals",
			Clients:    []FirstPartyClientSeed{{ID: "client", Name: "Client", Scope: "openid"}},
			Secrets: map[string]SecretReference{
				"example": {Provider: SecretProviderEnv, Locator: "SECRET", Version: "v1"},
			},
		}},
	}
	if err := manifest.ValidateForRequest(Request{Environment: EnvironmentDevelopment, Profile: ProfileBootstrap, Mode: ModeDryRun}); err != nil {
		t.Fatalf("development validation error = %v", err)
	}
	if err := manifest.ValidateForRequest(Request{Environment: EnvironmentDevelopment, Profile: ProfileTest, Mode: ModeDryRun}); err == nil {
		t.Fatal("profile mismatch succeeded, want error")
	}
	if err := manifest.ValidateForRequest(Request{Environment: EnvironmentProduction, Profile: ProfileBootstrap, Mode: ModeDryRun, FirstPartyRedirectURIs: []string{"https://id.example/callback"}}); err == nil {
		t.Fatal("production env provider succeeded, want error")
	}
}
