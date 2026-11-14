package usecases

// Authentication が所有する記録の保持期間 sweep。認証イベントの集計 bucket、期限切れの
// LoginSession、使われなくなった既知の端末を、種類ごとの保持期間に従って削除する。
// 監査の記録の保持期間は Audit が定め、Audit の掃引が消す。

import (
	"context"
	"time"
)

// RetentionPolicy は Authentication の記録の保持日数。0 以下の日数は「無期限保持」を意味する。
type RetentionPolicy struct {
	AggregatedDays int
	SessionDays    int
	// KnownDeviceDays は既知のサインイン端末を「既知」と呼び続ける期間 (wi-90)。
	// 成功イベントの監査の記録と同じ 365 日にそろえる。サインイン履歴から消えた端末を既知のまま
	// にすると、本人が確かめる手立てのない判定が残るためである。
	KnownDeviceDays int
}

// DefaultRetentionPolicy は 既定値。
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		AggregatedDays:  90,
		SessionDays:     90,
		KnownDeviceDays: 365,
	}
}

func cutoffBefore(now time.Time, days int) time.Time {
	if days <= 0 {
		return time.Time{}
	}
	return now.Add(-time.Duration(days) * 24 * time.Hour)
}

// BucketCutoff は authentication_event_buckets の削除境界 (集約は AggregatedDays)。
func (p RetentionPolicy) BucketCutoff(now time.Time) time.Time {
	return cutoffBefore(now, p.AggregatedDays)
}

// SessionCutoff は LoginSession (authentication_sessions) housekeeping の削除境界。
// expires_at がこれより前の行を物理削除できる (SessionDays を転用、wi-253 Plan §7)。
func (p RetentionPolicy) SessionCutoff(now time.Time) time.Time {
	return cutoffBefore(now, p.SessionDays)
}

// KnownDeviceCutoff は known_sign_in_devices の削除境界 (wi-90)。
func (p RetentionPolicy) KnownDeviceCutoff(now time.Time) time.Time {
	return cutoffBefore(now, p.KnownDeviceDays)
}

// AuthEventBucketPurger は sweep が要求する削除境界。store の read 契約
// (AuthEventBucketStore) とは分離し、sweep を持たない構成でも動く。
type AuthEventBucketPurger interface {
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// SessionPurger は LoginSession housekeeping cleanup が要求する削除境界。SessionStore の
// 認証解決契約とは分離し、sweep を持たない構成でも動く (wi-253 Plan §7)。
type SessionPurger interface {
	DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error)
}

// KnownDevicePurger は既知のサインイン端末の削除境界 (wi-90)。
type KnownDevicePurger interface {
	DeleteIdleBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// RetentionStores は sweep が触る保存先の束。nil の系統はスキップする。位置引数を
// 増やし続けると、どの store がどれか読めなくなるため名前付きで渡す。
type RetentionStores struct {
	Buckets      AuthEventBucketPurger
	Sessions     SessionPurger
	KnownDevices KnownDevicePurger
}

// Empty は掃除する先が 1 つも無いかを返す。
func (s RetentionStores) Empty() bool {
	return s.Buckets == nil && s.Sessions == nil && s.KnownDevices == nil
}

// sessionSweepBatchLimit は 1 回の sweep 呼び出しで物理削除する LoginSession の上限。
// 大きな DELETE 1 発による write amplification / lock 競合を避け、残りは次回の
// (外部 scheduler が起動する) sweep 呼び出しで収束させる。
const sessionSweepBatchLimit = 1000

// RetentionSweepResult は 1 回の sweep で削除した件数。
type RetentionSweepResult struct {
	Buckets      int64
	Sessions     int
	KnownDevices int64
}

// RunRetentionSweep は bucket・期限切れ LoginSession・使われなくなった既知の端末を
// 保持期間に従って一括削除する。store が nil の系統はスキップする。
// idempotent で、1 回で消し切れなくても次回で収束する。
func RunRetentionSweep(
	ctx context.Context,
	stores RetentionStores,
	policy RetentionPolicy,
	now time.Time,
) (RetentionSweepResult, error) {
	var result RetentionSweepResult
	now = now.UTC()
	if stores.Buckets != nil {
		if before := policy.BucketCutoff(now); !before.IsZero() {
			deleted, err := stores.Buckets.DeleteOlderThan(ctx, before)
			if err != nil {
				return result, err
			}
			result.Buckets = deleted
		}
	}
	if stores.Sessions != nil {
		if before := policy.SessionCutoff(now); !before.IsZero() {
			deleted, err := stores.Sessions.DeleteExpiredBatch(ctx, before, sessionSweepBatchLimit)
			if err != nil {
				return result, err
			}
			result.Sessions = deleted
		}
	}
	if stores.KnownDevices != nil {
		if before := policy.KnownDeviceCutoff(now); !before.IsZero() {
			deleted, err := stores.KnownDevices.DeleteIdleBefore(ctx, before)
			if err != nil {
				return result, err
			}
			result.KnownDevices = deleted
		}
	}
	return result, nil
}
