package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/ratelimit/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Limiter: NewRateLimiter(ports.RateLimitConfigs{"contract": {MaxRequests: 2, WindowSeconds: 60}, "other-policy": {MaxRequests: 2, WindowSeconds: 60}}), Context: context.Background(), Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	})
}
