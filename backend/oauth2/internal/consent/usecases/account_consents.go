package usecases

// エンドユーザー自身の Consent 操作 (self-service, wi-21)。
// SCL OAuth2 モジュールの self インターフェース ListMyConsents / RevokeMyConsent。
// 取り消しは admin の RevokeConsent と同じく Consent レコードの論理撤回 + ConsentRevoked
// イベントで、actor.sub == target.sub に固定する。

import (
	"context"
	"time"

	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ListConsentsForSub は指定 sub の active な (granted) Consent のみを返す。
// 接続済みアプリ一覧の用途で、revoked / expired は除外する。
func ListConsentsForSub(ctx context.Context, deps ConsentDeps, sub string) ([]*domain.Consent, error) {
	all, err := deps.ConsentRepo.FindAll(ctx, tenantports.TenantID(ctx))
	if err != nil {
		return nil, err
	}
	mine := make([]*domain.Consent, 0)
	for _, consent := range all {
		if consent.UserID == sub && consent.State == domain.ConsentGranted {
			mine = append(mine, consent)
		}
	}
	return mine, nil
}

// AccountConsents は consentports.AccountConsents を実装する。本人の撤回も管理者の撤回と
// 同じく、Granted からの遷移で consents のクォータの枠を空ける。
type AccountConsents struct {
	ConsentRepo consentports.ConsentRepository
	Emit        func(spec.DomainEvent)
	QuotaRepo   tenantports.QuotaRepository
}

var _ consentports.AccountConsents = AccountConsents{}

func (a AccountConsents) deps() ConsentDeps {
	return ConsentDeps(a)
}

func (a AccountConsents) ListGranted(ctx context.Context, sub string) ([]*domain.Consent, error) {
	return ListConsentsForSub(ctx, a.deps(), sub)
}

func (a AccountConsents) Revoke(ctx context.Context, sub, clientID string, now time.Time) error {
	return RevokeConsent(ctx, a.deps(), sub, sub, clientID, now)
}
