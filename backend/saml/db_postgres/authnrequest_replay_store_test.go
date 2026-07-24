package db_postgres

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

// TestAuthnRequestReplayStore は constraint SAML2Core-BearerAssertion
// (同一 tenant / SP / request ID に対する assertion は一度だけ発行する) を、
// PostgreSQL の実際の unique 制約・expires_at 述語・並行 INSERT に対して検証する。
// Valkey adapter (SETNX + TTL) と memory adapter の振る舞いパリティを取る (ADR-139)。
func TestAuthnRequestReplayStore(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	store := &AuthnRequestReplayStore{Pool: db}
	ctx := context.Background()
	now := time.Now().UTC()

	t.Run("records once, rejects duplicate, re-allows after expiry", func(t *testing.T) {
		if ok, err := store.RecordIfNew(ctx, tenant.ID, "sp", "req-1", time.Minute, now); err != nil || !ok {
			t.Fatalf("first reservation ok=%v err=%v (want true)", ok, err)
		}
		if ok, err := store.RecordIfNew(ctx, tenant.ID, "sp", "req-1", time.Minute, now); err != nil || ok {
			t.Fatalf("duplicate reservation ok=%v err=%v (want false)", ok, err)
		}
		if ok, err := store.RecordIfNew(ctx, tenant.ID, "sp", "req-1", time.Minute, now.Add(2*time.Minute)); err != nil || !ok {
			t.Fatalf("post-expiry reservation ok=%v err=%v (want true)", ok, err)
		}
	})

	t.Run("tenant isolation", func(t *testing.T) {
		if ok, _ := store.RecordIfNew(ctx, tenant.ID, "sp", "req-iso", time.Minute, now); !ok {
			t.Fatal("tenant a reservation failed")
		}
		if ok, err := store.RecordIfNew(ctx, other.ID, "sp", "req-iso", time.Minute, now); err != nil || !ok {
			t.Fatalf("tenant b reservation ok=%v err=%v (want true, isolated)", ok, err)
		}
	})

	t.Run("atomic under concurrency", func(t *testing.T) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		twins := 0
		for range 16 {
			wg.Go(func() {
				if ok, _ := store.RecordIfNew(ctx, tenant.ID, "sp", "req-parallel", time.Minute, now); ok {
					mu.Lock()
					twins++
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		if twins != 1 {
			t.Fatalf("winning reservations=%d, want 1", twins)
		}
	})

	t.Run("DeleteExpiredBatch reclaims expired rows", func(t *testing.T) {
		if _, err := store.RecordIfNew(ctx, tenant.ID, "sp", "req-gc", time.Minute, now); err != nil {
			t.Fatalf("seed reservation: %v", err)
		}
		n, err := store.DeleteExpiredBatch(ctx, now.Add(2*time.Minute), 100)
		if err != nil {
			t.Fatalf("gc: %v", err)
		}
		if n < 1 {
			t.Fatalf("deleted=%d, want >=1", n)
		}
	})
}
