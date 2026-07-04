package valkey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"time"

	authnports "idmagic/internal/authentication/ports"

	goredis "github.com/redis/go-redis/v9"
)

// LoginAttemptThrottle は login throttle を Valkey 上の共有カウンタで数える adapter。
// 複数レプリカでも per-account / per-IP のしきい値がクラスタ全体で一つになるよう、
// カウンタとロックを共有ストアに置く (ADR-077)。カウンタ / ロックのキーに載せる
// 識別子 (username / IP) は SHA-256 で hash 化して平文を残さず、tenant_id で名前空間を
// 分ける。到達不能時はエラーを返し、呼び出し側で fail-closed に倒れる (縮退時も攻撃を
// 素通しにしない)。
type LoginAttemptThrottle struct {
	Client  *goredis.Client
	Configs authnports.LoginThrottleConfigs
}

// recordFailure は失敗を 1 往復で原子的に数える。初回インクリメント時にだけ window の
// EXPIRE を張り、しきい値到達で counter を消してロックキーを lockout 秒で立てる。戻り値は
// 1 でロック確定、0 で未ロック。read-write を分割せず複数レプリカ間の競合を避ける。
var recordLoginFailure = goredis.NewScript(`
local failures = redis.call('INCR', KEYS[1])
if failures == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
if failures < tonumber(ARGV[2]) then
  return 0
end
redis.call('DEL', KEYS[1])
redis.call('SET', KEYS[2], '1', 'EX', ARGV[3])
return 1
`)

func (t *LoginAttemptThrottle) TryAcquire(
	ctx context.Context,
	kind authnports.LoginThrottleKind,
	key string,
	_ time.Time,
) (authnports.LoginThrottleResult, error) {
	pttl, err := t.Client.PTTL(ctx, t.lockKey(ctx, kind, key)).Result()
	if err != nil {
		return authnports.LoginThrottleResult{}, err
	}
	if pttl < 0 {
		return authnports.LoginThrottleResult{Allowed: true}, nil
	}
	return authnports.LoginThrottleResult{
		Allowed: false, Locked: true,
		RetryAfterSeconds: int(math.Ceil(pttl.Seconds())),
	}, nil
}

func (t *LoginAttemptThrottle) RecordFailure(
	ctx context.Context,
	kind authnports.LoginThrottleKind,
	key string,
	_ time.Time,
) (authnports.LoginThrottleResult, error) {
	config := t.config(kind)
	locked, err := recordLoginFailure.Run(
		ctx, t.Client,
		[]string{t.counterKey(ctx, kind, key), t.lockKey(ctx, kind, key)},
		config.WindowSeconds, config.MaxFailures, config.LockoutSeconds,
	).Int()
	if err != nil {
		return authnports.LoginThrottleResult{}, err
	}
	if locked == 0 {
		return authnports.LoginThrottleResult{Allowed: true}, nil
	}
	return authnports.LoginThrottleResult{
		Allowed: false, Locked: true, RetryAfterSeconds: config.LockoutSeconds,
	}, nil
}

func (t *LoginAttemptThrottle) RecordSuccess(
	ctx context.Context,
	kind authnports.LoginThrottleKind,
	key string,
) error {
	return t.Client.Del(ctx, t.counterKey(ctx, kind, key), t.lockKey(ctx, kind, key)).Err()
}

func (t *LoginAttemptThrottle) config(kind authnports.LoginThrottleKind) authnports.LoginThrottleConfig {
	if kind == authnports.LoginThrottleIP {
		return t.Configs.IP
	}
	return t.Configs.Account
}

func (t *LoginAttemptThrottle) counterKey(ctx context.Context, kind authnports.LoginThrottleKind, key string) string {
	return tenantKey(ctx, "login_throttle:failures:"+string(kind)+":"+hashThrottleIdentifier(key))
}

func (t *LoginAttemptThrottle) lockKey(ctx context.Context, kind authnports.LoginThrottleKind, key string) string {
	return tenantKey(ctx, "login_throttle:lock:"+string(kind)+":"+hashThrottleIdentifier(key))
}

func hashThrottleIdentifier(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
