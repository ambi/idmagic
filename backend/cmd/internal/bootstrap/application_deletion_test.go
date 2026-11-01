package bootstrap

import (
	"context"
	"testing"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-APPLICATION-018: 本番の PostgreSQL 組み立てが所有者の削除と同一 Tx の使用量を接続する。
func TestAssembledPostgresApplicationDeletion(t *testing.T) {
	db := pgtest.Require(t)
	settings := map[string]string{"PERSISTENCE": "postgres", "DATABASE_URL": db.Config().ConnString(), "KEY_PROVIDER": "local", "ISSUER": "http://idp.test"}
	loader := NewConfigLoader(func(key string) string { return settings[key] })
	config := LoadSharedConfig(loader)
	if err := loader.Err(); err != nil {
		t.Fatal(err)
	}
	deps, err := Assemble(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(deps.Close)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := tenantports.WithTenantID(t.Context(), tenant.ID)
	client := pgfixtures.SeedClient(t, deps.OAuth2.ClientRepo, tenant.ID)
	user := pgfixtures.SeedUser(t, deps.IdManagement.UserRepo, tenant.ID)
	now := pgfixtures.TestClock()
	app := &appdomain.Application{TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), Name: "Assembled deletion", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive, Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: client.ClientID}, CreatedAt: now, UpdatedAt: now}
	if err := deps.Application.Repo.Create(ctx, app); err != nil {
		t.Fatal(err)
	}
	token := &oauthdomain.RefreshTokenRecord{TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), FamilyID: pgfixtures.NewUUID(t), Hash: pgfixtures.UniqueID("refresh"), ClientID: client.ClientID, UserID: user.ID, Scopes: []string{"openid"}, IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}
	if err := deps.OAuth2.RefreshStore.Save(ctx, token); err != nil {
		t.Fatal(err)
	}
	if err := appusecases.DeleteApplication(ctx, appusecases.ApplicationDeps{Repo: deps.Application.Repo, QuotaRepo: deps.Tenancy.QuotaRepo}, user.ID, app.ID, now); err != nil {
		t.Fatal(err)
	}
	if token, err := deps.OAuth2.RefreshStore.FindByHash(ctx, token.Hash); err != nil || token != nil {
		t.Fatalf("token=%+v error=%v", token, err)
	}
}
