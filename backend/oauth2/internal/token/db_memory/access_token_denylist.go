package db_memory

import (
	"context"
	"sync"
	"time"
)

// =====================================================================
// AccessTokenDenylist (OAuth2)
// =====================================================================

type AccessTokenDenylist struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func NewAccessTokenDenylist() *AccessTokenDenylist {
	return &AccessTokenDenylist{entries: map[string]time.Time{}}
}

func (d *AccessTokenDenylist) Add(_ context.Context, jti string, expiresAt time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries[jti] = expiresAt
	return nil
}

func (d *AccessTokenDenylist) IsRevoked(_ context.Context, jti string, now time.Time) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	expiresAt, ok := d.entries[jti]
	if ok && !now.Before(expiresAt) {
		delete(d.entries, jti)
		return false, nil
	}
	return ok, nil
}
