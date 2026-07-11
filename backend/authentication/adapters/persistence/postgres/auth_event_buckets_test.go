package postgres

import (
	"context"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgfixtures"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
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

	list, err := store.List(ctx, tenant.ID, 10)
	if err != nil || len(list) != 1 || list[0].Count != 2 {
		t.Fatalf("list: %v %+v", err, list)
	}

	deleted, err := store.DeleteOlderThan(ctx, now.Add(time.Hour))
	if err != nil || deleted != 1 {
		t.Fatalf("delete older than: %v deleted=%d", err, deleted)
	}
	list, err = store.List(ctx, tenant.ID, 10)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected empty after sweep: %v %+v", err, list)
	}
}
