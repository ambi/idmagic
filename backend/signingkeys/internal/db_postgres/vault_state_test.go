package db_postgres

import (
	"context"
	"errors"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	"github.com/ambi/idmagic/backend/signingkeys/internal/keys_vault"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-SIGNINGKEYS-007: Vault の公開状態はコミットされた更新だけを共有し、失敗した更新を残さないこと。
func TestVaultStateStoreCommitsOnlySuccessfulUpdates(t *testing.T) {
	db := pgtest.Require(t)
	now := pgtest.Now()
	tenant := &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := tenancy.NewPostgresModule(db).TenantRepo.Save(context.Background(), tenant); err != nil {
		t.Fatal(err)
	}
	ctx := signingports.WithKeyUsage(tenantports.WithTenant(context.Background(), tenant, "", ""), signingdomain.KeyUsageXMLFederationSigning)
	store := VaultKeyStateStore{Pool: db}
	if !store.Healthy(ctx) {
		t.Fatal("healthy database reported unavailable")
	}
	if state, err := store.Load(ctx); err != nil || len(state.Records) != 0 {
		t.Fatalf("empty state: %v %+v", err, state)
	}
	if err := store.Update(ctx, func(state *keys_vault.KeySetState) error {
		state.Records = []keys_vault.KeyRecord{{Key: signingdomain.SigningKey{Kid: "old", Active: true}, Version: 1}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	failed := errors.New("persistence callback failed")
	if err := store.Update(ctx, func(state *keys_vault.KeySetState) error {
		state.Records[0].Version = 2
		return failed
	}); !errors.Is(err, failed) {
		t.Fatalf("update error: %v", err)
	}
	restarted := VaultKeyStateStore{Pool: db}
	state, err := restarted.Load(ctx)
	if err != nil || len(state.Records) != 1 || state.Records[0].Version != 1 || state.Records[0].Key.Kid != "old" {
		t.Fatalf("uncommitted state escaped: %v %+v", err, state)
	}
	otherScope := signingports.WithKeyScope(ctx, "profile")
	if state, err := restarted.Load(otherScope); err != nil || len(state.Records) != 0 {
		t.Fatal("selection escaped its key scope")
	}
	if err := store.Update(ctx, func(state *keys_vault.KeySetState) error {
		state.Records[0].Version = 2
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if state, err := restarted.Load(ctx); err != nil || state.Records[0].Version != 2 {
		t.Fatal("another reader retained the old state")
	}
}
