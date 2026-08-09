package db_postgres

import (
	"context"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestAuthEventBucketStoreRecordListAndSweep(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	store := &AuthEventBucketStore{Pool: db}
	ctx := context.Background()

	now := pgfixtures.TestClock()
	keyHash := pgfixtures.UniqueID("keyhash")
	first, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, tenant.ID, keyHash, now)
	if err != nil {
		t.Fatalf("record first: %v", err)
	}
	if !first.FirstInWindow || first.Bucket.Count != 1 {
		t.Fatalf("unexpected first record: %+v", first)
	}

	// now (03:04:05) と同じ 5 分窓 (03:00:00〜) に収まる時刻で 2 回目を記録する。
	second, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, tenant.ID, keyHash, now.Add(10*time.Second))
	if err != nil {
		t.Fatalf("record second: %v", err)
	}
	// 同一 5 分窓なので同じ bucket に畳み込まれ、最初の記録ではない。
	if second.FirstInWindow || second.Bucket.Count != 2 {
		t.Fatalf("unexpected second record: %+v", second)
	}

	list, err := store.List(ctx, tenant.ID, time.Time{}, "", 10)
	if err != nil || len(list) != 1 || list[0].Count != 2 {
		t.Fatalf("list: %v %+v", err, list)
	}

	deleted, err := store.DeleteOlderThan(ctx, now.Add(time.Hour))
	if err != nil || deleted != 1 {
		t.Fatalf("delete older than: %v deleted=%d", err, deleted)
	}
	list, err = store.List(ctx, tenant.ID, time.Time{}, "", 10)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected empty after sweep: %v %+v", err, list)
	}
}

func TestAuthEventBucketStoreListKeysetPagination(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	store := &AuthEventBucketStore{Pool: db}
	ctx := context.Background()
	base := pgfixtures.TestClock()

	// 5 distinct windows (20 minutes apart, well outside the 5-minute
	// aggregation window) so each Record call creates its own bucket.
	for i := range 5 {
		if _, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, tenant.ID, pgfixtures.UniqueID("keyhash"), base.Add(time.Duration(i)*20*time.Minute)); err != nil {
			t.Fatalf("record #%d: %v", i, err)
		}
	}

	// window_start DESC: newest bucket (i=4) first.
	first, err := store.List(ctx, tenant.ID, time.Time{}, "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("unexpected first page len: %+v", first)
	}
	if !first[0].WindowStart.After(first[1].WindowStart) {
		t.Fatalf("expected descending window_start: %+v", first)
	}

	last := first[len(first)-1]
	afterKey := string(last.Kind) + "|" + last.KeyHash
	next, err := store.List(ctx, tenant.ID, last.WindowStart, afterKey, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 {
		t.Fatalf("unexpected continuation page len: %+v", next)
	}
	if !next[0].WindowStart.Before(last.WindowStart) {
		t.Fatalf("continuation page must resume strictly after the cursor: last=%v next=%+v", last.WindowStart, next)
	}

	all, err := store.List(ctx, tenant.ID, time.Time{}, "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
