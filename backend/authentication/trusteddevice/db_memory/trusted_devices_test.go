package db_memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
)

func TestListActiveByUserUsesTheProvidedEvaluationTime(t *testing.T) {
	t.Parallel()
	repo := NewTrustedDeviceRepository()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	device, _, err := domain.NewTrustedDevice("tenant-1", "alice", "", 30*24*time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(context.Background(), device); err != nil {
		t.Fatal(err)
	}

	devices, err := repo.ListActiveByUser(context.Background(), "tenant-1", "alice", now)
	if err != nil || len(devices) != 1 {
		t.Fatalf("ListActiveByUser = %v (err %v), want exactly one device", devices, err)
	}
}
