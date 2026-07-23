package usecases

// 管理者向け Consent 操作 (List / Get / Revoke)。
// SCL OAuth2 bounded context の admin インターフェース群:
// ListAdminConsents / GetAdminConsent / RevokeAdminConsent。

import (
	"context"
	"errors"
	"time"

	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

var ErrConsentNotFound = errors.New("consent not found")

type ConsentDeps struct {
	ConsentRepo consentports.ConsentRepository
	Emit        func(spec.DomainEvent)
	// QuotaRepo frees the tenant's consents Hard Quota slot on revoke
	// (wi-160, ADR-134). The increment side lives in the /consent HTTP
	// handler (authorize_consent.go), which is where new consents are
	// created; nil skips enforcement.
	QuotaRepo tenantports.QuotaRepository
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
	consent, err := GetConsent(ctx, deps, sub, clientID)
	if err != nil {
		return err
	}
	tenantID := tenancy.TenantID(ctx)
	if err := deps.ConsentRepo.Revoke(ctx, tenantID, sub, clientID); err != nil {
		return err
	}
	// Revoke is idempotent (SCL); only free the quota slot the first time a
	// Granted consent transitions to Revoked, so repeat revokes don't
	// under-count usage (wi-160, ADR-134).
	if deps.QuotaRepo != nil && consent.State == domain.ConsentGranted {
		if err := tenancyusecases.DecrementQuota(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceConsents, 1); err != nil {
			return err
		}
	}
	emit(deps.Emit, &domain.ConsentRevokedEvent{
		At: adminNow(now), TenantID: tenantID, ActorUserID: actorUserID, UserID: sub, ClientID: clientID,
	})
	return nil
}
