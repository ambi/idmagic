package http

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func (d Deps) emitAuthenticationFailure(c *echo.Context, username, reason string) {
	if d.Emit != nil {
		d.Emit(&authdomain.AuthenticationFailed{
			At: time.Now().UTC(), TenantID: support.RequestTenantID(c), Username: username, Reason: reason,
			IP: extractClientIP(c.Request(), d.TrustedForwardedHops), UserAgent: c.Request().UserAgent(),
		})
	}
}

func (d Deps) acquireLoginThrottle(
	c *echo.Context,
	kind sessionports.LoginThrottleKind,
	key string,
) (sessionports.LoginThrottleResult, error) {
	if d.LoginAttemptThrottle == nil {
		return sessionports.LoginThrottleResult{Allowed: true}, nil
	}
	result, err := d.LoginAttemptThrottle.TryAcquire(c.Request().Context(), kind, key, time.Now().UTC())
	if d.Metrics != nil {
		outcome := "allowed"
		switch {
		case err != nil:
			outcome = "store_unavailable"
		case !result.Allowed:
			outcome = "throttled"
		}
		d.Metrics.RecordLoginThrottle(string(kind), outcome)
	}
	return result, err
}

// recordLoginFailure は失敗を throttle に記録し、閾値超過 (Locked) の key については
// LoginThrottled を emit したうえで失敗を集約 bucket に積む。集約に切り替わった場合は
// aggregated=true を返し、呼び出し側は個別の AuthenticationFailed を抑制する
// (これが攻撃時の行爆発を止める要点 / wi-20 スライス 3)。
func (d Deps) recordLoginFailure(c *echo.Context, username, clientIP string) (bool, error) {
	if d.LoginAttemptThrottle == nil {
		return false, nil
	}
	now := time.Now().UTC()
	aggregated := false
	for _, attempt := range []struct {
		kind sessionports.LoginThrottleKind
		key  string
	}{
		{sessionports.LoginThrottleAccount, username},
		{sessionports.LoginThrottleIP, clientIP},
	} {
		if attempt.key == "" {
			continue
		}
		result, err := d.LoginAttemptThrottle.RecordFailure(
			c.Request().Context(), attempt.kind, attempt.key, now,
		)
		if err != nil {
			return aggregated, err
		}
		if !result.Locked {
			continue
		}
		keyHash := d.correlationHash(c, attempt.key)
		if d.Emit != nil {
			d.Emit(&authdomain.LoginThrottled{
				At: now, TenantID: support.RequestTenantID(c), Kind: string(attempt.kind),
				KeyHash:           keyHash,
				RetryAfterSeconds: result.RetryAfterSeconds,
			})
		}
		if d.recordFailedLoginBucket(c, keyHash, now) {
			aggregated = true
		}
	}
	return aggregated, nil
}

// correlationHash は throttle / bucket の emit keyHash を tenant salt 付きで計算する
// (wi-145 / ADR-046)。username / IP の相関検索属性と同じ単一ヘルパ (spec.SaltedHash) を共有し、
// tenant salt により cross-tenant で相関を集約しない。salt store が無い構成 (一部テスト) では
// unsalted SHA-256 にフォールバックする。
func (d Deps) correlationHash(c *echo.Context, value string) string {
	if d.TenantSaltStore != nil {
		if salt, err := d.TenantSaltStore.GetSalt(c.Request().Context()); err == nil {
			return spec.SaltedHash(salt, value)
		}
	}
	return hashThrottleKey(value)
}

func hashThrottleKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// recordFailedLoginBucket は閾値超過後の失敗を 5 分窓の bucket に積み、その窓で最初の
// 記録だったときだけ AuthenticationEventAggregated を 1 件 emit する。bucket store が
// 無い構成では集約せず false を返し、呼び出し側は従来どおり個別イベントを残す。
func (d Deps) recordFailedLoginBucket(c *echo.Context, keyHash string, now time.Time) bool {
	if d.AuthEventBucketStore == nil {
		return false
	}
	result, err := d.AuthEventBucketStore.Record(
		c.Request().Context(), authnports.AuthEventBucketFailedLogin, support.RequestTenantID(c), keyHash, now,
	)
	if err != nil {
		return false
	}
	if result.FirstInWindow && d.Emit != nil {
		bucket := result.Bucket
		d.Emit(&authdomain.AuthenticationEventAggregated{
			At: now, TenantID: bucket.TenantID, Kind: string(bucket.Kind),
			BucketKey: failedLoginBucketKey(bucket),
			KeyHash:   bucket.KeyHash, Count: bucket.Count,
			FirstSeen: bucket.FirstSeen, LastSeen: bucket.LastSeen,
			TopKeys: []string{bucket.KeyHash},
		})
	}
	return true
}

func failedLoginBucketKey(bucket authnports.AuthEventBucket) string {
	return string(bucket.Kind) + ":" + bucket.KeyHash + ":" +
		strconv.FormatInt(bucket.WindowStart.Unix(), 10)
}

func extractClientIP(request *http.Request, trustedHops int) string {
	if request == nil || trustedHops <= 0 {
		return ""
	}
	parts := strings.Split(request.Header.Get("X-Forwarded-For"), ",")
	ips := make([]string, 0, len(parts))
	for _, part := range parts {
		if ip := strings.TrimSpace(part); ip != "" {
			ips = append(ips, ip)
		}
	}
	index := len(ips) - 1 - trustedHops
	if index < 0 || index >= len(ips) {
		return ""
	}
	return ips[index]
}

func writeLoginThrottled(c *echo.Context, retryAfterSeconds int) error {
	c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	return support.NoStoreJSON(c, http.StatusTooManyRequests, map[string]any{
		"error": "too_many_requests", "retry_after_seconds": retryAfterSeconds,
	})
}
