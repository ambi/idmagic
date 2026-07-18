package ports

import (
	"context"
	"time"
)

// AuthnRequestReplayStore atomically reserves a tenant/SP/request ID before assertion issuance.
// Implementations must fail closed: an unavailable store returns an error, never a successful reservation.
type AuthnRequestReplayStore interface {
	RecordIfNew(ctx context.Context, tenantID, entityID, requestID string, ttl time.Duration, now time.Time) (bool, error)
}
