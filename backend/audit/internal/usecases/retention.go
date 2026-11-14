package usecases

// 監査の記録の保持期間の掃引。種類ごとの保持期間を発生時刻から数え、過ぎた記録を消す。
// パーティション化を行わないので、この掃引が単一テーブルの肥大と個人データの滞留を抑える
// 唯一の機構になる。種類はイベントの公開契約の種別名で分類し、発行元のモジュールを import しない。

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
)

// RetentionPolicy は監査の記録の保持日数である（REQ-AUDIT-013）。DaysByType にない種類は
// DefaultDays を使う。どの種類も有限の期間を持ち、なりすましの記録もデフォルトの期間で消える。
type RetentionPolicy struct {
	DefaultDays int
	DaysByType  map[string]int
}

const (
	defaultRetentionDays    = 365
	failureRetentionDays    = 30
	aggregatedRetentionDays = 90
	mfaRetentionDays        = 90
	sessionEndRetentionDays = 90
)

// DefaultRetentionPolicy は製品の保持期間である。
func DefaultRetentionPolicy() RetentionPolicy {
	days := map[string]int{
		"AuthenticationFailed":          failureRetentionDays,
		"AuthenticationEventAggregated": aggregatedRetentionDays,
		"SessionEnded":                  sessionEndRetentionDays,
	}
	for _, t := range []string{
		"MfaChallengeIssued",
		"MfaChallengeSucceeded",
		"MfaChallengeFailed",
		"BackupCodeConsumed",
		"MfaEnrollmentRequired",
		"MfaEnrollmentCompleted",
		"MfaEnrollmentBypassIssued",
		"MfaEnrollmentBypassConsumed",
		"MfaEnrollmentBypassRevoked",
		"MfaEnrollmentBypassExpired",
	} {
		days[t] = mfaRetentionDays
	}
	return RetentionPolicy{
		DefaultDays: defaultRetentionDays,
		DaysByType:  days,
	}
}

// Cutoff は now を基準に、種類ごとの削除の境界を返す。境界より前に発生した記録が消える。
func (p RetentionPolicy) Cutoff(now time.Time) ports.RetentionCutoff {
	before := func(days int) time.Time { return now.Add(-time.Duration(days) * 24 * time.Hour) }
	byType := make(map[string]time.Time, len(p.DaysByType))
	for t, days := range p.DaysByType {
		byType[t] = before(days)
	}
	return ports.RetentionCutoff{
		Default: before(p.DefaultDays),
		ByType:  byType,
	}
}

// EventPurger は掃引が保存先に要求する削除である。
type EventPurger interface {
	DeleteOlderThan(ctx context.Context, cutoff ports.RetentionCutoff) (int64, error)
}

// PurgeExpiredEvents は保持期間を過ぎた監査の記録を消し、削除した件数を返す。
// 冪等で、一度で消し切れなくても次の実行で収束する。
func PurgeExpiredEvents(ctx context.Context, purger EventPurger, policy RetentionPolicy, now time.Time) (int64, error) {
	return purger.DeleteOlderThan(ctx, policy.Cutoff(now.UTC()))
}
