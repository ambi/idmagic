package domain_test

import (
	"testing"
	"time"

	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"
)

func TestEvaluateMfaEnrollment(t *testing.T) {
	start := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	now := start.Add(time.Minute)
	bypass := &mfadomain.MfaEnrollmentBypass{
		ID: "bypass", TenantID: "tenant", UserID: "user", IssuedBy: "admin",
		IssuedAt: start, ExpiresAt: start.Add(10 * time.Minute),
	}
	tests := []struct {
		name   string
		now    time.Time
		allow  bool
		bypass *mfadomain.MfaEnrollmentBypass
		want   mfadomain.MfaEnrollmentDecision
	}{
		{name: "before enforcement", now: start.Add(-time.Second), allow: true, bypass: bypass, want: mfadomain.MfaEnrollmentNotRequired},
		{name: "approved", now: now, allow: true, bypass: bypass, want: mfadomain.MfaEnrollmentRequired},
		{name: "missing bypass", now: now, allow: true, want: mfadomain.MfaEnrollmentDenied},
		{name: "disabled", now: now, allow: false, bypass: bypass, want: mfadomain.MfaEnrollmentDenied},
		{name: "grace expired", now: start.Add(11 * time.Minute), allow: true, bypass: bypass, want: mfadomain.MfaEnrollmentDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := mfadomain.EvaluateMfaEnrollment(tt.now, &start, 10*time.Minute, tt.allow, tt.bypass)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
