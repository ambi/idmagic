package resilience

import (
	"math/rand/v2"
	"time"
)

// RandomDelay は 0 以上 limit 以下の一様な待ち時間を返す。
// limit は非負とする。秘密値の生成には使わない。並行呼び出し可能である。
func RandomDelay(limit time.Duration) time.Duration {
	// uint64 で加算し、最大の duration でも上限を含めて抽選する。
	return time.Duration(rand.Uint64N(uint64(limit) + 1)) //nolint:gosec // G115: 抽選結果は非負の time.Duration の上限以下。
}
