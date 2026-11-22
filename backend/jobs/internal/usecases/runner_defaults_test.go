package usecases

import (
	"testing"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
)

//spec:covers REQ-JOBS-005: 乱数を省略した本番の構築でも、一様乱数の待ち時間を設定する。
func TestRunnerDefaultsProvideRandomDelay(t *testing.T) {
	runner := NewRunner(RunnerConfig{Lane: jobsdomain.LaneDefault}, RunnerDeps{})
	if runner.deps.RandomDelay == nil {
		t.Fatal("default random delay is missing")
	}
	const limit = 30 * time.Second
	allAtLimit := true
	for range 4 {
		delay := runner.deps.RandomDelay(limit)
		if delay < 0 || delay > limit {
			t.Fatalf("delay=%v, want [0,%v]", delay, limit)
		}
		allAtLimit = allAtLimit && delay == limit
	}
	// 4 回とも上限を選ぶ確率は 10^-41 未満。固定待機の配線を検出する。
	if allAtLimit {
		t.Fatal("default random delay always returns the limit")
	}
}
