package domain_test

import (
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"
)

func TestStepUpSatisfiedRecencyWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		ctx  *authdomain.AuthenticationContext
		want bool
	}{
		{"fresh auth", &authdomain.AuthenticationContext{AuthTime: now.Add(-time.Minute).Unix()}, true},
		{"stale auth", &authdomain.AuthenticationContext{AuthTime: now.Add(-10 * time.Minute).Unix()}, false},
		{
			"stale auth but recent step-up",
			&authdomain.AuthenticationContext{
				AuthTime: now.Add(-10 * time.Minute).Unix(), StepUpAt: now.Add(-2 * time.Minute).Unix(),
			},
			true,
		},
		{"boundary 300s", &authdomain.AuthenticationContext{AuthTime: now.Add(-300 * time.Second).Unix()}, true},
		{"just over 300s", &authdomain.AuthenticationContext{AuthTime: now.Add(-301 * time.Second).Unix()}, false},
		{"pending never", &authdomain.AuthenticationContext{AuthTime: now.Unix(), AuthenticationPending: true}, false},
		{"zero times", &authdomain.AuthenticationContext{}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		if got := mfadomain.StepUpSatisfied(tc.ctx, now); got != tc.want {
			t.Errorf("%s: StepUpSatisfied = %v, want %v", tc.name, got, tc.want)
		}
	}
}
