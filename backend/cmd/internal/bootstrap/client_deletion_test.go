package bootstrap

import (
	"context"
	"testing"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-OAUTH2-035, REQ-TENANCY-036: 本番の PostgreSQL 組み立てで、利用された単独のクライアントを同意、トークン、束縛とともに消し、使用量を減らす。
func TestAssembledPostgresStandaloneClientDeletion(t *testing.T) {
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
	now := pgfixtures.TestClock()
	user := pgfixtures.SeedUser(t, deps.IdManagement.UserRepo, tenant.ID)
	client := pgfixtures.SeedClient(t, deps.OAuth2.ClientRepo, tenant.ID)
	quota := deps.Tenancy.QuotaRepo
	if err := quota.CheckAndIncrement(ctx, tenant.ID, tenantdomain.ResourceOAuth2Clients, 1); err != nil {
		t.Fatal(err)
	}
	consent := &oauthdomain.Consent{UserID: user.ID, ClientID: client.ClientID, Scopes: []string{"openid"}, State: oauthdomain.ConsentGranted, GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	if err := deps.OAuth2.ConsentRepo.Save(ctx, tenant.ID, consent); err != nil {
		t.Fatal(err)
	}
	if err := quota.CheckAndIncrement(ctx, tenant.ID, tenantdomain.ResourceConsents, 1); err != nil {
		t.Fatal(err)
	}
	token := &oauthdomain.RefreshTokenRecord{TenantID: tenant.ID, ID: pgfixtures.NewUUID(t), FamilyID: pgfixtures.NewUUID(t), Hash: pgfixtures.UniqueID("refresh"), ClientID: client.ClientID, UserID: user.ID, Scopes: []string{"openid"}, IssuedAt: now, ExpiresAt: now.Add(time.Hour), AbsoluteExpiresAt: now.Add(24 * time.Hour)}
	if err := deps.OAuth2.RefreshStore.Save(ctx, token); err != nil {
		t.Fatal(err)
	}
	agent := &agentdomain.Agent{ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, Name: pgfixtures.UniqueID("agent"), Kind: idmdomain.AgentKindAutonomous, OwnerUserID: user.ID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := deps.IdManagement.AgentRepo.Save(ctx, agent); err != nil {
		t.Fatal(err)
	}
	if _, err := deps.IdManagement.AgentRepo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{AgentID: agent.ID, ClientID: client.ClientID, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	before, err := quota.GetUsage(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := deps.OAuth2.StandaloneClientDeletion().DeleteStandalone(ctx, tenant.ID, client.ClientID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if stored, err := deps.OAuth2.ClientRepo.FindByID(ctx, tenant.ID, client.ClientID); err != nil || stored != nil {
		t.Fatalf("client=%+v error=%v", stored, err)
	}
	if stored, err := deps.OAuth2.ConsentRepo.Find(ctx, tenant.ID, user.ID, client.ClientID); err != nil || stored != nil {
		t.Fatalf("consent=%+v error=%v", stored, err)
	}
	if stored, err := deps.OAuth2.RefreshStore.FindByHash(ctx, token.Hash); err != nil || stored != nil {
		t.Fatalf("token=%+v error=%v", stored, err)
	}
	bindings, err := deps.IdManagement.AgentRepo.ListBindings(ctx, tenant.ID, agent.ID)
	if err != nil || len(bindings) != 0 {
		t.Fatalf("bindings=%+v error=%v", bindings, err)
	}
	if stored, err := deps.IdManagement.AgentRepo.FindByID(ctx, tenant.ID, agent.ID); err != nil || stored == nil {
		t.Fatalf("agent=%+v error=%v", stored, err)
	}
	after, err := quota.GetUsage(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.OAuth2Clients != before.OAuth2Clients-1 || after.Consents != before.Consents-1 {
		t.Fatalf("usage before=%+v after=%+v", before, after)
	}
}
