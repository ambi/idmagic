package valkey

import (
	"context"
	"time"

	sharedvalkey "github.com/ambi/idmagic/backend/shared/adapters/persistence/valkey"

	goredis "github.com/redis/go-redis/v9"
)

// AuthnRequestReplayStore uses Redis SET NX with TTL so concurrent issuers share one reservation.
type AuthnRequestReplayStore struct{ Client goredis.Cmdable }

func (s *AuthnRequestReplayStore) RecordIfNew(ctx context.Context, tenantID, entityID, requestID string, ttl time.Duration, _ time.Time) (bool, error) {
	key := sharedvalkey.TenantKey(ctx, "saml:authnrequest-replay:"+tenantID+":"+entityID+":"+requestID)
	return s.Client.SetNX(ctx, key, "1", ttl).Result()
}
