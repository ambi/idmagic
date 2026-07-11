package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/scim/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// testClock は決定的なタイムスタンプ生成に用いる基準時刻。
func testClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

// newUUID は UUID 列向けの一意な UUID を生成する。
func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

// seedTenant / seedUser / seedGroup は pgfixtures を使わず自前で用意する。pgfixtures は
// 本パッケージ (scim/postgres) を import しており、本パッケージの内部テストから pgfixtures を
// 使うと postgres -> pgfixtures -> postgres の import cycle になるため (ADR-090, wi-172 と同じ制約)。

func seedTenant(t *testing.T, db sharedpg.DB) *spec.Tenant {
	t.Helper()
	now := testClock()
	tenant := &spec.Tenant{
		ID:          newUUID(t),
		Realm:       "tenant-" + newUUID(t)[:8],
		DisplayName: "Test Tenant",
		Status:      spec.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&sharedpg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

func seedUser(t *testing.T, db sharedpg.DB, tenantID string) *spec.User {
	t.Helper()
	now := testClock()
	user := &spec.User{
		ID:                newUUID(t),
		TenantID:          tenantID,
		PreferredUsername: "user-" + newUUID(t)[:8],
		PasswordHash:      "hash",
		Roles:             []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := (&sharedpg.UserRepository{Pool: db}).Save(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func seedGroup(t *testing.T, db sharedpg.DB, tenantID string) *spec.Group {
	t.Helper()
	now := testClock()
	group := &spec.Group{
		ID:        newUUID(t),
		TenantID:  tenantID,
		Name:      "group-" + newUUID(t)[:8],
		Roles:     []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := (&sharedpg.GroupRepository{Pool: db}).Save(context.Background(), group); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	return group
}

func TestScimRepositoryTokensAndRefs(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	group := seedGroup(t, db, tenant.ID)
	repo := &ScimRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	token := &ports.ScimToken{
		ID: newUUID(t), TenantID: tenant.ID, TokenHash: "hash-" + newUUID(t)[:8],
		Description: "provisioning", CreatedAt: now, ExpiresAt: new(now.Add(time.Hour)),
	}
	if err := repo.SaveToken(ctx, token); err != nil {
		t.Fatalf("save token: %v", err)
	}
	found, err := repo.FindToken(ctx, token.TokenHash)
	if err != nil || found == nil || found.ID != token.ID {
		t.Fatalf("find token: %v %+v", err, found)
	}
	tokens, err := repo.ListTokens(ctx, tenant.ID)
	if err != nil || len(tokens) != 1 {
		t.Fatalf("list tokens: %v len=%d", err, len(tokens))
	}

	userRef := &ports.ScimUserRef{TenantID: tenant.ID, ScimID: "scim-user-" + newUUID(t)[:8], UserID: user.ID}
	if err := repo.SaveUserRef(ctx, userRef); err != nil {
		t.Fatalf("save user ref: %v", err)
	}
	byScim, err := repo.FindUserRefByScimID(ctx, tenant.ID, userRef.ScimID)
	if err != nil || byScim == nil || byScim.UserID != user.ID {
		t.Fatalf("find user ref by scim id: %v %+v", err, byScim)
	}
	byUser, err := repo.FindUserRefByUserID(ctx, tenant.ID, user.ID)
	if err != nil || byUser == nil || byUser.ScimID != userRef.ScimID {
		t.Fatalf("find user ref by user id: %v %+v", err, byUser)
	}

	groupRef := &ports.ScimGroupRef{TenantID: tenant.ID, ScimID: "scim-group-" + newUUID(t)[:8], GroupID: group.ID}
	if err := repo.SaveGroupRef(ctx, groupRef); err != nil {
		t.Fatalf("save group ref: %v", err)
	}
	gByScim, err := repo.FindGroupRefByScimID(ctx, tenant.ID, groupRef.ScimID)
	if err != nil || gByScim == nil || gByScim.GroupID != group.ID {
		t.Fatalf("find group ref by scim id: %v %+v", err, gByScim)
	}
	gByGroup, err := repo.FindGroupRefByGroupID(ctx, tenant.ID, group.ID)
	if err != nil || gByGroup == nil || gByGroup.ScimID != groupRef.ScimID {
		t.Fatalf("find group ref by group id: %v %+v", err, gByGroup)
	}

	if err := repo.DeleteUserRef(ctx, tenant.ID, userRef.ScimID); err != nil {
		t.Fatalf("delete user ref: %v", err)
	}
	if err := repo.DeleteGroupRef(ctx, tenant.ID, groupRef.ScimID); err != nil {
		t.Fatalf("delete group ref: %v", err)
	}
	if err := repo.DeleteToken(ctx, tenant.ID, token.ID); err != nil {
		t.Fatalf("delete token: %v", err)
	}
}
