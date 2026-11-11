// Package ports は WsFederation モジュールの永続境界 (port) を定義する。
package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/wsfederation/domain"
)

// WsFedRelyingPartyRepository は WS-Federation relying party 登録の永続境界 (wi-61)。
// wtrealm で識別し、許可 wreply・audience・claim policy を保持する。
type WsFedRelyingPartyRepository interface {
	// FindByWtrealm は wtrealm に一致する RP を返す。存在しなければ (nil, nil)。
	FindByWtrealm(ctx context.Context, tenantID, wtrealm string) (*domain.WsFedRelyingParty, error)
	// ListAll はテナント内の RP を wtrealm 昇順で返す。
	ListAll(ctx context.Context, tenantID string) ([]*domain.WsFedRelyingParty, error)
	// Save は RP を upsert する。
	Save(ctx context.Context, rp *domain.WsFedRelyingParty) error
	// Delete は wtrealm に一致する RP を削除する (冪等)。
	Delete(ctx context.Context, tenantID, wtrealm string) error
	// Update は、wtrealm に一致する RP を、ほかの書き手を待たせて読み直し、change の結果を当てる。
	// RP がなければ change に nil を渡す。change がエラーを返せば何も変えずにそのエラーを返す。
	Update(ctx context.Context, tenantID, wtrealm string, change RelyingPartyChange) error
}

// RelyingPartyChange は、読み直した最新の RP への変更である。保存するなら rp を変えて RelyingPartySaved を返す。
type RelyingPartyChange func(rp *domain.WsFedRelyingParty) (RelyingPartyOutcome, error)

// RelyingPartyOutcome は、Update が RP に当てる結果である。
type RelyingPartyOutcome int

const (
	// RelyingPartyUnchanged は RP を変えない。
	RelyingPartyUnchanged RelyingPartyOutcome = iota
	// RelyingPartySaved は、change が変えた RP を保存する。
	RelyingPartySaved
	// RelyingPartyDeleted は RP を削除する。
	RelyingPartyDeleted
)
