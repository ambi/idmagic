package db_postgres

import (
	"context"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

func TestSsfStreamRepository_SaveFindDelete(t *testing.T) {
	db := pgtest.Require(t)
	repo := &SsfStreamRepository{Pool: db}
	tenant := seedTenant(t, db)

	s := newStream(t, tenant.ID, ssdomain.SsfStreamDirectionTransmit)
	if err := repo.Save(context.Background(), s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(context.Background(), tenant.ID, s.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil || got.Direction != ssdomain.SsfStreamDirectionTransmit || len(got.EventTypes) != 1 {
		t.Fatalf("FindByID = %+v, want match for %+v", got, s)
	}

	if err := repo.Delete(context.Background(), tenant.ID, s.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	gone, err := repo.FindByID(context.Background(), tenant.ID, s.ID)
	if err != nil {
		t.Fatalf("FindByID after delete: %v", err)
	}
	if gone != nil {
		t.Fatal("expected stream to be gone after Delete")
	}
}

func TestSsfStreamRepository_ListByTenantIsScoped(t *testing.T) {
	db := pgtest.Require(t)
	repo := &SsfStreamRepository{Pool: db}
	tenantA := seedTenant(t, db)
	tenantB := seedTenant(t, db)

	a := newStream(t, tenantA.ID, ssdomain.SsfStreamDirectionReceive)
	if err := repo.Save(context.Background(), a); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	b := newStream(t, tenantB.ID, ssdomain.SsfStreamDirectionReceive)
	if err := repo.Save(context.Background(), b); err != nil {
		t.Fatalf("Save b: %v", err)
	}

	list, err := repo.ListByTenant(context.Background(), tenantA.ID)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("ListByTenant(tenantA) = %+v, want only %q", list, a.ID)
	}
}

// TestSsfReceiverConfigRepository_CascadesOnStreamDelete は
// ssf_receiver_configs.stream_id の ON DELETE CASCADE を検証する。
func TestSsfReceiverConfigRepository_CascadesOnStreamDelete(t *testing.T) {
	db := pgtest.Require(t)
	streamRepo := &SsfStreamRepository{Pool: db}
	configRepo := &SsfReceiverConfigRepository{Pool: db}
	tenant := seedTenant(t, db)

	s := newStream(t, tenant.ID, ssdomain.SsfStreamDirectionReceive)
	if err := streamRepo.Save(context.Background(), s); err != nil {
		t.Fatalf("Save stream: %v", err)
	}
	uri := "https://issuer.example/.well-known/jwks.json"
	cfg := &ssdomain.SsfReceiverConfig{
		StreamID: s.ID, TrustedIssuer: "https://issuer.example", JWKSURI: &uri,
		AcceptedAudiences: []string{"https://idmagic.example/ssf"},
	}
	if err := configRepo.Save(context.Background(), tenant.ID, cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	if err := streamRepo.Delete(context.Background(), tenant.ID, s.ID); err != nil {
		t.Fatalf("Delete stream: %v", err)
	}
	gone, err := configRepo.FindByStream(context.Background(), tenant.ID, s.ID)
	if err != nil {
		t.Fatalf("FindByStream after cascade: %v", err)
	}
	if gone != nil {
		t.Fatalf("expected receiver config to be cascade-deleted with stream, got %+v", gone)
	}
}
