package domain

import (
	"errors"
	"testing"
)

func TestSamlIdentityProviderProfileValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		profile    SamlIdentityProviderProfile
		boundCount int
		wantErr    error
	}{
		{
			name: "default shared profile",
			profile: SamlIdentityProviderProfile{
				TenantID: "tenant-a", ProfileID: DefaultIDPProfileID, Name: "Default", Mode: IDPProfileModeShared, IsDefault: true,
			},
		},
		{
			name: "additional shared profile",
			profile: SamlIdentityProviderProfile{
				TenantID: "tenant-a", ProfileID: "partners", Name: "Partners", Mode: IDPProfileModeShared,
			},
			boundCount: 3,
		},
		{
			name: "dedicated profile with one binding",
			profile: SamlIdentityProviderProfile{
				TenantID: "tenant-a", ProfileID: "app-a", Name: "App A", Mode: IDPProfileModeDedicated,
			},
			boundCount: 1,
		},
		{
			name: "dedicated profile rejects a second binding",
			profile: SamlIdentityProviderProfile{
				TenantID: "tenant-a", ProfileID: "app-a", Name: "App A", Mode: IDPProfileModeDedicated,
			},
			boundCount: 2,
			wantErr:    ErrDedicatedIDPProfileCardinality,
		},
		{
			name: "default profile must be shared",
			profile: SamlIdentityProviderProfile{
				TenantID: "tenant-a", ProfileID: DefaultIDPProfileID, Name: "Default", Mode: IDPProfileModeDedicated, IsDefault: true,
			},
			wantErr: ErrInvalidIDPProfile,
		},
		{
			name: "additional profile cannot claim default identity",
			profile: SamlIdentityProviderProfile{
				TenantID: "tenant-a", ProfileID: "other", Name: "Other", Mode: IDPProfileModeShared, IsDefault: true,
			},
			wantErr: ErrInvalidIDPProfile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.profile.Validate(tt.boundCount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate(%d) error = %v, want %v", tt.boundCount, err, tt.wantErr)
			}
		})
	}
}

func TestSamlServiceProviderMatchesOnlyAssignedProfile(t *testing.T) {
	t.Parallel()

	sp := SamlServiceProvider{IDPProfileID: "profile-a"}
	if !sp.MatchesIDPProfile("profile-a") {
		t.Fatal("assigned profile must match")
	}
	if sp.MatchesIDPProfile("profile-b") {
		t.Fatal("cross-profile route must not match")
	}

	defaultSP := SamlServiceProvider{}
	if !defaultSP.MatchesIDPProfile(DefaultIDPProfileID) {
		t.Fatal("the model default must bind an unspecified Go value to the default profile")
	}
}
