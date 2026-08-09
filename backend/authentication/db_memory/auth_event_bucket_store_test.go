package db_memory

import (
	"context"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
)

func TestAuthEventBucketStoreAggregatesWithinWindow(t *testing.T) {
	store := NewAuthEventBucketStore()
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC) // 窓 [00:00, 00:05) 内

	first, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, "acme", "k1", base)
	if err != nil {
		t.Fatal(err)
	}
	if !first.FirstInWindow {
		t.Fatalf("expected first record to be FirstInWindow")
	}
	if first.Bucket.Count != 1 {
		t.Fatalf("expected count 1, got %d", first.Bucket.Count)
	}

	// 同窓・同 key の続く記録は集約され、FirstInWindow=false、count が伸びる。
	for i := 2; i <= 5; i++ {
		res, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, "acme", "k1", base.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if res.FirstInWindow {
			t.Fatalf("record #%d should not be FirstInWindow", i)
		}
		if res.Bucket.Count != i {
			t.Fatalf("record #%d expected count %d, got %d", i, i, res.Bucket.Count)
		}
	}

	buckets, err := store.List(ctx, "acme", time.Time{}, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	if buckets[0].Count != 5 {
		t.Fatalf("expected aggregated count 5, got %d", buckets[0].Count)
	}
	if !buckets[0].WindowStart.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected windowStart %v", buckets[0].WindowStart)
	}
}

func TestAuthEventBucketStoreNewWindowIsSeparate(t *testing.T) {
	store := NewAuthEventBucketStore()
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if _, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, "acme", "k1", base); err != nil {
		t.Fatal(err)
	}
	// 5 分窓の外なら別 bucket・再び FirstInWindow。
	next := base.Add(authnports.AuthEventBucketWindow)
	res, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, "acme", "k1", next)
	if err != nil {
		t.Fatal(err)
	}
	if !res.FirstInWindow {
		t.Fatalf("new window should be FirstInWindow")
	}
	buckets, err := store.List(ctx, "acme", time.Time{}, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
	// 新しい窓が先頭 (windowStart 降順)。
	if !buckets[0].WindowStart.After(buckets[1].WindowStart) {
		t.Fatalf("buckets not ordered by windowStart desc")
	}
}

func TestAuthEventBucketStoreIsolatesTenantsAndKeys(t *testing.T) {
	store := NewAuthEventBucketStore()
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, rec := range []struct {
		tenant, key string
	}{
		{"acme", "k1"}, {"acme", "k1"}, {"acme", "k2"}, {"other", "k1"},
	} {
		if _, err := store.Record(ctx, authnports.AuthEventBucketFailedLogin, rec.tenant, rec.key, base); err != nil {
			t.Fatal(err)
		}
	}

	acme, err := store.List(ctx, "acme", time.Time{}, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(acme) != 2 {
		t.Fatalf("expected 2 acme buckets, got %d", len(acme))
	}
	other, err := store.List(ctx, "other", time.Time{}, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0].Count != 1 {
		t.Fatalf("expected 1 other bucket with count 1, got %+v", other)
	}
}

func TestAuthEventBucketStoreListKeysetPagination(t *testing.T) {
	store := NewAuthEventBucketStore()
	ctx := context.Background()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range 5 {
		if _, err := store.Record(
			ctx, authnports.AuthEventBucketFailedLogin, "acme", "k"+string(rune('0'+i)),
			base.Add(time.Duration(i)*20*time.Minute),
		); err != nil {
			t.Fatalf("record #%d: %v", i, err)
		}
	}

	first, err := store.List(ctx, "acme", time.Time{}, "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || !first[0].WindowStart.After(first[1].WindowStart) {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	afterKey := string(last.Kind) + "|" + last.KeyHash
	next, err := store.List(ctx, "acme", last.WindowStart, afterKey, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 2 || !next[0].WindowStart.Before(last.WindowStart) {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := store.List(ctx, "acme", time.Time{}, "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
