package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgfixtures"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestMfaEnrollmentBypassRepositoryConsumesOnce(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	repo := &MfaEnrollmentBypassRepository{Pool: db}
	now := pgfixtures.TestClock()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	bypass := &domain.MfaEnrollmentBypass{
		ID: id, TenantID: tenant.ID, UserID: user.ID, IssuedBy: user.ID,
		IssuedAt: now, ExpiresAt: now.Add(15 * time.Minute),
	}
	if err := repo.Save(context.Background(), bypass); err != nil {
		t.Fatal(err)
	}
	consumed, err := repo.ConsumeActive(context.Background(), tenant.ID, user.ID, now.Add(time.Second))
	if err != nil || consumed == nil || consumed.ConsumedAt == nil {
		t.Fatalf("consume=%#v err=%v", consumed, err)
	}
	again, err := repo.ConsumeActive(context.Background(), tenant.ID, user.ID, now.Add(2*time.Second))
	if err != nil || again != nil {
		t.Fatalf("second consume=%#v err=%v", again, err)
	}
}
