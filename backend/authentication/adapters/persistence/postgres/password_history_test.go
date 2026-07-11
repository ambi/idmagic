package postgres

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgfixtures"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

func TestPasswordHistoryRepository(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &PasswordHistoryRepository{Pool: db}
	ctx := context.Background()

	now := pgfixtures.TestClock()
	if err := repo.Add(ctx, user.ID, "enc-1", now); err != nil {
		t.Fatalf("add 1: %v", err)
	}
	if err := repo.Add(ctx, user.ID, "enc-2", now.Add(time.Second)); err != nil {
		t.Fatalf("add 2: %v", err)
	}

	recent, err := repo.Recent(ctx, user.ID, 5)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 2 || recent[0].Encoded != "enc-2" {
		t.Fatalf("recent order/len wrong: %+v", recent)
	}

	if got, _ := repo.Recent(ctx, user.ID, 0); got != nil {
		t.Fatalf("depth 0 should be nil, got %+v", got)
	}
	if got, err := repo.Recent(ctx, user.ID, math.MaxInt); err != nil || len(got) != 2 {
		t.Fatalf("large depth should be capped safely: %v %+v", err, got)
	}

	if err := repo.DeleteAllForSub(ctx, user.ID); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	recent, err = repo.Recent(ctx, user.ID, 5)
	if err != nil || len(recent) != 0 {
		t.Fatalf("expected empty: %v %+v", err, recent)
	}
}
