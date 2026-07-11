package postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgfixtures"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestMfaFactorRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &MfaFactorRepository{Pool: db}
	ctx := context.Background()

	now := pgfixtures.TestClock()
	factor := &domain.MfaFactor{
		UserID:    user.ID,
		Type:      spec.MfaFactorTOTP,
		Secret:    new("secret"),
		Label:     new("Authenticator"),
		CreatedAt: now,
	}
	if err := repo.Save(ctx, factor); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Find(ctx, user.ID, spec.MfaFactorTOTP)
	if err != nil || got == nil || got.Secret == nil || *got.Secret != "secret" {
		t.Fatalf("find: %v %+v", err, got)
	}

	list, err := repo.ListBySub(ctx, user.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, user.ID, spec.MfaFactorTOTP); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.Find(ctx, user.ID, spec.MfaFactorTOTP)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}
