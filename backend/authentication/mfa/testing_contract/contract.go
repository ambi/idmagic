// Package testing_contract defines the shared MFA persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/mfa/domain"
	"github.com/ambi/idmagic/backend/authentication/mfa/ports"
)

type Fixture struct {
	Repository ports.MfaEnrollmentBypassRepository
	TenantA    string
	TenantB    string
	UserID     string
	IssuedBy   string
	BypassID   string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	for _, tc := range []struct {
		name  string
		act   func(context.Context, ports.MfaEnrollmentBypassRepository, Fixture) (*domain.MfaEnrollmentBypass, error)
		field func(*domain.MfaEnrollmentBypass) *time.Time
	}{
		{
			name: "consume active bypass once",
			act: func(ctx context.Context, repository ports.MfaEnrollmentBypassRepository, f Fixture) (*domain.MfaEnrollmentBypass, error) {
				return repository.ConsumeActive(ctx, f.TenantA, f.UserID, f.Now.Add(time.Second))
			},
			field: func(bypass *domain.MfaEnrollmentBypass) *time.Time { return bypass.ConsumedAt },
		},
		{
			name: "revoke active bypass once",
			act: func(ctx context.Context, repository ports.MfaEnrollmentBypassRepository, f Fixture) (*domain.MfaEnrollmentBypass, error) {
				return repository.RevokeActive(ctx, f.TenantA, f.UserID, f.Now.Add(time.Second))
			},
			field: func(bypass *domain.MfaEnrollmentBypass) *time.Time { return bypass.RevokedAt },
		},
		{
			name: "expire open bypass once",
			act: func(ctx context.Context, repository ports.MfaEnrollmentBypassRepository, f Fixture) (*domain.MfaEnrollmentBypass, error) {
				return repository.ExpireOpen(ctx, f.TenantA, f.UserID, f.Now.Add(2*time.Minute))
			},
			field: func(bypass *domain.MfaEnrollmentBypass) *time.Time { return bypass.ExpiredAt },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			ctx := context.Background()
			bypass := &domain.MfaEnrollmentBypass{
				ID: f.BypassID, TenantID: f.TenantA, UserID: f.UserID, IssuedBy: f.IssuedBy,
				IssuedAt: f.Now, ExpiresAt: f.Now.Add(time.Minute),
			}
			if err := f.Repository.Save(ctx, bypass); err != nil {
				t.Fatalf("Save: %v", err)
			}
			if leaked, err := f.Repository.FindActive(ctx, f.TenantB, f.UserID, f.Now); err != nil || leaked != nil {
				t.Fatalf("other tenant FindActive = (%+v, %v)", leaked, err)
			}
			changed, err := tc.act(ctx, f.Repository, f)
			if err != nil || changed == nil || tc.field(changed) == nil {
				t.Fatalf("first transition = (%+v, %v)", changed, err)
			}
			again, err := tc.act(ctx, f.Repository, f)
			if err != nil || again != nil {
				t.Fatalf("second transition = (%+v, %v)", again, err)
			}
		})
	}
}
