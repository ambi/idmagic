package audit

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/audit/internal/usecases"
)

// PurgeExpiredEvents は種類ごとの保持期間を過ぎた監査の記録を、検索属性とともに消す。
// 構築関数を通さず保存先を持たない Module では何もしない。
func (m Module) PurgeExpiredEvents(ctx context.Context, now time.Time) (int64, error) {
	if m.eventPurger == nil {
		return 0, nil
	}
	return usecases.PurgeExpiredEvents(ctx, m.eventPurger, usecases.DefaultRetentionPolicy(), now)
}
