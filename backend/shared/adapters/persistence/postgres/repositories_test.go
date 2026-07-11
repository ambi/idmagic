package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestRefreshTokenStoreSaveFindRotate(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	client := seedClient(t, db, tenant.ID)
	store := &RefreshTokenStore{Pool: db}
	ctx := context.Background()

	now := testClock()
	family := newUUID(t)
	rec := &domain.RefreshTokenRecord{
		ID:                newUUID(t),
		TenantID:          tenant.ID,
		Hash:              uniqueID("hash"),
		FamilyID:          family,
		ClientID:          client.ClientID,
		UserID:            user.ID,
		Scopes:            []string{"openid", "offline_access"},
		IssuedAt:          now,
		ExpiresAt:         now.Add(time.Hour),
		AbsoluteExpiresAt: now.AddDate(0, 0, 30),
	}
	if err := store.Save(ctx, rec); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.FindByHash(ctx, rec.Hash)
	if err != nil || got == nil || got.ID != rec.ID {
		t.Fatalf("find by hash: %v %+v", err, got)
	}

	next := &domain.RefreshTokenRecord{
		ID:                newUUID(t),
		TenantID:          tenant.ID,
		Hash:              uniqueID("hash"),
		FamilyID:          family,
		ParentID:          &rec.ID,
		ClientID:          client.ClientID,
		UserID:            user.ID,
		Scopes:            []string{"openid", "offline_access"},
		IssuedAt:          now.Add(time.Minute),
		ExpiresAt:         now.Add(2 * time.Hour),
		AbsoluteExpiresAt: now.AddDate(0, 0, 30),
	}
	rotated, err := store.Rotate(ctx, rec.ID, next)
	if err != nil || rotated == nil {
		t.Fatalf("rotate: %v %+v", err, rotated)
	}

	// 2度目のローテーションは親が rotated 済みのため nil を返す (reuse detection)。
	again, err := store.Rotate(ctx, rec.ID, next)
	if err != nil || again != nil {
		t.Fatalf("second rotate should be nil: %v %+v", err, again)
	}

	if err := store.RevokeFamily(ctx, family); err != nil {
		t.Fatalf("revoke family: %v", err)
	}
	got, err = store.FindByHash(ctx, next.Hash)
	if err != nil || got == nil || !got.Revoked {
		t.Fatalf("expected revoked token: %v %+v", err, got)
	}

	if err := store.DeleteAllForSub(ctx, user.ID); err != nil {
		t.Fatalf("delete all for sub: %v", err)
	}
}

func TestMfaFactorRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	repo := &MfaFactorRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	factor := &spec.MfaFactor{
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

func TestPasswordHistoryRepository(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	repo := &PasswordHistoryRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	if err := repo.Add(ctx, user.ID, "enc-1", now); err != nil {
		t.Fatalf("add 1: %v", err)
	}
	if err := repo.Add(ctx, user.ID, "enc-2", now.Add(time.Second)); err != nil {
		t.Fatalf("add 2: %v", err)
	}

	recent, err := repo.Recent(ctx, user.ID, 5)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 2 || recent[0].Encoded != "enc-2" {
		t.Fatalf("recent order/len wrong: %+v", recent)
	}

	if got, _ := repo.Recent(ctx, user.ID, 0); got != nil {
		t.Fatalf("depth 0 should be nil, got %+v", got)
	}

	if err := repo.DeleteAllForSub(ctx, user.ID); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	recent, err = repo.Recent(ctx, user.ID, 5)
	if err != nil || len(recent) != 0 {
		t.Fatalf("expected empty: %v %+v", err, recent)
	}
}

func TestAgentRepositoryRoundTripAndBindings(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	owner := seedUser(t, db, tenant.ID)
	client := seedClient(t, db, tenant.ID)
	repo := &AgentRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	agent := &spec.Agent{
		ID:          newUUID(t),
		TenantID:    tenant.ID,
		Name:        "svc-agent",
		Kind:        spec.AgentKindAutonomous,
		OwnerUserID: owner.ID,
		Status:      spec.AgentStatusActive,
		Roles:       []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatalf("save agent: %v", err)
	}

	got, err := repo.FindByID(ctx, tenant.ID, agent.ID)
	if err != nil || got == nil || got.OwnerUserID != owner.ID {
		t.Fatalf("find agent: %v %+v", err, got)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list agents: %v len=%d", err, len(list))
	}

	added, err := repo.AddBinding(ctx, &spec.AgentCredentialBinding{
		AgentID: agent.ID, ClientID: client.ClientID, CreatedAt: now,
	})
	if err != nil || !added {
		t.Fatalf("add binding: %v added=%v", err, added)
	}
	// 冪等: 同じ束縛の再追加は false。
	added, err = repo.AddBinding(ctx, &spec.AgentCredentialBinding{
		AgentID: agent.ID, ClientID: client.ClientID, CreatedAt: now,
	})
	if err != nil || added {
		t.Fatalf("duplicate binding should be false: %v added=%v", err, added)
	}

	bindings, err := repo.ListBindings(ctx, tenant.ID, agent.ID)
	if err != nil || len(bindings) != 1 {
		t.Fatalf("list bindings: %v len=%d", err, len(bindings))
	}

	byClient, err := repo.FindByClientID(ctx, tenant.ID, client.ClientID)
	if err != nil || byClient == nil || byClient.ID != agent.ID {
		t.Fatalf("find by client: %v %+v", err, byClient)
	}

	removed, err := repo.RemoveBinding(ctx, tenant.ID, agent.ID, client.ClientID)
	if err != nil || !removed {
		t.Fatalf("remove binding: %v removed=%v", err, removed)
	}

	if err := repo.Delete(ctx, tenant.ID, agent.ID); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
}

func TestGroupRepositoryRoundTripAndMembers(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	repo := &GroupRepository{Pool: db}
	ctx := context.Background()

	group := seedGroup(t, db, tenant.ID)

	got, err := repo.FindByID(ctx, tenant.ID, group.ID)
	if err != nil || got == nil || got.Name != group.Name {
		t.Fatalf("find group: %v %+v", err, got)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list groups: %v len=%d", err, len(list))
	}

	added, err := repo.AddMember(ctx, &spec.GroupMember{
		GroupID: group.ID, UserID: user.ID, CreatedAt: testClock(),
	})
	if err != nil || !added {
		t.Fatalf("add member: %v added=%v", err, added)
	}

	members, err := repo.ListMembersByGroup(ctx, tenant.ID, group.ID)
	if err != nil || len(members) != 1 || members[0].UserID != user.ID {
		t.Fatalf("list members: %v %+v", err, members)
	}

	count, err := repo.CountMembers(ctx, tenant.ID, group.ID)
	if err != nil || count != 1 {
		t.Fatalf("count members: %v count=%d", err, count)
	}

	groups, err := repo.ListGroupsByUser(ctx, tenant.ID, user.ID)
	if err != nil || len(groups) != 1 {
		t.Fatalf("groups by user: %v len=%d", err, len(groups))
	}

	removed, err := repo.RemoveMember(ctx, tenant.ID, group.ID, user.ID)
	if err != nil || !removed {
		t.Fatalf("remove member: %v removed=%v", err, removed)
	}

	if err := repo.Delete(ctx, tenant.ID, group.ID); err != nil {
		t.Fatalf("delete group: %v", err)
	}
}
