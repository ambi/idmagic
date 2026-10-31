// リプレイ防止 (DPoP / private_key_jwt assertion)。
package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

type DpopReplayStore interface {
	RecordIfNew(ctx context.Context, jti string, windowSeconds int, now time.Time) (bool, error)
}

type ClientAssertionReplayStore interface {
	RecordIfNew(ctx context.Context, jti string, windowSeconds int, now time.Time) (bool, error)
}

// TokenExchangeGrantStore は評価が発行した交換の許可を持つ (REQ-OAUTH2-079)。
type TokenExchangeGrantStore interface {
	Save(ctx context.Context, grantHash string, grant domain.TokenExchangeGrant) error
	// Claim は全項目が一致する期限内の許可を原子的に消費する。一致しない要求には消費させない。
	Claim(ctx context.Context, grantHash string, grant domain.TokenExchangeGrant, now time.Time) (bool, error)
}
