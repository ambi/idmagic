package oauth2_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/oauth2"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
)

//spec:covers REQ-OAUTH2-035: 単独のクライアントの削除の参加者が失敗すると、クライアント、同意、トークン、使用量を元のまま残す。
func TestPostgresStandaloneClientDeletionKeepsStateWhenAParticipantFails(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := tenantports.WithTenantID(t.Context(), tenant.ID)
	now := pgfixtures.TestClock()
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
	quota := tenancy.NewPostgresModule(db).QuotaRepo
	injected := errors.New("binding removal failed")
	module := oauth2.NewPostgresModule(db).WithPostgresClientDeletion(db, oauth2.PostgresClientDeletionParticipants{
		DeleteBindingsInTx: func(context.Context, pgx.Tx, string, string) error { return injected },
		QuotaInTx:          tenancy.QuotaRepositoryInTx,
	})
	client := pgfixtures.SeedClient(t, module.ClientRepo, tenant.ID)
	for _, resource := range []string{tenantdomain.ResourceOAuth2Clients, tenantdomain.ResourceConsents} {
		if err := quota.CheckAndIncrement(ctx, tenant.ID, resource, 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := module.ConsentRepo.Save(ctx, tenant.ID, &oauthdomain.Consent{UserID: user.ID, ClientID: client.ClientID, Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	token := &oauthdomain.RefreshTokenRecord{TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), FamilyID: pgfixtures.NewUUID(t), Hash: pgfixtures.UniqueID("refresh"), ClientID: client.ClientID, UserID: user.ID, Scopes: []string{"openid"}, IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}
	if err := module.RefreshStore.Save(ctx, token); err != nil {
		t.Fatal(err)
	}
	before, err := quota.GetUsage(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := module.StandaloneClientDeletion().DeleteStandalone(ctx, tenant.ID, client.ClientID); !errors.Is(err, injected) {
		t.Fatalf("delete error=%v, want %v", err, injected)
	}

	if stored, err := module.ClientRepo.FindByID(ctx, tenant.ID, client.ClientID); err != nil || stored == nil {
		t.Fatalf("client=%+v error=%v", stored, err)
	}
	if stored, err := module.ConsentRepo.Find(ctx, tenant.ID, user.ID, client.ClientID); err != nil || stored == nil {
		t.Fatalf("consent=%+v error=%v", stored, err)
	}
	if stored, err := module.RefreshStore.FindByHash(ctx, token.Hash); err != nil || stored == nil {
		t.Fatalf("token=%+v error=%v", stored, err)
	}
	if after, err := quota.GetUsage(ctx, tenant.ID); err != nil || *after != *before {
		t.Fatalf("usage before=%+v after=%+v error=%v", before, after, err)
	}
}

//spec:covers REQ-OAUTH2-035, REQ-OAUTH2-080: 確定の中で固定したクライアントがなければ拒否し、組み込みなら何も消さずに拒否する。
func TestPostgresStandaloneClientDeletionRechecksTheLockedClient(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := tenantports.WithTenantID(t.Context(), tenant.ID)
	module := oauth2.NewPostgresModule(db).WithPostgresClientDeletion(db, oauth2.PostgresClientDeletionParticipants{
		DeleteBindingsInTx: idmanagement.DeleteClientBindingsInTx, QuotaInTx: tenancy.QuotaRepositoryInTx,
	})
	deletion := module.StandaloneClientDeletion()
	if err := deletion.DeleteStandalone(ctx, tenant.ID, pgfixtures.NewUUID(t)); !errors.Is(err, clientports.ErrCatalogClientNotFound) {
		t.Fatalf("missing client error=%v", err)
	}
	firstParty := pgfixtures.SeedClient(t, module.ClientRepo, tenant.ID)
	firstParty.FirstParty = true
	if err := module.ClientRepo.Save(ctx, firstParty); err != nil {
		t.Fatal(err)
	}
	if err := deletion.DeleteStandalone(ctx, tenant.ID, firstParty.ClientID); !errors.Is(err, clientports.ErrFirstPartyClientProtected) {
		t.Fatalf("first-party error=%v", err)
	}
	if stored, err := module.ClientRepo.FindByID(ctx, tenant.ID, firstParty.ClientID); err != nil || stored == nil {
		t.Fatalf("first-party client=%+v error=%v", stored, err)
	}
}

func TestStandaloneClientDeletionIsNotWiredByDefault(t *testing.T) {
	if deletion := (oauth2.Module{}).StandaloneClientDeletion(); deletion != nil {
		t.Fatalf("deletion=%T, want nil", deletion)
	}
}
