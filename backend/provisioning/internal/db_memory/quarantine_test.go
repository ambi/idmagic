package db_memory

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
)

func FuzzPendingPaginationPreservesChronology(f *testing.F) {
	f.Add(int64(0), int64(100_000_000), int32(540))
	f.Add(int64(999_999_999), int64(1), int32(-480))
	f.Add(int64(0), int64(0), int32(0))
	f.Fuzz(func(t *testing.T, firstNanos, secondNanos int64, offsetMinutes int32) {
		ctx := t.Context()
		repo := NewProvisioningTaskRepository()
		zone := time.FixedZone("input", int(offsetMinutes%840)*60)
		first := time.Unix(1_700_000_000, int64(uint64(firstNanos)%1_000_000_000)).In(zone)
		second := time.Unix(1_700_000_000, int64(uint64(secondNanos)%1_000_000_000))
		var expected []*domain.ProvisioningTask
		for index, at := range []time.Time{first, second, first.UTC()} {
			id := []string{"a", "b", "c"}[index]
			task := &domain.ProvisioningTask{
				ID: id, TenantID: "tenant", ConnectionID: "connection", SourceType: domain.SourceTypeUser, SourceID: id,
				SourceVersion: 1, Operation: domain.OperationDelete, Status: domain.TaskPending, CreatedAt: at, UpdatedAt: at,
			}
			if _, err := repo.Save(ctx, task); err != nil {
				t.Fatal(err)
			}
			expected = append(expected, task)
		}
		slices.SortFunc(expected, func(a, b *domain.ProvisioningTask) int {
			if comparison := a.CreatedAt.Compare(b.CreatedAt); comparison != 0 {
				return comparison
			}
			return strings.Compare(a.ID, b.ID)
		})
		var afterAt time.Time
		var afterID string
		for _, want := range expected {
			page, err := repo.ListUnenqueued(ctx, afterAt, afterID, 1)
			if err != nil || len(page) != 1 || page[0].ID != want.ID {
				t.Fatalf("page=%v error=%v; want %s in chronological order", page, err, want.ID)
			}
			afterAt, afterID = page[0].CreatedAt, page[0].ID
		}
		if page, err := repo.ListUnenqueued(ctx, afterAt, afterID, 1); err != nil || len(page) != 0 {
			t.Fatalf("page=%v error=%v; want exhausted cursor", page, err)
		}
	})
}
