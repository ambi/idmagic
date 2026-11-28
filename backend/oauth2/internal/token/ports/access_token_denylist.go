package ports

import (
	"context"
	"time"
)

type AccessTokenDenylist interface {
	Add(ctx context.Context, jti string, expiresAt time.Time) error
	// IsRevoked は jti が失効済みで、その記録が now で期限内かを返す。
	IsRevoked(ctx context.Context, jti string, now time.Time) (bool, error)
}
