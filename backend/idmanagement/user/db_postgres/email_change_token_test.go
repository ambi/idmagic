package db_postgres

import (
	"context"
	"testing"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestEmailChangeTokenStoreSaveAndConsume(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	now := testClock()
	user := &userdomain.User{
		ID: newUUID(t), TenantID: tenant.ID, PreferredUsername: "email-change",
		PasswordHash: "hash", CreatedAt: now, UpdatedAt: now,
	}
	if err := (&UserRepository{Pool: db}).Save(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	store := &EmailChangeTokenStore{Pool: db}
	ctx := context.Background()

	record := userports.EmailChangeTokenRecord{
		Sub:       user.ID,
		TokenHash: uniqueID("token"),
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
