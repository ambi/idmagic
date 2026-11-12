package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ConsentLifetime は認可の画面で許可した同意の有効期間である。
const ConsentLifetime = 365 * 24 * time.Hour

// ConsentGrant は、認可の画面で利用者がクライアントへ許可した同意の入力である。
type ConsentGrant struct {
	UserID               string
	ClientID             string
	Scopes               []string
	AuthorizationDetails []spec.AuthorizationDetail
}

// GrantConsent は、利用者がクライアントへ許可した同意を保存する。同じ利用者とクライアントの同意は
// 置き換える。新しい同意と撤回済みの同意の再付与だけが consents の使用量を一つ使い、
// 保存に失敗した付与の分は戻す（REQ-TENANCY-050）。
func GrantConsent(ctx context.Context, deps ConsentDeps, tenantID string, in ConsentGrant, now time.Time) (*domain.Consent, error) {
	var reservation *tenantports.QuotaReservation
	if deps.QuotaRepo != nil {
		existing, err := deps.ConsentRepo.Find(ctx, tenantID, in.UserID, in.ClientID)
		if err != nil {
			return nil, err
		}
		if consumesConsentQuota(existing) {
			reservation, err = tenantports.ReserveQuota(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceConsents, 1)
			if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok {
				emit(deps.Emit, &tenancydomain.QuotaExceeded{At: now, TenantID: tenantID, Resource: qErr.Resource, HardLimit: true})
			}
			if err != nil {
				return nil, err
			}
			defer reservation.Release(ctx)
		}
	}
	consent := &domain.Consent{
		UserID: in.UserID, ClientID: in.ClientID,
		Scopes: in.Scopes, State: domain.ConsentGranted,
		GrantedAt: now, ExpiresAt: now.Add(ConsentLifetime),
		AuthorizationDetails: in.AuthorizationDetails,
	}
	if err := deps.ConsentRepo.Save(ctx, tenantID, consent); err != nil {
		return nil, err
	}
	reservation.Keep()
	emit(deps.Emit, &domain.ConsentGrantedEvent{At: now, TenantID: tenantID, UserID: in.UserID, ClientID: in.ClientID, Scopes: in.Scopes})
	if len(in.AuthorizationDetails) > 0 {
		emit(deps.Emit, &domain.AuthorizationDetailsConsented{
			At: now, TenantID: tenantID, UserID: in.UserID, ClientID: in.ClientID,
			DetailTypes: domain.DetailTypes(in.AuthorizationDetails),
		})
	}
	return consent, nil
}

// consumesConsentQuota は、同意の付与が consents の使用量を一つ使うかを返す。保存は
// (テナント, 利用者, クライアント) の upsert なので、新しい同意と撤回済みの同意の再付与だけが使い、
// 付与済みの同意のスコープの変更は二重に数えない（wi-160）。期限切れの行は枠を持ったままなので使わない。
func consumesConsentQuota(existing *domain.Consent) bool {
	return existing == nil || existing.State == domain.ConsentRevoked
}
