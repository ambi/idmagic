package ports

import (
	"context"
	"time"
)

// MessageIDReplayStore は WS-Trust の MessageID の再利用を防ぐ。
// 呼び出す側が名前空間を付けたキーと有効期間を渡す。
type MessageIDReplayStore interface {
	// RecordIfNew は未記録または期限切れのキーを原子的に記録して true を返す。
	// 有効な記録があれば false、保存できなければエラーを返す。
	RecordIfNew(ctx context.Context, key string, windowSeconds int, now time.Time) (bool, error)
}
