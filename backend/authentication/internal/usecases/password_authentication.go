package usecases

// セッションを作らないパスワード認証。対話のログインと WS-Trust の能動的な STS が同じ試行の制限、
// 照合、失敗の記録と集約、認証の失敗のイベントを使う (REQ-AUTHENTICATION-007、REQ-WSFEDERATION-004)。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/internal/ports"
	passworddomain "github.com/ambi/idmagic/backend/authentication/password/domain"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// LoginThrottleMetrics は、試行の制限の判定を指標へ記録する。
type LoginThrottleMetrics interface {
	RecordLoginThrottle(policy, outcome string)
}

// PasswordAuthentication は signinports.PasswordAuthenticator を実装する。
type PasswordAuthentication struct {
	UserRepo             userports.UserRepository
	PasswordHasher       passwordports.PasswordHasher
	SentinelPasswordHash string
	// LoginAttemptThrottle が nil なら試行の制限を数えない。
	LoginAttemptThrottle sessionports.LoginAttemptThrottle
	// AuthEventBucketStore が nil なら閾値を超えた失敗を集約せず、個別のイベントを残す。
	AuthEventBucketStore authnports.AuthEventBucketStore
	TenantSaltStore      auditports.TenantSaltStore
	Emit                 func(spec.DomainEvent)
	Metrics              LoginThrottleMetrics
}

var _ signinports.PasswordAuthenticator = PasswordAuthentication{}

// AuthenticatePassword は、試行の制限、照合、有効性とポリシーの確認をこの順に行う。
//
// 失敗の記録に失敗したときは、拒否の理由とエラーの両方を返す。呼び出し側はエラーを返す前に、
// 確定した失敗として結果の指標を記録できる。
func (p PasswordAuthentication) AuthenticatePassword(
	ctx context.Context, in signinports.PasswordAttempt,
) (signinports.PasswordOutcome, error) {
	normalizedUsername := strings.ToLower(in.Username)
	if result, err := p.acquireLoginThrottle(ctx, sessionports.LoginThrottleAccount, normalizedUsername); err != nil {
		return signinports.PasswordOutcome{}, err
	} else if !result.Allowed {
		return throttled(result), nil
	}
	if in.ClientIP != "" {
		if result, err := p.acquireLoginThrottle(ctx, sessionports.LoginThrottleIP, in.ClientIP); err != nil {
			return signinports.PasswordOutcome{}, err
		} else if !result.Allowed {
			return throttled(result), nil
		}
	}

	user, err := p.UserRepo.FindByUsername(ctx, tenantports.TenantID(ctx), in.Username)
	if err != nil {
		return signinports.PasswordOutcome{}, err
	}
	hashToVerify := p.SentinelPasswordHash
	if user != nil {
		hashToVerify = user.PasswordHash
	}
	ok := false
	if hashToVerify != "" {
		ok, err = p.PasswordHasher.Verify(in.Password, hashToVerify)
	}
	if user == nil || err != nil || !ok {
		refused := signinports.PasswordOutcome{Refusal: signinports.PasswordRefusalInvalidCredentials}
		aggregated, ferr := p.recordLoginFailure(ctx, normalizedUsername, in.ClientIP)
		if ferr != nil {
			return refused, ferr
		}
		// 閾値超過後は AuthenticationEventAggregated に集約し、個別行を出さない (爆発抑制)。
		if !aggregated {
			p.emitAuthenticationFailure(ctx, in, "invalid_credentials")
		}
		return refused, nil
	}
	if !user.IsActive() {
		p.emitAuthenticationFailure(ctx, in, "account_disabled")
		return signinports.PasswordOutcome{Refusal: signinports.PasswordRefusalAccountDisabled}, nil
	}
	if result := passworddomain.ValidatePassword(in.Password); !result.OK {
		return signinports.PasswordOutcome{Refusal: signinports.PasswordRefusalPasswordPolicy}, nil
	}
	if p.LoginAttemptThrottle != nil {
		if err := p.LoginAttemptThrottle.RecordSuccess(ctx, sessionports.LoginThrottleAccount, normalizedUsername); err != nil {
			return signinports.PasswordOutcome{}, err
		}
	}
	return signinports.PasswordOutcome{User: user, PasswordChangeRequired: passwordChangeRequired(ctx, user)}, nil
}

func throttled(result sessionports.LoginThrottleResult) signinports.PasswordOutcome {
	return signinports.PasswordOutcome{
		Refusal: signinports.PasswordRefusalThrottled, RetryAfterSeconds: result.RetryAfterSeconds,
	}
}

// passwordChangeRequired は、パスワードの期限切れか必須操作 update_password を、User を変えずに判定する。
func passwordChangeRequired(ctx context.Context, user *userdomain.User) bool {
	if slices.Contains(user.Lifecycle.RequiredActions, idmdomain.RequiredActionUpdatePassword) {
		return true
	}
	if user.PasswordHash == "" {
		return false
	}
	policy := passworddomain.PolicyForTenant(tenantports.Tenant(ctx))
	return passworddomain.PasswordExpired(policy, user.Lifecycle.PasswordChangedAt, time.Now().UTC())
}

func (p PasswordAuthentication) emitAuthenticationFailure(ctx context.Context, in signinports.PasswordAttempt, reason string) {
	if p.Emit != nil {
		p.Emit(&authdomain.AuthenticationFailed{
			At: time.Now().UTC(), TenantID: tenantports.TenantID(ctx), Username: in.Username, Reason: reason,
			IP: in.ClientIP, UserAgent: in.UserAgent,
		})
	}
}

func (p PasswordAuthentication) acquireLoginThrottle(
	ctx context.Context,
	kind sessionports.LoginThrottleKind,
	key string,
) (sessionports.LoginThrottleResult, error) {
	if p.LoginAttemptThrottle == nil {
		return sessionports.LoginThrottleResult{Allowed: true}, nil
	}
	result, err := p.LoginAttemptThrottle.TryAcquire(ctx, kind, key, time.Now().UTC())
	if p.Metrics != nil {
		outcome := "allowed"
		switch {
		case err != nil:
			outcome = "store_unavailable"
		case !result.Allowed:
			outcome = "throttled"
		}
		p.Metrics.RecordLoginThrottle(string(kind), outcome)
	}
	return result, err
}

// recordLoginFailure は失敗を throttle に記録し、閾値超過 (Locked) の key については
// LoginThrottled を emit したうえで失敗を集約 bucket に積む。集約に切り替わった場合は
// aggregated=true を返し、呼び出し側は個別の AuthenticationFailed を抑制する
// (これが攻撃時の行爆発を止める要点 / wi-20 スライス 3)。
func (p PasswordAuthentication) recordLoginFailure(ctx context.Context, username, clientIP string) (bool, error) {
	if p.LoginAttemptThrottle == nil {
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
		result, err := p.LoginAttemptThrottle.RecordFailure(ctx, attempt.kind, attempt.key, now)
		if err != nil {
			return aggregated, err
		}
		if !result.Locked {
			continue
		}
		keyHash := p.correlationHash(ctx, attempt.key)
		if p.Emit != nil {
			p.Emit(&authdomain.LoginThrottled{
				At: now, TenantID: tenantports.TenantID(ctx), Kind: string(attempt.kind),
				KeyHash:           keyHash,
				RetryAfterSeconds: result.RetryAfterSeconds,
			})
		}
		if p.recordFailedLoginBucket(ctx, keyHash, now) {
			aggregated = true
		}
	}
	return aggregated, nil
}

// correlationHash は throttle / bucket の emit keyHash を tenant salt 付きで計算する
// (wi-145)。username / IP の相関検索属性と同じ単一ヘルパ (spec.SaltedHash) を共有し、
// tenant salt により cross-tenant で相関を集約しない。salt store が無い構成 (一部テスト) では
// unsalted SHA-256 にフォールバックする。
func (p PasswordAuthentication) correlationHash(ctx context.Context, value string) string {
	if p.TenantSaltStore != nil {
		if salt, err := p.TenantSaltStore.GetSalt(ctx); err == nil {
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
func (p PasswordAuthentication) recordFailedLoginBucket(ctx context.Context, keyHash string, now time.Time) bool {
	if p.AuthEventBucketStore == nil {
		return false
	}
	result, err := p.AuthEventBucketStore.Record(
		ctx, authnports.AuthEventBucketFailedLogin, tenantports.TenantID(ctx), keyHash, now,
	)
	if err != nil {
		return false
	}
	if result.FirstInWindow && p.Emit != nil {
		bucket := result.Bucket
		p.Emit(&authdomain.AuthenticationEventAggregated{
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
