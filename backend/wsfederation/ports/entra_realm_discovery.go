package ports

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/wsfederation/domain"
)

// ErrEntraRealmDiscoveryDisabled は、運用者がレルム探索を無効にしたことを表す。
var ErrEntraRealmDiscoveryDisabled = errors.New("entra federation: realm discovery is disabled")

// EntraRealmDiscovery は、Microsoft のサインインの画面がドメインの利用者をどこへ送るかを問い合わせる。
// 通信の失敗、無効化、読めない応答はエラーで返し、呼び出し側は確かめられないものとして扱う。
type EntraRealmDiscovery interface {
	Discover(ctx context.Context, domain string) (domain.EntraDomainRealm, error)
}
