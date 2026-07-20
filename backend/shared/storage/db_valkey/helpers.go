package db_valkey

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/tenancy"

	goredis "github.com/redis/go-redis/v9"
)

// SetJSON は tenant-local store が共有する JSON 保存ヘルパー。
func SetJSON(ctx context.Context, client goredis.Cmdable, key string, value any, ttl time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, payload, ttl).Err()
}

// GetJSON は tenant-local store が共有する JSON 読み出しヘルパー。
func GetJSON(ctx context.Context, client goredis.Cmdable, key string, out any) error {
	payload, err := client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, out)
}

// TTLUntil は期限時刻から Valkey TTL を求める。
func TTLUntil(expiresAt time.Time) time.Duration {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return time.Millisecond
	}
	return ttl
}

// TenantKey はテナントで名前空間を分離した Valkey key を作る。
func TenantKey(ctx context.Context, suffix string) string {
	return "tenant:" + tenancy.TenantID(ctx) + ":" + suffix
}
