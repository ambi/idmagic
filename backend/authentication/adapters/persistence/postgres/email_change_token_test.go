package postgres

import (
	"context"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgfixtures"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

func TestEmailChangeTokenStoreSaveAndConsume(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	store := &EmailChangeTokenStore{Pool: db}
	ctx := context.Background()

	now := pgfixtures.TestClock()
	record := authnports.EmailChangeTokenRecord{
		Sub:       user.ID,
		TokenHash: pgfixtures.UniqueID("token"),
		NewEmail:  "new@example.com",
		CreatedAt: now,
		ExpiresAt: now.Add(time.Hour),
	}
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("save: %v", err)
	}

	// 期限切れ扱いの時刻では消費できない (nil, nil)。
	expired, err := store.Consume(ctx, record.TokenHash, now.Add(2*time.Hour))
	if err != nil || expired != nil {
		t.Fatalf("expired consume should be nil: %v %+v", err, expired)
	}

	// Save は user 単位で最新 1 本のみを残すため、消費対象を再度保存する。
	if err := store.Save(ctx, record); err != nil {
		t.Fatalf("resave: %v", err)
	}
	got, err := store.Consume(ctx, record.TokenHash, now.Add(time.Minute))
	if err != nil || got == nil || got.NewEmail != "new@example.com" {
		t.Fatalf("consume: %v %+v", err, got)
	}

	// 消費済みトークンは二度と使えない。
	again, err := store.Consume(ctx, record.TokenHash, now.Add(time.Minute))
	if err != nil || again != nil {
		t.Fatalf("second consume should be nil: %v %+v", err, again)
	}
}
