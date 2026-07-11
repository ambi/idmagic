package usecases

// 管理者向け Consent 操作 (List / Get / Revoke)。
// SCL OAuth2 bounded context の admin インターフェース群:
// ListAdminConsents / GetAdminConsent / RevokeAdminConsent。

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

var ErrConsentNotFound = errors.New("consent not found")

type ConsentDeps struct {
	ConsentRepo       oauthports.ConsentRepository
	Emit              func(spec.DomainEvent)
	TransactionalEmit func(spec.DomainEvent) error
}

func ListConsents(ctx context.Context, deps ConsentDeps) ([]*domain.Consent, error) {
	return deps.ConsentRepo.FindAll(ctx, tenancy.TenantID(ctx))
}

func GetConsent(
	ctx context.Context,
	deps ConsentDeps,
	sub, clientID string,
) (*domain.Consent, error) {
	consent, err := deps.ConsentRepo.Find(ctx, tenancy.TenantID(ctx), sub, clientID)
	if err != nil {
		return nil, err
	}
	if consent == nil {
		return nil, ErrConsentNotFound
	}
	return consent, nil
}

func RevokeConsent(
	ctx context.Context,
	deps ConsentDeps,
	actorUserID, sub, clientID string,
	now time.Time,
) error {
	if _, err := GetConsent(ctx, deps, sub, clientID); err != nil {
		return err
	}
	if err := deps.ConsentRepo.Revoke(ctx, tenancy.TenantID(ctx), sub, clientID); err != nil {
		return err
	}
	if err := emitTransactional(deps.TransactionalEmit, deps.Emit, &domain.ConsentRevokedEvent{
		At: adminNow(now), TenantID: tenancy.TenantID(ctx), ActorUserID: actorUserID, UserID: sub, ClientID: clientID,
	}); err != nil {
		return err
	}
	return nil
}
