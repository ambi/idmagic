package ports

import (
	"context"
	"time"

	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"
)

type MfaEnrollmentBypassRepository interface {
	Save(ctx context.Context, bypass *mfadomain.MfaEnrollmentBypass) error
	FindActive(ctx context.Context, tenantID, userID string, now time.Time) (*mfadomain.MfaEnrollmentBypass, error)
	ConsumeActive(ctx context.Context, tenantID, userID string, now time.Time) (*mfadomain.MfaEnrollmentBypass, error)
	RevokeActive(ctx context.Context, tenantID, userID string, now time.Time) (*mfadomain.MfaEnrollmentBypass, error)
	ExpireOpen(ctx context.Context, tenantID, userID string, now time.Time) (*mfadomain.MfaEnrollmentBypass, error)
}
