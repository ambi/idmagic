package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
)

type MfaEnrollmentBypassRepository interface {
	Save(ctx context.Context, bypass *domain.MfaEnrollmentBypass) error
	FindActive(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error)
	ConsumeActive(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error)
	RevokeActive(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error)
	ExpireOpen(ctx context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error)
}
