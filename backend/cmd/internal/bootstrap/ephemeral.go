package bootstrap

// 揮発性の認証 / OAuth2 一時状態 の空間回収 sweep。正しさは各 read の
// expires_at 述語が担保しており、本 sweep は best-effort な空間回収のみ。
// 常駐 idmagic-worker の周期 ticker から呼ぶ。

import (
	"context"
	"time"

	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
)

// ephemeralPurger は期限切れ行の一括削除境界。DeleteExpiredBatch を実装するのは postgres
// adapter だけで、memory adapter は実装しないため sweep から自動的に除外される
// (RunRetentionSweepOnce の Purger 型アサーションと同じパターン)。
type ephemeralPurger interface {
	DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error)
}

// EphemeralSweepBatchLimit は 1 ストア 1 tick あたりの削除上限。溢れた分は次 tick が回収する。
const EphemeralSweepBatchLimit = 1000

// RunEphemeralSweepOnce は全揮発性ストアの期限切れ行を現在時刻境界で一度だけ一括削除する。
// best-effort であり、あるストアの失敗は他ストアの回収を止めない。
func RunEphemeralSweepOnce(ctx context.Context, deps *Dependencies, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stores := []struct {
		name  string
		store any
	}{
		{"authorization_requests", deps.OAuth2.RequestStore},
		{"authorization_codes", deps.OAuth2.CodeStore},
		{"par_requests", deps.OAuth2.PARStore},
		{"device_codes", deps.OAuth2.DeviceCodeStore},
		{"dpop_replay", deps.OAuth2.DpopReplayStore},
		{"client_assertion_replay", deps.OAuth2.ClientAssertionReplayStore},
		{"access_token_denylist", deps.OAuth2.AccessTokenDenylist},
		{"webauthn_sessions", deps.Authentication.WebAuthnSessionStore},
		{"saml_authnrequest_replays", deps.Saml.ReplayStore},
	}
	// login throttle / endpoint rate limiter は factory 経由なので instance を作って加える
	// (GC は configs 非依存)。
	if deps.Authentication.NewLoginAttemptThrottle != nil {
		throttle := deps.Authentication.NewLoginAttemptThrottle(sessionports.LoginThrottleConfigs{})
		stores = append(stores, struct {
			name  string
			store any
		}{"login_throttle", throttle})
	}
	if deps.RateLimit.NewRateLimiter != nil {
		limiter := deps.RateLimit.NewRateLimiter(rlports.RateLimitConfigs{})
		stores = append(stores, struct {
			name  string
			store any
		}{"endpoint_rate_limit", limiter})
	}

	total := 0
	for _, s := range stores {
		purger, ok := s.store.(ephemeralPurger)
		if !ok {
			continue
		}
		deleted, err := purger.DeleteExpiredBatch(ctx, now, EphemeralSweepBatchLimit)
		if err != nil {
			logging.Warn(ctx, "ephemeral sweep store failed", "store", s.name, "error", err)
			continue
		}
		total += deleted
	}
	if total > 0 {
		logging.Info(ctx, "ephemeral sweep completed", "deleted", total)
	}
	return nil
}
