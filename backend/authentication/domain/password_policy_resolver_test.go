package domain

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestResolvePasswordPolicy(t *testing.T) {
	scl, err := spec.LoadSCL()
	if err != nil {
		t.Fatalf("load scl: %v", err)
	}
	defaults := PasswordPolicySnapshot{MinLength: 12, MaxLength: 128, HistoryDepth: 5}

	t.Run("nil tenant uses SCL global", func(t *testing.T) {
		snap := ResolvePasswordPolicy(scl, nil, defaults)
		if snap.MinLength == 0 || snap.MaxLength == 0 || snap.HistoryDepth == 0 {
			t.Fatalf("snapshot zero values: %+v", snap)
		}
	})

	t.Run("override only specified fields", func(t *testing.T) {
		minLength := 16
		tenant := &spec.Tenant{
			PasswordPolicyOverride: &spec.PasswordPolicyOverride{MinLength: &minLength},
		}
		base := ResolvePasswordPolicy(scl, nil, defaults)
		snap := ResolvePasswordPolicy(scl, tenant, defaults)
		if snap.MinLength != 16 {
			t.Fatalf("MinLength override not applied: %d", snap.MinLength)
		}
		if snap.MaxLength != base.MaxLength || snap.HistoryDepth != base.HistoryDepth {
			t.Fatalf("non-overridden fields drifted: %+v vs base %+v", snap, base)
		}
	})

	t.Run("zero or negative override ignored", func(t *testing.T) {
		zero := 0
		neg := -1
		tenant := &spec.Tenant{
			PasswordPolicyOverride: &spec.PasswordPolicyOverride{
				MinLength:    &zero,
				MaxLength:    &neg,
				HistoryDepth: &zero,
			},
		}
		base := ResolvePasswordPolicy(scl, nil, defaults)
		snap := ResolvePasswordPolicy(scl, tenant, defaults)
		if snap != base {
			t.Fatalf("guard rail breached: %+v vs base %+v", snap, base)
		}
	})
}
