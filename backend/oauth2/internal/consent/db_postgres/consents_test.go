package db_postgres

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	clientpostgres "github.com/ambi/idmagic/backend/oauth2/internal/client/db_postgres"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestConsentRepositoryListPageByTenant(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &ConsentRepository{Pool: db}
	ctx := context.Background()
	now := pgfixtures.TestClock()

	client := pgfixtures.SeedClient(t, &clientpostgres.OAuth2ClientRepository{Pool: db}, tenant.ID)
	userIDs := make([]string, 0, 5)
	for range 5 {
		u := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db).UserRepo, tenant.ID)
		userIDs = append(userIDs, u.ID)
		if err := repo.Save(ctx, tenant.ID, &domain.Consent{
			UserID: u.ID, ClientID: client.ClientID, Scopes: []string{"read"},
			GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		}); err != nil {
			t.Fatalf("save consent: %v", err)
		}
	}
	slices.SortFunc(userIDs, strings.Compare)

	first, err := repo.ListPage(ctx, tenant.ID, "", "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].UserID != userIDs[0] || first[1].UserID != userIDs[1] {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, tenant.ID, last.UserID, last.ClientID, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].UserID != userIDs[2] || next[1].UserID != userIDs[3] {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPage(ctx, tenant.ID, "", "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}

//spec:covers REQ-OAUTH2-058: 付与の行は同意の問い合わせに混ざらず、撤回と再付与で遷移し、再付与は last_used_at を保つ
//spec:covers REQ-IDMANAGEMENT-106: PostgreSQL でも対象 Agent とテナントの付与だけを返す。
func TestAgentDelegationRowsStayOutOfClientConsents(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	repo := &ConsentRepository{Pool: db}
	ctx := context.Background()
	now := pgfixtures.TestClock()
	module := idmanagement.NewPostgresModule(db)
	user := pgfixtures.SeedUser(t, module.UserRepo, tenant.ID)
	client := pgfixtures.SeedClient(t, &clientpostgres.OAuth2ClientRepository{Pool: db}, tenant.ID)
	agentID := uuid.NewV4().String()
	if err := module.AgentRepo.Save(ctx, &agentdomain.Agent{
		ID: agentID, TenantID: tenant.ID, Name: "bot", Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: user.ID, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if err := repo.Save(ctx, tenant.ID, &domain.Consent{
		UserID: user.ID, ClientID: client.ClientID, Scopes: []string{"read"},
		GrantedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("save consent: %v", err)
	}
	future := time.Now().UTC().Add(time.Hour)
	grant := &domain.Consent{
		UserID: user.ID, AgentID: agentID, Scopes: []string{"read", "write"},
		GrantedAt: time.Now().UTC(), ExpiresAt: future,
	}
	if err := repo.SaveAgentDelegation(ctx, tenant.ID, grant); err != nil {
		t.Fatalf("save grant: %v", err)
	}

	all, err := repo.FindAll(ctx, tenant.ID)
	if err != nil || len(all) != 1 || all[0].ClientID != client.ClientID || all[0].AgentID != "" {
		t.Fatalf("FindAll = (%+v, %v)", all, err)
	}
	page, err := repo.ListPage(ctx, tenant.ID, "", "", 10)
	if err != nil || len(page) != 1 {
		t.Fatalf("ListPage = (%+v, %v)", page, err)
	}

	usedAt := time.Now().UTC().Truncate(time.Second)
	if err := repo.TouchAgentDelegation(ctx, tenant.ID, user.ID, agentID, usedAt); err != nil {
		t.Fatalf("touch: %v", err)
	}
	found, err := repo.FindAgentDelegation(ctx, tenant.ID, user.ID, agentID)
	if err != nil || found == nil || found.State != domain.ConsentGranted || found.ClientID != "" ||
		found.LastUsedAt == nil || !found.LastUsedAt.Equal(usedAt) {
		t.Fatalf("FindAgentDelegation = (%+v, %v)", found, err)
	}
	if err := repo.RevokeAgentDelegation(ctx, tenant.ID, user.ID, agentID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	for _, tenantID := range []string{tenant.ID, pgfixtures.SeedTenant(t, db).ID} {
		rows, err := repo.ListDelegationsForAgent(ctx, tenantID, agentID)
		if err != nil {
			t.Fatal(err)
		}
		if tenantID != tenant.ID {
			if len(rows) != 0 {
				t.Fatalf("foreign tenant received grants=%v", rows)
			}
		} else if len(rows) != 1 || rows[0].UserID != user.ID || rows[0].State != domain.ConsentRevoked {
			t.Fatalf("target grants=%v", rows)
		}
	}
	rows, err := repo.ListDelegationsForAgent(ctx, tenant.ID, uuid.NewV4().String())
	if err != nil || len(rows) != 0 {
		t.Fatalf("other agent grants=%v error=%v", rows, err)
	}
	found, err = repo.FindAgentDelegation(ctx, tenant.ID, user.ID, agentID)
	if err != nil || found == nil || found.State != domain.ConsentRevoked || found.RevokedAt == nil {
		t.Fatalf("after revoke = (%+v, %v)", found, err)
	}
	if err := repo.SaveAgentDelegation(ctx, tenant.ID, grant); err != nil {
		t.Fatalf("re-grant: %v", err)
	}
	listed, err := repo.ListAgentDelegations(ctx, tenant.ID, user.ID)
	if err != nil || len(listed) != 1 || listed[0].State != domain.ConsentGranted || listed[0].LastUsedAt == nil {
		t.Fatalf("after re-grant = (%+v, %v)", listed, err)
	}
	tenantPage, err := repo.ListAgentDelegationPage(ctx, tenant.ID, "", "", 10)
	if err != nil || len(tenantPage) != 1 || tenantPage[0].AgentID != agentID {
		t.Fatalf("ListAgentDelegationPage = (%+v, %v)", tenantPage, err)
	}
	after, err := repo.ListAgentDelegationPage(ctx, tenant.ID, user.ID, agentID, 10)
	if err != nil || len(after) != 0 {
		t.Fatalf("page after the last row = (%+v, %v)", after, err)
	}
	before, err := repo.ListAgentDelegationPageBefore(ctx, tenant.ID, user.ID, agentID, 10)
	if err != nil || len(before) != 0 {
		t.Fatalf("page before the first row = (%+v, %v)", before, err)
	}
	// 付与済みの同意と再付与した代行は、どちらも consents の使用量を占める行として数える。
	if unrevoked, err := repo.DeleteAllForSub(ctx, user.ID); err != nil || unrevoked != 2 {
		t.Fatalf("delete = (%d, %v), want 2 unrevoked rows", unrevoked, err)
	}
	found, err = repo.FindAgentDelegation(ctx, tenant.ID, user.ID, agentID)
	if err != nil || found != nil {
		t.Fatalf("after delete = (%+v, %v)", found, err)
	}
}
