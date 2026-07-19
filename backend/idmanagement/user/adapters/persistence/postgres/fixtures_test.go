package postgres

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancypg "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres"
)

// 本パッケージは pgfixtures が依存する User repository 自身を所有するため、
// pgfixtures を import すると postgres -> pgfixtures -> postgres の import cycle に
// なる。shared/adapters/persistence/postgres 自身の内部テストと同じ理由で、
// 引き続き自前の unexported フィクスチャヘルパーを持つ (wi-178)。

// testClock は決定的なタイムスタンプ生成に用いる基準時刻。
func testClock() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }

var idSeq atomic.Uint64

// uniqueID はテスト間の行衝突を避けるための一意な識別子を生成する。
func uniqueID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, idSeq.Add(1))
}

// newUUID は UUID 列向けの一意な UUID を生成する。
func newUUID(t *testing.T) string {
	t.Helper()
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new uuid: %v", err)
	}
	return id
}

// seedTenant はテナントを作成して返す。FK 親が必要なテストの前提として使う。
func seedTenant(t *testing.T, db sharedpg.DB) *tenancydomain.Tenant {
	t.Helper()
	now := testClock()
	tenant := &tenancydomain.Tenant{
		ID:          newUUID(t),
		Realm:       uniqueID("tenant"),
		DisplayName: "Test Tenant",
		Status:      tenancydomain.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&tenancypg.TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}
