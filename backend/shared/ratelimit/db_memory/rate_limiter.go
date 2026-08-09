package db_memory

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
)

type counter struct {
	count     int
	expiresAt time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	configs  rlports.RateLimitConfigs
	counters map[string]counter
}

func NewRateLimiter(configs rlports.RateLimitConfigs) *RateLimiter {
	return &RateLimiter{configs: configs, counters: map[string]counter{}}
}

func (l *RateLimiter) Allow(
	_ context.Context,
	policyID, key string,
	now time.Time,
) (rlports.RateLimitResult, error) {
	config, ok := l.configs[policyID]
	if !ok {
		return rlports.RateLimitResult{}, fmt.Errorf("ratelimit: unknown policy %q", policyID)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	counterKey := policyID + ":" + key
	c, ok := l.counters[counterKey]
	if !ok || !now.Before(c.expiresAt) {
		c = counter{count: 1, expiresAt: now.Add(time.Duration(config.WindowSeconds) * time.Second)}
	} else {
		c.count++
	}
	l.counters[counterKey] = c
	if c.count > config.MaxRequests {
		return rlports.RateLimitResult{
			Allowed:           false,
			RetryAfterSeconds: int(math.Ceil(c.expiresAt.Sub(now).Seconds())),
		}, nil
	}
	return rlports.RateLimitResult{Allowed: true}, nil
}
