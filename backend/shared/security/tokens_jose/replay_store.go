package tokens_jose

import (
	"context"
	"time"
)

// ReplayStore は検証器が受理した jti を期限付きで一度だけ記録する。
// 用途ごとの保存先と有効期間は、呼び出す側が選ぶ。
type ReplayStore interface {
	RecordIfNew(ctx context.Context, jti string, windowSeconds int, now time.Time) (bool, error)
}
