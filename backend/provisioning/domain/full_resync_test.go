package domain_test

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

//spec:covers REQ-PROVISIONING-013: 全対象のプロビジョニングタスクが終端になったときだけ完了し、成功数と失敗数と対象数を完了イベントへ写す。
func TestFullResyncSettleCompletesWhenEveryTaskIsTerminal(t *testing.T) {
	started := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	now := started.Add(time.Minute)
	resync := domain.NewFullResync("resync-1", "tenant-1", "app-1", 3, started)

	completed, ok := resync.Settle(domain.FullResyncTally{Succeeded: 2, Failed: 1}, now)
	if !ok {
		t.Fatalf("Settle(all terminal) = not completed, want completed")
	}
	if completed.Status != domain.FullResyncStatusCompleted || completed.CompletedAt == nil || !completed.CompletedAt.Equal(now) {
		t.Fatalf("completed = %+v, want completed at %v", completed, now)
	}
	if completed.SucceededCount != 2 || completed.FailedCount != 1 {
		t.Fatalf("counts = %d/%d, want 2 succeeded, 1 failed", completed.SucceededCount, completed.FailedCount)
	}
	if resync.Status != domain.FullResyncStatusRunning {
		t.Fatalf("receiver status = %s, want Settle to leave the original running", resync.Status)
	}
	event := completed.CompletedEvent()
	want := domain.FullResyncCompleted{At: now, TenantID: "tenant-1", ApplicationID: "app-1", TotalSubjects: 3, SucceededCount: 2, FailedCount: 1}
	if *event != want {
		t.Fatalf("CompletedEvent() = %+v, want %+v", *event, want)
	}
}

//spec:covers REQ-PROVISIONING-013: 終端の件数が対象数に満たないとき、および完了済みのフル同期では完了しない。
func TestFullResyncSettleRefusesEarlyOrRepeatedCompletion(t *testing.T) {
	started := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	now := started.Add(time.Minute)
	running := *domain.NewFullResync("resync-1", "tenant-1", "app-1", 3, started)
	completed, ok := running.Settle(domain.FullResyncTally{Succeeded: 3}, now)
	if !ok {
		t.Fatal("Settle(all succeeded) = not completed")
	}

	cases := []struct {
		name   string
		resync domain.FullResync
		tally  domain.FullResyncTally
	}{
		// 一件が in_flight のまま、またはまだ作られていない。
		{"終端が対象数に満たない", running, domain.FullResyncTally{Succeeded: 1, Failed: 1}},
		{"完了済み", completed, domain.FullResyncTally{Succeeded: 3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := tc.resync.Settle(tc.tally, now.Add(time.Minute)); ok {
				t.Fatalf("Settle(%+v) = %+v, want not completed", tc.tally, got)
			}
		})
	}
}

//spec:covers REQ-PROVISIONING-013: 対象が 0 件のフル同期は開始と同時に完了できる。
func TestFullResyncSettleCompletesAnEmptyScope(t *testing.T) {
	now := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	completed, ok := domain.NewFullResync("resync-1", "tenant-1", "app-1", 0, now).Settle(domain.FullResyncTally{}, now)
	if !ok || completed.CompletedEvent().TotalSubjects != 0 {
		t.Fatalf("Settle(empty) = %+v, %v; want completed with zero subjects", completed, ok)
	}
}
