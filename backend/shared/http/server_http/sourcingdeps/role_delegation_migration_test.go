package sourcingdeps_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/sourcing"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/google/uuid"
)

//spec:covers REQ-IDMANAGEMENT-092, REQ-SOURCING-005: 既存 SCIM 対応表のある Group だけを上流管理へ移し、所属を保って未承認ロールを除く。再実行しても同じ結果になる。
func TestSCIMGroupRoleDelegationBackfill(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx := tenantports.WithTenant(t.Context(), tenant, "", "")
	identity := idmanagement.NewPostgresModule(pool)
	source := sourcing.NewPostgresModule(pool)
	user := pgfixtures.SeedUser(t, identity.UserRepo, tenant.ID)
	now := time.Now().UTC()
	var groups []*groupdomain.Group
	for _, name := range []string{"local", "legacy-scim"} {
		g := &groupdomain.Group{ID: uuid.NewString(), TenantID: tenant.ID, Name: name, Roles: []string{"admin", "catalog:read"}, CreatedAt: now, UpdatedAt: now}
		if err := identity.GroupRepo.Save(ctx, g); err != nil {
			t.Fatal(err)
		}
		if _, err := identity.GroupRepo.AddMember(ctx, &groupdomain.GroupMember{GroupID: g.ID, UserID: user.ID, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		groups = append(groups, g)
	}
	if err := source.ScimRepo.SaveGroupRef(ctx, &scimports.ScimGroupRef{TenantID: tenant.ID, ScimID: "external", GroupID: groups[1].ID}); err != nil {
		t.Fatal(err)
	}
	_, file, _, _ := runtime.Caller(0)
	migration, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "infra", "schema", "data-migrations", "2026-10-26-mark-upstream-managed-groups.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		//sql:raw データ移行 SQL のファイルを実際の既存行へ適用する。
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatal(err)
		}
		memberships, err := identity.GroupRepo.ListGroupsByUser(ctx, tenant.ID, user.ID)
		if err != nil || len(memberships) != 2 {
			t.Fatalf("memberships=%+v err=%v", memberships, err)
		}
		for _, g := range memberships {
			if g.ID == groups[1].ID {
				if !g.UpstreamManaged || len(g.Roles) != 0 || g.RoleDelegation != nil {
					t.Fatalf("migrated group=%+v", g)
				}
			} else if g.UpstreamManaged || len(g.Roles) != 2 {
				t.Fatalf("local group changed=%+v", g)
			}
		}
	}
}
