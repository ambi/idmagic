package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

var (
	ErrMfaEnrollmentNotAllowed      = errors.New("MFA enrollment is not allowed")
	ErrMfaEnrollmentExpired         = errors.New("MFA enrollment expired")
	ErrMfaEnrollmentAlreadyComplete = errors.New("MFA enrollment already complete")
)

type MfaEnrollmentDeps struct {
	UserRepo               idmports.UserRepository
	MfaFactorRepo          ports.MfaFactorRepository
	WebAuthnCredentialRepo ports.WebAuthnCredentialRepository
	BypassRepo             ports.MfaEnrollmentBypassRepository
	Emit                   func(spec.DomainEvent)
}

func HasMfaEnrollment(ctx context.Context, deps MfaEnrollmentDeps, userID string) (bool, error) {
	factors, err := deps.MfaFactorRepo.ListBySub(ctx, userID)
	if err != nil {
		return false, err
	}
	if len(factors) > 0 {
		return true, nil
	}
	if deps.WebAuthnCredentialRepo == nil {
		return false, nil
	}
	credentials, err := deps.WebAuthnCredentialRepo.ListBySub(ctx, userID)
	return len(credentials) > 0, err
}

func IssueMfaEnrollmentBypass(ctx context.Context, deps MfaEnrollmentDeps, actorID, userID string, ttl time.Duration, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	if ttl < time.Minute || ttl > time.Hour || deps.BypassRepo == nil {
		return nil, ErrMfaEnrollmentNotAllowed
	}
	user, err := deps.UserRepo.FindBySub(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) || !user.IsActive() {
		return nil, ErrMfaEnrollmentNotAllowed
	}
	enrolled, err := HasMfaEnrollment(ctx, deps, userID)
	if err != nil {
		return nil, err
	}
	if enrolled {
		return nil, ErrMfaEnrollmentAlreadyComplete
	}
	if expired, err := deps.BypassRepo.ExpireOpen(ctx, user.TenantID, userID, now); err != nil {
		return nil, err
	} else if expired != nil {
		emitMfaEnrollmentEvent(deps.Emit, &domain.MfaEnrollmentBypassExpired{At: now, TenantID: user.TenantID, UserID: userID, BypassID: expired.ID})
	}
	if existing, err := deps.BypassRepo.RevokeActive(ctx, user.TenantID, userID, now); err != nil {
		return nil, err
	} else if existing != nil {
		emitMfaEnrollmentEvent(deps.Emit, &domain.MfaEnrollmentBypassRevoked{At: now, TenantID: user.TenantID, ActorUserID: actorID, UserID: userID, BypassID: existing.ID})
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	bypass := &domain.MfaEnrollmentBypass{ID: id, TenantID: user.TenantID, UserID: userID, IssuedBy: actorID, IssuedAt: now, ExpiresAt: now.Add(ttl)}
	if err := deps.BypassRepo.Save(ctx, bypass); err != nil {
		return nil, err
	}
	emitMfaEnrollmentEvent(deps.Emit, &domain.MfaEnrollmentBypassIssued{At: now, TenantID: user.TenantID, ActorUserID: actorID, UserID: userID, BypassID: id, ExpiresAt: bypass.ExpiresAt})
	return bypass, nil
}

func RevokeMfaEnrollmentBypass(ctx context.Context, deps MfaEnrollmentDeps, actorID, userID string, now time.Time) error {
	if deps.BypassRepo == nil {
		return nil
	}
	bypass, err := deps.BypassRepo.RevokeActive(ctx, tenancy.TenantID(ctx), userID, now)
	if err != nil {
		return err
	}
	if bypass != nil {
		emitMfaEnrollmentEvent(deps.Emit, &domain.MfaEnrollmentBypassRevoked{At: now, TenantID: bypass.TenantID, ActorUserID: actorID, UserID: userID, BypassID: bypass.ID})
	}
	return nil
}

func emitMfaEnrollmentEvent(emit func(spec.DomainEvent), event spec.DomainEvent) {
	if emit != nil {
		emit(event)
	}
}
