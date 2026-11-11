package ports

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/wsfederation/domain"
)

// ErrEntraObservationOutdated は、保存済みの記録より前に取得した観測で記録を上書きしようとしたことを表す。
var ErrEntraObservationOutdated = errors.New("entra federation: the observation is older than the stored one")

// EntraFederationObservationRepository は、テナントとドメインごとに最新の一件の観測記録を保存する。
// ドメインは小文字で渡す。
type EntraFederationObservationRepository interface {
	// Find は記録を返す。なければ (nil, nil)。
	Find(ctx context.Context, tenantID, domain string) (*domain.EntraFederationObservationRecord, error)
	// ListAll はテナントの記録をドメインの順で返す。
	ListAll(ctx context.Context, tenantID string) ([]*domain.EntraFederationObservationRecord, error)
	// Save は記録を置き換える。保存済みの記録の取得時刻より前の観測なら、置き換えずに
	// ErrEntraObservationOutdated を返す。同じ取得時刻は置き換える。
	Save(ctx context.Context, record domain.EntraFederationObservationRecord) error
	// Delete は記録を消す（冪等）。
	Delete(ctx context.Context, tenantID, domain string) error
}
