package usecases

// 認証イベント / 監査イベントの保持期間 sweep (ADR-045)。種類ごとに根拠ある保持期間を
// 決め、cutoff より古い行を確実に削除する。partition 化は行わない判断のため、retention が
// 単一テーブルの肥大と PII 滞留を抑える唯一の機構になる。
//
// 種類別の既定: 成功 / 一般監査 365 日・失敗詳細 30 日・bucket 集約 90 日・MFA 90 日・
// セッション 90 日。impersonation は本人保護のため短縮対象外 (global cap 未設定なら無期限)。
// global cap (MaxDays) はどの種類もこれを超えて保持しない上限。

import (
	"context"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
)

// RetentionPolicy は種類別の保持日数と global cap。0 以下の日数は「無期限保持」を意味する。
type RetentionPolicy struct {
	SuccessDays    int
	FailDays       int
	AggregatedDays int
	MfaDays        int
	SessionDays    int
	// MaxDays は global cap (0 = 上限なし)。各種類はこれを超えて保持しない。
	MaxDays int
}

// DefaultRetentionPolicy は ADR-045 の既定値。
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		SuccessDays:    365,
		FailDays:       30,
		AggregatedDays: 90,
		MfaDays:        90,
		SessionDays:    90,
		MaxDays:        0,
	}
}

// 保持期間の種類分け。impersonation は短縮対象外なので Keep に入れる。
var (
	retentionFailTypes = []string{
		(&authdomain.AuthenticationFailed{}).EventType(),
		(&authdomain.AuthenticationStepFailed{}).EventType(),
	}
	retentionAggregatedTypes = []string{
		(&authdomain.AuthenticationEventAggregated{}).EventType(),
	}
	retentionMfaTypes = []string{
		(&authdomain.MfaChallengeIssued{}).EventType(),
		(&authdomain.MfaChallengeSucceeded{}).EventType(),
		(&authdomain.MfaChallengeFailed{}).EventType(),
		(&authdomain.BackupCodeConsumed{}).EventType(),
		(&authdomain.MfaEnrollmentRequiredEvent{}).EventType(),
		(&authdomain.MfaEnrollmentCompleted{}).EventType(),
		(&authdomain.MfaEnrollmentBypassIssued{}).EventType(),
		(&authdomain.MfaEnrollmentBypassConsumed{}).EventType(),
		(&authdomain.MfaEnrollmentBypassRevoked{}).EventType(),
		(&authdomain.MfaEnrollmentBypassExpired{}).EventType(),
	}
	retentionSessionTypes = []string{
		(&authdomain.SessionStarted{}).EventType(),
		(&authdomain.SessionRefreshed{}).EventType(),
		(&authdomain.SessionEnded{}).EventType(),
	}
	retentionImpersonationTypes = []string{
		(&authdomain.SessionImpersonationStarted{}).EventType(),
		(&authdomain.SessionImpersonationEnded{}).EventType(),
	}
)

// capDays は global cap を適用する。days <= 0 (無期限) は cap があればそれに丸める。
func (p RetentionPolicy) capDays(days int) int {
	if p.MaxDays > 0 {
		if days <= 0 || days > p.MaxDays {
			return p.MaxDays
		}
	}
	return days
}

// AuditCutoff は now を基準に、監査イベント sweep の type 別 cutoff を組み立てる。
func (p RetentionPolicy) AuditCutoff(now time.Time) auditports.RetentionCutoff {
	byType := map[string]time.Time{}
	assign := func(types []string, days int) {
		days = p.capDays(days)
		if days <= 0 {
			return
		}
		before := now.Add(-time.Duration(days) * 24 * time.Hour)
		for _, t := range types {
			byType[t] = before
		}
	}
	assign(retentionFailTypes, p.FailDays)
	assign(retentionAggregatedTypes, p.AggregatedDays)
	assign(retentionMfaTypes, p.MfaDays)
	assign(retentionSessionTypes, p.SessionDays)

	cutoff := auditports.RetentionCutoff{ByType: byType}
	// impersonation は short 化対象外。global cap がある場合のみ cap で消す。
	if p.MaxDays > 0 {
		before := now.Add(-time.Duration(p.MaxDays) * 24 * time.Hour)
		for _, t := range retentionImpersonationTypes {
			byType[t] = before
		}
	} else {
		cutoff.Keep = append(cutoff.Keep, retentionImpersonationTypes...)
	}
	if d := p.capDays(p.SuccessDays); d > 0 {
		cutoff.Default = now.Add(-time.Duration(d) * 24 * time.Hour)
	}
	return cutoff
}

// BucketCutoff は authentication_event_buckets の削除境界 (集約は AggregatedDays)。
func (p RetentionPolicy) BucketCutoff(now time.Time) time.Time {
	days := p.capDays(p.AggregatedDays)
	if days <= 0 {
		return time.Time{}
	}
	return now.Add(-time.Duration(days) * 24 * time.Hour)
}

// SessionCutoff は LoginSession (authentication_sessions) housekeeping の削除境界。
// expires_at がこれより前の行を物理削除できる (SessionDays を転用、wi-253 Plan §7)。
func (p RetentionPolicy) SessionCutoff(now time.Time) time.Time {
	days := p.capDays(p.SessionDays)
	if days <= 0 {
		return time.Time{}
	}
	return now.Add(-time.Duration(days) * 24 * time.Hour)
}

// AuditEventPurger / AuthEventBucketPurger は sweep が要求する削除境界。store の read 契約
// (AuditEventRepository / AuthEventBucketStore) とは分離し、sweep を持たない構成でも動く。
type AuditEventPurger interface {
	DeleteOlderThan(ctx context.Context, cutoff auditports.RetentionCutoff) (int64, error)
}

type AuthEventBucketPurger interface {
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// SessionPurger は LoginSession housekeeping cleanup が要求する削除境界。SessionStore の
// 認証解決契約とは分離し、sweep を持たない構成でも動く (wi-253 Plan §7)。
type SessionPurger interface {
	DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error)
}

// sessionSweepBatchLimit は 1 回の sweep 呼び出しで物理削除する LoginSession の上限。
// 大きな DELETE 1 発による write amplification / lock 競合を避け、残りは次回の
// (外部 scheduler が起動する) sweep 呼び出しで収束させる。
const sessionSweepBatchLimit = 1000

// RetentionSweepResult は 1 回の sweep で削除した件数。
type RetentionSweepResult struct {
	AuditEvents int64
	Buckets     int64
	Sessions    int
}

// RunRetentionSweep は監査イベント・bucket・期限切れ LoginSession を保持期間に従って
// 一括削除する。store が nil の系統はスキップする。idempotent で、1 回で消し切れなくても
// 次回で収束する。
func RunRetentionSweep(
	ctx context.Context,
	audit AuditEventPurger,
	buckets AuthEventBucketPurger,
	sessions SessionPurger,
	policy RetentionPolicy,
	now time.Time,
) (RetentionSweepResult, error) {
	var result RetentionSweepResult
	now = now.UTC()
	if audit != nil {
		deleted, err := audit.DeleteOlderThan(ctx, policy.AuditCutoff(now))
		if err != nil {
			return result, err
		}
		result.AuditEvents = deleted
	}
	if buckets != nil {
		if before := policy.BucketCutoff(now); !before.IsZero() {
			deleted, err := buckets.DeleteOlderThan(ctx, before)
			if err != nil {
				return result, err
			}
			result.Buckets = deleted
		}
	}
	if sessions != nil {
		if before := policy.SessionCutoff(now); !before.IsZero() {
			deleted, err := sessions.DeleteExpiredBatch(ctx, before, sessionSweepBatchLimit)
			if err != nil {
				return result, err
			}
			result.Sessions = deleted
		}
	}
	return result, nil
}
