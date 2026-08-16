package db_postgres

import (
	"context"
	"strings"
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func groupAtNameLength(t *testing.T, tenantID string, length int) *groupdomain.Group {
	t.Helper()
	now := testClock()
	return &groupdomain.Group{
		ID: newUUID(t), TenantID: tenantID,
		Name: strings.Repeat("あ", length), Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}
}

// CHECK は最後の防壁であって、通常の入力を止める場所ではない。domain が受ける値は
// データベースも受けなければならない。char_length はコードポイントを数えるので、
// 1 文字 3 バイトの日本語でも上限ちょうどまで保存できる。
func TestGroupRepositoryStoresNamesUpToTheDomainLimit(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	group := groupAtNameLength(t, tenant.ID, spec.LengthName)
	if err := group.Validate(); err != nil {
		t.Fatalf("domain rejected a name at the limit: %v", err)
	}
	if err := (&GroupRepository{Pool: db}).Save(context.Background(), group); err != nil {
		t.Fatalf("database rejected a name the domain accepts: %v", err)
	}
}

// domain を迂回した書き込みは CHECK が止める。上限が実装の外側にも存在することを
// 固定し、片側だけ緩めたときに気づけるようにする。
func TestGroupRepositoryRejectsNamesOverTheDomainLimit(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	group := groupAtNameLength(t, tenant.ID, spec.LengthName+1)
	if err := group.Validate(); err == nil {
		t.Fatal("domain accepted a name over the limit")
	}
	err := (&GroupRepository{Pool: db}).Save(context.Background(), group)
	if err == nil {
		t.Fatal("database accepted a name over the limit")
	}
	if !strings.Contains(err.Error(), "groups_name_length") {
		t.Fatalf("a constraint other than the length check fired: %v", err)
	}
}
