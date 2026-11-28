package usecases

// 管理者向け Consent 操作 (List / Get / Revoke)。
// SCL OAuth2 モジュールの admin インターフェース群:
// ListAdminConsents / GetAdminConsent / RevokeAdminConsent。

import (
	"context"
	"time"

	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ErrConsentNotFound は公開パッケージの同名のエラーである。
var ErrConsentNotFound = consentports.ErrConsentNotFound

type ConsentDeps struct {
	ConsentRepo consentports.ConsentRepository
	Emit        func(spec.DomainEvent)
	// QuotaRepo enforces the tenant's consents Hard Quota (wi-160): GrantConsent
	// and GrantAgentDelegation reserve a slot, and revocation frees it; nil skips
	// enforcement.
	QuotaRepo tenantports.QuotaRepository
}

// ListConsents returns up to limit+1 consents after the given keyset
// (wi-159) — callers pass limit+1 to detect whether a next page
// exists, then trim to limit before responding.
func ListConsents(ctx context.Context, deps ConsentDeps, afterUserID, afterClientID string, limit int) ([]*oauthdomain.Consent, error) {
	return deps.ConsentRepo.ListPage(ctx, tenantports.TenantID(ctx), afterUserID, afterClientID, limit)
}

func ListConsentsBefore(ctx context.Context, deps ConsentDeps, beforeUserID, beforeClientID string, limit int) ([]*oauthdomain.Consent, error) {
	return deps.ConsentRepo.ListPageBefore(ctx, tenantports.TenantID(ctx), beforeUserID, beforeClientID, limit)
}

func GetConsent(
	ctx context.Context,
	deps ConsentDeps,
	sub, clientID string,
) (*oauthdomain.Consent, error) {
	consent, err := deps.ConsentRepo.Find(ctx, tenantports.TenantID(ctx), sub, clientID)
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
	now = adminNow(now)
	consent, err := GetConsent(ctx, deps, sub, clientID)
	if err != nil {
		return err
	}
	tenantID := tenantports.TenantID(ctx)
	if err := deps.ConsentRepo.Revoke(ctx, tenantID, sub, clientID, now); err != nil {
		return err
	}
	// Revoke is idempotent (SCL); only free the quota slot the first time a
	// consent that is Granted at now transitions to Revoked, so repeat revokes
	// and revokes of expired consents don't under-count usage (wi-160).
	if deps.QuotaRepo != nil && consent.StateAt(now) == oauthdomain.ConsentGranted {
		if err := deps.QuotaRepo.Decrement(ctx, tenantID, tenancydomain.ResourceConsents, 1); err != nil {
			return err
		}
	}
	emit(deps.Emit, &oauthdomain.ConsentRevokedEvent{
		At: now, TenantID: tenantID, ActorUserID: actorUserID, UserID: sub, ClientID: clientID,
	})
	return nil
}
