package domain

import (
	"testing"
	"time"
)

//spec:covers EX-PROVISIONING-006-01: 予約は削除時刻から grace_period_days × 24 時間が経った瞬間に期限へ達し、その直前には達しない。
func TestScheduledDeprovision_IsDueExactlyWhenTheGracePeriodElapses(t *testing.T) {
	deletedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	s := NewScheduledDeprovision("r-1", "tenant-1", "app-1", "user-1", 42, deletedAt, 7)

	if want := deletedAt.Add(7 * 24 * time.Hour); !s.DueAt.Equal(want) {
		t.Fatalf("DueAt = %v, want %v", s.DueAt, want)
	}
	if s.IsDue(s.DueAt.Add(-time.Nanosecond)) {
		t.Error("IsDue(just before DueAt) = true, want false")
	}
	if !s.IsDue(s.DueAt) {
		t.Error("IsDue(DueAt) = false, want true")
	}
	cancelled := *s
	cancelled.Status = ScheduledDeprovisionCancelled
	if cancelled.IsDue(s.DueAt.Add(time.Hour)) {
		t.Error("IsDue() of a cancelled reservation = true, want false")
	}
}

func TestScheduledDeprovision_TaskDeletesTheUserAtTheDeletionVersion(t *testing.T) {
	deletedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	s := NewScheduledDeprovision("r-1", "tenant-1", "app-1", "user-1", 42, deletedAt, 7)
	now := s.DueAt.Add(time.Minute)

	d := s.Task("d-1", now)

	want := ProvisioningTask{
		ID: "d-1", TenantID: "tenant-1", ConnectionID: "app-1", SourceType: SourceTypeUser, SourceID: "user-1",
		SourceVersion: 42, Operation: OperationDelete, Status: TaskPending, CreatedAt: now, UpdatedAt: now,
	}
	if *d != want {
		t.Fatalf("Task() = %+v, want %+v", *d, want)
	}
	if err := d.Validate(); err != nil {
		t.Fatalf("Task().Validate() error = %v", err)
	}
}
