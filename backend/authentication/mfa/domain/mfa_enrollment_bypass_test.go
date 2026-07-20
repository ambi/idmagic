package domain_test

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/mfa/domain"
)

func TestEvaluateMfaEnrollment(t *testing.T) {
	start := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	now := start.Add(time.Minute)
	bypass := &domain.MfaEnrollmentBypass{
		ID: "bypass", TenantID: "tenant", UserID: "user", IssuedBy: "admin",
		IssuedAt: start, ExpiresAt: start.Add(10 * time.Minute),
	}
	tests := []struct {
		name   string
		now    time.Time
		allow  bool
		bypass *domain.MfaEnrollmentBypass
		want   domain.MfaEnrollmentDecision
	}{
		{name: "before enforcement", now: start.Add(-time.Second), allow: true, bypass: bypass, want: domain.MfaEnrollmentNotRequired},
		{name: "approved", now: now, allow: true, bypass: bypass, want: domain.MfaEnrollmentRequired},
		{name: "missing bypass", now: now, allow: true, want: domain.MfaEnrollmentDenied},
		{name: "disabled", now: now, allow: false, bypass: bypass, want: domain.MfaEnrollmentDenied},
		{name: "grace expired", now: start.Add(11 * time.Minute), allow: true, bypass: bypass, want: domain.MfaEnrollmentDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := domain.EvaluateMfaEnrollment(tt.now, &start, 10*time.Minute, tt.allow, tt.bypass)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
