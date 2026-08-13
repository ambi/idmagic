package domain

import (
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestResolvePasswordPolicy(t *testing.T) {
	defaults := PasswordPolicySnapshot{MinLength: 12, MaxLength: 128, HistoryDepth: 5}

	t.Run("nil tenant uses global defaults", func(t *testing.T) {
		snap := ResolvePasswordPolicy(nil, defaults)
		if snap.MinLength == 0 || snap.MaxLength == 0 || snap.HistoryDepth == 0 {
			t.Fatalf("snapshot zero values: %+v", snap)
		}
	})

	t.Run("override only specified fields", func(t *testing.T) {
		minLength := 16
		tenant := &tenancydomain.Tenant{
			PasswordPolicyOverride: &tenancydomain.PasswordPolicyOverride{MinLength: &minLength},
		}
		base := ResolvePasswordPolicy(nil, defaults)
		snap := ResolvePasswordPolicy(tenant, defaults)
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
		tenant := &tenancydomain.Tenant{
			PasswordPolicyOverride: &tenancydomain.PasswordPolicyOverride{
				MinLength:    &zero,
				MaxLength:    &neg,
				HistoryDepth: &zero,
			},
		}
		base := ResolvePasswordPolicy(nil, defaults)
		snap := ResolvePasswordPolicy(tenant, defaults)
		if snap != base {
			t.Fatalf("guard rail breached: %+v vs base %+v", snap, base)
		}
	})

	// REQ-AUTHENTICATION-024: expiry is a tenant opt-in, and the snapshot also
	// carries the policy update time the evaluation starts from.
	t.Run("max age and policy update time are resolved from the tenant", func(t *testing.T) {
		maxAge := 90
		updatedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		tenant := &tenancydomain.Tenant{
			PasswordPolicyOverride:  &tenancydomain.PasswordPolicyOverride{MaxAgeDays: &maxAge},
			PasswordPolicyUpdatedAt: &updatedAt,
		}
		snap := ResolvePasswordPolicy(tenant, defaults)
		if snap.MaxAgeDays != 90 {
			t.Fatalf("MaxAgeDays override not applied: %d", snap.MaxAgeDays)
		}
		if snap.PolicyUpdatedAt == nil || !snap.PolicyUpdatedAt.Equal(updatedAt) {
			t.Fatalf("PolicyUpdatedAt not carried: %+v", snap.PolicyUpdatedAt)
		}
	})

	t.Run("max age defaults to disabled", func(t *testing.T) {
		if snap := ResolvePasswordPolicy(nil, defaults); snap.MaxAgeDays != 0 {
			t.Fatalf("expiry must be off by default: %d", snap.MaxAgeDays)
		}
	})
}

// REQ-AUTHENTICATION-024
func TestPasswordExpired(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	daysAgo := func(d int) *time.Time {
		t := now.AddDate(0, 0, -d)
		return &t
	}
	policyUpdated := daysAgo(365)

	tests := []struct {
		name              string
		snap              PasswordPolicySnapshot
		passwordChangedAt *time.Time
		want              bool
	}{
		{
			name:              "expiry disabled never expires",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 0, PolicyUpdatedAt: policyUpdated},
			passwordChangedAt: daysAgo(4000),
			want:              false,
		},
		{
			name:              "older than max age expires",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90, PolicyUpdatedAt: policyUpdated},
			passwordChangedAt: daysAgo(91),
			want:              true,
		},
		{
			name:              "within max age does not expire",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90, PolicyUpdatedAt: policyUpdated},
			passwordChangedAt: daysAgo(89),
			want:              false,
		},
		{
			name:              "exactly at the boundary does not expire yet",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90, PolicyUpdatedAt: policyUpdated},
			passwordChangedAt: daysAgo(90),
			want:              false,
		},
		{
			// A policy change must not expire everyone at once (grace).
			name:              "recent policy change grants a full window",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90, PolicyUpdatedAt: daysAgo(10)},
			passwordChangedAt: daysAgo(4000),
			want:              false,
		},
		{
			name:              "policy changed longer ago than max age stops granting grace",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90, PolicyUpdatedAt: daysAgo(91)},
			passwordChangedAt: nil,
			want:              true,
		},
		{
			// A user with no recorded change starts from the policy update time.
			name:              "never changed with a recent policy update does not expire",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90, PolicyUpdatedAt: daysAgo(10)},
			passwordChangedAt: nil,
			want:              false,
		},
		{
			// A tenant with no policy update time (no override) is measured from
			// password_changed_at alone.
			name:              "no policy update time falls back to the password change time",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90},
			passwordChangedAt: daysAgo(91),
			want:              true,
		},
		{
			name:              "no policy update time and no password change time never expires",
			snap:              PasswordPolicySnapshot{MaxAgeDays: 90},
			passwordChangedAt: nil,
			want:              false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PasswordExpired(tc.snap, tc.passwordChangedAt, now); got != tc.want {
				t.Fatalf("PasswordExpired = %v, want %v", got, tc.want)
			}
		})
	}
}
